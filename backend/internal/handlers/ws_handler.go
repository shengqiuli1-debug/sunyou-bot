package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"sunyou-bot/backend/internal/hub"
	"sunyou-bot/backend/internal/middleware"
)

func (h *Handler) RoomWS(c *gin.Context) {
	user := middleware.CurrentUser(c)
	roomID := c.Param("id")

	member, err := h.rooms.GetMember(c.Request.Context(), roomID, user.ID)
	if err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "join room first"})
		return
	}

	conn, err := h.upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}

	roomHub := h.hub.GetOrCreate(roomID)
	client := hub.NewClient(conn, roomID, user.ID, user.Nickname, string(member.Identity))

	messages, _ := h.rooms.LatestMessages(c.Request.Context(), roomID, 40)
	members, _ := h.rooms.ListMembers(c.Request.Context(), roomID)
	room, _ := h.rooms.GetRoom(c.Request.Context(), roomID)

	_ = conn.SetWriteDeadline(time.Now().Add(3 * time.Second))
	_ = conn.WriteJSON(map[string]any{
		"type": "bootstrap",
		"payload": gin.H{
			"room":     room,
			"member":   member,
			"members":  members,
			"messages": messages,
		},
		"timestamp": time.Now().Format(time.RFC3339),
	})

	roomHub.ServeClient(client)

	h.hub.Broadcast(roomID, "system", gin.H{"content": user.Nickname + " 已进入房间", "userId": user.ID})
}
