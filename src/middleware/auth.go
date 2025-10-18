package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"www.github.com/Wanderer0074348/HybridLM/src/auth"
)

type AuthMiddleware struct {
	sessionStore *auth.SessionStore
	userStore    *auth.UserStore
}

func NewAuthMiddleware(sessionStore *auth.SessionStore, userStore *auth.UserStore) *AuthMiddleware {
	return &AuthMiddleware{
		sessionStore: sessionStore,
		userStore:    userStore,
	}
}

func (m *AuthMiddleware) RequireAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		sessionID, err := c.Cookie("session_id")
		if err != nil {
			authHeader := c.GetHeader("Authorization")
			if authHeader != "" && strings.HasPrefix(authHeader, "Bearer ") {
				sessionID = strings.TrimPrefix(authHeader, "Bearer ")
			}
		}

		if sessionID == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
			c.Abort()
			return
		}

		session, err := m.sessionStore.GetSession(c.Request.Context(), sessionID)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired session"})
			c.Abort()
			return
		}

		user, err := m.userStore.GetUser(c.Request.Context(), session.UserID)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found"})
			c.Abort()
			return
		}

		c.Set("user", user)
		c.Set("session", session)

		if err := m.sessionStore.RefreshSession(c.Request.Context(), sessionID); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to refresh session"})
			c.Abort()
			return
		}

		c.Next()
	}
}

func (m *AuthMiddleware) OptionalAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		sessionID, err := c.Cookie("session_id")
		if err != nil {
			authHeader := c.GetHeader("Authorization")
			if authHeader != "" && strings.HasPrefix(authHeader, "Bearer ") {
				sessionID = strings.TrimPrefix(authHeader, "Bearer ")
			}
		}

		if sessionID == "" {
			c.Next()
			return
		}

		session, err := m.sessionStore.GetSession(c.Request.Context(), sessionID)
		if err != nil {
			c.Next()
			return
		}

		user, err := m.userStore.GetUser(c.Request.Context(), session.UserID)
		if err != nil {
			c.Next()
			return
		}

		c.Set("user", user)
		c.Set("session", session)
		m.sessionStore.RefreshSession(c.Request.Context(), sessionID)

		c.Next()
	}
}
