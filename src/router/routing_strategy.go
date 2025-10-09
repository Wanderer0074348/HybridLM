package router

import (
	"www.github.com/Wanderer0074348/HybridLM/src/config"
	"www.github.com/Wanderer0074348/HybridLM/src/models"
)

type RoutingStrategy interface {
	Decide(metrics *models.QueryMetrics) *models.RoutingDecision
}

type HybridRoutingStrategy struct {
	config *config.RouterConfig
}

func NewHybridRoutingStrategy(cfg *config.RouterConfig) *HybridRoutingStrategy {
	return &HybridRoutingStrategy{
		config: cfg,
	}
}

func (s *HybridRoutingStrategy) Decide(metrics *models.QueryMetrics) *models.RoutingDecision {
	decision := &models.RoutingDecision{
		ComplexityScore: metrics.Complexity,
	}

	// Multi-factor routing decision
	if metrics.Complexity > s.config.ComplexityThreshold {
		decision.UseLLM = true
		decision.Reason = "High complexity query requires LLM reasoning"
		decision.Confidence = 0.9
		return decision
	}

	if metrics.TokenCount > 100 {
		decision.UseLLM = true
		decision.Reason = "Long query requires cloud LLM processing"
		decision.Confidence = 0.85
		return decision
	}

	if metrics.HasContext {
		decision.UseLLM = true
		decision.Reason = "Context-aware query routed to LLM"
		decision.Confidence = 0.8
		return decision
	}

	// Default to SLM for simple queries
	decision.UseLLM = false
	decision.Reason = "Simple query suitable for edge SLM"
	decision.Confidence = 0.95

	return decision
}
