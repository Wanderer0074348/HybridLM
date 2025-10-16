package cache

import (
	"context"
	"encoding/json"
	"time"

	"github.com/redis/go-redis/v9"
	"www.github.com/Wanderer0074348/HybridLM/src/config"
	"www.github.com/Wanderer0074348/HybridLM/src/models"
)

type RedisCache struct {
	client *redis.Client
	ttl    time.Duration
}

func NewRedisCache(cfg *config.RedisConfig) (*RedisCache, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     cfg.Address,
		Password: cfg.Password,
		DB:       cfg.DB,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, err
	}

	return &RedisCache{
		client: client,
		ttl:    cfg.CacheTTL,
	}, nil
}

func (c *RedisCache) Get(ctx context.Context, key string) (*models.InferenceResponse, error) {
	val, err := c.client.Get(ctx, key).Result()
	if err == redis.Nil {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	var response models.InferenceResponse
	if err := json.Unmarshal([]byte(val), &response); err != nil {
		return nil, err
	}

	return &response, nil
}

func (c *RedisCache) Set(ctx context.Context, key string, response *models.InferenceResponse) error {
	data, err := json.Marshal(response)
	if err != nil {
		return err
	}

	return c.client.Set(ctx, key, data, c.ttl).Err()
}

func (c *RedisCache) Delete(ctx context.Context, key string) error {
	return c.client.Del(ctx, key).Err()
}

func (c *RedisCache) Close() error {
	return c.client.Close()
}

// GetClient returns the underlying Redis client for direct access
func (c *RedisCache) GetClient() *redis.Client {
	return c.client
}
