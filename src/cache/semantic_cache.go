package cache

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/sashabaranov/go-openai"
	"www.github.com/Wanderer0074348/HybridLM/src/config"
	"www.github.com/Wanderer0074348/HybridLM/src/models"
)

const (
	embeddingPrefix = "embedding:"
	queryPrefix     = "query:"
	embeddingModel  = "text-embedding-ada-002"
)

// CachedEntry represents a cached query with its embedding
type CachedEntry struct {
	Query     string                    `json:"query"`
	Embedding []float32                 `json:"embedding"`
	Response  *models.InferenceResponse `json:"response"`
	CachedAt  time.Time                 `json:"cached_at"`
}

// SemanticCache implements semantic similarity-based caching
type SemanticCache struct {
	client         *redis.Client
	openaiClient   *openai.Client
	ttl            time.Duration
	similarityThreshold float64
}

// NewSemanticCache creates a new semantic cache instance
func NewSemanticCache(redisCfg *config.RedisConfig, semanticCfg *config.SemanticCacheConfig) (*SemanticCache, error) {
	// Initialize Redis client
	client := redis.NewClient(&redis.Options{
		Addr:     redisCfg.Address,
		Password: redisCfg.Password,
		DB:       redisCfg.DB,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	// Initialize OpenAI client for embeddings
	openaiClient := openai.NewClient(semanticCfg.APIKey)

	return &SemanticCache{
		client:              client,
		openaiClient:        openaiClient,
		ttl:                 redisCfg.CacheTTL,
		similarityThreshold: semanticCfg.SimilarityThreshold,
	}, nil
}

// Get retrieves a cached response by exact key match
func (c *SemanticCache) Get(ctx context.Context, key string) (*models.InferenceResponse, error) {
	val, err := c.client.Get(ctx, queryPrefix+key).Result()
	if err == redis.Nil {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get from cache: %w", err)
	}

	var entry CachedEntry
	if err := json.Unmarshal([]byte(val), &entry); err != nil {
		return nil, fmt.Errorf("failed to unmarshal cache entry: %w", err)
	}

	return entry.Response, nil
}

// Set stores a response with exact key (backward compatibility)
func (c *SemanticCache) Set(ctx context.Context, key string, response *models.InferenceResponse) error {
	entry := CachedEntry{
		Query:     key,
		Embedding: nil, // No embedding for basic set
		Response:  response,
		CachedAt:  time.Now(),
	}

	data, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("failed to marshal cache entry: %w", err)
	}

	return c.client.Set(ctx, queryPrefix+key, data, c.ttl).Err()
}

// Delete removes a cached entry
func (c *SemanticCache) Delete(ctx context.Context, key string) error {
	pipe := c.client.Pipeline()
	pipe.Del(ctx, queryPrefix+key)
	pipe.Del(ctx, embeddingPrefix+key)
	_, err := pipe.Exec(ctx)
	return err
}

// Close closes the Redis connection
func (c *SemanticCache) Close() error {
	return c.client.Close()
}

// GetSimilar finds semantically similar cached queries
func (c *SemanticCache) GetSimilar(ctx context.Context, query string, threshold float64) (*models.SemanticCacheResult, error) {
	// Generate embedding for the query
	queryEmbedding, err := c.generateEmbedding(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to generate query embedding: %w", err)
	}

	// Get all cached embeddings
	keys, err := c.client.Keys(ctx, queryPrefix+"*").Result()
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve cache keys: %w", err)
	}

	var bestMatch *models.SemanticCacheResult
	maxSimilarity := threshold

	// Compare with each cached entry
	for _, key := range keys {
		val, err := c.client.Get(ctx, key).Result()
		if err != nil {
			continue
		}

		var entry CachedEntry
		if err := json.Unmarshal([]byte(val), &entry); err != nil {
			continue
		}

		// Skip entries without embeddings
		if len(entry.Embedding) == 0 {
			continue
		}

		// Calculate cosine similarity
		similarity := cosineSimilarity(queryEmbedding, entry.Embedding)

		if similarity > maxSimilarity {
			maxSimilarity = similarity
			cacheKey := key[len(queryPrefix):] // Remove prefix
			bestMatch = &models.SemanticCacheResult{
				Response:   entry.Response,
				Similarity: similarity,
				CacheKey:   cacheKey,
			}
		}
	}

	return bestMatch, nil
}

// SetWithEmbedding stores a response with its query embedding
func (c *SemanticCache) SetWithEmbedding(ctx context.Context, key string, query string, response *models.InferenceResponse) error {
	// Generate embedding for the query
	embedding, err := c.generateEmbedding(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to generate embedding: %w", err)
	}

	entry := CachedEntry{
		Query:     query,
		Embedding: embedding,
		Response:  response,
		CachedAt:  time.Now(),
	}

	data, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("failed to marshal cache entry: %w", err)
	}

	// Store the entry with TTL
	if err := c.client.Set(ctx, queryPrefix+key, data, c.ttl).Err(); err != nil {
		return fmt.Errorf("failed to set cache entry: %w", err)
	}

	return nil
}

// generateEmbedding generates an embedding vector for the given text
func (c *SemanticCache) generateEmbedding(ctx context.Context, text string) ([]float32, error) {
	if text == "" {
		return nil, errors.New("text cannot be empty")
	}

	resp, err := c.openaiClient.CreateEmbeddings(ctx, openai.EmbeddingRequest{
		Input: []string{text},
		Model: openai.AdaEmbeddingV2,
	})
	if err != nil {
		return nil, fmt.Errorf("openai embedding request failed: %w", err)
	}

	if len(resp.Data) == 0 {
		return nil, errors.New("no embedding returned from OpenAI")
	}

	return resp.Data[0].Embedding, nil
}

// cosineSimilarity calculates the cosine similarity between two vectors
func cosineSimilarity(a, b []float32) float64 {
	if len(a) != len(b) {
		return 0.0
	}

	var dotProduct, normA, normB float64
	for i := range a {
		dotProduct += float64(a[i]) * float64(b[i])
		normA += float64(a[i]) * float64(a[i])
		normB += float64(b[i]) * float64(b[i])
	}

	if normA == 0 || normB == 0 {
		return 0.0
	}

	return dotProduct / (math.Sqrt(normA) * math.Sqrt(normB))
}
