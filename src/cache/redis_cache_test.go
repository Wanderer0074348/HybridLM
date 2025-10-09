package cache

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"www.github.com/Wanderer0074348/HybridLM/src/config"
	"www.github.com/Wanderer0074348/HybridLM/src/models"
)

func setupTestRedis(t *testing.T) (*RedisCache, *miniredis.Miniredis) {
	mr, err := miniredis.Run()
	require.NoError(t, err)

	cfg := &config.RedisConfig{
		Address:  mr.Addr(),
		Password: "",
		DB:       0,
		CacheTTL: time.Hour,
	}

	cache, err := NewRedisCache(cfg)
	require.NoError(t, err)

	return cache, mr
}

func TestRedisCache_SetAndGet(t *testing.T) {
	cache, mr := setupTestRedis(t)
	defer mr.Close()
	defer cache.Close()

	ctx := context.Background()
	key := "test:key"

	response := &models.InferenceResponse{
		Response:      "Test response",
		ModelUsed:     "test-model",
		RoutingReason: "test reason",
		Latency:       100 * time.Millisecond,
		CacheHit:      false,
		Timestamp:     time.Now(),
	}

	err := cache.Set(ctx, key, response)
	assert.NoError(t, err)

	retrieved, err := cache.Get(ctx, key)
	assert.NoError(t, err)
	assert.NotNil(t, retrieved)
	assert.Equal(t, response.Response, retrieved.Response)
	assert.Equal(t, response.ModelUsed, retrieved.ModelUsed)
}

func TestRedisCache_GetNonExistent(t *testing.T) {
	cache, mr := setupTestRedis(t)
	defer mr.Close()
	defer cache.Close()

	ctx := context.Background()

	retrieved, err := cache.Get(ctx, "nonexistent:key")
	assert.NoError(t, err)
	assert.Nil(t, retrieved)
}

func TestRedisCache_Delete(t *testing.T) {
	cache, mr := setupTestRedis(t)
	defer mr.Close()
	defer cache.Close()

	ctx := context.Background()
	key := "test:delete"

	response := &models.InferenceResponse{
		Response: "Test",
	}

	cache.Set(ctx, key, response)
	err := cache.Delete(ctx, key)
	assert.NoError(t, err)

	retrieved, _ := cache.Get(ctx, key)
	assert.Nil(t, retrieved)
}

func TestRedisCache_Expiration(t *testing.T) {
	mr, err := miniredis.Run()
	require.NoError(t, err)
	defer mr.Close()

	cfg := &config.RedisConfig{
		Address:  mr.Addr(),
		Password: "",
		DB:       0,
		CacheTTL: 1 * time.Second,
	}

	cache, err := NewRedisCache(cfg)
	require.NoError(t, err)
	defer cache.Close()

	ctx := context.Background()
	key := "test:expiry"

	response := &models.InferenceResponse{Response: "Test"}
	cache.Set(ctx, key, response)

	mr.FastForward(2 * time.Second)

	retrieved, _ := cache.Get(ctx, key)
	assert.Nil(t, retrieved, "Key should be expired")
}

func BenchmarkRedisCache_Set(b *testing.B) {
	mr, _ := miniredis.Run()
	defer mr.Close()

	cfg := &config.RedisConfig{
		Address:  mr.Addr(),
		CacheTTL: time.Hour,
	}
	cache, _ := NewRedisCache(cfg)
	defer cache.Close()

	response := &models.InferenceResponse{Response: "Benchmark"}
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cache.Set(ctx, "bench:key", response)
	}
}
