package features

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"www.github.com/Wanderer0074348/HybridLM/src/models"

	"github.com/sashabaranov/go-openai"
)

type SLMComplexityAssessor struct {
	client   *openai.Client
	model    string
	endpoint string
}

func NewSLMComplexityAssessor(apiKey, endpoint, model string) *SLMComplexityAssessor {
	config := openai.DefaultConfig(apiKey)
	if endpoint != "" {
		config.BaseURL = endpoint + "/v1"
	}

	return &SLMComplexityAssessor{
		client:   openai.NewClientWithConfig(config),
		model:    model,
		endpoint: endpoint,
	}
}

func (s *SLMComplexityAssessor) AssessComplexity(ctx context.Context, query string) (*models.SLMComplexityAssessment, error) {
	systemPrompt := `You are a query complexity analyzer. Your task is to determine if a query is complex or simple.

Complex queries typically:
- Require multi-step reasoning
- Involve comparisons or analysis
- Need deep domain knowledge
- Request detailed explanations
- Involve creative or open-ended thinking
- Require synthesis of multiple concepts

Simple queries typically:
- Ask for factual information
- Have straightforward yes/no answers
- Request basic definitions
- Involve simple calculations
- Can be answered with direct lookup

Respond ONLY with valid JSON in this exact format:
{"is_complex": true/false, "confidence": 0.0-1.0, "reasoning": "brief explanation"}

Do not include any other text, formatting, or markdown.`

	userPrompt := fmt.Sprintf("Analyze this query: %s", query)

	resp, err := s.client.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
		Model: s.model,
		Messages: []openai.ChatCompletionMessage{
			{
				Role:    openai.ChatMessageRoleSystem,
				Content: systemPrompt,
			},
			{
				Role:    openai.ChatMessageRoleUser,
				Content: userPrompt,
			},
		},
		Temperature: 0.1,
		MaxTokens:   150,
	})

	if err != nil {
		return nil, fmt.Errorf("SLM assessment failed: %w", err)
	}

	if len(resp.Choices) == 0 {
		return nil, fmt.Errorf("no response from SLM assessor")
	}

	content := strings.TrimSpace(resp.Choices[0].Message.Content)

	content = strings.TrimPrefix(content, "```json")
	content = strings.TrimPrefix(content, "```")
	content = strings.TrimSuffix(content, "```")
	content = strings.TrimSpace(content)

	var assessment models.SLMComplexityAssessment
	if err := json.Unmarshal([]byte(content), &assessment); err != nil {
		return nil, fmt.Errorf("failed to parse SLM assessment: %w", err)
	}

	return &assessment, nil
}
