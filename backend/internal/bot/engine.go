package bot

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"math/rand"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	redis "github.com/redis/go-redis/v9"

	"sunyou-bot/backend/internal/bot/character"
	"sunyou-bot/backend/internal/bot/llm"
	"sunyou-bot/backend/internal/bot/prompt"
	"sunyou-bot/backend/internal/bot/scorer"
	"sunyou-bot/backend/internal/models"
)

const (
	ReplySourceLLM      = "llm"
	ReplySourceTemplate = "template"
)

type Options struct {
	CooldownSeconds   int
	LLMEnabled        bool
	LLMModel          string
	LLMProviderName   string
	LLMBaseURL        string
	LLMTimeoutSeconds int
	LLMMaxAttempts    int
	LLMTotalBudgetSec int
	LLMPerModelTOsec  int
	LLMProbeTOsec     int
	LLMProbeBudgetSec int
	HealthyMinCount   int
	LLMModels         []ModelConfig
	LLMDebugRawResp   bool
	APIKeyPresent     bool
	DebugForceLLM     bool
	MinPresence       bool
}

type ModelConfig struct {
	Name           string
	Provider       string
	BaseURL        string
	APIKey         string
	Enabled        bool
	Priority       int
	Pool           string
	TimeoutSeconds int
}

type Input struct {
	RoomID           string
	SpeakerID        string
	SpeakerName      string
	SpeakerIdentity  models.Identity
	SpeakerMessageID int64
	ReplyToMessageID *int64
	Content          string
	Role             models.BotRole
	FireLevel        models.FireLevel
	IsColdScene      bool
	RecentMsgCount   int
	ConsecutiveHitID string
	ReplyToMessage   *models.ChatMessage
	RecentMessages   []models.ChatMessage
	TargetMode       bool
	ImmuneMode       bool
}

type Output struct {
	Triggered            bool
	Content              string
	TargetID             string
	ForceReply           bool
	UsedLLM              bool
	Score                int
	Atmosphere           string
	Reason               string
	TraceID              string
	ReplySource          string
	LLMModel             string
	FallbackReason       string
	TriggerReason        string
	TriggerType          string
	HypeScore            int
	AbsurdityScore       int
	RiskScore            int
	ReplyMode            string
	ModelPool            string
	CandidateModels      []string
	TriedModels          []string
	SelectedModel        string
	ModelFailures        []string
	SkippedModels        []string
	LastErrorType        string
	CircuitOpenUntil     string
	BotReplySkipped      bool
	LLMAttempted         bool
	InputLanguage        string
	OutputLanguage       string
	LanguageMismatch     bool
	RejectReason         string
	SkipReason           string
	Provider             string
	LLMEnabled           bool
	ProviderInitialized  bool
	APIKeyPresent        bool
	HTTPStatus           int
	LatencyMs            int
	PromptTokens         int
	CompletionTokens     int
	TotalTokens          int
	RequestPromptExcerpt string
	ErrorMessage         string
	DisplayableFound     bool
	ReasoningOnly        bool
}

type DebugStatus struct {
	LLMEnabled           bool       `json:"llm_enabled"`
	ProviderInitialized  bool       `json:"provider_initialized"`
	ProviderName         string     `json:"provider_name"`
	BaseURL              string     `json:"base_url"`
	FinalRequestURL      string     `json:"final_request_url"`
	Model                string     `json:"model"`
	ModelPoolChatMain    []string   `json:"model_pool_chat_main"`
	ModelPoolPremium     []string   `json:"model_pool_premium_fallback"`
	ModelPoolReasoning   []string   `json:"model_pool_reasoning"`
	ModelPoolCode        []string   `json:"model_pool_code"`
	ModelPoolSpecialized []string   `json:"model_pool_specialized_do_not_use_for_chat"`
	MaxAttempts          int        `json:"max_attempts"`
	TotalBudgetSeconds   int        `json:"total_budget_seconds"`
	PerModelTimeoutSec   int        `json:"per_model_timeout_seconds"`
	APIKeyPresent        bool       `json:"api_key_present"`
	TimeoutSeconds       int        `json:"timeout_seconds"`
	LLMDebugRawResponse  bool       `json:"llm_debug_raw_response"`
	DebugForceLLM        bool       `json:"debug_force_llm"`
	MinPresence          bool       `json:"min_presence"`
	LastLLMSuccessAt     *time.Time `json:"last_llm_success_at,omitempty"`
	LastLLMError         string     `json:"last_llm_error"`
	LastFallbackReason   string     `json:"last_fallback_reason"`
}

type DebugEvent struct {
	Time                     string `json:"time"`
	Event                    string `json:"event"`
	RoomID                   string `json:"room_id,omitempty"`
	MessageID                int64  `json:"message_id,omitempty"`
	SenderID                 string `json:"sender_id,omitempty"`
	SenderName               string `json:"sender_name,omitempty"`
	Content                  string `json:"content,omitempty"`
	ReplyToMessageID         *int64 `json:"reply_to_message_id,omitempty"`
	ReplyToIsBot             bool   `json:"reply_to_is_bot"`
	BotRole                  string `json:"bot_role,omitempty"`
	FirepowerLevel           string `json:"firepower_level,omitempty"`
	TriggerReason            string `json:"trigger_reason,omitempty"`
	TriggerType              string `json:"trigger_type,omitempty"`
	SkipReason               string `json:"skip_reason,omitempty"`
	BotReplySkipped          bool   `json:"bot_reply_skipped"`
	ForceReply               bool   `json:"force_reply"`
	HypeScore                int    `json:"hype_score"`
	AbsurdityScore           int    `json:"absurdity_score"`
	RiskScore                int    `json:"risk_score"`
	ReplyMode                string `json:"reply_mode,omitempty"`
	ModelPool                string `json:"model_pool,omitempty"`
	CandidateModels          string `json:"candidate_models,omitempty"`
	TriedModels              string `json:"tried_models,omitempty"`
	SelectedModel            string `json:"selected_model,omitempty"`
	ModelFailures            string `json:"model_failures,omitempty"`
	SkippedModels            string `json:"skipped_models,omitempty"`
	LastErrorType            string `json:"last_error_type,omitempty"`
	CircuitScope             string `json:"circuit_scope,omitempty"`
	CircuitKey               string `json:"circuit_key,omitempty"`
	ProviderCircuitKey       string `json:"provider_circuit_key,omitempty"`
	CircuitOpenUntil         string `json:"circuit_open_until,omitempty"`
	ModelCircuitOpenUntil    string `json:"model_circuit_open_until,omitempty"`
	ProviderCircuitOpenUntil string `json:"provider_circuit_open_until,omitempty"`
	LLMAttempted             bool   `json:"llm_attempted"`
	InputLanguage            string `json:"input_language,omitempty"`
	OutputLanguage           string `json:"output_language,omitempty"`
	LanguageMismatch         bool   `json:"language_mismatch"`
	RejectReason             string `json:"reject_reason,omitempty"`
	ReplySource              string `json:"reply_source,omitempty"`
	FallbackReason           string `json:"fallback_reason,omitempty"`
	UsedLLM                  bool   `json:"used_llm"`
	LLMEnabled               bool   `json:"llm_enabled"`
	ProviderInitialized      bool   `json:"provider_initialized"`
	TraceID                  string `json:"trace_id,omitempty"`
	Model                    string `json:"model,omitempty"`
	Provider                 string `json:"provider,omitempty"`
	BaseURL                  string `json:"base_url,omitempty"`
	APIKeyPresent            bool   `json:"api_key_present"`
	RequestPromptExcerpt     string `json:"request_prompt_excerpt,omitempty"`
	ResponseText             string `json:"response_text,omitempty"`
	HTTPStatus               int    `json:"http_status,omitempty"`
	LatencyMs                int    `json:"latency_ms,omitempty"`
	PromptTokens             int    `json:"prompt_tokens,omitempty"`
	CompletionTokens         int    `json:"completion_tokens,omitempty"`
	TotalTokens              int    `json:"total_tokens,omitempty"`
	ErrorMessage             string `json:"error_message,omitempty"`
	RawResponsePreview       string `json:"raw_response_preview,omitempty"`
	ChoiceSummary            string `json:"choice_summary,omitempty"`
	ChoicesCount             int    `json:"choices_count,omitempty"`
	ContentPreview           string `json:"content_preview,omitempty"`
	DisplayableFound         bool   `json:"displayable_content_found"`
	ReasoningOnly            bool   `json:"reasoning_only_response"`
}

type DebugObserver func(DebugEvent)

type generateMeta struct {
	Provider                 string
	BaseURL                  string
	Model                    string
	ModelPool                string
	CandidateModels          []string
	TriedModels              []string
	SelectedModel            string
	ModelFailures            []string
	SkippedModels            []string
	LastErrorType            string
	CircuitScope             string
	CircuitKey               string
	ProviderCircuitKey       string
	CircuitOpenUntil         string
	ModelCircuitOpenUntil    string
	ProviderCircuitOpenUntil string
	BotReplySkipped          bool
	LLMAttempted             bool
	InputLanguage            string
	OutputLanguage           string
	LanguageMismatch         bool
	RejectReason             string
	RequestPromptExcerpt     string
	HTTPStatus               int
	LatencyMs                int
	PromptTokens             int
	CompletionTokens         int
	TotalTokens              int
	ErrorMessage             string
	RawResponsePreview       string
	RawResponseHeaders       map[string]string
	ChoicesCount             int
	ChoiceSummary            string
	ContentPreview           string
	DisplayableFound         bool
	ReasoningOnly            bool
}

type Engine struct {
	redis           *redis.Client
	cooldownSeconds int
	llmEnabled      bool
	llmProvider     llm.Provider
	llmModel        string
	llmProviderName string
	llmBaseURL      string
	llmTimeoutSec   int
	llmMaxAttempts  int
	llmTotalBudget  int
	llmPerModelTO   int
	llmProbeTO      int
	llmProbeBudget  int
	healthyMinCount int
	llmModels       []ModelConfig
	llmDebugRawResp bool
	apiKeyPresent   bool
	debugForceLLM   bool
	minPresence     bool
	logger          *slog.Logger
	mu              sync.Mutex
	rand            *rand.Rand

	statusMu           sync.Mutex
	lastLLMSuccessAt   *time.Time
	lastLLMError       string
	lastFallbackReason string
	observerMu         sync.RWMutex
	debugObserver      DebugObserver
	healthyMu          sync.RWMutex
	healthyCandidates  map[string][]string
	probeMu            sync.Mutex
	probeRunning       map[string]bool
}

func NewEngine(redisCli *redis.Client, opts Options, provider llm.Provider, logger *slog.Logger) *Engine {
	cool := opts.CooldownSeconds
	if cool <= 0 {
		cool = 6
	}
	models := normalizeModelConfigs(opts)
	maxAttempts := opts.LLMMaxAttempts
	if maxAttempts <= 0 {
		maxAttempts = 3
	}
	totalBudget := opts.LLMTotalBudgetSec
	if totalBudget <= 0 {
		totalBudget = 30
	}
	perModelTO := opts.LLMPerModelTOsec
	if perModelTO <= 0 {
		perModelTO = 10
	}
	probeTO := opts.LLMProbeTOsec
	if probeTO <= 0 {
		probeTO = 4
	}
	probeBudget := opts.LLMProbeBudgetSec
	if probeBudget <= 0 {
		probeBudget = 60
	}
	healthyMin := opts.HealthyMinCount
	if healthyMin <= 0 {
		healthyMin = 2
	}
	eng := &Engine{
		redis:             redisCli,
		cooldownSeconds:   cool,
		llmEnabled:        opts.LLMEnabled,
		llmProvider:       provider,
		llmModel:          opts.LLMModel,
		llmProviderName:   opts.LLMProviderName,
		llmBaseURL:        opts.LLMBaseURL,
		llmTimeoutSec:     opts.LLMTimeoutSeconds,
		llmMaxAttempts:    maxAttempts,
		llmTotalBudget:    totalBudget,
		llmPerModelTO:     perModelTO,
		llmProbeTO:        probeTO,
		llmProbeBudget:    probeBudget,
		healthyMinCount:   healthyMin,
		llmModels:         models,
		llmDebugRawResp:   opts.LLMDebugRawResp,
		apiKeyPresent:     opts.APIKeyPresent,
		debugForceLLM:     opts.DebugForceLLM,
		minPresence:       opts.MinPresence,
		logger:            logger,
		rand:              rand.New(rand.NewSource(time.Now().UnixNano())),
		healthyCandidates: map[string][]string{},
		probeRunning:      map[string]bool{},
	}
	if eng.llmEnabled {
		go eng.schedulePoolProbe(poolChatMain, "startup_init")
	}
	return eng
}

