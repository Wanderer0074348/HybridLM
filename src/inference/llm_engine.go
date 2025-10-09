package inference

import (
	"context"
	"fmt"

	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/openai"

	"www.github.com/Wanderer0074348/HybridLM/src/config"
	"www.github.com/Wanderer0074348/HybridLM/src/models"
)

type LLMClient struct {
	config *config.LLMConfig
	llm    llms.Model
}

func NewLLMClient(cfg *config.LLMConfig) (*LLMClient, error) {

	llm, err := openai.New(
		openai.WithToken(cfg.APIKey),
		openai.WithModel(cfg.Model),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create OpenAI client: %w", err)
	}

	return &LLMClient{
		config: cfg,
		llm:    llm,
	}, nil
}

func (c *LLMClient) Infer(ctx context.Context, req *models.InferenceRequest) (string, error) {

	prompt := req.Query
	if req.Context != "" {
		prompt = fmt.Sprintf("Context: %s\n\nQuestion: %s", req.Context, req.Query)
	}

	temperature := float64(req.Temperature)
	if temperature == 0 {
		temperature = 0.7
	}

	callOptions := []llms.CallOption{
		llms.WithTemperature(temperature),
		llms.WithMaxTokens(c.config.MaxTokens),
	}

	response, err := llms.GenerateFromSinglePrompt(
		ctx,
		c.llm,
		prompt,
		callOptions...,
	)
	if err != nil {
		return "", fmt.Errorf("OpenAI generation failed: %w", err)
	}

	return response, nil
}

func (c *LLMClient) InferStreaming(ctx context.Context, req *models.InferenceRequest, callback func(string) error) error {
	prompt := req.Query
	if req.Context != "" {
		prompt = fmt.Sprintf("Context: %s\n\nQuestion: %s", req.Context, req.Query)
	}

	temperature := float64(req.Temperature)
	if temperature == 0 {
		temperature = 0.7
	}

	streamingFunc := func(ctx context.Context, chunk []byte) error {
		if len(chunk) > 0 {
			return callback(string(chunk))
		}
		return nil
	}

	_, err := llms.GenerateFromSinglePrompt(
		ctx,
		c.llm,
		prompt,
		llms.WithTemperature(temperature),
		llms.WithMaxTokens(c.config.MaxTokens),
		llms.WithStreamingFunc(streamingFunc),
	)

	return err
}
