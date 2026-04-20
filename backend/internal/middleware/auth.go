package middleware

import (
	"database/sql"
	"net/http"

	"github.com/gin-gonic/gin"

	"sunyou-bot/backend/internal/models"
	"sunyou-bot/backend/internal/services"
)

const userContextKey = "auth_user"

func RequireAuth(userSvc *services.UserService) gin.HandlerFunc {
	return func(c *gin.Context) {
		token := c.GetHeader("X-User-Token")
		if token == "" {
			token = c.Query("token")
		}
		if token == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing user token"})
			return
		}
		user, err := userSvc.GetByToken(c.Request.Context(), token)
		if err != nil {
			if err == sql.ErrNoRows {
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid user token"})
				return
			}
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "load user failed"})
			return
		}
		c.Set(userContextKey, user)
		c.Next()
	}
}

func CurrentUser(c *gin.Context) *models.User {
	v, ok := c.Get(userContextKey)
	if !ok {
		return nil
	}
	u, _ := v.(*models.User)
	return u
}