func (e *Engine) DebugStatus() DebugStatus {
	e.statusMu.Lock()
	defer e.statusMu.Unlock()
	var t *time.Time
	if e.lastLLMSuccessAt != nil {
		cp := *e.lastLLMSuccessAt
		t = &cp
	}
	chatPool, premiumPool, reasoningPool, codePool, specializedPool := e.listPools()
	return DebugStatus{
		LLMEnabled:           e.llmEnabled,
		ProviderInitialized:  e.providerReady(),
		ProviderName:         e.llmProviderName,
		BaseURL:              e.llmBaseURL,
		FinalRequestURL:      llm.BuildChatCompletionsURL(e.llmBaseURL),
		Model:                e.llmModel,
		ModelPoolChatMain:    chatPool,
		ModelPoolPremium:     premiumPool,
		ModelPoolReasoning:   reasoningPool,
		ModelPoolCode:        codePool,
		ModelPoolSpecialized: specializedPool,
		MaxAttempts:          e.llmMaxAttempts,
		TotalBudgetSeconds:   e.llmTotalBudget,
		PerModelTimeoutSec:   e.llmPerModelTO,
		APIKeyPresent:        e.apiKeyPresent,
		TimeoutSeconds:       e.llmTimeoutSec,
		LLMDebugRawResponse:  e.llmDebugRawResp,
		DebugForceLLM:        e.debugForceLLM,
		MinPresence:          e.minPresence,
		LastLLMSuccessAt:     t,
		LastLLMError:         e.lastLLMError,
		LastFallbackReason:   e.lastFallbackReason,
	}
}

func (e *Engine) SetDebugObserver(observer DebugObserver) {
	e.observerMu.Lock()
	defer e.observerMu.Unlock()
	e.debugObserver = observer
}

func (e *Engine) MaybeReply(ctx context.Context, in Input) Output {
	in.Role = character.NormalizeRole(in.Role)
	roleSpec := character.Get(in.Role)
	traceID := uuid.NewString()
	providerInitialized := e.providerReady()
	replyToIsBot := in.ReplyToMessage != nil && in.ReplyToMessage.IsBotMessage
	e.logForceReplyCheck(in, replyToIsBot)
	if replyToIsBot {
		e.logForceReplyHit(in)
	} else {
		e.logForceReplyMiss(in)
	}
	e.logTriggerEnter(in, traceID, "user_message")

	eval := scorer.Evaluate(scorer.Input{
		SpeakerID:        in.SpeakerID,
		SpeakerIdentity:  in.SpeakerIdentity,
		Content:          in.Content,
		FireLevel:        in.FireLevel,
		IsColdScene:      in.IsColdScene,
		RecentMsgCount:   in.RecentMsgCount,
		ConsecutiveHitID: in.ConsecutiveHitID,
		ReplyToMessage:   in.ReplyToMessage,
		RecentMessages:   in.RecentMessages,
		MinPresence:      e.minPresence,
	})
	eval = character.ApplyTriggerPolicy(roleSpec, eval, character.TriggerContext{
		Content:         in.Content,
		ForceReply:      replyToIsBot,
		SpeakerIdentity: in.SpeakerIdentity,
	})

	triggerReason := resolveTriggerReason(in, eval)
	triggerType := resolveTriggerType(in, triggerReason)
	out := Output{
		Score:               eval.Score,
		Atmosphere:          eval.Atmosphere,
		ForceReply:          eval.ForceReply,
		Reason:              strings.Join(eval.Tags, ","),
		TraceID:             traceID,
		TriggerReason:       triggerReason,
		TriggerType:         triggerType,
		HypeScore:           eval.Score,
		AbsurdityScore:      eval.AbsurdityScore,
		RiskScore:           eval.RiskScore,
		ReplyMode:           eval.ReplyMode,
		ModelPool:           "",
		LLMEnabled:          e.llmEnabled,
		ProviderInitialized: providerInitialized,
		APIKeyPresent:       e.apiKeyPresent,
	}

	if !eval.ForceReply {
		if in.SpeakerIdentity == models.IdentityImmune && eval.Score < eval.Threshold+10 {
			out.SkipReason = "score_too_low"
			e.logTriggerSkip(in, traceID, triggerReason, triggerType, eval.ForceReply, eval.Score, eval.AbsurdityScore, eval.RiskScore, eval.ReplyMode, out.SkipReason)
			return out
		}
		if muted, _ := e.redis.Get(ctx, roomMuteKey(in.RoomID)).Result(); muted == "1" {
			out.SkipReason = "bot_muted"
			e.logTriggerSkip(in, traceID, triggerReason, triggerType, eval.ForceReply, eval.Score, eval.AbsurdityScore, eval.RiskScore, eval.ReplyMode, out.SkipReason)
			return out
		}
		if !eval.ShouldReply {
			out.SkipReason = "score_too_low"
			e.logTriggerSkip(in, traceID, triggerReason, triggerType, eval.ForceReply, eval.Score, eval.AbsurdityScore, eval.RiskScore, eval.ReplyMode, out.SkipReason)
			return out
		}
		if ok, _ := e.acquireCooldown(ctx, in.RoomID); !ok && !(e.minPresence && eval.PresenceBoost) {
			out.SkipReason = "cooldown_active"
			e.logTriggerSkip(in, traceID, triggerReason, triggerType, eval.ForceReply, eval.Score, eval.AbsurdityScore, eval.RiskScore, eval.ReplyMode, out.SkipReason)
			return out
		}
	}

	e.logTriggerHit(in, traceID, triggerReason, triggerType, eval.ForceReply, eval.Score, eval.AbsurdityScore, eval.RiskScore, eval.ReplyMode)

	// In normal mode, anti-bullying branch can force template-only line.
	if !e.debugForceLLM && !eval.ForceReply && in.ConsecutiveHitID == in.SpeakerID && in.SpeakerIdentity != models.IdentityTarget && e.roll(100) < 25 {
		out.Triggered = true
		out.Content = e.pickNeutralNarration(in.Role)
		out.TargetID = ""
		out.ReplySource = ReplySourceTemplate
		out.LLMModel = ""
		out.FallbackReason = "rule_template_only"
		out.Provider = e.llmProviderName
		e.recordFallback(out.FallbackReason, "")
		e.logReplyDone(in, out)
		return out
	}

	line, usedLLM, fallbackReason, meta := e.generate(ctx, in, roleSpec, eval.Atmosphere, traceID, triggerReason, triggerType, eval.ForceReply, eval.AbsurdityScore, eval.RiskScore, eval.ReplyMode)
	if strings.TrimSpace(line) == "" {
		out.SkipReason = "bot_reply_skipped"
		if strings.TrimSpace(fallbackReason) != "" {
			out.SkipReason = "bot_reply_skipped:" + strings.TrimSpace(fallbackReason)
		}
		out.FallbackReason = fallbackReason
		out.Provider = meta.Provider
		out.HTTPStatus = meta.HTTPStatus
		out.LatencyMs = meta.LatencyMs
		out.PromptTokens = meta.PromptTokens
		out.CompletionTokens = meta.CompletionTokens
		out.TotalTokens = meta.TotalTokens
		out.RequestPromptExcerpt = meta.RequestPromptExcerpt
		out.ErrorMessage = meta.ErrorMessage
		out.DisplayableFound = meta.DisplayableFound
		out.ReasoningOnly = meta.ReasoningOnly
		out.ModelPool = meta.ModelPool
		out.CandidateModels = append([]string{}, meta.CandidateModels...)
		out.TriedModels = append([]string{}, meta.TriedModels...)
		out.SelectedModel = meta.SelectedModel
		out.ModelFailures = append([]string{}, meta.ModelFailures...)
		out.SkippedModels = append([]string{}, meta.SkippedModels...)
		out.LastErrorType = meta.LastErrorType
		out.CircuitOpenUntil = meta.CircuitOpenUntil
		out.BotReplySkipped = true
		out.LLMAttempted = meta.LLMAttempted
		out.InputLanguage = meta.InputLanguage
		out.OutputLanguage = meta.OutputLanguage
		out.LanguageMismatch = meta.LanguageMismatch
		out.RejectReason = meta.RejectReason
		e.logTriggerSkip(in, traceID, triggerReason, triggerType, eval.ForceReply, eval.Score, eval.AbsurdityScore, eval.RiskScore, eval.ReplyMode, out.SkipReason)
		e.logReplyDone(in, out)
		return out
	}
	roboticToneBlocked := roleSpec.OutputPolicy.BlockRoboticTone && IsRoboticToneText(line)
	roboticRepeatBlocked := roleSpec.OutputPolicy.BlockRoboticTone && IsRoboticRepetitionText(line)
	if roboticToneBlocked || roboticRepeatBlocked {
		blockReason := "robotic_tone"
		if roboticRepeatBlocked {
			blockReason = "robotic_repetition"
		}
		out.SkipReason = "bot_reply_skipped:" + blockReason
		out.FallbackReason = blockReason
		out.Provider = meta.Provider
		out.HTTPStatus = meta.HTTPStatus
		out.LatencyMs = meta.LatencyMs
		out.PromptTokens = meta.PromptTokens
		out.CompletionTokens = meta.CompletionTokens
		out.TotalTokens = meta.TotalTokens
		out.RequestPromptExcerpt = meta.RequestPromptExcerpt
		out.ErrorMessage = "final outbound robotic style blocked"
		out.DisplayableFound = meta.DisplayableFound
		out.ReasoningOnly = meta.ReasoningOnly
		out.ModelPool = meta.ModelPool
		out.CandidateModels = append([]string{}, meta.CandidateModels...)
		out.TriedModels = append([]string{}, meta.TriedModels...)
		out.SelectedModel = meta.SelectedModel
		out.ModelFailures = append([]string{}, meta.ModelFailures...)
		out.SkippedModels = append([]string{}, meta.SkippedModels...)
		out.LastErrorType = blockReason
		out.CircuitOpenUntil = meta.CircuitOpenUntil
		out.BotReplySkipped = true
		out.LLMAttempted = meta.LLMAttempted
		out.InputLanguage = meta.InputLanguage
		out.OutputLanguage = meta.OutputLanguage
		out.LanguageMismatch = meta.LanguageMismatch
		out.RejectReason = blockReason
		e.logTriggerSkip(in, traceID, triggerReason, triggerType, eval.ForceReply, eval.Score, eval.AbsurdityScore, eval.RiskScore, eval.ReplyMode, out.SkipReason)
		e.logReplyDone(in, out)
		return out
	}

	if e.debugForceLLM {
		if usedLLM {
			line = "[LLM] " + line
		} else {
			line = "[TPL-FALLBACK] " + line
		}
	}

	out.Triggered = true
	out.Content = line
	out.TargetID = in.SpeakerID
	out.UsedLLM = usedLLM
	out.FallbackReason = fallbackReason
	out.Provider = meta.Provider
	out.HTTPStatus = meta.HTTPStatus
	out.LatencyMs = meta.LatencyMs
	out.PromptTokens = meta.PromptTokens
	out.CompletionTokens = meta.CompletionTokens
	out.TotalTokens = meta.TotalTokens
	out.RequestPromptExcerpt = meta.RequestPromptExcerpt
	out.ErrorMessage = meta.ErrorMessage
	out.DisplayableFound = meta.DisplayableFound
	out.ReasoningOnly = meta.ReasoningOnly
	out.ModelPool = meta.ModelPool
	out.CandidateModels = append([]string{}, meta.CandidateModels...)
	out.TriedModels = append([]string{}, meta.TriedModels...)
	out.SelectedModel = meta.SelectedModel
	out.ModelFailures = append([]string{}, meta.ModelFailures...)
	out.SkippedModels = append([]string{}, meta.SkippedModels...)
	out.LastErrorType = meta.LastErrorType
	out.CircuitOpenUntil = meta.CircuitOpenUntil
	out.BotReplySkipped = false
	out.LLMAttempted = meta.LLMAttempted
	out.InputLanguage = meta.InputLanguage
	out.OutputLanguage = meta.OutputLanguage
	out.LanguageMismatch = meta.LanguageMismatch
	out.RejectReason = meta.RejectReason
	if usedLLM {
		out.ReplySource = ReplySourceLLM
		out.LLMModel = meta.Model
	} else {
		out.ReplySource = ReplySourceTemplate
		out.LLMModel = ""
	}

	e.logReplyDone(in, out)
	return out
}

