package router

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"www.github.com/Wanderer0074348/HybridLM/src/config"
	"www.github.com/Wanderer0074348/HybridLM/src/models"
)

func TestRoutingStrategy_HighComplexity(t *testing.T) {
	cfg := &config.RouterConfig{ComplexityThreshold: 0.65}
	strategy := NewHybridRoutingStrategy(cfg)

	metrics := &models.QueryMetrics{
		Complexity:  0.8,
		TokenCount:  50,
		HasContext:  false,
		QueryLength: 200,
	}

	decision := strategy.Decide(metrics)

	assert.True(t, decision.UseLLM)
	assert.Contains(t, decision.Reason, "High complexity")
	assert.GreaterOrEqual(t, decision.Confidence, 0.8)
}

func TestRoutingStrategy_LongQuery(t *testing.T) {
	cfg := &config.RouterConfig{ComplexityThreshold: 0.65}
	strategy := NewHybridRoutingStrategy(cfg)

	metrics := &models.QueryMetrics{
		Complexity: 0.4,
		TokenCount: 150,
		HasContext: false,
	}

	decision := strategy.Decide(metrics)

	assert.True(t, decision.UseLLM)
	assert.Contains(t, decision.Reason, "Long query")
}

func TestRoutingStrategy_SimpleQuery(t *testing.T) {
	cfg := &config.RouterConfig{ComplexityThreshold: 0.65}
	strategy := NewHybridRoutingStrategy(cfg)

	metrics := &models.QueryMetrics{
		Complexity: 0.3,
		TokenCount: 10,
		HasContext: false,
	}

	decision := strategy.Decide(metrics)

	assert.False(t, decision.UseLLM)
	assert.Contains(t, decision.Reason, "Simple query")
}
