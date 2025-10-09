package models

import (
	"context"
)

// LLMInferencer defines the interface for LLM clients
type LLMInferencer interface {
	Infer(ctx context.Context, req *InferenceRequest) (string, error)
}

// SLMInferencer defines the interface for SLM engines
type SLMInferencer interface {
	Infer(ctx context.Context, req *InferenceRequest) (string, error)
	Close() error
}

// CacheStore defines the interface for cache operations
type CacheStore interface {
	Get(ctx context.Context, key string) (*InferenceResponse, error)
	Set(ctx context.Context, key string, response *InferenceResponse) error
	Delete(ctx context.Context, key string) error
	Close() error
}