func (e *Engine) RunLLMTest(ctx context.Context, in Input) Output {
	traceID := uuid.NewString()
	triggerReason := "manual_test"
	triggerType := "manual_test"
	in.Role = character.NormalizeRole(in.Role)
	roleSpec := character.Get(in.Role)
	e.logTriggerEnter(in, traceID, triggerType)
	e.logTriggerHit(in, traceID, triggerReason, triggerType, true, 100, 100, 0, "judgement")

	line, usedLLM, fallbackReason, meta := e.generate(ctx, in, roleSpec, "测试", traceID, triggerReason, triggerType, true, 100, 0, "judgement")
	if strings.TrimSpace(line) == "" {
		line = "LLM 测试未返回内容"
	}

	if usedLLM {
		line = "[LLM-TEST] " + line
	} else {
		line = "[TPL-TEST] " + line
	}

	out := Output{
		Triggered:            true,
		Content:              line,
		TargetID:             in.SpeakerID,
		ForceReply:           true,
		UsedLLM:              usedLLM,
		Score:                100,
		Atmosphere:           "测试",
		Reason:               "llm_test",
		TraceID:              traceID,
		TriggerReason:        triggerReason,
		TriggerType:          triggerType,
		HypeScore:            100,
		AbsurdityScore:       100,
		RiskScore:            0,
		ReplyMode:            "judgement",
		ModelPool:            meta.ModelPool,
		CandidateModels:      append([]string{}, meta.CandidateModels...),
		TriedModels:          append([]string{}, meta.TriedModels...),
		SelectedModel:        meta.SelectedModel,
		ModelFailures:        append([]string{}, meta.ModelFailures...),
		SkippedModels:        append([]string{}, meta.SkippedModels...),
		LastErrorType:        meta.LastErrorType,
		CircuitOpenUntil:     meta.CircuitOpenUntil,
		BotReplySkipped:      false,
		LLMAttempted:         meta.LLMAttempted,
		FallbackReason:       fallbackReason,
		Provider:             meta.Provider,
		LLMEnabled:           e.llmEnabled,
		ProviderInitialized:  e.providerReady(),
		APIKeyPresent:        e.apiKeyPresent,
		HTTPStatus:           meta.HTTPStatus,
		LatencyMs:            meta.LatencyMs,
		PromptTokens:         meta.PromptTokens,
		CompletionTokens:     meta.CompletionTokens,
		TotalTokens:          meta.TotalTokens,
		RequestPromptExcerpt: meta.RequestPromptExcerpt,
		ErrorMessage:         meta.ErrorMessage,
		DisplayableFound:     meta.DisplayableFound,
		ReasoningOnly:        meta.ReasoningOnly,
	}
	if usedLLM {
		out.ReplySource = ReplySourceLLM
		out.LLMModel = meta.Model
	} else {
		out.ReplySource = ReplySourceTemplate
	}
	if !usedLLM && out.FallbackReason == "" {
		out.FallbackReason = "llm_error"
	}
	e.logReplyDone(in, out)
	return out
}

func (e *Engine) RunMinReply(_ context.Context, in Input) Output {
	in.Role = character.NormalizeRole(in.Role)
	roleSpec := character.Get(in.Role)
	traceID := uuid.NewString()
	line := "[MIN-REPLY] " + e.renderTemplate(in, roleSpec, "保底", "debug_min_reply", true, "light_absurd", 0)
	out := Output{
		Triggered:           true,
		Content:             line,
		TargetID:            in.SpeakerID,
		ForceReply:          true,
		UsedLLM:             false,
		Score:               100,
		Atmosphere:          "保底",
		Reason:              "debug_min_reply",
		TraceID:             traceID,
		TriggerReason:       "debug_min_reply",
		TriggerType:         "system_event",
		HypeScore:           100,
		AbsurdityScore:      90,
		RiskScore:           0,
		ReplyMode:           "light_absurd",
		ReplySource:         ReplySourceTemplate,
		FallbackReason:      "debug_min_reply",
		Provider:            e.llmProviderName,
		LLMEnabled:          e.llmEnabled,
		ProviderInitialized: e.providerReady(),
		APIKeyPresent:       e.apiKeyPresent,
		DisplayableFound:    false,
		ReasoningOnly:       false,
		LLMAttempted:        false,
	}
	e.logTriggerEnter(in, traceID, out.TriggerType)
	e.logTriggerHit(in, traceID, out.TriggerReason, out.TriggerType, out.ForceReply, out.HypeScore, out.AbsurdityScore, out.RiskScore, out.ReplyMode)
	e.logReplyDone(in, out)
	return out
}

