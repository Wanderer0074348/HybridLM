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

type StateStore struct {
	client *redis.Client
}

func NewStateStore(client *redis.Client) *StateStore {
	return &StateStore{
		client: client,
	}
}

func (s *StateStore) GenerateState() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("failed to generate random state: %w", err)
	}
	return base64.URLEncoding.EncodeToString(b), nil
}

func (s *StateStore) SaveState(ctx context.Context, state string, ttl time.Duration) error {
	oauthState := OAuthState{
		State:     state,
		ExpiresAt: time.Now().Add(ttl),
	}

	data, err := json.Marshal(oauthState)
	if err != nil {
		return fmt.Errorf("failed to marshal state: %w", err)
	}

	key := fmt.Sprintf("oauth_state:%s", state)
	return s.client.Set(ctx, key, data, ttl).Err()
}

func (s *StateStore) ValidateState(ctx context.Context, state string) (bool, error) {
	key := fmt.Sprintf("oauth_state:%s", state)

	data, err := s.client.Get(ctx, key).Result()
	if err == redis.Nil {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("failed to get state: %w", err)
	}

	var oauthState OAuthState
	if err := json.Unmarshal([]byte(data), &oauthState); err != nil {
		return false, fmt.Errorf("failed to unmarshal state: %w", err)
	}

	if time.Now().After(oauthState.ExpiresAt) {
		s.client.Del(ctx, key)
		return false, nil
	}

	s.client.Del(ctx, key)
	return true, nil
}
