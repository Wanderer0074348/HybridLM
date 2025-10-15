package inference

/*
Hybrid SLM Inference Engine

This engine implements three inference strategies for Small Language Models (SLMs):

1. PARALLEL Strategy (like parallel resistors):
   - Runs all models simultaneously with the same prompt
   - Aggregates results using weighted voting, longest response, or similarity-based voting
   - Fast but uses more resources
   - Best for: Quick, diverse perspectives on the same query

2. SERIES Strategy (like series resistors):
   - Chains models sequentially, each refining the previous output
   - Model 1 generates initial response → Model 2 refines → Model 3 further refines
   - Slower but produces highly refined outputs
   - Best for: Complex queries requiring iterative improvement

3. HYBRID Strategy (parallel + series combination):
   - Phase 1: First N-1 models run in parallel
   - Phase 2: Best aggregated response is refined by the last (most capable) model
   - Balances speed and quality
   - Best for: General use cases requiring both diversity and refinement

Configuration (config.yaml):
- strategy: "parallel" | "series" | "hybrid"
- aggregation_fn: "weighted" | "longest" | "voting"
- models: Array of models with name, endpoint, api_key, and weight

Example:
  models:
    - name: "llama-3.1-8b-instant"    # Fast model
      weight: 1.5
    - name: "llama-3.3-70b-versatile"  # Capable model
      weight: 2.0
    - name: "mixtral-8x7b-32768"       # Diverse reasoning
      weight: 1.8
*/

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"

	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/openai"

	"www.github.com/Wanderer0074348/HybridLM/src/config"
	"www.github.com/Wanderer0074348/HybridLM/src/models"
)

type modelClient struct {
	name   string
	llm    llms.Model
	weight float64
}

type inferenceResult struct {
	modelName string
	response  string
	weight    float64
	err       error
}

type SLMEngine struct {
	config     *config.SLMConfig
	clients    []modelClient
	workerPool chan struct{}
	mu         sync.RWMutex
}

func NewSLMEngine(cfg *config.SLMConfig) (*SLMEngine, error) {

	if len(cfg.Models) == 0 {
		return nil, fmt.Errorf("no models configured in SLM config")
	}

	// Create clients for all configured models
	clients := make([]modelClient, 0, len(cfg.Models))

	for _, modelCfg := range cfg.Models {
		// Validate model config
		if modelCfg.Name == "" {
			return nil, fmt.Errorf("model name is empty in config")
		}
		if modelCfg.Endpoint == "" {
			return nil, fmt.Errorf("endpoint is empty for model %s", modelCfg.Name)
		}
		if modelCfg.APIKey == "" {
			return nil, fmt.Errorf("API key is empty for model %s (check GROQ_API_KEY environment variable)", modelCfg.Name)
		}

		llm, err := openai.New(
			openai.WithBaseURL(modelCfg.Endpoint),
			openai.WithToken(modelCfg.APIKey),
			openai.WithModel(modelCfg.Name),
		)
		if err != nil {
			return nil, fmt.Errorf("failed to create client for model %s: %w", modelCfg.Name, err)
		}

		clients = append(clients, modelClient{
			name:   modelCfg.Name,
			llm:    llm,
			weight: modelCfg.Weight,
		})
	}

	workerPool := make(chan struct{}, cfg.MaxConcurrent)

	return &SLMEngine{
		config:     cfg,
		clients:    clients,
		workerPool: workerPool,
	}, nil
}

func (e *SLMEngine) Infer(ctx context.Context, req *models.InferenceRequest) (string, error) {

	select {
	case e.workerPool <- struct{}{}:
		defer func() { <-e.workerPool }()
	case <-ctx.Done():
		return "", ctx.Err()
	}

	e.mu.RLock()
	defer e.mu.RUnlock()

	// Choose strategy based on configuration
	switch e.config.Strategy {
	case "parallel":
		return e.inferParallel(ctx, req)
	case "series":
		return e.inferSeries(ctx, req)
	case "hybrid":
		return e.inferHybrid(ctx, req)
	default:
		// Default to first model if strategy not recognized
		return e.inferSingleModel(ctx, req, e.clients[0])
	}
}