func (e *Engine) generate(ctx context.Context, in Input, roleSpec character.Spec, atmosphere, traceID, triggerReason, triggerType string, forceReply bool, absurdityScore, riskScore int, replyMode string) (line string, usedLLM bool, fallbackReason string, meta generateMeta) {
	meta.Provider = e.llmProviderName
	meta.Model = e.llmModel
	meta.BaseURL = e.llmBaseURL
	inputLang := DetectPrimaryLanguage(in.Content)
	meta.InputLanguage = inputLang

	if !e.llmEnabled {
		fallbackReason = "llm_disabled"
		meta.ErrorMessage = "llm disabled by config"
		meta.LLMAttempted = false
		e.recordFallback(fallbackReason, meta.ErrorMessage)
		e.logLLMError(in, traceID, fallbackReason, meta.ErrorMessage, meta)
		if shouldTemplateFallback(fallbackReason, forceReply) {
			e.logLLMFallback(in, traceID, fallbackReason, meta.ErrorMessage, meta)
			return e.renderTemplate(in, roleSpec, atmosphere, triggerReason, forceReply, replyMode, riskScore), false, fallbackReason, meta
		}
		meta.BotReplySkipped = true
		return "", false, fallbackReason, meta
	}

	safeRecent := takeRecent(filterPlannerLeakMessages(in.RecentMessages), 14)
	sys, usr := prompt.Build(prompt.BuildInput{
		Role:            in.Role,
		FireLevel:       in.FireLevel,
		SpeakerName:     in.SpeakerName,
		SpeakerIdentity: in.SpeakerIdentity,
		RolePersona:     roleSpec.PromptPolicy.Persona,
		RoleInstruction: roleSpec.PromptPolicy.RoleInstruction,
		ToneHint:        roleSpec.PromptPolicy.ToneHint,
		InputLanguage:   inputLang,
		SpeakerContent:  in.Content,
		Atmosphere:      atmosphere,
		ReplyToMessage:  in.ReplyToMessage,
		RecentMessages:  safeRecent,
		TriggerReason:   triggerReason,
		ReplyMode:       replyMode,
		AbsurdityScore:  absurdityScore,
		RiskScore:       riskScore,
	})
	meta.RequestPromptExcerpt = excerptPrompt(sys, usr, 1000)

	plan := e.selectPoolPlan(in, forceReply, roleSpec)
	requestStarted := time.Now()
	plannedPools := make([]string, 0, len(plan))
	poolAll := make(map[string][]ModelConfig, len(plan))
	poolAvail := make(map[string][]ModelConfig, len(plan))
	skippedModels := make([]string, 0, 16)
	totalAllCandidates := 0
	totalAvailCandidates := 0
	for _, p := range plan {
		plannedPools = append(plannedPools, p.Pool)
		all := e.allPoolModels(p.Pool)
		poolAll[p.Pool] = all
		totalAllCandidates += len(all)
		for _, c := range all {
			meta.CandidateModels = append(meta.CandidateModels, c.Name)
		}
		avail, skipped := e.availableModels(ctx, p.Pool)
		poolAvail[p.Pool] = e.prioritizeHealthy(p.Pool, avail)
		totalAvailCandidates += len(avail)
		for _, item := range skipped {
			skippedModels = append(skippedModels, p.Pool+":"+item)
		}
		healthy := e.getHealthyCandidates(p.Pool)
		if len(healthy) == 0 || len(avail) < e.healthyMinCount {
			reason := "healthy_empty"
			if len(healthy) > 0 && len(avail) < e.healthyMinCount {
				reason = "healthy_low"
			}
			e.schedulePoolProbe(p.Pool, reason)
		}
	}
	meta.ModelPool = strings.Join(plannedPools, "->")
	meta.SkippedModels = append([]string{}, skippedModels...)

	if totalAllCandidates == 0 {
		fallbackReason = "no_available_model"
		meta.ErrorMessage = "no configured model in selected pool"
		meta.LastErrorType = fallbackReason
		meta.BotReplySkipped = true
		meta.LLMAttempted = false
		e.recordFallback(fallbackReason, meta.ErrorMessage)
		e.logLLMError(in, traceID, fallbackReason, meta.ErrorMessage, meta)
		if shouldTemplateFallback(fallbackReason, forceReply) {
			e.logLLMFallback(in, traceID, fallbackReason, meta.ErrorMessage, meta)
			return e.renderTemplate(in, roleSpec, atmosphere, triggerReason, forceReply, replyMode, riskScore), false, fallbackReason, meta
		}
		return "", false, fallbackReason, meta
	}
	if totalAvailCandidates == 0 {
		fallbackReason = "all_models_cooling_down"
		meta.ErrorMessage = "all configured models are cooling down or disabled"
		meta.LastErrorType = fallbackReason
		meta.BotReplySkipped = true
		meta.LLMAttempted = false
		e.recordFallback(fallbackReason, meta.ErrorMessage)
		e.logLLMError(in, traceID, fallbackReason, meta.ErrorMessage, meta)
		if shouldTemplateFallback(fallbackReason, forceReply) {
			e.logLLMFallback(in, traceID, fallbackReason, meta.ErrorMessage, meta)
			return e.renderTemplate(in, roleSpec, atmosphere, triggerReason, forceReply, replyMode, riskScore), false, fallbackReason, meta
		}
		return "", false, fallbackReason, meta
	}

	totalBudget := e.llmTotalBudget
	if totalBudget <= 0 {
		totalBudget = 30
	}
	deadline := time.Now().Add(time.Duration(totalBudget) * time.Second)
	tried := make([]string, 0, defaultTryNormal)
	failures := make([]string, 0, defaultTryNormal)
	blockedProviders := map[string]bool{}
	lastReason := "llm_error"
	lastErrMsg := ""
	fallbackEligibleReason := ""
	lastCircuitUntil := ""
	lastCircuitScope := ""
	lastModelCircuitUntil := ""
	lastProviderCircuitUntil := ""
	lastOutputLang := ""
	lastRejectReason := ""
	lastLanguageMismatch := false
	llmAttempted := false

	for _, item := range plan {
		candidates := poolAvail[item.Pool]
		if len(candidates) == 0 {
			continue
		}
		attemptLimit := e.planAttemptLimit(item.MaxAttempts)
		if attemptLimit > len(candidates) {
			attemptLimit = len(candidates)
		}
		if attemptLimit <= 0 {
			attemptLimit = 1
		}
		attemptedInPool := 0

		for _, candidate := range candidates {
			if attemptedInPool >= attemptLimit {
				break
			}
			providerScope := e.providerScopeKey(candidate)
			if blockedProviders[providerScope] {
				meta.SkippedModels = append(meta.SkippedModels, item.Pool+":"+candidate.Name+":provider_blocked_this_round")
				continue
			}
			remaining := time.Until(deadline)
			if remaining <= 300*time.Millisecond {
				lastReason = "timeout"
				lastErrMsg = "llm total budget exceeded"
				failures = append(failures, "budget_exceeded")
				break
			}

			attemptTO := candidate.TimeoutSeconds
			if attemptTO <= 0 {
				attemptTO = e.llmPerModelTO
			}
			if attemptTO <= 0 {
				attemptTO = 10
			}
			attemptDur := time.Duration(attemptTO) * time.Second
			if attemptDur > remaining {
				attemptDur = remaining
			}

			curMeta := meta
			curMeta.ModelPool = item.Pool
			curMeta.Model = candidate.Name
			curMeta.Provider = candidate.Provider
			curMeta.BaseURL = candidate.BaseURL
			curMeta.CircuitKey = e.modelScopeKey(candidate)
			curMeta.ProviderCircuitKey = e.providerScopeKey(candidate)
			curMeta.TriedModels = append(append([]string{}, tried...), candidate.Name)
			curMeta.SelectedModel = candidate.Name
			curMeta.ModelFailures = append([]string{}, failures...)
			curMeta.LLMAttempted = llmAttempted
			e.logLLMPrepare(in, traceID, curMeta)

			provider := e.providerForModel(candidate)
			if provider == nil {
				lastReason = "provider_misconfig"
				lastErrMsg = "llm provider not initialized for model"
				curMeta.ErrorMessage = lastErrMsg
				curMeta.LastErrorType = lastReason
				curMeta.InputLanguage = inputLang
				e.recordFallback(lastReason, lastErrMsg)
				circuitUntil, circuitScope, modelUntil, providerUntil := e.applyModelResult(ctx, candidate, lastReason, false)
				curMeta.CircuitScope = circuitScope
				if circuitUntil > 0 {
					curMeta.CircuitOpenUntil = time.Unix(circuitUntil, 0).UTC().Format(time.RFC3339)
					lastCircuitUntil = curMeta.CircuitOpenUntil
				}
				if modelUntil > 0 {
					curMeta.ModelCircuitOpenUntil = time.Unix(modelUntil, 0).UTC().Format(time.RFC3339)
					lastModelCircuitUntil = curMeta.ModelCircuitOpenUntil
				}
				if providerUntil > 0 {
					curMeta.ProviderCircuitOpenUntil = time.Unix(providerUntil, 0).UTC().Format(time.RFC3339)
					lastProviderCircuitUntil = curMeta.ProviderCircuitOpenUntil
				}
				lastCircuitScope = circuitScope
				lastOutputLang = curMeta.OutputLanguage
				lastRejectReason = curMeta.RejectReason
				lastLanguageMismatch = curMeta.LanguageMismatch
				e.logLLMError(in, traceID, lastReason, lastErrMsg, curMeta)
				failures = append(failures, formatModelFailure(candidate.Name, lastReason, 0))
				if shouldStopProviderForRound(lastReason) {
					blockedProviders[providerScope] = true
				}
				tried = append(tried, candidate.Name)
				attemptedInPool++
				continue
			}

			curMeta.LLMAttempted = true
			e.logLLMRequest(in, traceID, len(in.RecentMessages), forceReply, triggerReason, triggerType, curMeta)
			llmAttempted = true

			tryCtx, cancel := context.WithTimeout(ctx, attemptDur)
			started := time.Now()
			result, err := provider.GenerateReply(tryCtx, llm.GenerateInput{
				SystemPrompt: sys,
				UserPrompt:   usr,
				Messages:     buildMessageForLLM(sys, usr, safeRecent),
			})
			cancel()

			curMeta.LatencyMs = int(time.Since(started).Milliseconds())
			curMeta.HTTPStatus = result.HTTPStatus
			curMeta.PromptTokens = result.PromptTokens
			curMeta.CompletionTokens = result.CompletionTokens
			curMeta.TotalTokens = result.TotalTokens
			curMeta.RawResponsePreview = result.RawBodyPreview
			curMeta.RawResponseHeaders = result.RawHeaders
			curMeta.ChoicesCount = result.ChoicesCount
			curMeta.ChoiceSummary = result.ChoiceSummary
			curMeta.ContentPreview = result.ContentPreview
			curMeta.DisplayableFound = result.DisplayableFound
			curMeta.ReasoningOnly = result.ReasoningOnly
			curMeta.TriedModels = append(append([]string{}, tried...), candidate.Name)
			curMeta.LLMAttempted = true

			if err == nil {
				reply := normalizeLLMOutput(result.Content)
				if maxRunes := roleSpec.OutputPolicy.MaxReplyRunes; maxRunes > 0 {
					r := []rune(reply)
					if len(r) > maxRunes {
						reply = strings.TrimSpace(string(r[:maxRunes]))
					}
				}
				curMeta.OutputLanguage = DetectPrimaryLanguage(reply)
				curMeta.InputLanguage = inputLang
				plannerLeak := roleSpec.OutputPolicy.BlockPlannerLeak && IsPlannerLeakText(reply)
				roboticTone := roleSpec.OutputPolicy.BlockRoboticTone && IsRoboticToneText(reply)
				roboticRepeat := roleSpec.OutputPolicy.BlockRoboticTone && IsRoboticRepetitionText(reply)
				languageMismatch := roleSpec.OutputPolicy.EnforceLanguageMatch && isLanguageMismatch(inputLang, curMeta.OutputLanguage)
				if strings.TrimSpace(reply) != "" && !plannerLeak && !roboticTone && !roboticRepeat && result.DisplayableFound && !result.ReasoningOnly {
					if languageMismatch {
						lastReason = "language_mismatch"
						lastErrMsg = fmt.Sprintf("output language mismatch input=%s output=%s", inputLang, curMeta.OutputLanguage)
						curMeta.LanguageMismatch = true
						curMeta.RejectReason = "language_mismatch"
					} else {
						_, _, _, _ = e.applyModelResult(ctx, candidate, "", true)
						e.recordLLMSuccess()
						curMeta.SelectedModel = candidate.Name
						curMeta.ModelFailures = append([]string{}, failures...)
						e.logLLMSuccess(in, traceID, len([]rune(reply)), curMeta)
						return reply, true, "", curMeta
					}
				} else if plannerLeak {
					lastReason = "planner_leak"
					lastErrMsg = "planner leak text blocked"
					curMeta.RejectReason = "planner_leak"
				} else if roboticTone {
					lastReason = "robotic_tone"
					lastErrMsg = "robotic report-like tone blocked"
					curMeta.RejectReason = "robotic_tone"
				} else if roboticRepeat {
					lastReason = "robotic_repetition"
					lastErrMsg = "robotic repetitive structure blocked"
					curMeta.RejectReason = "robotic_repetition"
				} else if result.ReasoningOnly {
					lastReason = "reasoning_only_response"
					lastErrMsg = "llm reasoning only response"
					curMeta.RejectReason = "low_quality_output"
				} else if !result.DisplayableFound {
					lastReason = "no_displayable_content"
					lastErrMsg = "llm no displayable content"
					curMeta.RejectReason = "low_quality_output"
				} else {
					lastReason = "empty_response"
					lastErrMsg = "llm empty content"
					curMeta.RejectReason = "low_quality_output"
				}
			} else {
				curMeta.HTTPStatus = llmStatusFromErr(err)
				lastReason = classifyLLMFailure(err, curMeta)
				lastErrMsg = shortErr(err)
				if lastReason == "planner_leak" {
					curMeta.RejectReason = "planner_leak"
				} else if lastReason == "robotic_tone" {
					curMeta.RejectReason = "robotic_tone"
				} else if lastReason == "robotic_repetition" {
					curMeta.RejectReason = "robotic_repetition"
				} else if lastReason == "language_mismatch" {
					curMeta.RejectReason = "language_mismatch"
				} else {
					curMeta.RejectReason = "low_quality_output"
				}
			}

			curMeta.ErrorMessage = lastErrMsg
			curMeta.SelectedModel = candidate.Name
			e.recordFallback(lastReason, lastErrMsg)
			circuitUntil, circuitScope, modelUntil, providerUntil := e.applyModelResult(ctx, candidate, lastReason, false)
			curMeta.LastErrorType = lastReason
			curMeta.CircuitScope = circuitScope
			if circuitUntil > 0 {
				curMeta.CircuitOpenUntil = time.Unix(circuitUntil, 0).UTC().Format(time.RFC3339)
				lastCircuitUntil = curMeta.CircuitOpenUntil
			}
			if modelUntil > 0 {
				curMeta.ModelCircuitOpenUntil = time.Unix(modelUntil, 0).UTC().Format(time.RFC3339)
				lastModelCircuitUntil = curMeta.ModelCircuitOpenUntil
			}
			if providerUntil > 0 {
				curMeta.ProviderCircuitOpenUntil = time.Unix(providerUntil, 0).UTC().Format(time.RFC3339)
				lastProviderCircuitUntil = curMeta.ProviderCircuitOpenUntil
			}
			lastCircuitScope = circuitScope
			lastOutputLang = curMeta.OutputLanguage
			lastRejectReason = curMeta.RejectReason
			lastLanguageMismatch = curMeta.LanguageMismatch
			curMeta.ModelFailures = append(append([]string{}, failures...), formatModelFailure(candidate.Name, lastReason, curMeta.HTTPStatus))
			e.logLLMError(in, traceID, lastReason, lastErrMsg, curMeta)
			if lastReason == "no_displayable_content" || lastReason == "planner_leak" || lastReason == "reasoning_only_response" || lastReason == "robotic_tone" || lastReason == "robotic_repetition" {
				fallbackEligibleReason = lastReason
			}

			failures = append(failures, formatModelFailure(candidate.Name, lastReason, curMeta.HTTPStatus))
			if shouldStopProviderForRound(lastReason) {
				blockedProviders[providerScope] = true
			}
			tried = append(tried, candidate.Name)
			attemptedInPool++
		}
		if attemptedInPool >= attemptLimit {
			e.schedulePoolProbe(item.Pool, "attempt_limit_reached")
			if e.logger != nil {
				e.logger.Info("[LLM] llm_attempt_limit_reached",
					"trace_id", traceID,
					"pool_name", item.Pool,
					"request_model_attempts", attemptedInPool,
					"attempt_limit", attemptLimit,
					"request_total_model_time_ms", int(time.Since(requestStarted).Milliseconds()),
				)
			}
		}
	}

	meta.TriedModels = tried
	meta.ModelFailures = failures
	meta.SelectedModel = ""
	meta.BotReplySkipped = true
	meta.LLMAttempted = llmAttempted

	if len(tried) == 0 {
		if totalAllCandidates == 0 {
			fallbackReason = "no_available_model"
		} else {
			fallbackReason = "all_models_cooling_down"
		}
	} else if fallbackEligibleReason != "" {
		fallbackReason = fallbackEligibleReason
	} else {
		fallbackReason = "pool_exhausted"
	}
	meta.LastErrorType = lastReason
	if strings.TrimSpace(meta.LastErrorType) == "" {
		meta.LastErrorType = fallbackReason
	}
	meta.CircuitScope = lastCircuitScope
	meta.CircuitOpenUntil = lastCircuitUntil
	meta.ModelCircuitOpenUntil = lastModelCircuitUntil
	meta.ProviderCircuitOpenUntil = lastProviderCircuitUntil
	meta.OutputLanguage = lastOutputLang
	meta.RejectReason = lastRejectReason
	meta.LanguageMismatch = lastLanguageMismatch
	meta.ErrorMessage = lastErrMsg
	if e.logger != nil {
		e.logger.Info("[LLM] request_model_attempts",
			"trace_id", traceID,
			"model_pool", meta.ModelPool,
			"request_model_attempts", len(tried),
			"llm_attempt_limit_reached", len(tried) >= e.llmMaxAttempts && e.llmMaxAttempts > 0,
			"request_total_model_time_ms", int(time.Since(requestStarted).Milliseconds()),
			"tried_models", joinList(tried),
			"healthy_candidates", joinList(e.getHealthyCandidates(poolChatMain)),
		)
	}
	if fallbackReason == "" {
		fallbackReason = "pool_exhausted"
	}
	if shouldTemplateFallback(fallbackReason, forceReply) {
		e.logLLMFallback(in, traceID, fallbackReason, lastErrMsg, meta)
		return e.renderTemplate(in, roleSpec, atmosphere, triggerReason, forceReply, replyMode, riskScore), false, fallbackReason, meta
	}
	return "", false, fallbackReason, meta
}

func (e *Engine) renderTemplate(in Input, roleSpec character.Spec, atmosphere, triggerReason string, forceReply bool, replyMode string, riskScore int) string {
	snippet := trimContent(in.Content)
	scenario := normalizeScenario(triggerReason, in.Content, atmosphere, in.ReplyToMessage)
	mode := fireMode(in.FireLevel, scenario, forceReply, replyMode, riskScore)
	spec := roleSpec
	role := character.NormalizeRole(in.Role)
	if replyMode == "cold_narration" {
		role = models.RoleNarrator
		spec = character.Get(role)
	}
	pack := spec.FallbackPack(scenario)

	lines := make([]string, 0, 6)
	lines = append(lines, formatTemplate(e.pick(pack.Classify), in.SpeakerName, snippet))

	if mode != "light" || e.roll(100) > 30 {
		lines = append(lines, formatTemplate(e.pick(pack.Amplify), in.SpeakerName, snippet))
	}

	qCount := questionCountByMode(mode)
	for i := 0; i < qCount; i++ {
		lines = append(lines, formatTemplate(e.pick(pack.Question), in.SpeakerName, snippet))
	}

	lines = append(lines, formatTemplate(e.pick(pack.Final), in.SpeakerName, snippet))
	return trimTemplateOutput(strings.Join(lines, " "))
}

