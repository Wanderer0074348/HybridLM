package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"www.github.com/Wanderer0074348/HybridLM/src/cache"
	"www.github.com/Wanderer0074348/HybridLM/src/inference"
	"www.github.com/Wanderer0074348/HybridLM/src/models"
	"www.github.com/Wanderer0074348/HybridLM/src/router"
)

type InferenceHandler struct {
	router    *router.QueryRouter
	slmEngine *inference.SLMEngine
	llmClient *inference.LLMClient
	cache     *cache.RedisCache
}

func NewInferenceHandler(
	r *router.QueryRouter,
	slm *inference.SLMEngine,
	llm *inference.LLMClient,
	c *cache.RedisCache,
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

	cacheKey := h.router.GenerateCacheKey(&req)
	cachedResp, err := h.cache.Get(c.Request.Context(), cacheKey)
	if err == nil && cachedResp != nil {
		cachedResp.CacheHit = true
		cachedResp.Latency = time.Since(startTime)
		c.JSON(http.StatusOK, cachedResp)
		return
	}

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

	_ = h.cache.Set(c.Request.Context(), cacheKey, result)

	c.JSON(http.StatusOK, result)
}

func (h *InferenceHandler) HealthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":    "healthy",
		"timestamp": time.Now(),
	})
}
