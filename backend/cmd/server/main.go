package main

import (
	"context"
	"errors"
	"log/slog"
	"math/rand"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"sunyou-bot/backend/internal/bot"
	"sunyou-bot/backend/internal/bot/llm"
	"sunyou-bot/backend/internal/config"
	"sunyou-bot/backend/internal/database"
	"sunyou-bot/backend/internal/handlers"
	"sunyou-bot/backend/internal/hub"
	"sunyou-bot/backend/internal/logger"
	"sunyou-bot/backend/internal/models"
	"sunyou-bot/backend/internal/risk"
	"sunyou-bot/backend/internal/scheduler"
	"sunyou-bot/backend/internal/services"
)

func main() {
	rand.Seed(time.Now().UnixNano())

	cfg, err := config.Load()
	if err != nil {
		panic(err)
	}
	log, closeLog := logger.New(cfg.AppEnv, cfg.LogDir)
	defer closeLog()
	log.Info("logger initialized", "log_dir", cfg.LogDir, "log_file", "app.log")

	db, err := database.NewPostgres(cfg.PostgresDSN)
	if err != nil {
		log.Error("connect postgres failed", "error", err)
		os.Exit(1)
	}
	defer db.Close()
	if err := database.EnsureSchema(context.Background(), db); err != nil {
		log.Error("ensure schema failed", "error", err)
		os.Exit(1)
	}

	redisCli, err := database.NewRedis(cfg.RedisAddr, cfg.RedisPass, cfg.RedisDB)
	if err != nil {
		log.Error("connect redis failed", "error", err)
		os.Exit(1)
	}
	defer redisCli.Close()

	userSvc := services.NewUserService(db)
	pointSvc := services.NewPointService(db)
	reportSvc := services.NewReportService(db, redisCli)
	botAuditSvc := services.NewBotAuditService(db, log)
	riskSvc := risk.NewService(db, redisCli, cfg.SensitiveWords, cfg.RiskMaxMsgPer10s)
	modelPool := buildBotModelConfigs(cfg)

	var llmProvider llm.Provider
	rawDebug := cfg.LLMDebugRawResp && strings.EqualFold(strings.TrimSpace(cfg.AppEnv), "development")
	if cfg.LLMEnabled {
		llmProvider = llm.NewOpenAICompatibleProvider(
			cfg.LLMBaseURL,
			cfg.LLMAPIKey,
			cfg.LLMModel,
			time.Duration(cfg.LLMTimeoutSeconds)*time.Second,
			rawDebug,
		)
		if strings.EqualFold(strings.TrimSpace(cfg.AppEnv), "development") {
			log.Info("llm request endpoint resolved",
				"base_url", cfg.LLMBaseURL,
				"request_path", llm.ChatCompletionsPath(),
				"final_request_url", llm.BuildChatCompletionsURL(cfg.LLMBaseURL),
				"model", cfg.LLMModel,
			)
		}
		log.Info("llm enabled",
			"provider", cfg.LLMProvider,
			"model", cfg.LLMModel,
			"llm_debug_raw_response", rawDebug,
			"debug_force_llm", cfg.BotDebugForceLLM,
			"debug_min_reply", cfg.BotDebugMinReply,
			"max_attempts", cfg.LLMMaxAttempts,
			"total_budget_seconds", cfg.LLMTotalBudgetSec,
			"per_model_timeout_seconds", cfg.LLMPerModelTOsec,
			"pool_probe_timeout_seconds", cfg.LLMProbeTOsec,
			"pool_probe_total_budget_seconds", cfg.LLMProbeBudgetSec,
			"healthy_min_candidates", cfg.LLMHealthyMinCnt,
			"chat_main_pool_size", len(cfg.LLMChatMainPool),
			"premium_fallback_pool_size", len(cfg.LLMPremiumPool),
			"reasoning_pool_size", len(cfg.LLMReasoningPool),
			"code_pool_size", len(cfg.LLMCodePool),
			"specialized_pool_size", len(cfg.LLMSpecializedPool),
			"reasoning_pool_enabled", cfg.LLMReasoningEnabled,
			"specialized_pool_enabled", cfg.LLMSpecializedEnabled,
			"model_pool_size", len(modelPool),
		)
	} else {
		log.Info("llm disabled", "provider", cfg.LLMProvider, "model", cfg.LLMModel, "llm_debug_raw_response", false, "debug_force_llm", cfg.BotDebugForceLLM, "debug_min_reply", cfg.BotDebugMinReply)
	}

	botEngine := bot.NewEngine(redisCli, bot.Options{
		CooldownSeconds:   cfg.RoomBotCooldownSeconds,
		LLMEnabled:        cfg.LLMEnabled,
		LLMModel:          cfg.LLMModel,
		LLMProviderName:   cfg.LLMProvider,
		LLMBaseURL:        cfg.LLMBaseURL,
		LLMTimeoutSeconds: cfg.LLMTimeoutSeconds,
		LLMMaxAttempts:    cfg.LLMMaxAttempts,
		LLMTotalBudgetSec: cfg.LLMTotalBudgetSec,
		LLMPerModelTOsec:  cfg.LLMPerModelTOsec,
		LLMProbeTOsec:     cfg.LLMProbeTOsec,
		LLMProbeBudgetSec: cfg.LLMProbeBudgetSec,
		HealthyMinCount:   cfg.LLMHealthyMinCnt,
		LLMModels:         modelPool,
		LLMDebugRawResp:   rawDebug,
		APIKeyPresent:     modelPoolHasAPIKey(modelPool),
		DebugForceLLM:     cfg.BotDebugForceLLM,
		MinPresence:       cfg.BotMinPresence,
	}, llmProvider, log)
	roomSvc := services.NewRoomService(db, redisCli, botEngine, botAuditSvc, riskSvc, reportSvc, cfg.RoomColdSeconds, cfg.BotDebugMinReply, log)

	var hubMgr *hub.Manager
	hubMgr = hub.NewManager(log,
		func(ctx context.Context, roomID string, c *hub.Client, payload hub.ChatInbound) {
			log.Info("[CHAT] chat.message.received",
				"room_id", roomID,
				"message_id", 0,
				"sender_id", c.UserID,
				"sender_name", c.Nickname,
				"content", shortContent(payload.Content),
				"reply_to_message_id", payload.ReplyToMessageID,
			)
			hubMgr.Broadcast(roomID, "bot_debug", bot.DebugEvent{
				Time:             time.Now().Format(time.RFC3339Nano),
				Event:            "chat.message.received",
				RoomID:           roomID,
				MessageID:        0,
				SenderID:         c.UserID,
				SenderName:       c.Nickname,
				Content:          shortContent(payload.Content),
				ReplyToMessageID: payload.ReplyToMessageID,
			})

			member, err := roomSvc.GetMember(ctx, roomID, c.UserID)
			if err != nil {
				return
			}
			userMsg, botInput, blocked, err := roomSvc.HandleChat(ctx, roomID, services.ChatUser{
				ID:       c.UserID,
				Nickname: c.Nickname,
				Identity: member.Identity,
			}, services.ChatPayload{
				Content:           payload.Content,
				ClientMsgID:       payload.ClientMsgID,
				ReplyToMessageID:  payload.ReplyToMessageID,
				ReplyToSenderName: payload.ReplyToSenderName,
				ReplyToPreview:    payload.ReplyToPreview,
			})
			if blocked != "" {
				hubMgr.Broadcast(roomID, "system", map[string]any{"content": blocked, "userId": c.UserID})
				hubMgr.Broadcast(roomID, "bot_debug", bot.DebugEvent{
					Time:             time.Now().Format(time.RFC3339Nano),
					Event:            "bot.trigger.skip",
					RoomID:           roomID,
					SenderID:         c.UserID,
					SenderName:       c.Nickname,
					Content:          shortContent(payload.Content),
					ReplyToMessageID: payload.ReplyToMessageID,
					SkipReason:       "not_triggered",
				})
				return
			}
			if err != nil {
				if errors.Is(err, services.ErrRoomEnded) {
					report, _ := roomSvc.GetOrBuildReport(context.Background(), roomID)
					hubMgr.Broadcast(roomID, "room_end", map[string]any{"roomId": roomID, "report": report})
				}
				return
			}
			if userMsg != nil {
				hubMgr.Broadcast(roomID, "bot_debug", bot.DebugEvent{
					Time:             time.Now().Format(time.RFC3339Nano),
					Event:            "chat.message.saved",
					RoomID:           roomID,
					MessageID:        userMsg.ID,
					SenderID:         c.UserID,
					SenderName:       c.Nickname,
					Content:          shortContent(userMsg.Content),
					ReplyToMessageID: userMsg.ReplyToMessageID,
				})
				log.Info("[CHAT] chat.message.broadcasted",
					"room_id", roomID,
					"message_id", userMsg.ID,
					"sender_id", c.UserID,
					"sender_name", c.Nickname,
					"content", shortContent(userMsg.Content),
					"reply_to_message_id", userMsg.ReplyToMessageID,
				)
				hubMgr.Broadcast(roomID, "chat", userMsg)
				hubMgr.Broadcast(roomID, "bot_debug", bot.DebugEvent{
					Time:             time.Now().Format(time.RFC3339Nano),
					Event:            "chat.message.broadcasted",
					RoomID:           roomID,
					MessageID:        userMsg.ID,
					SenderID:         c.UserID,
					SenderName:       c.Nickname,
					Content:          shortContent(userMsg.Content),
					ReplyToMessageID: userMsg.ReplyToMessageID,
				})
			}
			if userMsg != nil && botInput != nil {
				savedUser := *userMsg
				input := *botInput
				go func(roomID string, userMessage models.ChatMessage, in bot.Input) {
					timeout := time.Duration(cfg.LLMTimeoutSeconds+8) * time.Second
					if timeout < 8*time.Second {
						timeout = 8 * time.Second
					}
					botCtx, cancel := context.WithTimeout(context.Background(), timeout)
					defer cancel()

					botMsg, botErr := roomSvc.HandleBotReply(botCtx, roomID, &userMessage, in)
					if botErr != nil {
						if errors.Is(botErr, services.ErrRoomEnded) {
							report, _ := roomSvc.GetOrBuildReport(context.Background(), roomID)
							hubMgr.Broadcast(roomID, "room_end", map[string]any{"roomId": roomID, "report": report})
						}
						return
					}
					if botMsg == nil {
						return
					}
					hubMgr.Broadcast(roomID, "bot_debug", bot.DebugEvent{
						Time:             time.Now().Format(time.RFC3339Nano),
						Event:            "chat.message.saved",
						RoomID:           roomID,
						MessageID:        botMsg.ID,
						SenderName:       botMsg.Nickname,
						Content:          shortContent(botMsg.Content),
						ReplyToMessageID: botMsg.ReplyToMessageID,
					})
					log.Info("[CHAT] chat.message.broadcasted",
						"room_id", roomID,
						"message_id", botMsg.ID,
						"sender_id", "",
						"sender_name", botMsg.Nickname,
						"content", shortContent(botMsg.Content),
						"reply_to_message_id", botMsg.ReplyToMessageID,
					)
					hubMgr.Broadcast(roomID, "chat", botMsg)
					hubMgr.Broadcast(roomID, "bot_debug", bot.DebugEvent{
						Time:             time.Now().Format(time.RFC3339Nano),
						Event:            "chat.message.broadcasted",
						RoomID:           roomID,
						MessageID:        botMsg.ID,
						SenderName:       botMsg.Nickname,
						Content:          shortContent(botMsg.Content),
						ReplyToMessageID: botMsg.ReplyToMessageID,
					})
				}(roomID, savedUser, input)
			}
		},
		func(ctx context.Context, roomID string, c *hub.Client) {
			roomSvc.MarkOnline(ctx, roomID, c.UserID)
			members, _ := roomSvc.ListMembers(ctx, roomID)
			hubMgr.Broadcast(roomID, "member_update", map[string]any{"members": members})
		},
		func(ctx context.Context, roomID string, c *hub.Client) {
			roomSvc.MarkOffline(ctx, roomID, c.UserID)
			members, _ := roomSvc.ListMembers(ctx, roomID)
			hubMgr.Broadcast(roomID, "member_update", map[string]any{"members": members})
		},
	)
	botEngine.SetDebugObserver(func(evt bot.DebugEvent) {
		if strings.TrimSpace(evt.RoomID) == "" {
			return
		}
		hubMgr.Broadcast(evt.RoomID, "bot_debug", evt)
		if err := botAuditSvc.CaptureDebugEvent(context.Background(), evt); err != nil {
			log.Error("persist bot audit failed", "trace_id", evt.TraceID, "event", evt.Event, "error", err)
		}
	})

	h := handlers.New(log, db, redisCli, userSvc, pointSvc, roomSvc, botAuditSvc, hubMgr, cfg.AllowedOrigin)
	router := h.Router()

	expirer := scheduler.NewRoomExpirer(log, roomSvc)
	expirer.Start()

	srv := &http.Server{
		Addr:         cfg.Host + ":" + cfg.Port,
		Handler:      router,
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
	}

	go func() {
		log.Info("backend started", "addr", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Error("server stopped unexpectedly", "error", err)
		}
	}()

	waitForShutdown(log, srv, cfg.GracefulTimeout, expirer)
}