// Parallel inference: Run all models simultaneously and aggregate results
func (e *SLMEngine) inferParallel(ctx context.Context, req *models.InferenceRequest) (string, error) {
	results := make(chan inferenceResult, len(e.clients))
	var wg sync.WaitGroup

	prompt := e.buildPrompt(req)

	// Run all models in parallel
	for _, client := range e.clients {
		wg.Add(1)
		go func(c modelClient) {
			defer wg.Done()

			response, err := e.runModel(ctx, c, prompt, req.Temperature)
			results <- inferenceResult{
				modelName: c.name,
				response:  response,
				weight:    c.weight,
				err:       err,
			}
		}(client)
	}

	// Wait for all models to complete
	go func() {
		wg.Wait()
		close(results)
	}()

	// Collect results
	var allResults []inferenceResult
	for result := range results {
		allResults = append(allResults, result)
	}

	// Aggregate results
	return e.aggregateResults(allResults)
}

// Series inference: Chain models sequentially, each refining the previous output
func (e *SLMEngine) inferSeries(ctx context.Context, req *models.InferenceRequest) (string, error) {
	prompt := e.buildPrompt(req)

	// First model generates initial response
	response, err := e.runModel(ctx, e.clients[0], prompt, req.Temperature)
	if err != nil {
		return "", fmt.Errorf("first model failed: %w", err)
	}

	// Subsequent models refine the response
	for i := 1; i < len(e.clients); i++ {
		refinementPrompt := fmt.Sprintf(
			"Original query: %s\n\nPrevious response: %s\n\nPlease refine and improve the above response, making it more accurate and comprehensive:",
			req.Query,
			response,
		)

		refined, err := e.runModel(ctx, e.clients[i], refinementPrompt, req.Temperature)
		if err != nil {
			// If refinement fails, return previous response
			return response, nil
		}
		response = refined
	}

	return response, nil
}

// Hybrid inference: Parallel first, then series refinement with best result
func (e *SLMEngine) inferHybrid(ctx context.Context, req *models.InferenceRequest) (string, error) {
	// Phase 1: Parallel inference with first N-1 models
	parallelCount := len(e.clients) - 1
	if parallelCount < 1 {
		parallelCount = 1
	}

	results := make(chan inferenceResult, parallelCount)
	var wg sync.WaitGroup

	prompt := e.buildPrompt(req)

	// Run parallel inference
	for i := 0; i < parallelCount; i++ {
		wg.Add(1)
		go func(c modelClient) {
			defer wg.Done()

			response, err := e.runModel(ctx, c, prompt, req.Temperature)
			results <- inferenceResult{
				modelName: c.name,
				response:  response,
				weight:    c.weight,
				err:       err,
			}
		}(e.clients[i])
	}

	go func() {
		wg.Wait()
		close(results)
	}()

	// Collect parallel results
	var allResults []inferenceResult
	for result := range results {
		allResults = append(allResults, result)
	}

	// Get best response from parallel phase
	bestResponse, err := e.aggregateResults(allResults)
	if err != nil {
		return "", err
	}

	// Phase 2: Refine with the last (usually most capable) model
	if len(e.clients) > 1 {
		lastModel := e.clients[len(e.clients)-1]
		refinementPrompt := fmt.Sprintf(
			"Original query: %s\n\nAggregated response from multiple models: %s\n\nPlease provide a refined, comprehensive answer:",
			req.Query,
			bestResponse,
		)

		refined, err := e.runModel(ctx, lastModel, refinementPrompt, req.Temperature)
		if err != nil {
			// If refinement fails, return aggregated response
			return bestResponse, nil
		}
		return refined, nil
	}

	return bestResponse, nil
}

// Helper: Run a single model
func (e *SLMEngine) inferSingleModel(ctx context.Context, req *models.InferenceRequest, client modelClient) (string, error) {
	prompt := e.buildPrompt(req)
	return e.runModel(ctx, client, prompt, req.Temperature)
}

// Helper: Build prompt from request
func (e *SLMEngine) buildPrompt(req *models.InferenceRequest) string {
	if req.Context != "" {
		return fmt.Sprintf("Context: %s\n\nQuestion: %s", req.Context, req.Query)
	}
	return req.Query
}

