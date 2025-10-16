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
	CostMetrics   *CostMetrics  `json:"cost_metrics,omitempty"`
}

type CostMetrics struct {
	InputTokens      int     `json:"input_tokens"`
	OutputTokens     int     `json:"output_tokens"`
	TotalTokens      int     `json:"total_tokens"`
	Cost             float64 `json:"cost"`              // Actual cost in USD
	CacheCost        float64 `json:"cache_cost"`        // Cost of cache operation (embeddings)
	TotalCost        float64 `json:"total_cost"`        // Cost + CacheCost
	EstimatedSavings float64 `json:"estimated_savings"` // Money saved by using SLM instead of LLM
	Model            string  `json:"model"`             // Specific model used
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

// Chat-specific types for conversational interactions

type ChatMessage struct {
	Role      string    `json:"role"`      // "user" or "assistant"
	Content   string    `json:"content"`   // The actual message text
	Timestamp time.Time `json:"timestamp"` // When the message was created
}

type ChatSession struct {
	SessionID        string        `json:"session_id"`
	Messages         []ChatMessage `json:"messages"`
	CreatedAt        time.Time     `json:"created_at"`
	LastInteraction  time.Time     `json:"last_interaction"`
	TotalTokens      int           `json:"total_tokens"`       // Running token count
	MessageCount     int           `json:"message_count"`      // Number of messages in session
	ModelPreference  string        `json:"model_preference"`   // "llm", "slm", or "auto"
}

type ChatRequest struct {
	SessionID   string  `json:"session_id,omitempty"`   // Optional: if not provided, creates new session
	Message     string  `json:"message" binding:"required"` // User's message
	MaxTokens   int     `json:"max_tokens,omitempty"`
	Temperature float32 `json:"temperature,omitempty"`
	Stream      bool    `json:"stream,omitempty"`       // Enable streaming response
}

type ChatResponse struct {
	SessionID      string        `json:"session_id"`
	Response       string        `json:"response"`
	ModelUsed      string        `json:"model_used"`
	RoutingReason  string        `json:"routing_reason"`
	Latency        time.Duration `json:"latency"`
	CacheHit       bool          `json:"cache_hit"`
	Timestamp      time.Time     `json:"timestamp"`
	MessageCount   int           `json:"message_count"`   // Total messages in this session
	CostMetrics    *CostMetrics  `json:"cost_metrics,omitempty"`
}