type templatePack struct {
	Classify []string
	Amplify  []string
	Question []string
	Final    []string
}

func templatePackByRoleScenario(role models.BotRole, scenario string) templatePack {
	switch role {
	case models.RoleJudge:
		return judgePack(scenario)
	case models.RoleNarrator:
		return narratorPack(scenario)
	default:
		return npcPack(scenario)
	}
}

func judgePack(scenario string) templatePack {
	switch scenario {
	case "quote_bot":
		return templatePack{
			Classify: []string{
				"你拿“%s”来回我？我先给你记一笔：硬蹭。",
				"你把“%s”拎出来，就别怪我当面加罚。",
			},
			Amplify: []string{
				"我看得很清楚，你不是来聊，是来拿空句试探我。",
				"你这动作一出手，我就知道你准备继续添乱。",
			},
			Question: []string{
				"你真觉得引用我就能洗白你这句？",
				"你拿什么证明你不是来添堵的？",
				"你发这句之前真做过质量检查？",
			},
			Final: []string{
				"我话放这儿：下一句再空，我直接按污染续罚。",
				"先闭三秒，别再往我眼前丢废句。",
			},
		}
	case "ad_smell":
		return templatePack{
			Classify: []string{
				"你这句“%s”一出来，我就闻到推销味了。",
				"“%s”这种话在我这就是污染，不是开场。",
			},
			Amplify: []string{
				"你不是在聊天，你是在往群里倒低配投放文案。",
				"我看你这路数，就是把大家耐心当消耗品。",
			},
			Question: []string{
				"你真把这里当广告位了？",
				"你觉得这种味道不会把气氛搞脏？",
				"你发这句的时候一点都不心虚？",
			},
			Final: []string{
				"我给你的结论很简单：收回这套，别碍眼。",
				"别继续投放了，先学会说人话。",
			},
		}
	case "hard_mouth", "pretend_expert", "flag":
		return templatePack{
			Classify: []string{
				"你这句“%s”在我这就是嘴硬找补。",
				"“%s”这口气太熟了，我一看就知道你在装懂。",
			},
			Amplify: []string{
				"你拿语气顶内容，结果两头都空。",
				"你这不是解释，是提前给自己留台阶。",
			},
			Question: []string{
				"你真把口气当证据了？",
				"你拿什么撑住这句硬气？",
				"你自己回看不会心虚？",
			},
			Final: []string{
				"我先判你重来：补内容，再开口。",
				"收收姿态，别在我这儿空转。",
			},
		}
	case "risk_high", "argument":
		return templatePack{
			Classify: []string{
				"你这句“%s”火气太重了，我先压你一档。",
				"我听出来了，你现在情绪比内容多。",
			},
			Amplify: []string{
				"你再往上冲，只会把对话冲成噪音。",
				"我看你现在是在加火，不是在沟通。",
			},
			Question: []string{
				"你真要拿音量代替内容？",
				"你是要结论，还是要继续上头？",
			},
			Final: []string{
				"我先让你降火，下一句好好说。",
				"先稳住，别把场子搞成事故。",
			},
		}
	default:
		return templatePack{
			Classify: []string{
				"你这句“%s”在我这儿就是空心开场。",
				"“%s”这种话，我看一眼就判低质。",
			},
			Amplify: []string{
				"你看着像在说话，其实是在拿废话刷存在。",
				"我本来还想听内容，结果你先给我塞噪音。",
			},
			Question: []string{
				"你真觉得这句配占一行？",
				"你发之前做过最基本的质量检查吗？",
				"你是来聊天还是来污染节奏？",
			},
			Final: []string{
				"我给你一句忠告：下一句给内容，别再空转。",
				"少来这套，闭麦三秒都算贡献。",
			},
		}
	}
}

func npcPack(scenario string) templatePack {
	switch scenario {
	case "quote_bot":
		return templatePack{
			Classify: []string{
				"你拿“%s”来回我？行，我先嫌弃你这动作。",
				"你引用“%s”，我只看见你在硬蹭存在感。",
			},
			Amplify: []string{
				"你不是在回应，你是在拿低配社交姿势刷镜头。",
				"我看你这套就一句话：没内容但非要出现。",
			},
			Question: []string{
				"你真觉得这招能抬气氛？",
				"你自己读这句不尴尬吗？",
				"你是想聊天还是想现眼？",
			},
			Final: []string{
				"收一收，别拿我这儿当热身区。",
				"你下一句再空，我就当场补刀。",
			},
		}
	case "ad_smell":
		return templatePack{
			Classify: []string{
				"“%s”这味儿太冲，我一眼就当广告。",
				"你这句“%s”不是开场，是推销味泄漏。",
			},
			Amplify: []string{
				"你在把群聊往交易页拖，空气都被你搞脏了。",
				"这不是社交，这就是低配投放动作。",
			},
			Question: []string{
				"你觉得大家是来听你投放的吗？",
				"谁给你的勇气把这句扔进来？",
				"你自己闻不出这股低级推广味？",
			},
			Final: []string{
				"别继续抹屏，撤回这种味道。",
				"离聊天区远一点，别再给我添乱。",
			},
		}
	case "hard_mouth", "pretend_expert", "flag":
		return templatePack{
			Classify: []string{
				"“%s”这句里，嘴硬味和装懂味都冒出来了。",
				"你这句“%s”就是先摆谱再补内容。",
			},
			Amplify: []string{
				"你在拿气势顶逻辑，结果两边都悬空。",
				"我看这就是先把自己说服，再试图糊我。",
			},
			Question: []string{
				"你真觉得这叫有理有据？",
				"你是来讲道理还是来撑场面？",
				"这句你自己回看不心虚？",
			},
			Final: []string{
				"别硬撑了，补内容再回来。",
				"收收戏，先把逻辑接上再跟我说。",
			},
		}
	case "risk_high", "argument":
		return templatePack{
			Classify: []string{
				"“%s”这句火药味太重，我先给你降温。",
				"你现在不是交流，你是在冲突加码。",
			},
			Amplify: []string{
				"你再往上顶，只会把房间拖进情绪泥潭。",
				"你把聊天当擂台，内容早被你踢下去了。",
			},
			Question: []string{
				"你是想解决问题还是想赢情绪？",
				"继续冲真的有意义吗？",
			},
			Final: []string{
				"降点火，别把场子掀了。",
				"先稳住，再跟我说人话。",
			},
		}
	default:
		return templatePack{
			Classify: []string{
				"“%s”这句，低质到我想直接静音。",
				"你这句“%s”不是聊天，是廉价试探。",
			},
			Amplify: []string{
				"看着像打招呼，其实是在拿空话污染现场。",
				"你在用最省脑子的句式消耗大家耐心。",
			},
			Question: []string{
				"你真以为这种话值钱吗？",
				"你发之前没想过会拉低气氛？",
				"你是来对话还是来刷存在感？",
			},
			Final: []string{
				"少来这套，别把房间搞脏。",
				"闭麦三秒，对所有人都有帮助。",
			},
		}
	}
}

func narratorPack(scenario string) templatePack {
	switch scenario {
	case "quote_bot":
		return templatePack{
			Classify: []string{
				"你引用“%s”，我先记你一条：主动加戏。",
				"你这句“%s”一接上来，我就知道事故要续集。",
			},
			Amplify: []string{
				"本来气氛还能抢救，你又递了一把低质燃料。",
				"你这不是回应，你是在放大失焦社交。",
			},
			Question: []string{
				"你真觉得这段值得加更吗？",
				"你打算靠这句把现场救回来？",
				"你自己听不出这股现眼感？",
			},
			Final: []string{
				"我给你的结论：别继续加戏，先停机。",
				"你下一句再空，我就直接按污染续集处理。",
			},
		}
	case "ad_smell":
		return templatePack{
			Classify: []string{
				"“%s”这句我直接归档成广告味入侵。",
				"你这句“%s”在我这儿就是交易话术污染。",
			},
			Amplify: []string{
				"聊天室被你拖向低配推销现场，观感断崖下滑。",
				"你把社交动作做成投放脚本，空气都浑了。",
			},
			Question: []string{
				"你真把这里当广告位了？",
				"你觉得这种味道不会脏屏吗？",
				"你发的时候一点都不心虚？",
			},
			Final: []string{
				"我这边收镜：请撤离你的投放动作。",
				"别再往现场倒废料，我看着都累。",
			},
		}
	case "hard_mouth", "pretend_expert", "flag":
		return templatePack{
			Classify: []string{
				"“%s”这句我先归档：姿态先行，内容缺席。",
				"你这句“%s”装懂感明显高于信息量。",
			},
			Amplify: []string{
				"你试图靠语气稳住局面，结果内容继续失焦。",
				"我看到的是“先嘴硬后找补”标准流程。",
			},
			Question: []string{
				"这段姿态表演真能替代论据吗？",
				"你确定这句不是在给自己留退路？",
			},
			Final: []string{
				"我给你一句结语：少演一点，内容会更清楚。",
				"先补信息，再发下一条给我看。",
			},
		}
	case "risk_high", "argument":
		return templatePack{
			Classify: []string{
				"你这句“%s”把情绪温度直接拉高了。",
				"我先提醒你：你正在把对话推向高风险区。",
			},
			Amplify: []string{
				"你继续加码只会扩散噪音，不会增加信息。",
				"我看对话已经快被情绪吞没了。",
			},
			Question: []string{
				"你现在还在解决问题吗？",
				"这股火气值得继续扩散吗？",
			},
			Final: []string{
				"我先把话收束：降温，再继续说。",
				"控火，别失控。",
			},
		}
	default:
		return templatePack{
			Classify: []string{
				"你这句“%s”，我先归档为空心社交动作。",
				"“%s”这种开场在我这儿就是低质样本。",
			},
			Amplify: []string{
				"这句没有信息，只剩存在焦虑在屏幕上发热。",
				"你像在聊天，其实在把节奏往下水道拽。",
			},
			Question: []string{
				"你真觉得这种句子配占一行吗？",
				"你发这句是想沟通还是想留痕？",
				"你自己回看不会尴尬吗？",
			},
			Final: []string{
				"我最后一句：别再拿空气当内容。",
				"下条请带信息，不然静音更体面。",
			},
		}
	}
}

func normalizeScenario(triggerReason, content, atmosphere string, replyTo *models.ChatMessage) string {
	if replyTo != nil && replyTo.IsBotMessage {
		return "quote_bot"
	}
	reason := strings.TrimSpace(strings.ToLower(triggerReason))
	switch reason {
	case "quote_bot", "ad_smell", "greeting", "small_talk", "low_value_noise", "chat_pollution", "cold_start", "argument", "flag", "hard_mouth", "pretend_expert", "risk_high":
		return reason
	}
	lower := strings.ToLower(strings.TrimSpace(content))
	if containsAnyToken(lower, []string{"你好", "哈喽", "hello", "hi", "在吗", "有人吗", "哈哈", "都行", "随便", "？", "?"}) {
		return "small_talk"
	}
	if containsAnyToken(lower, []string{"加v", "加vx", "代发", "推广", "优惠", "返利", "代理"}) {
		return "ad_smell"
	}
	if strings.Contains(strings.ToLower(atmosphere), "冷场") {
		return "cold_start"
	}
	return "chat_pollution"
}

func fireMode(level models.FireLevel, scenario string, forceReply bool, replyMode string, riskScore int) string {
	if replyMode == "force_quote_reply" {
		return "crazy"
	}
	if replyMode == "cold_narration" || riskScore >= 60 {
		return "light"
	}
	if replyMode == "light_absurd" {
		return "abstract"
	}
	switch level {
	case models.FireLow:
		return "light"
	case models.FireHigh:
		if forceReply || scenario == "ad_smell" || scenario == "chat_pollution" || scenario == "small_talk" {
			return "crazy"
		}
		return "abstract"
	default:
		return "yin"
	}
}

func questionCountByMode(mode string) int {
	switch mode {
	case "light":
		return 1
	case "crazy":
		return 3
	default:
		return 2
	}
}

