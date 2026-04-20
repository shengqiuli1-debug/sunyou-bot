package handlers

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"sunyou-bot/backend/internal/middleware"
)

func (h *Handler) CreateGuest(c *gin.Context) {
	var req struct {
		Nickname string `json:"nickname"`
	}
	_ = c.ShouldBindJSON(&req)
	user, err := h.users.CreateGuest(c.Request.Context(), strings.TrimSpace(req.Nickname))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"user":  user,
		"token": user.Token,
	})
}

func (h *Handler) Me(c *gin.Context) {
	user := middleware.CurrentUser(c)
	c.JSON(http.StatusOK, gin.H{"user": user})
}

func (h *Handler) GetPoints(c *gin.Context) {
	user := middleware.CurrentUser(c)
	points, err := h.points.EnsureBalance(c.Request.Context(), user.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"points": points})
}

func (h *Handler) MockRecharge(c *gin.Context) {
	user := middleware.CurrentUser(c)
	var req struct {
		Amount  int    `json:"amount"`
		Channel string `json:"channel"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		req.Amount = 20
	}
	if req.Channel == "" {
		req.Channel = "mock_wechat"
	}
	balance, err := h.points.Recharge(c.Request.Context(), user.ID, req.Amount, req.Channel)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"balance": balance})
}

func (h *Handler) PointLedger(c *gin.Context) {
	user := middleware.CurrentUser(c)
	ledger, err := h.users.ListPointLedger(c.Request.Context(), user.ID, 50)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": ledger})
}

func (h *Handler) RoomRecords(c *gin.Context) {
	user := middleware.CurrentUser(c)
	rooms, err := h.users.ListRoomRecords(c.Request.Context(), user.ID, 50)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": rooms})
}
