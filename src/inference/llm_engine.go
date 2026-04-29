package inference

import (
	"context"
	"fmt"
	"strings"

	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/openai"

	"www.github.com/Wanderer0074348/HybridLM/src/config"
	"www.github.com/Wanderer0074348/HybridLM/src/models"
)

func supportsCustomTemperature(model string) bool {
	m := strings.ToLower(model)
	return !strings.HasPrefix(m, "gpt-5") && !strings.HasPrefix(m, "o1") && !strings.HasPrefix(m, "o3")
}

func (c *LLMClient) buildCallOptions(req *models.InferenceRequest, extra ...llms.CallOption) []llms.CallOption {
	temperature := 1.0
	if supportsCustomTemperature(c.config.Model) {
		temperature = float64(req.Temperature)
		if temperature == 0 {
			temperature = 0.7
		}
	}
	opts := []llms.CallOption{
		llms.WithTemperature(temperature),
		llms.WithMaxTokens(c.config.MaxTokens),
	}
	return append(opts, extra...)
}

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

	response, err := llms.GenerateFromSinglePrompt(
		ctx,
		c.llm,
		prompt,
		c.buildCallOptions(req)...,
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

	streamingFunc := func(_ context.Context, chunk []byte) error {
		if len(chunk) > 0 {
			return callback(string(chunk))
		}
		return nil
	}

	_, err := llms.GenerateFromSinglePrompt(
		ctx,
		c.llm,
		prompt,
		c.buildCallOptions(req, llms.WithStreamingFunc(streamingFunc))...,
	)

	return err
}
