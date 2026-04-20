package handlers

import (
	"database/sql"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"sunyou-bot/backend/internal/middleware"
	"sunyou-bot/backend/internal/services"
)

func (h *Handler) DebugLLMStatus(c *gin.Context) {
	c.JSON(http.StatusOK, h.rooms.DebugLLMStatus())
}

func (h *Handler) DebugRoomBotAudits(c *gin.Context) {
	if h.audits == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "bot audit service disabled"})
		return
	}
	user := middleware.CurrentUser(c)
	roomID := c.Param("roomId")
	if _, err := h.rooms.GetMember(c.Request.Context(), roomID, user.ID); err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "join room first"})
		return
	}
	page, _ := strconv.Atoi(strings.TrimSpace(c.Query("page")))
	pageSize, _ := strconv.Atoi(strings.TrimSpace(c.Query("pageSize")))
	replySource := strings.TrimSpace(c.Query("replySource"))
	botRole := strings.TrimSpace(c.Query("botRole"))

	items, total, err := h.audits.ListRoomAudits(c.Request.Context(), roomID, services.BotAuditListFilter{
		Page:        page,
		PageSize:    pageSize,
		ReplySource: replySource,
		BotRole:     botRole,
	})
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	c.JSON(http.StatusOK, gin.H{
		"items":    items,
		"total":    total,
		"page":     page,
		"pageSize": pageSize,
	})
}

func (h *Handler) DebugMessageBotAudit(c *gin.Context) {
	if h.audits == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "bot audit service disabled"})
		return
	}
	user := middleware.CurrentUser(c)
	messageID, err := strconv.ParseInt(strings.TrimSpace(c.Param("messageId")), 10, 64)
	if err != nil || messageID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid message id"})
		return
	}
	item, err := h.audits.GetByMessageID(c.Request.Context(), messageID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "bot audit not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if _, err := h.rooms.GetMember(c.Request.Context(), item.RoomID, user.ID); err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "join room first"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"item": item})
}

func (h *Handler) DebugBotAuditByID(c *gin.Context) {
	if h.audits == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "bot audit service disabled"})
		return
	}
	user := middleware.CurrentUser(c)
	id, err := strconv.ParseInt(strings.TrimSpace(c.Param("id")), 10, 64)
	if err != nil || id <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid bot audit id"})
		return
	}
	item, err := h.audits.GetByID(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "bot audit not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if _, err := h.rooms.GetMember(c.Request.Context(), item.RoomID, user.ID); err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "join room first"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"item": item})
}

func (h *Handler) DebugLLMTest(c *gin.Context) {
	user := middleware.CurrentUser(c)
	roomID := c.Param("id")

	msg, err := h.rooms.RunLLMTest(c.Request.Context(), roomID, user.ID)
	if err != nil {
		switch {
		case errors.Is(err, services.ErrNotOwner):
			c.JSON(http.StatusForbidden, gin.H{"error": "only owner can run llm test"})
		case errors.Is(err, services.ErrRoomEnded):
			c.JSON(http.StatusBadRequest, gin.H{"error": "room ended"})
		default:
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		}
		return
	}
	if msg != nil {
		h.hub.Broadcast(roomID, "chat", msg)
	}
	c.JSON(http.StatusOK, gin.H{"message": msg})
}