func buildBotModelConfigs(cfg *config.Config) []bot.ModelConfig {
	out := make([]bot.ModelConfig, 0, 20)

	if len(cfg.LLMModels) > 0 {
		for _, item := range cfg.LLMModels {
			apiKey := strings.TrimSpace(cfg.LLMAPIKey)
			if envName := strings.TrimSpace(item.APIKeyEnv); envName != "" {
				if v := strings.TrimSpace(os.Getenv(envName)); v != "" {
					apiKey = v
				}
			}
			out = append(out, bot.ModelConfig{
				Name:           strings.TrimSpace(item.Name),
				Provider:       strings.TrimSpace(item.Provider),
				BaseURL:        strings.TrimSpace(item.BaseURL),
				APIKey:         apiKey,
				Enabled:        item.Enabled,
				Priority:       item.Priority,
				Pool:           strings.TrimSpace(item.Pool),
				TimeoutSeconds: item.TimeoutSeconds,
			})
		}
		return out
	}

	build := func(poolName string, models []string) {
		for idx, m := range models {
			name := strings.TrimSpace(m)
			if name == "" {
				continue
			}
			out = append(out, bot.ModelConfig{
				Name:           name,
				Provider:       cfg.LLMProvider,
				BaseURL:        cfg.LLMBaseURL,
				APIKey:         cfg.LLMAPIKey,
				Enabled:        true,
				Priority:       idx + 1,
				Pool:           poolName,
				TimeoutSeconds: cfg.LLMPerModelTOsec,
			})
		}
	}
	build("chat_main_pool", cfg.LLMChatMainPool)
	build("premium_fallback_pool", cfg.LLMPremiumPool)
	build("code_pool", cfg.LLMCodePool)
	if cfg.LLMReasoningEnabled {
		build("reasoning_pool", cfg.LLMReasoningPool)
	}
	if cfg.LLMSpecializedEnabled {
		build("specialized_do_not_use_for_chat", cfg.LLMSpecializedPool)
	}
	return out
}

func modelPoolHasAPIKey(models []bot.ModelConfig) bool {
	for _, item := range models {
		if strings.TrimSpace(item.APIKey) != "" {
			return true
		}
	}
	return false
}

func waitForShutdown(log *slog.Logger, srv *http.Server, timeout time.Duration, expirer *scheduler.RoomExpirer) {
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGTERM, syscall.SIGINT)
	<-quit

	expirer.Stop()
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Error("graceful shutdown failed", "error", err)
	}
	log.Info("server exited")
}

func shortContent(content string) string {
	content = strings.TrimSpace(content)
	r := []rune(content)
	if len(r) > 32 {
		return string(r[:32]) + "..."
	}
	return content
}
