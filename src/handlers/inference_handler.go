package handlers

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"www.github.com/Wanderer0074348/HybridLM/src/models"
	"www.github.com/Wanderer0074348/HybridLM/src/router"
)

type InferenceHandler struct {
	router        *router.QueryRouter
	slmEngine     models.SLMInferencer     // Changed to interface
	llmClient     models.LLMInferencer     // Changed to interface
	cache         models.CacheStore        // Changed to interface
	semanticCache models.SemanticCacheStore // Semantic cache for similarity search
	useSemanticCache bool
	similarityThreshold float64
}

func NewInferenceHandler(
	r *router.QueryRouter,
	slm models.SLMInferencer, // Changed to interface
	llm models.LLMInferencer, // Changed to interface
	c models.CacheStore, // Changed to interface
) *InferenceHandler {
	return &InferenceHandler{
		router:              r,
		slmEngine:           slm,
		llmClient:           llm,
		cache:               c,
		semanticCache:       nil, // Will be set via SetSemanticCache if enabled
		useSemanticCache:    false,
		similarityThreshold: 0.85,
	}
}

// SetSemanticCache enables semantic caching with the provided cache store
func (h *InferenceHandler) SetSemanticCache(sc models.SemanticCacheStore, threshold float64) {
	h.semanticCache = sc
	h.useSemanticCache = true
	h.similarityThreshold = threshold
}

func (h *InferenceHandler) HandleInference(c *gin.Context) {
	var req models.InferenceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	startTime := time.Now()

	// Check semantic cache first if enabled
	if h.useSemanticCache && h.semanticCache != nil {
		semanticResult, err := h.semanticCache.GetSimilar(c.Request.Context(), req.Query, h.similarityThreshold)
		if err == nil && semanticResult != nil {
			// Found a semantically similar cached response
			semanticResult.Response.CacheHit = true
			semanticResult.Response.Latency = time.Since(startTime)
			semanticResult.Response.RoutingReason = semanticResult.Response.RoutingReason +
				" (semantic cache hit, similarity: " + formatFloat(semanticResult.Similarity) + ")"
			c.JSON(http.StatusOK, semanticResult.Response)
			return
		}
	}

	// Fall back to exact cache check
	cacheKey := h.router.GenerateCacheKey(&req)
	cachedResp, err := h.cache.Get(c.Request.Context(), cacheKey)
	if err == nil && cachedResp != nil {
		cachedResp.CacheHit = true
		cachedResp.Latency = time.Since(startTime)
		c.JSON(http.StatusOK, cachedResp)
		return
	}

	// Route query
	decision, err := h.router.Route(c.Request.Context(), &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "routing failed"})
		return
	}

	var response string
	var modelUsed string

	if decision.UseLLM {
		response, err = h.llmClient.Infer(c.Request.Context(), &req)
		modelUsed = "cloud-llm"
	} else {
		response, err = h.slmEngine.Infer(c.Request.Context(), &req)
		modelUsed = "edge-slm"
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   err.Error(),
			"model":   modelUsed,
			"routing": decision.Reason,
		})
		return
	}

	result := &models.InferenceResponse{
		Response:      response,
		ModelUsed:     modelUsed,
		RoutingReason: decision.Reason,
		Latency:       time.Since(startTime),
		CacheHit:      false,
		Timestamp:     time.Now(),
	}

	// Cache the response
	if h.useSemanticCache && h.semanticCache != nil {
		// Store with embedding for semantic similarity search
		_ = h.semanticCache.SetWithEmbedding(c.Request.Context(), cacheKey, req.Query, result)
	} else {
		// Store with exact key only
		_ = h.cache.Set(c.Request.Context(), cacheKey, result)
	}

	c.JSON(http.StatusOK, result)
}

// formatFloat formats a float64 to 3 decimal places
func formatFloat(f float64) string {
	return fmt.Sprintf("%.3f", f)
}

func (h *InferenceHandler) HealthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":    "healthy",
		"timestamp": time.Now(),
	})
}
