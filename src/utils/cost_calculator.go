package utils

import (
	"strings"

	"www.github.com/Wanderer0074348/HybridLM/src/models"
)

// Pricing per 1M tokens (as of 2025)
const (
	// OpenAI GPT-3.5-turbo
	GPT35InputPer1M  = 0.50  // $0.50 per 1M input tokens
	GPT35OutputPer1M = 1.50  // $1.50 per 1M output tokens

	// OpenAI GPT-4
	GPT4InputPer1M  = 30.00 // $30 per 1M input tokens
	GPT4OutputPer1M = 60.00 // $60 per 1M output tokens

	// Groq (free tier, but for estimation we use costs)
	GroqInputPer1M  = 0.10 // $0.10 per 1M tokens (estimate for Llama)
	GroqOutputPer1M = 0.10 // $0.10 per 1M tokens

	// OpenAI Embeddings
	EmbeddingPer1M = 0.10 // $0.10 per 1M tokens (text-embedding-ada-002)
)

// EstimateTokenCount estimates token count from text (rough approximation)
// More accurate: ~1 token per 4 characters for English
func EstimateTokenCount(text string) int {
	// Remove extra whitespace
	text = strings.TrimSpace(text)

	// Rough estimate: 1 token ≈ 4 characters
	charCount := len(text)
	tokenCount := charCount / 4

	// Add some buffer for special tokens
	if tokenCount < 10 {
		tokenCount = 10
	}

	return tokenCount
}

// CalculateLLMCost calculates the cost for LLM inference
func CalculateLLMCost(inputTokens, outputTokens int, model string) float64 {
	var inputCost, outputCost float64

	// Determine pricing based on model
	switch {
	case strings.Contains(strings.ToLower(model), "gpt-4"):
		inputCost = float64(inputTokens) * GPT4InputPer1M / 1000000
		outputCost = float64(outputTokens) * GPT4OutputPer1M / 1000000
	case strings.Contains(strings.ToLower(model), "gpt-3.5"):
		inputCost = float64(inputTokens) * GPT35InputPer1M / 1000000
		outputCost = float64(outputTokens) * GPT35OutputPer1M / 1000000
	default:
		// Default to GPT-3.5 pricing
		inputCost = float64(inputTokens) * GPT35InputPer1M / 1000000
		outputCost = float64(outputTokens) * GPT35OutputPer1M / 1000000
	}

	return inputCost + outputCost
}

// CalculateSLMCost calculates the cost for SLM inference (Groq models)
func CalculateSLMCost(inputTokens, outputTokens int) float64 {
	inputCost := float64(inputTokens) * GroqInputPer1M / 1000000
	outputCost := float64(outputTokens) * GroqOutputPer1M / 1000000
	return inputCost + outputCost
}

// CalculateEmbeddingCost calculates the cost for generating embeddings
func CalculateEmbeddingCost(tokens int) float64 {
	return float64(tokens) * EmbeddingPer1M / 1000000
}

// CalculateCostMetrics calculates comprehensive cost metrics for an inference
func CalculateCostMetrics(
	query string,
	response string,
	modelUsed string,
	specificModel string,
	cacheHit bool,
	semanticCacheEnabled bool,
) *models.CostMetrics {
	inputTokens := EstimateTokenCount(query)
	outputTokens := EstimateTokenCount(response)
	totalTokens := inputTokens + outputTokens

	metrics := &models.CostMetrics{
		InputTokens:  inputTokens,
		OutputTokens: outputTokens,
		TotalTokens:  totalTokens,
		Model:        specificModel,
	}

	// If cache hit, only count embedding cost (if semantic cache is enabled)
	if cacheHit {
		if semanticCacheEnabled {
			// Only paid for embedding generation to check similarity
			metrics.CacheCost = CalculateEmbeddingCost(inputTokens)
			metrics.TotalCost = metrics.CacheCost
		} else {
			// Exact cache hit - no cost at all
			metrics.Cost = 0
			metrics.CacheCost = 0
			metrics.TotalCost = 0
		}

		// Calculate what it would have cost without cache
		if modelUsed == "cloud-llm" {
			metrics.EstimatedSavings = CalculateLLMCost(inputTokens, outputTokens, specificModel)
		} else {
			metrics.EstimatedSavings = CalculateSLMCost(inputTokens, outputTokens)
		}

		return metrics
	}

	// Calculate inference cost based on model used
	if modelUsed == "cloud-llm" {
		metrics.Cost = CalculateLLMCost(inputTokens, outputTokens, specificModel)
		// No savings since we used the expensive model
		metrics.EstimatedSavings = 0
	} else {
		// SLM used
		metrics.Cost = CalculateSLMCost(inputTokens, outputTokens)
		// Calculate savings compared to if we had used LLM
		llmCost := CalculateLLMCost(inputTokens, outputTokens, "gpt-3.5-turbo")
		metrics.EstimatedSavings = llmCost - metrics.Cost
	}

	// Add embedding cost if semantic cache is enabled (we generate embeddings for caching)
	if semanticCacheEnabled {
		metrics.CacheCost = CalculateEmbeddingCost(inputTokens)
	}

	metrics.TotalCost = metrics.Cost + metrics.CacheCost

	return metrics
}