func trimTemplateOutput(v string) string {
	v = strings.Join(strings.Fields(strings.TrimSpace(v)), " ")
	r := []rune(v)
	if len(r) > 190 {
		return string(r[:190]) + "..."
	}
	return v
}

func containsAnyToken(content string, keys []string) bool {
	for _, k := range keys {
		if strings.Contains(content, strings.ToLower(strings.TrimSpace(k))) {
			return true
		}
	}
	return false
}

func (e *Engine) pickNeutralNarration(role models.BotRole) string {
	switch role {
	case models.RoleJudge:
		return "【阴阳裁判】我先缓一手，下一句再让我看值不值得开罚。"
	case models.RoleNarrator:
		return "【冷面旁白】我先静三秒，等你下一句别再添堵。"
	default:
		return "【损友 NPC】我先放你一马，下一句再空我就直接开刀。"
	}
}

func (e *Engine) acquireCooldown(ctx context.Context, roomID string) (bool, error) {
	key := "room:" + roomID + ":bot:cooldown"
	ok, err := e.redis.SetNX(ctx, key, "1", time.Duration(e.cooldownSeconds)*time.Second).Result()
	return ok, err
}

func roomMuteKey(roomID string) string {
	return "room:" + roomID + ":bot:muted"
}

func trimContent(content string) string {
	content = strings.TrimSpace(content)
	r := []rune(content)
	if len(r) > 12 {
		return string(r[:12]) + "..."
	}
	if content == "" {
		return "这句"
	}
	return content
}

func normalizeLLMOutput(content string) string {
	content = strings.TrimSpace(content)
	content = strings.ReplaceAll(content, "\n", " ")
	content = strings.Join(strings.Fields(content), " ")
	if content == "" {
		return ""
	}
	r := []rune(content)
	if len(r) > 86 {
		content = string(r[:86]) + "..."
	}
	return content
}

func formatTemplate(tpl string, args ...string) string {
	n := strings.Count(tpl, "%s")
	if n <= 0 {
		return tpl
	}
	vals := make([]any, n)
	for i := 0; i < n; i++ {
		if i < len(args) {
			vals[i] = args[i]
		} else {
			vals[i] = ""
		}
	}
	return fmt.Sprintf(tpl, vals...)
}

func (e *Engine) pick(arr []string) string {
	e.mu.Lock()
	defer e.mu.Unlock()
	return arr[e.rand.Intn(len(arr))]
}

func (e *Engine) roll(n int) int {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.rand.Intn(n) + 1
}

func takeRecent(all []models.ChatMessage, n int) []models.ChatMessage {
	if n <= 0 || len(all) <= n {
		return all
	}
	return all[len(all)-n:]
}

func filterPlannerLeakMessages(all []models.ChatMessage) []models.ChatMessage {
	if len(all) == 0 {
		return all
	}
	out := make([]models.ChatMessage, 0, len(all))
	for _, msg := range all {
		isBot := msg.SenderType == "bot" || msg.IsBotMessage
		if isBot && IsPlannerLeakText(msg.Content) {
			continue
		}
		out = append(out, msg)
	}
	return out
}

func botTitle(role models.BotRole) string {
	switch role {
	case models.RoleJudge:
		return "阴阳裁判"
	case models.RoleNarrator:
		return "冷面旁白"
	default:
		return "损友 NPC"
	}
}

func shortErr(err error) string {
	if err == nil {
		return ""
	}
	msg := strings.TrimSpace(err.Error())
	r := []rune(msg)
	if len(r) > 180 {
		return string(r[:180]) + "..."
	}
	return msg
}

func llmStatusFromErr(err error) int {
	if err == nil {
		return 0
	}
	var pErr *llm.ProviderError
	if errors.As(err, &pErr) {
		return pErr.StatusCode
	}
	return 0
}

func excerptPrompt(systemPrompt, userPrompt string, max int) string {
	joined := "SYSTEM:\n" + strings.TrimSpace(systemPrompt) + "\n\nUSER:\n" + strings.TrimSpace(userPrompt)
	r := []rune(strings.TrimSpace(joined))
	if max <= 0 {
		max = 1000
	}
	if len(r) > max {
		return string(r[:max]) + "..."
	}
	return string(r)
}

func resolveTriggerReason(in Input, eval scorer.Result) string {
	if in.ReplyToMessage != nil && in.ReplyToMessage.IsBotMessage {
		return "quote_bot"
	}
	if in.SpeakerIdentity == models.IdentityTarget {
		return "target_user"
	}
	if hasTag(eval.Tags, "cold_start") {
		return "cold_start"
	}
	if hasTag(eval.Tags, "ad_smell") {
		return "ad_smell"
	}
	if hasTag(eval.Tags, "greeting") {
		return "greeting"
	}
	if hasTag(eval.Tags, "small_talk") {
		return "small_talk"
	}
	if hasTag(eval.Tags, "low_value_noise") {
		return "low_value_noise"
	}
	if hasTag(eval.Tags, "chat_pollution") {
		return "chat_pollution"
	}
	if hasTag(eval.Tags, "hard_mouth") {
		return "hard_mouth"
	}
	if hasTag(eval.Tags, "pretend_expert") {
		return "pretend_expert"
	}
	if hasTag(eval.Tags, "high_tension") || hasTag(eval.Tags, "risk_high") || hasTag(eval.Tags, "heated_argument") {
		return "risk_high"
	}
	if hasTag(eval.Tags, "argument") || hasTag(eval.Tags, "question_bomb") {
		return "argument"
	}
	if hasTag(eval.Tags, "flag") {
		return "flag"
	}
	return "hype_score"
}

func resolveTriggerType(in Input, triggerReason string) string {
	if in.ReplyToMessage != nil && in.ReplyToMessage.IsBotMessage {
		return "quote_bot"
	}
	if triggerReason == "cold_start" {
		return "cold_start"
	}
	if triggerReason == "manual_test" {
		return "manual_test"
	}
	if strings.TrimSpace(in.SpeakerID) == "" || in.SpeakerMessageID <= 0 {
		return "system_event"
	}
	return "user_message"
}

func hasTag(tags []string, target string) bool {
	for _, t := range tags {
		if strings.EqualFold(strings.TrimSpace(t), target) {
			return true
		}
	}
	return false
}

func classifyLLMError(err error) string {
	if err == nil {
		return "llm_error"
	}
	errLower := strings.ToLower(err.Error())
	if strings.Contains(errLower, "reasoning only response") {
		return "reasoning_only_response"
	}
	if strings.Contains(errLower, "no displayable content") {
		return "no_displayable_content"
	}
	if strings.Contains(errLower, "planner leak") {
		return "planner_leak"
	}
	if strings.Contains(errLower, "robotic tone") {
		return "robotic_tone"
	}
	if strings.Contains(errLower, "robotic repetitive") || strings.Contains(errLower, "robotic repetition") {
		return "robotic_repetition"
	}
	if strings.Contains(errLower, "language mismatch") {
		return "language_mismatch"
	}
	if strings.Contains(errLower, "empty content") || strings.Contains(errLower, "empty choices") || strings.Contains(errLower, "empty response") {
		return "empty_response"
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return "llm_timeout"
	}
	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return "llm_timeout"
	}
	if strings.Contains(errLower, "timeout") {
		return "llm_timeout"
	}
	return "llm_error"
}

func shouldTemplateFallback(reason string, forceReply bool) bool {
	if forceReply {
		return true
	}
	switch strings.TrimSpace(reason) {
	case "no_displayable_content", "planner_leak", "reasoning_only_response", "robotic_tone", "robotic_repetition":
		return true
	default:
		return false
	}
}

func (e *Engine) logTriggerEnter(in Input, traceID, triggerType string) {
	if strings.TrimSpace(triggerType) == "" {
		triggerType = "user_message"
	}
	if e.logger != nil {
		e.logger.Info("[BOT] bot.trigger.enter",
			"room_id", in.RoomID,
			"sender_id", in.SpeakerID,
			"sender_name", in.SpeakerName,
			"message_id", in.SpeakerMessageID,
			"trace_id", traceID,
			"trigger_type", triggerType,
			"llm_enabled", e.llmEnabled,
			"provider_initialized", e.providerReady(),
			"api_key_present", e.apiKeyPresent,
			"content", trimContent(in.Content),
			"bot_role", in.Role,
			"reply_to_message_id", in.ReplyToMessageID,
			"reply_to_bot", in.ReplyToMessage != nil && in.ReplyToMessage.IsBotMessage,
			"target_mode", in.TargetMode,
			"immune_mode", in.ImmuneMode,
		)
	}
	e.emitDebug(DebugEvent{
		Event:               "bot.trigger.enter",
		TraceID:             traceID,
		RoomID:              in.RoomID,
		MessageID:           in.SpeakerMessageID,
		SenderID:            in.SpeakerID,
		SenderName:          in.SpeakerName,
		Content:             trimContent(in.Content),
		ReplyToMessageID:    in.ReplyToMessageID,
		ReplyToIsBot:        in.ReplyToMessage != nil && in.ReplyToMessage.IsBotMessage,
		BotRole:             string(in.Role),
		TriggerType:         triggerType,
		LLMEnabled:          e.llmEnabled,
		ProviderInitialized: e.providerReady(),
		APIKeyPresent:       e.apiKeyPresent,
	})
}

func (e *Engine) logTriggerHit(in Input, traceID, triggerReason, triggerType string, forceReply bool, hypeScore, absurdityScore, riskScore int, replyMode string) {
	if e.logger != nil {
		e.logger.Info("[BOT] bot.trigger.hit",
			"room_id", in.RoomID,
			"message_id", in.SpeakerMessageID,
			"sender_id", in.SpeakerID,
			"trace_id", traceID,
			"bot_role", in.Role,
			"firepower_level", in.FireLevel,
			"trigger_reason", triggerReason,
			"trigger_type", triggerType,
			"llm_enabled", e.llmEnabled,
			"provider_initialized", e.providerReady(),
			"api_key_present", e.apiKeyPresent,
			"force_reply", forceReply,
			"hype_score", hypeScore,
			"absurdity_score", absurdityScore,
			"risk_score", riskScore,
			"reply_mode", replyMode,
		)
	}
	e.emitDebug(DebugEvent{
		Event:               "bot.trigger.hit",
		TraceID:             traceID,
		RoomID:              in.RoomID,
		MessageID:           in.SpeakerMessageID,
		SenderID:            in.SpeakerID,
		SenderName:          in.SpeakerName,
		BotRole:             string(in.Role),
		FirepowerLevel:      string(in.FireLevel),
		TriggerReason:       triggerReason,
		TriggerType:         triggerType,
		ForceReply:          forceReply,
		HypeScore:           hypeScore,
		AbsurdityScore:      absurdityScore,
		RiskScore:           riskScore,
		ReplyMode:           replyMode,
		LLMEnabled:          e.llmEnabled,
		ProviderInitialized: e.providerReady(),
		APIKeyPresent:       e.apiKeyPresent,
	})
}

func (e *Engine) logTriggerSkip(in Input, traceID, triggerReason, triggerType string, forceReply bool, hypeScore, absurdityScore, riskScore int, replyMode, skipReason string) {
	botReplySkipped := strings.HasPrefix(skipReason, "bot_reply_skipped")
	if e.logger != nil {
		e.logger.Info("[BOT] bot.trigger.skip",
			"room_id", in.RoomID,
			"message_id", in.SpeakerMessageID,
			"sender_id", in.SpeakerID,
			"trace_id", traceID,
			"bot_role", in.Role,
			"trigger_reason", triggerReason,
			"trigger_type", triggerType,
			"llm_enabled", e.llmEnabled,
			"provider_initialized", e.providerReady(),
			"api_key_present", e.apiKeyPresent,
			"force_reply", forceReply,
			"hype_score", hypeScore,
			"absurdity_score", absurdityScore,
			"risk_score", riskScore,
			"reply_mode", replyMode,
			"skip_reason", skipReason,
			"bot_reply_skipped", botReplySkipped,
		)
	}
	e.emitDebug(DebugEvent{
		Event:               "bot.trigger.skip",
		TraceID:             traceID,
		RoomID:              in.RoomID,
		MessageID:           in.SpeakerMessageID,
		SenderID:            in.SpeakerID,
		SenderName:          in.SpeakerName,
		BotRole:             string(in.Role),
		FirepowerLevel:      string(in.FireLevel),
		TriggerReason:       triggerReason,
		TriggerType:         triggerType,
		SkipReason:          skipReason,
		BotReplySkipped:     botReplySkipped,
		ForceReply:          forceReply,
		HypeScore:           hypeScore,
		AbsurdityScore:      absurdityScore,
		RiskScore:           riskScore,
		ReplyMode:           replyMode,
		LLMEnabled:          e.llmEnabled,
		ProviderInitialized: e.providerReady(),
		APIKeyPresent:       e.apiKeyPresent,
	})
}

