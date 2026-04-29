package utils

import (
	"strings"

	"www.github.com/Wanderer0074348/HybridLM/src/models"
)

// Pricing per 1M tokens (as of April 2026)
const (
	// OpenAI GPT-3.5-turbo
	GPT35InputPer1M  = 0.50
	GPT35OutputPer1M = 1.50

	// OpenAI GPT-4.1
	GPT4InputPer1M  = 2.00
	GPT4OutputPer1M = 8.00

	// OpenAI GPT-5.5 (released 2026-04-23)
	GPT55InputPer1M  = 5.00
	GPT55OutputPer1M = 30.00

	// OpenAI GPT-5.4 mini (used as RL judge)
	GPT54MiniInputPer1M  = 0.25
	GPT54MiniOutputPer1M = 2.00

	// Groq — per-model pricing (as of 2025)
	// llama-3.1-8b-instant
	GroqLlama8bInputPer1M  = 0.05 // $0.05 per 1M input tokens
	GroqLlama8bOutputPer1M = 0.08 // $0.08 per 1M output tokens
	// llama-3.3-70b-versatile
	GroqLlama70bInputPer1M  = 0.59 // $0.59 per 1M input tokens
	GroqLlama70bOutputPer1M = 0.79 // $0.79 per 1M output tokens
	// openai/gpt-oss-20b (on Groq)
	GroqGPTOss20bInputPer1M  = 0.90 // $0.90 per 1M input tokens
	GroqGPTOss20bOutputPer1M = 0.90 // $0.90 per 1M output tokens
	// meta-llama/llama-4-scout-17b-16e-instruct
	GroqLlama4ScoutInputPer1M  = 0.11 // $0.11 per 1M input tokens
	GroqLlama4ScoutOutputPer1M = 0.34 // $0.34 per 1M output tokens
	// Fallback for unknown Groq models
	GroqDefaultInputPer1M  = 0.10
	GroqDefaultOutputPer1M = 0.10

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

	m := strings.ToLower(model)
	switch {
	case strings.Contains(m, "gpt-5.4-mini") || strings.Contains(m, "gpt-5-4-mini"):
		inputCost = float64(inputTokens) * GPT54MiniInputPer1M / 1000000
		outputCost = float64(outputTokens) * GPT54MiniOutputPer1M / 1000000
	case strings.Contains(m, "gpt-5"):
		inputCost = float64(inputTokens) * GPT55InputPer1M / 1000000
		outputCost = float64(outputTokens) * GPT55OutputPer1M / 1000000
	case strings.Contains(m, "gpt-4"):
		inputCost = float64(inputTokens) * GPT4InputPer1M / 1000000
		outputCost = float64(outputTokens) * GPT4OutputPer1M / 1000000
	case strings.Contains(m, "gpt-3.5"):
		inputCost = float64(inputTokens) * GPT35InputPer1M / 1000000
		outputCost = float64(outputTokens) * GPT35OutputPer1M / 1000000
	default:
		inputCost = float64(inputTokens) * GPT55InputPer1M / 1000000
		outputCost = float64(outputTokens) * GPT55OutputPer1M / 1000000
	}

	return inputCost + outputCost
}

// CalculateSLMCost calculates the cost for SLM inference (Groq models)
// model should be the specific Groq model name for accurate per-model pricing.
func CalculateSLMCost(inputTokens, outputTokens int, model string) float64 {
	var inputRate, outputRate float64
	switch {
	case strings.Contains(model, "llama-3.3-70b") || strings.Contains(model, "llama3-70b"):
		inputRate, outputRate = GroqLlama70bInputPer1M, GroqLlama70bOutputPer1M
	case strings.Contains(model, "llama-3.1-8b") || strings.Contains(model, "llama3-8b"):
		inputRate, outputRate = GroqLlama8bInputPer1M, GroqLlama8bOutputPer1M
	case strings.Contains(model, "llama-4-scout"):
		inputRate, outputRate = GroqLlama4ScoutInputPer1M, GroqLlama4ScoutOutputPer1M
	case strings.Contains(model, "gpt-oss-20b"):
		inputRate, outputRate = GroqGPTOss20bInputPer1M, GroqGPTOss20bOutputPer1M
	default:
		inputRate, outputRate = GroqDefaultInputPer1M, GroqDefaultOutputPer1M
	}
	return float64(inputTokens)*inputRate/1000000 + float64(outputTokens)*outputRate/1000000
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
			metrics.EstimatedSavings = CalculateSLMCost(inputTokens, outputTokens, specificModel)
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
		metrics.Cost = CalculateSLMCost(inputTokens, outputTokens, specificModel)
		// Calculate savings compared to if we had used LLM
		llmCost := CalculateLLMCost(inputTokens, outputTokens, "gpt-5.5")
		metrics.EstimatedSavings = llmCost - metrics.Cost
	}

	// Add embedding cost if semantic cache is enabled (we generate embeddings for caching)
	if semanticCacheEnabled {
		metrics.CacheCost = CalculateEmbeddingCost(inputTokens)
	}

	metrics.TotalCost = metrics.Cost + metrics.CacheCost

	return metrics
}
