package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/oauth2"
)

type Handler struct {
	oauthConfig  *oauth2.Config
	stateStore   *StateStore
	sessionStore *SessionStore
	userStore    *UserStore
	config       *Config
}

func NewHandler(
	oauthConfig *oauth2.Config,
	stateStore *StateStore,
	sessionStore *SessionStore,
	userStore *UserStore,
	config *Config,
) *Handler {
	return &Handler{
		oauthConfig:  oauthConfig,
		stateStore:   stateStore,
		sessionStore: sessionStore,
		userStore:    userStore,
		config:       config,
	}
}

func (h *Handler) Login(c *gin.Context) {
	state, err := h.stateStore.GenerateState()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate state"})
		return
	}

	if err := h.stateStore.SaveState(c.Request.Context(), state, 10*time.Minute); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save state"})
		return
	}

	url := h.oauthConfig.AuthCodeURL(state, oauth2.AccessTypeOffline, oauth2.ApprovalForce)
	c.JSON(http.StatusOK, gin.H{"url": url})
}

func (h *Handler) Callback(c *gin.Context) {
	state := c.Query("state")
	code := c.Query("code")

	if state == "" || code == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing state or code parameter"})
		return
	}

	valid, err := h.stateStore.ValidateState(c.Request.Context(), state)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to validate state"})
		return
	}
	if !valid {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired state"})
		return
	}

	token, err := h.oauthConfig.Exchange(context.Background(), code)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Failed to exchange code for token"})
		return
	}

	googleUser, err := h.fetchGoogleUserInfo(token.AccessToken)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch user info"})
		return
	}

	if !googleUser.VerifiedEmail {
		c.JSON(http.StatusForbidden, gin.H{"error": "Email not verified"})
		return
	}

	user, err := h.userStore.GetOrCreateUser(c.Request.Context(), googleUser)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create user"})
		return
	}

	session, err := h.sessionStore.CreateSession(c.Request.Context(), user.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create session"})
		return
	}

	sameSite := http.SameSiteLaxMode
	if h.config.CookieSameSite == "strict" {
		sameSite = http.SameSiteStrictMode
	} else if h.config.CookieSameSite == "none" {
		sameSite = http.SameSiteNoneMode
	}

	c.SetSameSite(sameSite)

	cookieDomain := h.config.CookieDomain
	if cookieDomain == "localhost" {
		cookieDomain = ""
	}

	c.SetCookie(
		"session_id",
		session.ID,
		h.config.SessionDuration,
		"/",
		cookieDomain,
		h.config.CookieSecure,
		true,
	)

	frontendURL := h.config.FrontendURL
	if frontendURL == "" {
		frontendURL = "http://localhost:3000"
	}

	c.Redirect(http.StatusFound, frontendURL+"/auth/callback")
}

func (h *Handler) Logout(c *gin.Context) {
	sessionID, err := c.Cookie("session_id")
	if err == nil {
		h.sessionStore.DeleteSession(c.Request.Context(), sessionID)
	}

	sameSite := http.SameSiteLaxMode
	if h.config.CookieSameSite == "strict" {
		sameSite = http.SameSiteStrictMode
	} else if h.config.CookieSameSite == "none" {
		sameSite = http.SameSiteNoneMode
	}

	c.SetSameSite(sameSite)

	cookieDomain := h.config.CookieDomain
	if cookieDomain == "localhost" {
		cookieDomain = ""
	}

	c.SetCookie(
		"session_id",
		"",
		-1,
		"/",
		cookieDomain,
		h.config.CookieSecure,
		true,
	)

	c.JSON(http.StatusOK, gin.H{"message": "Logged out successfully"})
}

func (h *Handler) Me(c *gin.Context) {
	userInterface, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Not authenticated"})
		return
	}

	user, ok := userInterface.(*User)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid user data"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"user": user})
}

func (h *Handler) fetchGoogleUserInfo(accessToken string) (*GoogleUserInfo, error) {
	req, err := http.NewRequest("GET", "https://www.googleapis.com/oauth2/v2/userinfo", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+accessToken)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch user info: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to fetch user info: status %d, body: %s", resp.StatusCode, string(body))
	}

	var googleUser GoogleUserInfo
	if err := json.NewDecoder(resp.Body).Decode(&googleUser); err != nil {
		return nil, fmt.Errorf("failed to decode user info: %w", err)
	}

	return &googleUser, nil
}
