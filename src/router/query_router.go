package router

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"strings"
	"unicode"

	"www.github.com/Wanderer0074348/HybridLM/src/config"
	"www.github.com/Wanderer0074348/HybridLM/src/models"
)

type QueryRouter struct {
	config      *config.RouterConfig
	strategy    RoutingStrategy
	mlStrategy  *MLRoutingStrategy
	rlStrategy  *RLRoutingStrategy
	useMLRouter bool
	useRLRouter bool
}

func NewQueryRouter(cfg *config.RouterConfig) *QueryRouter {
	return &QueryRouter{
		config:      cfg,
		strategy:    NewHybridRoutingStrategy(cfg),
		useMLRouter: false,
		useRLRouter: false,
	}
}

func NewQueryRouterWithML(cfg *config.RouterConfig, mlStrategy *MLRoutingStrategy) *QueryRouter {
	return &QueryRouter{
		config:      cfg,
		strategy:    NewHybridRoutingStrategy(cfg),
		mlStrategy:  mlStrategy,
		useMLRouter: true,
		useRLRouter: false,
	}
}

func NewQueryRouterWithRL(cfg *config.RouterConfig, rlStrategy *RLRoutingStrategy) *QueryRouter {
	return &QueryRouter{
		config:      cfg,
		strategy:    NewHybridRoutingStrategy(cfg),
		rlStrategy:  rlStrategy,
		useMLRouter: false,
		useRLRouter: true,
	}
}

func (r *QueryRouter) Route(ctx context.Context, req *models.InferenceRequest) (*models.RoutingDecision, error) {
	if r.useRLRouter && r.rlStrategy != nil {
		decision, _, _ := r.rlStrategy.DecideWithFeatures(ctx, req.Query, req.Context)
		return decision, nil
	}

	if r.useMLRouter && r.mlStrategy != nil {
		decision, _, _ := r.mlStrategy.DecideWithFeatures(ctx, req.Query, req.Context)
		return decision, nil
	}

	metrics := r.analyzeQuery(req)
	decision := r.strategy.Decide(metrics)

	return decision, nil
}

func (r *QueryRouter) RouteWithFeatures(ctx context.Context, req *models.InferenceRequest) (*models.RoutingDecision, *models.QueryFeatures, string, error) {
	if r.useRLRouter && r.rlStrategy != nil {
		decision, features, abTestGroup := r.rlStrategy.DecideWithFeatures(ctx, req.Query, req.Context)
		return decision, features, abTestGroup, nil
	}

	if r.useMLRouter && r.mlStrategy != nil {
		decision, features, abTestGroup := r.mlStrategy.DecideWithFeatures(ctx, req.Query, req.Context)
		return decision, features, abTestGroup, nil
	}

	metrics := r.analyzeQuery(req)
	decision := r.strategy.Decide(metrics)

	return decision, nil, "control", nil
}

func (r *QueryRouter) analyzeQuery(req *models.InferenceRequest) *models.QueryMetrics {
	metrics := &models.QueryMetrics{
		QueryLength: len(req.Query),
		HasContext:  len(req.Context) > 0,
	}

	// Estimate token count (rough approximation)
	metrics.TokenCount = len(strings.Fields(req.Query))

	// Calculate complexity score
	metrics.Complexity = r.calculateComplexity(req.Query)

	return metrics
}

func (r *QueryRouter) calculateComplexity(query string) float64 {
	var score float64

	// Length factor
	lengthScore := float64(len(query)) / 1000.0
	if lengthScore > 1.0 {
		lengthScore = 1.0
	}

	// Word diversity
	words := strings.Fields(strings.ToLower(query))
	uniqueWords := make(map[string]bool)
	for _, word := range words {
		uniqueWords[word] = true
	}
	diversityScore := float64(len(uniqueWords)) / float64(len(words))

	// Question complexity indicators
	complexityKeywords := []string{
		"explain", "analyze", "compare", "evaluate", "why",
		"how does", "what if", "reasoning", "detailed",
	}
	keywordScore := 0.0
	queryLower := strings.ToLower(query)
	for _, keyword := range complexityKeywords {
		if strings.Contains(queryLower, keyword) {
			keywordScore += 0.15
		}
	}

	// Punctuation complexity
	punctCount := 0
	for _, char := range query {
		if unicode.IsPunct(char) {
			punctCount++
		}
	}
	punctScore := float64(punctCount) / 100.0
	if punctScore > 0.3 {
		punctScore = 0.3
	}

	score = (lengthScore * 0.3) + (diversityScore * 0.3) +
		(keywordScore * 0.3) + (punctScore * 0.1)

	return score
}

func (r *QueryRouter) GenerateCacheKey(req *models.InferenceRequest) string {
	data := req.Query + "|" + req.Context
	hash := md5.Sum([]byte(data))
	return "inference:" + hex.EncodeToString(hash[:])
}
