package handlers

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"sunyou-bot/backend/internal/middleware"
	"sunyou-bot/backend/internal/models"
	"sunyou-bot/backend/internal/services"
)

func (h *Handler) CreateRoom(c *gin.Context) {
	user := middleware.CurrentUser(c)
	raw, err := io.ReadAll(c.Request.Body)
	if err != nil {
		if h.logger != nil {
			h.logger.Info("create room request body read failed", "user_id", user.ID, "error", err.Error())
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}
	req, err := parseCreateRoomInput(raw)
	if err != nil {
		if h.logger != nil {
			h.logger.Info("create room body parse fallback to defaults",
				"user_id", user.ID,
				"parse_error", err.Error(),
				"raw_body", string(raw),
			)
		}
		req = services.CreateRoomInput{
			DurationMinutes: 5,
			BotRole:         models.RoleJudge,
			FireLevel:       models.FireMedium,
			GenerateReport:  true,
		}
	}
	room, freeUsed, balance, err := h.rooms.CreateRoom(c.Request.Context(), user.ID, req)
	if err != nil {
		if errors.Is(err, services.ErrInsufficientPoints) {
			cost := services.RoomCost(req.DurationMinutes, req.FireLevel)
			if h.logger != nil {
				h.logger.Info("create room rejected: insufficient points",
					"user_id", user.ID,
					"duration_minutes", req.DurationMinutes,
					"fire_level", req.FireLevel,
					"required_cost", cost,
					"current_balance", balance,
					"error", err.Error(),
				)
			}
			c.JSON(http.StatusPaymentRequired, gin.H{
				"error":          "points not enough",
				"code":           "POINTS_NOT_ENOUGH",
				"requiredCost":   cost,
				"currentBalance": balance,
			})
			return
		}
		if errors.Is(err, services.ErrInvalidRoomOptions) {
			if h.logger != nil {
				h.logger.Info("create room rejected: invalid room options",
					"user_id", user.ID,
					"duration_minutes", req.DurationMinutes,
					"bot_role", req.BotRole,
					"fire_level", req.FireLevel,
					"error", err.Error(),
				)
			}
			c.JSON(http.StatusBadRequest, gin.H{"error": "duration only supports 5/15/30"})
			return
		}
		if h.logger != nil {
			h.logger.Error("create room failed", "user_id", user.ID, "error", err.Error())
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if h.logger != nil {
		h.logger.Info("create room success",
			"user_id", user.ID,
			"room_id", room.ID,
			"duration_minutes", req.DurationMinutes,
			"bot_role", req.BotRole,
			"fire_level", req.FireLevel,
			"remaining_point_balance", balance,
			"free_trial_used", freeUsed,
		)
	}

	_, _ = h.rooms.JoinRoom(c.Request.Context(), room.ID, user.ID, models.IdentityNormal, true)

	c.JSON(http.StatusOK, gin.H{
		"room":                  room,
		"freeTrialUsed":         freeUsed,
		"remainingPointBalance": balance,
	})
}

func parseCreateRoomInput(raw []byte) (services.CreateRoomInput, error) {
	type strictInput struct {
		DurationMinutes int              `json:"durationMinutes"`
		BotRole         models.BotRole   `json:"botRole"`
		FireLevel       models.FireLevel `json:"fireLevel"`
		GenerateReport  bool             `json:"generateReport"`
	}
	out := services.CreateRoomInput{}
	if len(strings.TrimSpace(string(raw))) == 0 {
		return out, errors.New("empty body")
	}

	var strict strictInput
	if err := json.Unmarshal(raw, &strict); err == nil && strict.DurationMinutes > 0 {
		out.DurationMinutes = strict.DurationMinutes
		out.BotRole = strict.BotRole
		out.FireLevel = strict.FireLevel
		out.GenerateReport = strict.GenerateReport
		return normalizeCreateRoomInput(out)
	}

	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		return out, errors.New("malformed json")
	}

	duration, err := readIntField(m,
		"durationMinutes", "duration_minutes", "duration", "durationMin", "duration_min", "minutes")
	if err != nil {
		return out, fmt.Errorf("duration invalid: %w", err)
	}
	out.DurationMinutes = duration

	role := readStringField(m, "botRole", "bot_role", "role")
	if role != "" {
		out.BotRole = normalizeBotRole(role)
	}

	fire := readStringField(m, "fireLevel", "fire_level", "firepower", "firepowerLevel", "firepower_level")
	if fire != "" {
		out.FireLevel = normalizeFireLevel(fire)
	}

	if v, ok := readBoolField(m, "generateReport", "generate_report", "withReport", "with_report"); ok {
		out.GenerateReport = v
	}

	return normalizeCreateRoomInput(out)
}

func normalizeCreateRoomInput(in services.CreateRoomInput) (services.CreateRoomInput, error) {
	if in.DurationMinutes <= 0 {
		in.DurationMinutes = 5
	}
	if in.DurationMinutes != 5 && in.DurationMinutes != 15 && in.DurationMinutes != 30 {
		return in, errors.New("durationMinutes must be 5/15/30")
	}
	if in.BotRole == "" {
		in.BotRole = models.RoleJudge
	}
	if in.FireLevel == "" {
		in.FireLevel = models.FireMedium
	}
	return in, nil
}

func readStringField(m map[string]any, keys ...string) string {
	for _, k := range keys {
		v, ok := m[k]
		if !ok || v == nil {
			continue
		}
		s := strings.TrimSpace(fmt.Sprint(v))
		if s != "" && s != "<nil>" {
			return s
		}
	}
	return ""
}

func readIntField(m map[string]any, keys ...string) (int, error) {
	for _, k := range keys {
		v, ok := m[k]
		if !ok || v == nil {
			continue
		}
		switch n := v.(type) {
		case float64:
			return int(n), nil
		case int:
			return n, nil
		case json.Number:
			i, err := n.Int64()
			if err != nil {
				return 0, errors.New("not a valid number")
			}
			return int(i), nil
		case string:
			s := strings.TrimSpace(n)
			if s == "" {
				continue
			}
			i, err := strconv.Atoi(s)
			if err != nil {
				return 0, errors.New("not a valid integer string")
			}
			return i, nil
		default:
			return 0, errors.New("unsupported type")
		}
	}
	return 0, errors.New("missing field")
}

func readBoolField(m map[string]any, keys ...string) (bool, bool) {
	for _, k := range keys {
		v, ok := m[k]
		if !ok || v == nil {
			continue
		}
		switch b := v.(type) {
		case bool:
			return b, true
		case string:
			s := strings.TrimSpace(strings.ToLower(b))
			if s == "" {
				continue
			}
			if s == "1" || s == "true" || s == "yes" || s == "on" {
				return true, true
			}
			if s == "0" || s == "false" || s == "no" || s == "off" {
				return false, true
			}
		}
	}
	return false, false
}

func normalizeBotRole(v string) models.BotRole {
	s := strings.ToLower(strings.TrimSpace(v))
	switch {
	case strings.Contains(s, "judge"), strings.Contains(s, "裁判"):
		return models.RoleJudge
	case strings.Contains(s, "narr"), strings.Contains(s, "旁白"):
		return models.RoleNarrator
	case strings.Contains(s, "npc"), strings.Contains(s, "损友"):
		return models.RoleNpc
	default:
		return models.BotRole(s)
	}
}

func normalizeFireLevel(v string) models.FireLevel {
	s := strings.ToLower(strings.TrimSpace(v))
	switch s {
	case "low", "轻嘴", "light":
		return models.FireLow
	case "medium", "mid", "normal", "阴阳":
		return models.FireMedium
	case "high", "abstract", "crazy", "发疯", "抽象":
		return models.FireHigh
	default:
		return models.FireLevel(s)
	}
}

func (h *Handler) GetRoomByShareCode(c *gin.Context) {
	code := c.Param("code")
	room, err := h.rooms.GetRoomByShareCode(c.Request.Context(), code)
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "room not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"room": room})
}

func (h *Handler) GetRoom(c *gin.Context) {
	user := middleware.CurrentUser(c)
	roomID := c.Param("id")
	room, err := h.rooms.GetRoom(c.Request.Context(), roomID)
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "room not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	members, err := h.rooms.ListMembers(c.Request.Context(), roomID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	self, _ := h.rooms.GetMember(c.Request.Context(), roomID, user.ID)
	c.JSON(http.StatusOK, gin.H{
		"room":    room,
		"members": members,
		"self":    self,
	})
}

func (h *Handler) JoinRoom(c *gin.Context) {
	user := middleware.CurrentUser(c)
	roomID := c.Param("id")
	var req struct {
		Identity      models.Identity `json:"identity"`
		ConfirmTarget bool            `json:"confirmTarget"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}
	member, err := h.rooms.JoinRoom(c.Request.Context(), roomID, user.ID, req.Identity, req.ConfirmTarget)
	if err != nil {
		switch {
		case errors.Is(err, services.ErrTargetNeedConfirm):
			c.JSON(http.StatusBadRequest, gin.H{"error": "target identity needs confirm"})
		case errors.Is(err, services.ErrRoomEnded):
			c.JSON(http.StatusBadRequest, gin.H{"error": "room ended"})
		case errors.Is(err, services.ErrInvalidIdentity):
			c.JSON(http.StatusBadRequest, gin.H{"error": "identity invalid"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}
	members, _ := h.rooms.ListMembers(c.Request.Context(), roomID)
	h.hub.Broadcast(roomID, "member_update", gin.H{"members": members})
	c.JSON(http.StatusOK, gin.H{"member": member, "ws": "/ws/rooms/" + roomID})
}

func (h *Handler) UpdateIdentity(c *gin.Context) {
	user := middleware.CurrentUser(c)
	roomID := c.Param("id")
	var req struct {
		Identity      models.Identity `json:"identity"`
		ConfirmTarget bool            `json:"confirmTarget"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}
	member, err := h.rooms.UpdateIdentity(c.Request.Context(), roomID, user.ID, req.Identity, req.ConfirmTarget)
	if err != nil {
		switch {
		case errors.Is(err, services.ErrTargetNeedConfirm):
			c.JSON(http.StatusBadRequest, gin.H{"error": "target identity needs confirm"})
		case errors.Is(err, services.ErrNotRoomMember):
			c.JSON(http.StatusNotFound, gin.H{"error": "not room member"})
		case errors.Is(err, services.ErrInvalidIdentity):
			c.JSON(http.StatusBadRequest, gin.H{"error": "identity invalid"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}
	members, _ := h.rooms.ListMembers(c.Request.Context(), roomID)
	h.hub.Broadcast(roomID, "member_update", gin.H{"members": members})
	c.JSON(http.StatusOK, gin.H{"member": member})
}

func (h *Handler) RoomControl(c *gin.Context) {
	user := middleware.CurrentUser(c)
	roomID := c.Param("id")
	var req services.HostControlInput
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}
	room, err := h.rooms.HostControl(c.Request.Context(), roomID, user.ID, req)
	if err != nil {
		switch {
		case errors.Is(err, services.ErrNotOwner):
			c.JSON(http.StatusForbidden, gin.H{"error": "only owner can control"})
		default:
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		}
		return
	}
	h.hub.Broadcast(roomID, "control_update", gin.H{"room": room, "action": req.Action, "value": req.Value})
	if strings.EqualFold(req.Action, "end_room") {
		report, _ := h.rooms.GetOrBuildReport(c.Request.Context(), roomID)
		h.hub.Broadcast(roomID, "room_end", gin.H{"roomId": roomID, "report": report})
	}
	c.JSON(http.StatusOK, gin.H{"room": room})
}

func (h *Handler) EndRoom(c *gin.Context) {
	user := middleware.CurrentUser(c)
	roomID := c.Param("id")
	_, err := h.rooms.HostControl(c.Request.Context(), roomID, user.ID, services.HostControlInput{Action: "end_room"})
	if err != nil {
		if errors.Is(err, services.ErrNotOwner) {
			c.JSON(http.StatusForbidden, gin.H{"error": "only owner can end room"})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	report, _ := h.rooms.GetOrBuildReport(c.Request.Context(), roomID)
	h.hub.Broadcast(roomID, "room_end", gin.H{"roomId": roomID, "report": report})
	c.JSON(http.StatusOK, gin.H{"report": report})
}

func (h *Handler) RoomMessages(c *gin.Context) {
	roomID := c.Param("id")
	messages, err := h.rooms.LatestMessages(c.Request.Context(), roomID, 80)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": messages})
}

func (h *Handler) RoomReport(c *gin.Context) {
	roomID := c.Param("id")
	report, err := h.rooms.GetOrBuildReport(c.Request.Context(), roomID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"report": report})
}

func (h *Handler) ReportAbuse(c *gin.Context) {
	user := middleware.CurrentUser(c)
	roomID := c.Param("id")
	var req struct {
		TargetUserID string `json:"targetUserId"`
		Message      string `json:"message"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}
	if strings.TrimSpace(req.Message) == "" {
		req.Message = "用户反馈：内容令人不适"
	}
	if err := h.rooms.SubmitAbuseReport(c.Request.Context(), roomID, user.ID, req.TargetUserID, req.Message); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}
