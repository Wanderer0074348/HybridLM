package auth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

type SessionStore struct {
	client          *redis.Client
	sessionDuration time.Duration
}

func NewSessionStore(client *redis.Client, sessionDuration time.Duration) *SessionStore {
	return &SessionStore{
		client:          client,
		sessionDuration: sessionDuration,
	}
}

func (s *SessionStore) GenerateSessionID() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("failed to generate session ID: %w", err)
	}
	return base64.URLEncoding.EncodeToString(b), nil
}

func (s *SessionStore) CreateSession(ctx context.Context, userID string) (*Session, error) {
	sessionID, err := s.GenerateSessionID()
	if err != nil {
		return nil, err
	}

	session := &Session{
		ID:        sessionID,
		UserID:    userID,
		ExpiresAt: time.Now().Add(s.sessionDuration),
		CreatedAt: time.Now(),
	}

	data, err := json.Marshal(session)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal session: %w", err)
	}

	key := fmt.Sprintf("session:%s", sessionID)
	if err := s.client.Set(ctx, key, data, s.sessionDuration).Err(); err != nil {
		return nil, fmt.Errorf("failed to save session: %w", err)
	}

	return session, nil
}

func (s *SessionStore) GetSession(ctx context.Context, sessionID string) (*Session, error) {
	key := fmt.Sprintf("session:%s", sessionID)

	data, err := s.client.Get(ctx, key).Result()
	if err == redis.Nil {
		return nil, fmt.Errorf("session not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get session: %w", err)
	}

	var session Session
	if err := json.Unmarshal([]byte(data), &session); err != nil {
		return nil, fmt.Errorf("failed to unmarshal session: %w", err)
	}

	if time.Now().After(session.ExpiresAt) {
		s.DeleteSession(ctx, sessionID)
		return nil, fmt.Errorf("session expired")
	}

	return &session, nil
}

func (s *SessionStore) DeleteSession(ctx context.Context, sessionID string) error {
	key := fmt.Sprintf("session:%s", sessionID)
	return s.client.Del(ctx, key).Err()
}

func (s *SessionStore) RefreshSession(ctx context.Context, sessionID string) error {
	session, err := s.GetSession(ctx, sessionID)
	if err != nil {
		return err
	}

	session.ExpiresAt = time.Now().Add(s.sessionDuration)

	data, err := json.Marshal(session)
	if err != nil {
		return fmt.Errorf("failed to marshal session: %w", err)
	}

	key := fmt.Sprintf("session:%s", sessionID)
	return s.client.Set(ctx, key, data, s.sessionDuration).Err()
}
