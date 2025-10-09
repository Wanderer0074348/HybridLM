package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"www.github.com/Wanderer0074348/HybridLM/src/models"
	"www.github.com/Wanderer0074348/HybridLM/src/router"
)

type InferenceHandler struct {
	router    *router.QueryRouter
	slmEngine models.SLMInferencer // Changed to interface
	llmClient models.LLMInferencer // Changed to interface
	cache     models.CacheStore    // Changed to interface
}

func NewInferenceHandler(
	r *router.QueryRouter,
	slm models.SLMInferencer, // Changed to interface
	llm models.LLMInferencer, // Changed to interface
	c models.CacheStore, // Changed to interface
) *InferenceHandler {
	return &InferenceHandler{
		router:    r,
		slmEngine: slm,
		llmClient: llm,
		cache:     c,
	}
}

func (h *InferenceHandler) HandleInference(c *gin.Context) {
	var req models.InferenceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	startTime := time.Now()

	// Check cache first
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
	_ = h.cache.Set(c.Request.Context(), cacheKey, result)

	c.JSON(http.StatusOK, result)
}

func (h *InferenceHandler) HealthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":    "healthy",
		"timestamp": time.Now(),
	})
}
