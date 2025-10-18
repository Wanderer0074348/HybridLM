package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

type UserStore struct {
	client *redis.Client
}

func NewUserStore(client *redis.Client) *UserStore {
	return &UserStore{
		client: client,
	}
}

func (u *UserStore) GetOrCreateUser(ctx context.Context, googleUser *GoogleUserInfo) (*User, error) {
	existingUser, err := u.GetUserByEmail(ctx, googleUser.Email)
	if err == nil {
		existingUser.Picture = googleUser.Picture
		existingUser.Name = googleUser.Name
		existingUser.UpdatedAt = time.Now()
		if err := u.SaveUser(ctx, existingUser); err != nil {
			return nil, err
		}
		return existingUser, nil
	}

	user := &User{
		ID:            googleUser.ID,
		Email:         googleUser.Email,
		Name:          googleUser.Name,
		Picture:       googleUser.Picture,
		EmailVerified: googleUser.VerifiedEmail,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	if err := u.SaveUser(ctx, user); err != nil {
		return nil, err
	}

	return user, nil
}

func (u *UserStore) SaveUser(ctx context.Context, user *User) error {
	data, err := json.Marshal(user)
	if err != nil {
		return fmt.Errorf("failed to marshal user: %w", err)
	}

	userKey := fmt.Sprintf("user:%s", user.ID)
	emailKey := fmt.Sprintf("user_email:%s", user.Email)

	pipe := u.client.Pipeline()
	pipe.Set(ctx, userKey, data, 0)
	pipe.Set(ctx, emailKey, user.ID, 0)

	if _, err := pipe.Exec(ctx); err != nil {
		return fmt.Errorf("failed to save user: %w", err)
	}

	return nil
}

func (u *UserStore) GetUser(ctx context.Context, userID string) (*User, error) {
	key := fmt.Sprintf("user:%s", userID)

	data, err := u.client.Get(ctx, key).Result()
	if err == redis.Nil {
		return nil, fmt.Errorf("user not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	var user User
	if err := json.Unmarshal([]byte(data), &user); err != nil {
		return nil, fmt.Errorf("failed to unmarshal user: %w", err)
	}

	return &user, nil
}

func (u *UserStore) GetUserByEmail(ctx context.Context, email string) (*User, error) {
	emailKey := fmt.Sprintf("user_email:%s", email)

	userID, err := u.client.Get(ctx, emailKey).Result()
	if err == redis.Nil {
		return nil, fmt.Errorf("user not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get user ID: %w", err)
	}

	return u.GetUser(ctx, userID)
}
