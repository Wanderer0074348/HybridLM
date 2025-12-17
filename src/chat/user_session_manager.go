package chat

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	userSessionsKey = "user_sessions:" // Key prefix for user's session list
	maxSessions     = 3                 // Maximum sessions to keep per user
)

type UserSessionManager struct {
	client       *redis.Client
	sessionStore *SessionStore
}

type SessionMetadata struct {
	SessionID       string    `json:"session_id"`
	Title           string    `json:"title"`
	LastInteraction time.Time `json:"last_interaction"`
	MessageCount    int       `json:"message_count"`
	CreatedAt       time.Time `json:"created_at"`
}

func NewUserSessionManager(client *redis.Client, sessionStore *SessionStore) *UserSessionManager {
	return &UserSessionManager{
		client:       client,
		sessionStore: sessionStore,
	}
}

// GetUserSessions returns all sessions for a user, sorted by last interaction
func (m *UserSessionManager) GetUserSessions(ctx context.Context, userID string) ([]SessionMetadata, error) {
	key := userSessionsKey + userID

	data, err := m.client.Get(ctx, key).Result()
	if err == redis.Nil {
		return []SessionMetadata{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get user sessions: %w", err)
	}

	var sessions []SessionMetadata
	if err := json.Unmarshal([]byte(data), &sessions); err != nil {
		return nil, fmt.Errorf("failed to unmarshal sessions: %w", err)
	}

	// Sort by last interaction, newest first
	sort.Slice(sessions, func(i, j int) bool {
		return sessions[i].LastInteraction.After(sessions[j].LastInteraction)
	})

	return sessions, nil
}

// AddUserSession adds a new session to user's session list, maintaining max limit
func (m *UserSessionManager) AddUserSession(ctx context.Context, userID, sessionID, title string) error {
	sessions, err := m.GetUserSessions(ctx, userID)
	if err != nil {
		return err
	}

	// Check if session already exists (update it)
	found := false
	for i, s := range sessions {
		if s.SessionID == sessionID {
			sessions[i].LastInteraction = time.Now()
			sessions[i].Title = title
			found = true
			break
		}
	}

	if !found {
		// Add new session
		newSession := SessionMetadata{
			SessionID:       sessionID,
			Title:           title,
			LastInteraction: time.Now(),
			MessageCount:    0,
			CreatedAt:       time.Now(),
		}
		sessions = append(sessions, newSession)
	}

	// Sort by last interaction, newest first
	sort.Slice(sessions, func(i, j int) bool {
		return sessions[i].LastInteraction.After(sessions[j].LastInteraction)
	})

	// Keep only the last N sessions
	if len(sessions) > maxSessions {
		// Delete old sessions from Redis
		for i := maxSessions; i < len(sessions); i++ {
			_ = m.sessionStore.DeleteSession(ctx, sessions[i].SessionID)
		}
		sessions = sessions[:maxSessions]
	}

	// Save updated list
	key := userSessionsKey + userID
	data, err := json.Marshal(sessions)
	if err != nil {
		return fmt.Errorf("failed to marshal sessions: %w", err)
	}

	// Keep user sessions for 30 days
	if err := m.client.Set(ctx, key, data, 30*24*time.Hour).Err(); err != nil {
		return fmt.Errorf("failed to save user sessions: %w", err)
	}

	return nil
}

// UpdateSessionMetadata updates session metadata after each message
func (m *UserSessionManager) UpdateSessionMetadata(ctx context.Context, userID, sessionID string, messageCount int) error {
	sessions, err := m.GetUserSessions(ctx, userID)
	if err != nil {
		return err
	}

	// Update session metadata
	for i, s := range sessions {
		if s.SessionID == sessionID {
			sessions[i].LastInteraction = time.Now()
			sessions[i].MessageCount = messageCount
			break
		}
	}

	// Save updated list
	key := userSessionsKey + userID
	data, err := json.Marshal(sessions)
	if err != nil {
		return fmt.Errorf("failed to marshal sessions: %w", err)
	}

	if err := m.client.Set(ctx, key, data, 30*24*time.Hour).Err(); err != nil {
		return fmt.Errorf("failed to save user sessions: %w", err)
	}

	return nil
}

// DeleteUserSession removes a session from user's list
func (m *UserSessionManager) DeleteUserSession(ctx context.Context, userID, sessionID string) error {
	sessions, err := m.GetUserSessions(ctx, userID)
	if err != nil {
		return err
	}

	// Remove session from list
	filtered := []SessionMetadata{}
	for _, s := range sessions {
		if s.SessionID != sessionID {
			filtered = append(filtered, s)
		}
	}

	// Delete the actual session
	if err := m.sessionStore.DeleteSession(ctx, sessionID); err != nil {
		return err
	}

	// Save updated list
	key := userSessionsKey + userID
	data, err := json.Marshal(filtered)
	if err != nil {
		return fmt.Errorf("failed to marshal sessions: %w", err)
	}

	if err := m.client.Set(ctx, key, data, 30*24*time.Hour).Err(); err != nil {
		return fmt.Errorf("failed to save user sessions: %w", err)
	}

	return nil
}

// GenerateSessionTitle generates a title from the first message
func GenerateSessionTitle(firstMessage string) string {
	maxLen := 50
	if len(firstMessage) <= maxLen {
		return firstMessage
	}
	return firstMessage[:maxLen] + "..."
}
