package rl

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/sashabaranov/go-openai"
)

type RewardModel struct {
	client *openai.Client
	model  string
}

type RewardScore struct {
	Score      float64 `json:"score"`
	Reasoning  string  `json:"reasoning"`
	Coherence  float64 `json:"coherence"`
	Accuracy   float64 `json:"accuracy"`
	Relevance  float64 `json:"relevance"`
	Completeness float64 `json:"completeness"`
}

func NewRewardModel(apiKey, model string) *RewardModel {
	config := openai.DefaultConfig(apiKey)
	return &RewardModel{
		client: openai.NewClientWithConfig(config),
		model:  model,
	}
}

func (r *RewardModel) EvaluateResponse(ctx context.Context, query, response string) (*RewardScore, error) {
	systemPrompt := `You are an expert evaluator of AI responses. Your task is to evaluate the quality of a response to a given query.

Evaluate based on:
1. COHERENCE: Is the response well-structured and logical?
2. ACCURACY: Is the information factually correct?
3. RELEVANCE: Does it directly address the query?
4. COMPLETENESS: Is it sufficiently detailed?

Respond ONLY with valid JSON in this exact format:
{
  "score": 0.0-1.0,
  "reasoning": "brief explanation",
  "coherence": 0.0-1.0,
  "accuracy": 0.0-1.0,
  "relevance": 0.0-1.0,
  "completeness": 0.0-1.0
}

Score guidelines:
- 0.9-1.0: Excellent response, comprehensive and accurate
- 0.7-0.9: Good response, mostly accurate and helpful
- 0.5-0.7: Adequate response, but missing depth or has minor issues
- 0.3-0.5: Poor response, significant issues or irrelevant
- 0.0-0.3: Very poor, incorrect or unhelpful

Do not include any other text, formatting, or markdown.`

	userPrompt := fmt.Sprintf(`Query: %s

Response: %s

Evaluate this response.`, query, response)

	resp, err := r.client.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
		Model: r.model,
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
		MaxTokens:   300,
	})

	if err != nil {
		return nil, fmt.Errorf("reward model evaluation failed: %w", err)
	}

	if len(resp.Choices) == 0 {
		return nil, fmt.Errorf("no response from reward model")
	}

	content := strings.TrimSpace(resp.Choices[0].Message.Content)
	content = strings.TrimPrefix(content, "```json")
	content = strings.TrimPrefix(content, "```")
	content = strings.TrimSuffix(content, "```")
	content = strings.TrimSpace(content)

	var rewardScore RewardScore
	if err := json.Unmarshal([]byte(content), &rewardScore); err != nil {
		return nil, fmt.Errorf("failed to parse reward score: %w", err)
	}

	return &rewardScore, nil
}

func (r *RewardModel) CalculateReward(
	qualityScore float64,
	latencyMs int,
	costUSD float64,
	usedLLM bool,
) float64 {
	qualityWeight := 0.7
	latencyWeight := 0.15
	costWeight := 0.15

	normalizedLatency := 1.0 - min(float64(latencyMs)/5000.0, 1.0)
	normalizedCost := 1.0
	if usedLLM {
		normalizedCost = 1.0 - min(costUSD/0.01, 1.0)
	}

	reward := (qualityScore * qualityWeight) +
		(normalizedLatency * latencyWeight) +
		(normalizedCost * costWeight)

	return reward
}

func min(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}
