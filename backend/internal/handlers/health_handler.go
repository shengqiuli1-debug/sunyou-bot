package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func (h *Handler) Health(c *gin.Context) {
	if err := h.db.PingContext(c.Request.Context()); err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"status": "db_down", "error": err.Error()})
		return
	}
	if err := h.redis.Ping(c.Request.Context()).Err(); err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"status": "redis_down", "error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}
