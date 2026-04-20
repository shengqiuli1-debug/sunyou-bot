package handlers

import (
	"database/sql"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	redis "github.com/redis/go-redis/v9"
	"log/slog"

	"sunyou-bot/backend/internal/hub"
	"sunyou-bot/backend/internal/middleware"
	"sunyou-bot/backend/internal/services"
)

type Handler struct {
	logger        *slog.Logger
	db            *sql.DB
	redis         *redis.Client
	users         *services.UserService
	points        *services.PointService
	rooms         *services.RoomService
	audits        *services.BotAuditService
	hub           *hub.Manager
	upgrader      websocket.Upgrader
	allowedOrigin string
}

func New(
	logger *slog.Logger,
	db *sql.DB,
	redisCli *redis.Client,
	users *services.UserService,
	points *services.PointService,
	rooms *services.RoomService,
	audits *services.BotAuditService,
	hubMgr *hub.Manager,
	allowedOrigin string,
) *Handler {
	return &Handler{
		logger:        logger,
		db:            db,
		redis:         redisCli,
		users:         users,
		points:        points,
		rooms:         rooms,
		audits:        audits,
		hub:           hubMgr,
		allowedOrigin: allowedOrigin,
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				origin := r.Header.Get("Origin")
				if origin == "" {
					return true
				}
				return origin == allowedOrigin || allowedOrigin == "*"
			},
		},
	}
}

func (h *Handler) Router() *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		if origin == "" {
			origin = h.allowedOrigin
		}
		if h.allowedOrigin == "*" || origin == h.allowedOrigin {
			c.Header("Access-Control-Allow-Origin", origin)
			c.Header("Vary", "Origin")
		}
		c.Header("Access-Control-Allow-Headers", "Content-Type, X-User-Token")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, OPTIONS")
		c.Header("Access-Control-Allow-Credentials", "true")
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	})
	r.Use(func(c *gin.Context) {
		started := time.Now()
		c.Next()
		h.logger.Info("http",
			"method", c.Request.Method,
			"path", c.Request.URL.Path,
			"status", c.Writer.Status(),
			"latency", time.Since(started).String(),
		)
	})

	r.GET("/healthz", h.Health)
	r.HEAD("/healthz", h.Health)

	api := r.Group("/api/v1")
	{
		api.POST("/users/guest", h.CreateGuest)
		api.GET("/rooms/by-share/:code", h.GetRoomByShareCode)

		auth := api.Group("")
		auth.Use(middleware.RequireAuth(h.users))
		{
			auth.GET("/users/me", h.Me)
			auth.GET("/points", h.GetPoints)
			auth.POST("/points/recharge", h.MockRecharge)
			auth.GET("/points/ledger", h.PointLedger)
			auth.GET("/rooms/records", h.RoomRecords)
			auth.GET("/debug/llm-status", h.DebugLLMStatus)
			auth.GET("/debug/rooms/:roomId/bot-audits", h.DebugRoomBotAudits)
			auth.GET("/debug/messages/:messageId/bot-audit", h.DebugMessageBotAudit)
			auth.GET("/debug/bot-audits/:id", h.DebugBotAuditByID)

			auth.POST("/rooms", h.CreateRoom)
			auth.GET("/rooms/:id", h.GetRoom)
			auth.POST("/rooms/:id/join", h.JoinRoom)
			auth.POST("/rooms/:id/identity", h.UpdateIdentity)
			auth.POST("/rooms/:id/control", h.RoomControl)
			auth.POST("/rooms/:id/debug/llm-test", h.DebugLLMTest)
			auth.POST("/rooms/:id/end", h.EndRoom)
			auth.GET("/rooms/:id/messages", h.RoomMessages)
			auth.GET("/rooms/:id/report", h.RoomReport)
			auth.POST("/rooms/:id/report-abuse", h.ReportAbuse)
		}
	}

	// Debug-compatible aliases for manual curl checks without version prefix.
	apiCompat := r.Group("/api")
	apiCompat.Use(middleware.RequireAuth(h.users))
	{
		apiCompat.GET("/debug/llm-status", h.DebugLLMStatus)
		apiCompat.GET("/debug/rooms/:roomId/bot-audits", h.DebugRoomBotAudits)
		apiCompat.GET("/debug/messages/:messageId/bot-audit", h.DebugMessageBotAudit)
		apiCompat.GET("/debug/bot-audits/:id", h.DebugBotAuditByID)
		apiCompat.POST("/rooms/:id/debug/llm-test", h.DebugLLMTest)
	}

	ws := r.Group("/ws")
	ws.Use(middleware.RequireAuth(h.users))
	ws.GET("/rooms/:id", h.RoomWS)

	return r
}