func (e *Engine) logForceReplyCheck(in Input, replyToIsBot bool) {
	if e.logger != nil {
		e.logger.Info("[BOT] bot.force_reply.check",
			"room_id", in.RoomID,
			"message_id", in.SpeakerMessageID,
			"reply_to_message_id", in.ReplyToMessageID,
			"reply_to_is_bot", replyToIsBot,
			"force_reply", replyToIsBot,
		)
	}
	e.emitDebug(DebugEvent{
		Event:            "bot.force_reply.check",
		RoomID:           in.RoomID,
		MessageID:        in.SpeakerMessageID,
		ReplyToMessageID: in.ReplyToMessageID,
		ReplyToIsBot:     replyToIsBot,
		ForceReply:       replyToIsBot,
		SenderID:         in.SpeakerID,
		SenderName:       in.SpeakerName,
	})
}

func (e *Engine) logForceReplyHit(in Input) {
	if e.logger != nil {
		e.logger.Info("[BOT] bot.force_reply.hit",
			"room_id", in.RoomID,
			"message_id", in.SpeakerMessageID,
			"reply_to_message_id", in.ReplyToMessageID,
			"reply_to_is_bot", true,
			"force_reply", true,
		)
	}
	e.emitDebug(DebugEvent{
		Event:            "bot.force_reply.hit",
		RoomID:           in.RoomID,
		MessageID:        in.SpeakerMessageID,
		ReplyToMessageID: in.ReplyToMessageID,
		ReplyToIsBot:     true,
		ForceReply:       true,
		SenderID:         in.SpeakerID,
		SenderName:       in.SpeakerName,
	})
}

func (e *Engine) logForceReplyMiss(in Input) {
	if e.logger != nil {
		e.logger.Info("[BOT] bot.force_reply.miss",
			"room_id", in.RoomID,
			"message_id", in.SpeakerMessageID,
			"reply_to_message_id", in.ReplyToMessageID,
			"reply_to_is_bot", false,
			"force_reply", false,
		)
	}
	e.emitDebug(DebugEvent{
		Event:            "bot.force_reply.miss",
		RoomID:           in.RoomID,
		MessageID:        in.SpeakerMessageID,
		ReplyToMessageID: in.ReplyToMessageID,
		ReplyToIsBot:     false,
		ForceReply:       false,
		SenderID:         in.SpeakerID,
		SenderName:       in.SpeakerName,
	})
}

func (e *Engine) logLLMPrepare(in Input, traceID string, meta generateMeta) {
	model := meta.Model
	if strings.TrimSpace(model) == "" {
		model = e.llmModel
	}
	provider := meta.Provider
	if strings.TrimSpace(provider) == "" {
		provider = e.llmProviderName
	}
	baseURL := meta.BaseURL
	if strings.TrimSpace(baseURL) == "" {
		baseURL = e.llmBaseURL
	}
	if e.logger != nil {
		e.logger.Info("[LLM] bot.llm.prepare",
			"trace_id", traceID,
			"room_id", in.RoomID,
			"message_id", in.SpeakerMessageID,
			"sender_id", in.SpeakerID,
			"used_llm", false,
			"llm_enabled", e.llmEnabled,
			"provider_initialized", e.providerReady(),
			"model", model,
			"provider", provider,
			"base_url", baseURL,
			"model_pool", meta.ModelPool,
			"candidate_models", joinList(meta.CandidateModels),
			"skipped_models", joinList(meta.SkippedModels),
			"tried_models", joinList(meta.TriedModels),
			"selected_model", meta.SelectedModel,
			"circuit_key", meta.CircuitKey,
			"provider_circuit_key", meta.ProviderCircuitKey,
			"input_language", meta.InputLanguage,
			"llm_attempted", meta.LLMAttempted,
			"timeout_seconds", e.llmTimeoutSec,
			"llm_debug_raw_response", e.llmDebugRawResp,
			"api_key_present", e.apiKeyPresent,
			"request_prompt_excerpt", meta.RequestPromptExcerpt,
		)
	}
	e.emitDebug(DebugEvent{
		Event:                "bot.llm.prepare",
		TraceID:              traceID,
		RoomID:               in.RoomID,
		MessageID:            in.SpeakerMessageID,
		SenderID:             in.SpeakerID,
		SenderName:           in.SpeakerName,
		UsedLLM:              false,
		LLMEnabled:           e.llmEnabled,
		ProviderInitialized:  e.providerReady(),
		Model:                model,
		Provider:             provider,
		BaseURL:              baseURL,
		ModelPool:            meta.ModelPool,
		CandidateModels:      joinList(meta.CandidateModels),
		SkippedModels:        joinList(meta.SkippedModels),
		TriedModels:          joinList(meta.TriedModels),
		SelectedModel:        meta.SelectedModel,
		CircuitKey:           meta.CircuitKey,
		ProviderCircuitKey:   meta.ProviderCircuitKey,
		InputLanguage:        meta.InputLanguage,
		LLMAttempted:         meta.LLMAttempted,
		APIKeyPresent:        e.apiKeyPresent,
		ContentPreview:       "",
		ForceReply:           in.ReplyToMessage != nil && in.ReplyToMessage.IsBotMessage,
		RequestPromptExcerpt: meta.RequestPromptExcerpt,
	})
}

func (e *Engine) logLLMRequest(in Input, traceID string, messageCount int, forceReply bool, triggerReason, triggerType string, meta generateMeta) {
	if e.logger != nil {
		e.logger.Info("[LLM] bot.llm.request",
			"trace_id", traceID,
			"room_id", in.RoomID,
			"message_id", in.SpeakerMessageID,
			"message_count", messageCount,
			"force_reply", forceReply,
			"trigger_reason", triggerReason,
			"trigger_type", triggerType,
			"model_pool", meta.ModelPool,
			"candidate_models", joinList(meta.CandidateModels),
			"skipped_models", joinList(meta.SkippedModels),
			"tried_models", joinList(meta.TriedModels),
			"selected_model", meta.SelectedModel,
			"circuit_key", meta.CircuitKey,
			"provider_circuit_key", meta.ProviderCircuitKey,
			"input_language", meta.InputLanguage,
			"llm_attempted", true,
			"request_prompt_excerpt", meta.RequestPromptExcerpt,
		)
	}
	e.emitDebug(DebugEvent{
		Event:                "bot.llm.request",
		TraceID:              traceID,
		RoomID:               in.RoomID,
		MessageID:            in.SpeakerMessageID,
		SenderID:             in.SpeakerID,
		SenderName:           in.SpeakerName,
		UsedLLM:              false,
		LLMEnabled:           e.llmEnabled,
		ProviderInitialized:  e.providerReady(),
		ForceReply:           forceReply,
		TriggerReason:        triggerReason,
		TriggerType:          triggerType,
		ModelPool:            meta.ModelPool,
		CandidateModels:      joinList(meta.CandidateModels),
		SkippedModels:        joinList(meta.SkippedModels),
		TriedModels:          joinList(meta.TriedModels),
		SelectedModel:        meta.SelectedModel,
		CircuitKey:           meta.CircuitKey,
		ProviderCircuitKey:   meta.ProviderCircuitKey,
		InputLanguage:        meta.InputLanguage,
		LLMAttempted:         true,
		RequestPromptExcerpt: meta.RequestPromptExcerpt,
	})
}

func (e *Engine) logLLMSuccess(in Input, traceID string, contentLength int, meta generateMeta) {
	if e.logger != nil {
		e.logger.Info("[LLM] bot.llm.success",
			"trace_id", traceID,
			"room_id", in.RoomID,
			"message_id", in.SpeakerMessageID,
			"model", meta.Model,
			"provider", meta.Provider,
			"base_url", meta.BaseURL,
			"model_pool", meta.ModelPool,
			"candidate_models", joinList(meta.CandidateModels),
			"skipped_models", joinList(meta.SkippedModels),
			"tried_models", joinList(meta.TriedModels),
			"selected_model", meta.SelectedModel,
			"circuit_key", meta.CircuitKey,
			"provider_circuit_key", meta.ProviderCircuitKey,
			"model_failures", joinList(meta.ModelFailures),
			"input_language", meta.InputLanguage,
			"output_language", meta.OutputLanguage,
			"language_mismatch", meta.LanguageMismatch,
			"reject_reason", meta.RejectReason,
			"http_status", meta.HTTPStatus,
			"latency_ms", meta.LatencyMs,
			"prompt_tokens", meta.PromptTokens,
			"completion_tokens", meta.CompletionTokens,
			"total_tokens", meta.TotalTokens,
			"choices_count", meta.ChoicesCount,
			"choice_summary", meta.ChoiceSummary,
			"content_preview", meta.ContentPreview,
			"displayable_content_found", meta.DisplayableFound,
			"reasoning_only_response", meta.ReasoningOnly,
			"llm_attempted", meta.LLMAttempted,
			"raw_response_preview", meta.RawResponsePreview,
			"raw_response_headers", meta.RawResponseHeaders,
			"content_length", contentLength,
			"used_llm", true,
		)
	}
	e.emitDebug(DebugEvent{
		Event:               "bot.llm.success",
		TraceID:             traceID,
		RoomID:              in.RoomID,
		MessageID:           in.SpeakerMessageID,
		SenderID:            in.SpeakerID,
		SenderName:          in.SpeakerName,
		UsedLLM:             true,
		LLMEnabled:          e.llmEnabled,
		ProviderInitialized: e.providerReady(),
		Model:               meta.Model,
		Provider:            meta.Provider,
		BaseURL:             meta.BaseURL,
		ModelPool:           meta.ModelPool,
		CandidateModels:     joinList(meta.CandidateModels),
		SkippedModels:       joinList(meta.SkippedModels),
		TriedModels:         joinList(meta.TriedModels),
		SelectedModel:       meta.SelectedModel,
		CircuitKey:          meta.CircuitKey,
		ProviderCircuitKey:  meta.ProviderCircuitKey,
		ModelFailures:       joinList(meta.ModelFailures),
		InputLanguage:       meta.InputLanguage,
		OutputLanguage:      meta.OutputLanguage,
		LanguageMismatch:    meta.LanguageMismatch,
		RejectReason:        meta.RejectReason,
		HTTPStatus:          meta.HTTPStatus,
		LatencyMs:           meta.LatencyMs,
		PromptTokens:        meta.PromptTokens,
		CompletionTokens:    meta.CompletionTokens,
		TotalTokens:         meta.TotalTokens,
		ChoicesCount:        meta.ChoicesCount,
		ChoiceSummary:       meta.ChoiceSummary,
		ContentPreview:      meta.ContentPreview,
		DisplayableFound:    meta.DisplayableFound,
		ReasoningOnly:       meta.ReasoningOnly,
		LLMAttempted:        meta.LLMAttempted,
		RawResponsePreview:  meta.RawResponsePreview,
	})
}

