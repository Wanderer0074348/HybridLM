package inference

import (
	"context"
	"fmt"
	"sync"

	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/ollama"

	"www.github.com/Wanderer0074348/HybridLM/src/config"
	"www.github.com/Wanderer0074348/HybridLM/src/models"
)

type SLMEngine struct {
	config     *config.SLMConfig
	llm        llms.Model
	workerPool chan struct{}
	mu         sync.RWMutex
}

func NewSLMEngine(cfg *config.SLMConfig) (*SLMEngine, error) {
	// Initialize Ollama with langchaingo
	llm, err := ollama.New(
		ollama.WithModel(cfg.ModelName),
		ollama.WithServerURL(cfg.OllamaHost),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create Ollama client: %w", err)
	}

	// Create worker pool for concurrent inference
	workerPool := make(chan struct{}, cfg.MaxConcurrent)

	return &SLMEngine{
		config:     cfg,
		llm:        llm,
		workerPool: workerPool,
	}, nil
}

func (e *SLMEngine) Infer(ctx context.Context, req *models.InferenceRequest) (string, error) {
	// Acquire worker slot
	select {
	case e.workerPool <- struct{}{}:
		defer func() { <-e.workerPool }()
	case <-ctx.Done():
		return "", ctx.Err()
	}

	e.mu.RLock()
	defer e.mu.RUnlock()

	// Build prompt
	prompt := req.Query
	if req.Context != "" {
		prompt = fmt.Sprintf("Context: %s\n\nQuestion: %s", req.Context, req.Query)
	}

	// Set default temperature
	temperature := float64(req.Temperature)
	if temperature == 0 {
		temperature = 0.7
	}

	// Call options
	callOptions := []llms.CallOption{
		llms.WithTemperature(temperature),
		llms.WithMaxTokens(e.config.MaxTokens),
	}

	// Generate response using unified interface (same as OpenAI!)
	response, err := llms.GenerateFromSinglePrompt(
		ctx,
		e.llm,
		prompt,
		callOptions...,
	)
	if err != nil {
		return "", fmt.Errorf("Ollama generation failed: %w", err)
	}

	return response, nil
}

// Streaming support
func (e *SLMEngine) InferStreaming(ctx context.Context, req *models.InferenceRequest, callback func(string) error) error {
	// Acquire worker slot
	select {
	case e.workerPool <- struct{}{}:
		defer func() { <-e.workerPool }()
	case <-ctx.Done():
		return ctx.Err()
	}

	e.mu.RLock()
	defer e.mu.RUnlock()

	prompt := req.Query
	if req.Context != "" {
		prompt = fmt.Sprintf("Context: %s\n\nQuestion: %s", req.Context, req.Query)
	}

	temperature := float64(req.Temperature)
	if temperature == 0 {
		temperature = 0.7
	}

	// Streaming callback
	streamingFunc := func(ctx context.Context, chunk []byte) error {
		if len(chunk) > 0 {
			return callback(string(chunk))
		}
		return nil
	}

	_, err := llms.GenerateFromSinglePrompt(
		ctx,
		e.llm,
		prompt,
		llms.WithTemperature(temperature),
		llms.WithMaxTokens(e.config.MaxTokens),
		llms.WithStreamingFunc(streamingFunc),
	)

	return err
}

func (e *SLMEngine) Close() error {
	close(e.workerPool)
	return nil
}
