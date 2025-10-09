package models

import "time"

type InferenceRequest struct {
	Query       string            `json:"query" binding:"required"`
	Context     string            `json:"context,omitempty"`
	MaxTokens   int               `json:"max_tokens,omitempty"`
	Temperature float32           `json:"temperature,omitempty"`
	Metadata    map[string]string `json:"metadata,omitempty"`
}

type InferenceResponse struct {
	Response      string        `json:"response"`
	ModelUsed     string        `json:"model_used"`
	RoutingReason string        `json:"routing_reason"`
	Latency       time.Duration `json:"latency"`
	CacheHit      bool          `json:"cache_hit"`
	Timestamp     time.Time     `json:"timestamp"`
}

type RoutingDecision struct {
	UseLLM          bool
	Reason          string
	Confidence      float64
	ComplexityScore float64
}

type QueryMetrics struct {
	TokenCount  int
	Complexity  float64
	HasContext  bool
	QueryLength int
}