// Helper: Run inference on a specific model
func (e *SLMEngine) runModel(ctx context.Context, client modelClient, prompt string, temperature float32) (string, error) {
	temp := float64(temperature)
	if temp == 0 {
		temp = 0.7
	}

	callOptions := []llms.CallOption{
		llms.WithTemperature(temp),
		llms.WithMaxTokens(e.config.MaxTokens),
	}

	response, err := llms.GenerateFromSinglePrompt(
		ctx,
		client.llm,
		prompt,
		callOptions...,
	)
	if err != nil {
		return "", fmt.Errorf("model %s generation failed: %w", client.name, err)
	}

	return response, nil
}

// Helper: Aggregate results from multiple models
func (e *SLMEngine) aggregateResults(results []inferenceResult) (string, error) {
	// Filter out errors and collect error messages
	validResults := make([]inferenceResult, 0)
	var errorMessages []string

	for _, r := range results {
		if r.err == nil && r.response != "" {
			validResults = append(validResults, r)
		} else if r.err != nil {
			errorMessages = append(errorMessages, fmt.Sprintf("%s: %v", r.modelName, r.err))
		}
	}

	if len(validResults) == 0 {
		errorDetail := ""
		if len(errorMessages) > 0 {
			errorDetail = " - Errors: " + strings.Join(errorMessages, "; ")
		}
		return "", fmt.Errorf("all models failed to generate responses%s", errorDetail)
	}

	switch e.config.AggregationFn {
	case "weighted":
		return e.aggregateWeighted(validResults), nil
	case "longest":
		return e.aggregateLongest(validResults), nil
	case "voting":
		return e.aggregateVoting(validResults), nil
	default:
		// Default to weighted
		return e.aggregateWeighted(validResults), nil
	}
}

// Weighted aggregation: Choose response from highest weighted model
func (e *SLMEngine) aggregateWeighted(results []inferenceResult) string {
	sort.Slice(results, func(i, j int) bool {
		return results[i].weight > results[j].weight
	})
	return results[0].response
}

// Longest aggregation: Choose the most detailed response
func (e *SLMEngine) aggregateLongest(results []inferenceResult) string {
	sort.Slice(results, func(i, j int) bool {
		return len(results[i].response) > len(results[j].response)
	})
	return results[0].response
}

// Voting aggregation: Simple similarity-based voting (returns most common pattern)
func (e *SLMEngine) aggregateVoting(results []inferenceResult) string {
	if len(results) == 1 {
		return results[0].response
	}

	// For simplicity, use weighted approach with a twist:
	// Find the response with the most similar responses (by length and first words)
	type scored struct {
		result inferenceResult
		score  float64
	}

	scores := make([]scored, len(results))

	for i, r1 := range results {
		score := r1.weight // Start with base weight
		for j, r2 := range results {
			if i != j {
				// Add similarity bonus
				similarity := e.calculateSimilarity(r1.response, r2.response)
				score += similarity * r2.weight
			}
		}
		scores[i] = scored{result: r1, score: score}
	}

	// Return highest scoring response
	sort.Slice(scores, func(i, j int) bool {
		return scores[i].score > scores[j].score
	})

	return scores[0].result.response
}

// Simple similarity metric based on length and common words
func (e *SLMEngine) calculateSimilarity(s1, s2 string) float64 {
	words1 := strings.Fields(strings.ToLower(s1))
	words2 := strings.Fields(strings.ToLower(s2))

	if len(words1) == 0 || len(words2) == 0 {
		return 0.0
	}

	// Count common words
	wordSet := make(map[string]bool)
	for _, w := range words1 {
		wordSet[w] = true
	}

	common := 0
	for _, w := range words2 {
		if wordSet[w] {
			common++
		}
	}

	// Jaccard similarity
	union := len(words1) + len(words2) - common
	if union == 0 {
		return 0.0
	}

	return float64(common) / float64(union)
}

func (e *SLMEngine) InferStreaming(ctx context.Context, req *models.InferenceRequest, callback func(string) error) error {

	select {
	case e.workerPool <- struct{}{}:
		defer func() { <-e.workerPool }()
	case <-ctx.Done():
		return ctx.Err()
	}

	e.mu.RLock()
	defer e.mu.RUnlock()

	// For streaming, use the first (fastest) model only
	// Hybrid/parallel strategies don't work well with streaming
	prompt := e.buildPrompt(req)

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
		e.clients[0].llm,
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