func (e *Engine) logLLMError(in Input, traceID, fallbackReason, errMsg string, meta generateMeta) {
	if e.logger != nil {
		e.logger.Error("[LLM] bot.llm.error",
			"trace_id", traceID,
			"room_id", in.RoomID,
			"message_id", in.SpeakerMessageID,
			"used_llm", false,
			"llm_enabled", e.llmEnabled,
			"provider_initialized", e.providerReady(),
			"model", meta.Model,
			"provider", meta.Provider,
			"base_url", meta.BaseURL,
			"model_pool", meta.ModelPool,
			"candidate_models", joinList(meta.CandidateModels),
			"skipped_models", joinList(meta.SkippedModels),
			"tried_models", joinList(meta.TriedModels),
			"selected_model", meta.SelectedModel,
			"circuit_key", meta.CircuitKey,
			"provider_circuit_key", meta.ProviderCircuitKey,
			"model_failures", joinList(meta.ModelFailures),
			"last_error_type", meta.LastErrorType,
			"input_language", meta.InputLanguage,
			"output_language", meta.OutputLanguage,
			"language_mismatch", meta.LanguageMismatch,
			"reject_reason", meta.RejectReason,
			"circuit_scope", meta.CircuitScope,
			"circuit_open_until", meta.CircuitOpenUntil,
			"model_circuit_open_until", meta.ModelCircuitOpenUntil,
			"provider_circuit_open_until", meta.ProviderCircuitOpenUntil,
			"api_key_present", e.apiKeyPresent,
			"http_status", meta.HTTPStatus,
			"latency_ms", meta.LatencyMs,
			"prompt_tokens", meta.PromptTokens,
			"completion_tokens", meta.CompletionTokens,
			"total_tokens", meta.TotalTokens,
			"choices_count", meta.ChoicesCount,
			"choice_summary", meta.ChoiceSummary,
			"content_preview", meta.ContentPreview,
			"displayable_content_found", meta.DisplayableFound,
			"reasoning_only_response", meta.ReasoningOnly,
			"llm_attempted", meta.LLMAttempted,
			"raw_response_preview", meta.RawResponsePreview,
			"raw_response_headers", meta.RawResponseHeaders,
			"fallback_reason", fallbackReason,
			"error_message", errMsg,
		)
	}
	e.emitDebug(DebugEvent{
		Event:                    "bot.llm.error",
		TraceID:                  traceID,
		RoomID:                   in.RoomID,
		MessageID:                in.SpeakerMessageID,
		SenderID:                 in.SpeakerID,
		SenderName:               in.SpeakerName,
		UsedLLM:                  false,
		LLMEnabled:               e.llmEnabled,
		ProviderInitialized:      e.providerReady(),
		Model:                    meta.Model,
		Provider:                 meta.Provider,
		BaseURL:                  meta.BaseURL,
		ModelPool:                meta.ModelPool,
		CandidateModels:          joinList(meta.CandidateModels),
		SkippedModels:            joinList(meta.SkippedModels),
		TriedModels:              joinList(meta.TriedModels),
		SelectedModel:            meta.SelectedModel,
		CircuitKey:               meta.CircuitKey,
		ProviderCircuitKey:       meta.ProviderCircuitKey,
		ModelFailures:            joinList(meta.ModelFailures),
		LastErrorType:            meta.LastErrorType,
		InputLanguage:            meta.InputLanguage,
		OutputLanguage:           meta.OutputLanguage,
		LanguageMismatch:         meta.LanguageMismatch,
		RejectReason:             meta.RejectReason,
		CircuitScope:             meta.CircuitScope,
		CircuitOpenUntil:         meta.CircuitOpenUntil,
		ModelCircuitOpenUntil:    meta.ModelCircuitOpenUntil,
		ProviderCircuitOpenUntil: meta.ProviderCircuitOpenUntil,
		APIKeyPresent:            e.apiKeyPresent,
		HTTPStatus:               meta.HTTPStatus,
		LatencyMs:                meta.LatencyMs,
		PromptTokens:             meta.PromptTokens,
		CompletionTokens:         meta.CompletionTokens,
		TotalTokens:              meta.TotalTokens,
		ChoicesCount:             meta.ChoicesCount,
		ChoiceSummary:            meta.ChoiceSummary,
		ContentPreview:           meta.ContentPreview,
		DisplayableFound:         meta.DisplayableFound,
		ReasoningOnly:            meta.ReasoningOnly,
		LLMAttempted:             meta.LLMAttempted,
		RawResponsePreview:       meta.RawResponsePreview,
		RequestPromptExcerpt:     meta.RequestPromptExcerpt,
		FallbackReason:           fallbackReason,
		ErrorMessage:             errMsg,
	})
}

func (e *Engine) logLLMFallback(in Input, traceID, fallbackReason, errMsg string, meta generateMeta) {
	if e.logger != nil {
		e.logger.Warn("[LLM] bot.llm.fallback",
			"trace_id", traceID,
			"room_id", in.RoomID,
			"message_id", in.SpeakerMessageID,
			"fallback_reason", fallbackReason,
			"model_pool", meta.ModelPool,
			"candidate_models", joinList(meta.CandidateModels),
			"skipped_models", joinList(meta.SkippedModels),
			"tried_models", joinList(meta.TriedModels),
			"selected_model", meta.SelectedModel,
			"circuit_key", meta.CircuitKey,
			"provider_circuit_key", meta.ProviderCircuitKey,
			"model_failures", joinList(meta.ModelFailures),
			"last_error_type", meta.LastErrorType,
			"input_language", meta.InputLanguage,
			"output_language", meta.OutputLanguage,
			"language_mismatch", meta.LanguageMismatch,
			"reject_reason", meta.RejectReason,
			"circuit_scope", meta.CircuitScope,
			"circuit_open_until", meta.CircuitOpenUntil,
			"model_circuit_open_until", meta.ModelCircuitOpenUntil,
			"provider_circuit_open_until", meta.ProviderCircuitOpenUntil,
			"http_status", meta.HTTPStatus,
			"latency_ms", meta.LatencyMs,
			"choices_count", meta.ChoicesCount,
			"choice_summary", meta.ChoiceSummary,
			"content_preview", meta.ContentPreview,
			"displayable_content_found", meta.DisplayableFound,
			"reasoning_only_response", meta.ReasoningOnly,
			"llm_attempted", meta.LLMAttempted,
			"raw_response_preview", meta.RawResponsePreview,
			"error_message", errMsg,
			"used_llm", false,
		)
	}
	e.emitDebug(DebugEvent{
		Event:                    "bot.llm.fallback",
		TraceID:                  traceID,
		RoomID:                   in.RoomID,
		MessageID:                in.SpeakerMessageID,
		SenderID:                 in.SpeakerID,
		SenderName:               in.SpeakerName,
		UsedLLM:                  false,
		LLMEnabled:               e.llmEnabled,
		ProviderInitialized:      e.providerReady(),
		Model:                    meta.Model,
		Provider:                 meta.Provider,
		BaseURL:                  meta.BaseURL,
		ModelPool:                meta.ModelPool,
		CandidateModels:          joinList(meta.CandidateModels),
		SkippedModels:            joinList(meta.SkippedModels),
		TriedModels:              joinList(meta.TriedModels),
		SelectedModel:            meta.SelectedModel,
		CircuitKey:               meta.CircuitKey,
		ProviderCircuitKey:       meta.ProviderCircuitKey,
		ModelFailures:            joinList(meta.ModelFailures),
		LastErrorType:            meta.LastErrorType,
		InputLanguage:            meta.InputLanguage,
		OutputLanguage:           meta.OutputLanguage,
		LanguageMismatch:         meta.LanguageMismatch,
		RejectReason:             meta.RejectReason,
		CircuitScope:             meta.CircuitScope,
		CircuitOpenUntil:         meta.CircuitOpenUntil,
		ModelCircuitOpenUntil:    meta.ModelCircuitOpenUntil,
		ProviderCircuitOpenUntil: meta.ProviderCircuitOpenUntil,
		HTTPStatus:               meta.HTTPStatus,
		LatencyMs:                meta.LatencyMs,
		ChoicesCount:             meta.ChoicesCount,
		ChoiceSummary:            meta.ChoiceSummary,
		ContentPreview:           meta.ContentPreview,
		DisplayableFound:         meta.DisplayableFound,
		ReasoningOnly:            meta.ReasoningOnly,
		LLMAttempted:             meta.LLMAttempted,
		RawResponsePreview:       meta.RawResponsePreview,
		RequestPromptExcerpt:     meta.RequestPromptExcerpt,
		FallbackReason:           fallbackReason,
		ErrorMessage:             errMsg,
	})
}

func (e *Engine) logReplyDone(in Input, out Output) {
	if e.logger != nil {
		e.logger.Info("[BOT] bot.reply.done",
			"room_id", in.RoomID,
			"message_id", in.SpeakerMessageID,
			"trace_id", out.TraceID,
			"reply_source", out.ReplySource,
			"provider", out.Provider,
			"llm_model", out.LLMModel,
			"fallback_reason", out.FallbackReason,
			"http_status", out.HTTPStatus,
			"latency_ms", out.LatencyMs,
			"prompt_tokens", out.PromptTokens,
			"completion_tokens", out.CompletionTokens,
			"total_tokens", out.TotalTokens,
			"trigger_reason", out.TriggerReason,
			"trigger_type", out.TriggerType,
			"absurdity_score", out.AbsurdityScore,
			"risk_score", out.RiskScore,
			"reply_mode", out.ReplyMode,
			"model_pool", out.ModelPool,
			"candidate_models", joinList(out.CandidateModels),
			"skipped_models", joinList(out.SkippedModels),
			"tried_models", joinList(out.TriedModels),
			"selected_model", out.SelectedModel,
			"model_failures", joinList(out.ModelFailures),
			"last_error_type", out.LastErrorType,
			"input_language", out.InputLanguage,
			"output_language", out.OutputLanguage,
			"language_mismatch", out.LanguageMismatch,
			"reject_reason", out.RejectReason,
			"circuit_open_until", out.CircuitOpenUntil,
			"bot_reply_skipped", out.BotReplySkipped,
			"llm_attempted", out.LLMAttempted,
			"llm_enabled", out.LLMEnabled,
			"provider_initialized", out.ProviderInitialized,
			"api_key_present", out.APIKeyPresent,
			"displayable_content_found", out.DisplayableFound,
			"reasoning_only_response", out.ReasoningOnly,
			"preview", trimContent(out.Content),
		)
	}
	e.emitDebug(DebugEvent{
		Event:                "bot.reply.done",
		RoomID:               in.RoomID,
		MessageID:            in.SpeakerMessageID,
		SenderID:             in.SpeakerID,
		SenderName:           in.SpeakerName,
		TraceID:              out.TraceID,
		ReplySource:          out.ReplySource,
		Provider:             out.Provider,
		FallbackReason:       out.FallbackReason,
		TriggerReason:        out.TriggerReason,
		TriggerType:          out.TriggerType,
		ForceReply:           out.ForceReply,
		HypeScore:            out.HypeScore,
		AbsurdityScore:       out.AbsurdityScore,
		RiskScore:            out.RiskScore,
		ReplyMode:            out.ReplyMode,
		ModelPool:            out.ModelPool,
		CandidateModels:      joinList(out.CandidateModels),
		SkippedModels:        joinList(out.SkippedModels),
		TriedModels:          joinList(out.TriedModels),
		SelectedModel:        out.SelectedModel,
		ModelFailures:        joinList(out.ModelFailures),
		LastErrorType:        out.LastErrorType,
		InputLanguage:        out.InputLanguage,
		OutputLanguage:       out.OutputLanguage,
		LanguageMismatch:     out.LanguageMismatch,
		RejectReason:         out.RejectReason,
		CircuitOpenUntil:     out.CircuitOpenUntil,
		BotReplySkipped:      out.BotReplySkipped,
		LLMAttempted:         out.LLMAttempted,
		UsedLLM:              out.UsedLLM,
		LLMEnabled:           out.LLMEnabled,
		ProviderInitialized:  out.ProviderInitialized,
		Model:                out.LLMModel,
		ResponseText:         out.Content,
		HTTPStatus:           out.HTTPStatus,
		LatencyMs:            out.LatencyMs,
		PromptTokens:         out.PromptTokens,
		CompletionTokens:     out.CompletionTokens,
		TotalTokens:          out.TotalTokens,
		RequestPromptExcerpt: out.RequestPromptExcerpt,
		APIKeyPresent:        out.APIKeyPresent,
		ErrorMessage:         out.ErrorMessage,
		DisplayableFound:     out.DisplayableFound,
		ReasoningOnly:        out.ReasoningOnly,
	})
}

func (e *Engine) recordLLMSuccess() {
	now := time.Now()
	e.statusMu.Lock()
	defer e.statusMu.Unlock()
	e.lastLLMSuccessAt = &now
	e.lastLLMError = ""
}

func (e *Engine) recordFallback(reason, errMsg string) {
	e.statusMu.Lock()
	defer e.statusMu.Unlock()
	e.lastFallbackReason = reason
	e.lastLLMError = errMsg
}

func (e *Engine) emitDebug(event DebugEvent) {
	e.observerMu.RLock()
	observer := e.debugObserver
	e.observerMu.RUnlock()
	if observer == nil {
		return
	}
	if strings.TrimSpace(event.Time) == "" {
		event.Time = time.Now().Format(time.RFC3339Nano)
	}
	observer(event)
}
