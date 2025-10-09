package router

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"www.github.com/Wanderer0074348/HybridLM/src/config"
	"www.github.com/Wanderer0074348/HybridLM/src/models"
)

func TestQueryRouter_SimpleQuery(t *testing.T) {
	cfg := &config.RouterConfig{
		ComplexityThreshold: 0.65,
	}
	router := NewQueryRouter(cfg)

	req := &models.InferenceRequest{
		Query: "What is 2+2?",
	}

	decision, err := router.Route(context.Background(), req)

	assert.NoError(t, err)
	assert.NotNil(t, decision)
	assert.False(t, decision.UseLLM, "Simple query should route to SLM")
	assert.Contains(t, decision.Reason, "Simple query")
}

func TestQueryRouter_ComplexQuery(t *testing.T) {
	cfg := &config.RouterConfig{
		ComplexityThreshold: 0.65,
	}
	router := NewQueryRouter(cfg)

	req := &models.InferenceRequest{
		Query: "Explain in comprehensive and exhaustive detail the architectural decisions behind implementing a hybrid Large Language Model inference system that intelligently routes queries between cloud-based LLMs and edge-deployed Small Language Models. Analyze the multi-factor complexity scoring algorithm that evaluates query characteristics including length, vocabulary diversity, keyword presence, and punctuation patterns to determine optimal routing decisions. Compare and contrast the trade-offs between using local Ollama deployments versus cloud-based HuggingFace Inference API endpoints for edge inference, considering factors such as deployment complexity, infrastructure requirements, free-tier limitations, cold-start latency, and model availability constraints. Evaluate the effectiveness of Redis caching strategies in reducing duplicate inference costs and improving response latency, discussing optimal TTL configurations, cache key generation methods, and strategies for handling cache invalidation during model updates. Discuss the implications of using LangChainGo as a unified abstraction layer across multiple LLM providers, analyzing how this design pattern facilitates provider switching and reduces vendor lock-in while maintaining consistent inference interfaces. Examine the performance characteristics and scalability bottlenecks when handling 10,000+ concurrent users across both routing pathways, considering connection pooling, worker pool management, circuit breaker patterns, and graceful degradation strategies. Why do these architectural patterns matter for production deployments on resource-constrained platforms like Railway, Render, or Fly.io free tiers? Provide detailed reasoning about cost optimization strategies that balance response quality against API usage costs, and recommend specific implementation approaches for monitoring, alerting, and auto-scaling based on query complexity distributions and traffic patterns.",
	}

	decision, err := router.Route(context.Background(), req)

	assert.NoError(t, err)
	assert.NotNil(t, decision)
	assert.True(t, decision.UseLLM, "Complex query should route to LLM")
}

func TestQueryRouter_WithContext(t *testing.T) {
	cfg := &config.RouterConfig{
		ComplexityThreshold: 0.65,
	}
	router := NewQueryRouter(cfg)

	req := &models.InferenceRequest{
		Query:   "What are the bottlenecks?",
		Context: "We have a distributed system with Redis caching.",
	}

	decision, err := router.Route(context.Background(), req)

	assert.NoError(t, err)
	assert.True(t, decision.UseLLM, "Queries with context should route to LLM")
}

func TestQueryRouter_CacheKey(t *testing.T) {
	cfg := &config.RouterConfig{
		ComplexityThreshold: 0.65,
	}
	router := NewQueryRouter(cfg)

	req1 := &models.InferenceRequest{Query: "Test"}
	req2 := &models.InferenceRequest{Query: "Test"}
	req3 := &models.InferenceRequest{Query: "Different"}

	key1 := router.GenerateCacheKey(req1)
	key2 := router.GenerateCacheKey(req2)
	key3 := router.GenerateCacheKey(req3)

	assert.Equal(t, key1, key2)
	assert.NotEqual(t, key1, key3)
}

func BenchmarkQueryRouter_Route(b *testing.B) {
	cfg := &config.RouterConfig{
		ComplexityThreshold: 0.65,
	}
	router := NewQueryRouter(cfg)

	req := &models.InferenceRequest{
		Query: "Explain how caching works in detail",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		router.Route(context.Background(), req)
	}
}
