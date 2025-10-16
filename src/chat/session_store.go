package chat

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"

	"www.github.com/Wanderer0074348/HybridLM/src/models"
)

const (
	sessionKeyPrefix = "chat_session:"
	sessionTTL       = 24 * time.Hour // Sessions expire after 24 hours of inactivity
	maxContextWindow = 20             // Keep last 20 messages for context
)

type SessionStore struct {
	client *redis.Client
}

func NewSessionStore(client *redis.Client) *SessionStore {
	return &SessionStore{
		client: client,
	}
}

// CreateSession creates a new chat session
func (s *SessionStore) CreateSession(ctx context.Context) (*models.ChatSession, error) {
	sessionID := "sess_" + uuid.New().String()

	session := &models.ChatSession{
		SessionID:       sessionID,
		Messages:        []models.ChatMessage{},
		CreatedAt:       time.Now(),
		LastInteraction: time.Now(),
		TotalTokens:     0,
		MessageCount:    0,
		ModelPreference: "auto",
	}

	if err := s.SaveSession(ctx, session); err != nil {
		return nil, err
	}

	return session, nil
}

// GetSession retrieves a session by ID
func (s *SessionStore) GetSession(ctx context.Context, sessionID string) (*models.ChatSession, error) {
	key := sessionKeyPrefix + sessionID

	data, err := s.client.Get(ctx, key).Result()
	if err == redis.Nil {
		return nil, fmt.Errorf("session not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get session: %w", err)
	}

	var session models.ChatSession
	if err := json.Unmarshal([]byte(data), &session); err != nil {
		return nil, fmt.Errorf("failed to unmarshal session: %w", err)
	}

	return &session, nil
}

// SaveSession saves or updates a session
func (s *SessionStore) SaveSession(ctx context.Context, session *models.ChatSession) error {
	key := sessionKeyPrefix + session.SessionID

	data, err := json.Marshal(session)
	if err != nil {
		return fmt.Errorf("failed to marshal session: %w", err)
	}

	if err := s.client.Set(ctx, key, data, sessionTTL).Err(); err != nil {
		return fmt.Errorf("failed to save session: %w", err)
	}

	return nil
}

// AddMessage adds a message to the session and updates it
func (s *SessionStore) AddMessage(ctx context.Context, sessionID string, role string, content string, tokens int) error {
	session, err := s.GetSession(ctx, sessionID)
	if err != nil {
		return err
	}

	message := models.ChatMessage{
		Role:      role,
		Content:   content,
		Timestamp: time.Now(),
	}

	session.Messages = append(session.Messages, message)
	session.LastInteraction = time.Now()
	session.MessageCount++
	session.TotalTokens += tokens

	// Trim old messages if exceeding context window
	if len(session.Messages) > maxContextWindow {
		// Keep the most recent messages
		session.Messages = session.Messages[len(session.Messages)-maxContextWindow:]
	}

	return s.SaveSession(ctx, session)
}

// DeleteSession deletes a session
func (s *SessionStore) DeleteSession(ctx context.Context, sessionID string) error {
	key := sessionKeyPrefix + sessionID

	if err := s.client.Del(ctx, key).Err(); err != nil {
		return fmt.Errorf("failed to delete session: %w", err)
	}

	return nil
}

// GetRecentSessions returns all active session IDs (for admin/debugging)
func (s *SessionStore) GetRecentSessions(ctx context.Context) ([]string, error) {
	pattern := sessionKeyPrefix + "*"

	keys, err := s.client.Keys(ctx, pattern).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get sessions: %w", err)
	}

	// Strip prefix from keys
	sessionIDs := make([]string, len(keys))
	for i, key := range keys {
		sessionIDs[i] = key[len(sessionKeyPrefix):]
	}

	return sessionIDs, nil
}

// BuildConversationContext builds a conversation history string for the LLM
func (s *SessionStore) BuildConversationContext(session *models.ChatSession) string {
	if len(session.Messages) == 0 {
		return ""
	}

	context := "Previous conversation:\n"
	for _, msg := range session.Messages {
		context += fmt.Sprintf("%s: %s\n", msg.Role, msg.Content)
	}

	return context
}
