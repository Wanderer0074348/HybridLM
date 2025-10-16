package handlers

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"www.github.com/Wanderer0074348/HybridLM/src/chat"
	"www.github.com/Wanderer0074348/HybridLM/src/models"
	"www.github.com/Wanderer0074348/HybridLM/src/router"
	"www.github.com/Wanderer0074348/HybridLM/src/utils"
)

type ChatHandler struct {
	queryRouter  *router.QueryRouter
	slmEngine    models.SLMInferencer
	llmClient    models.LLMInferencer
	cache        models.CacheStore
	sessionStore *chat.SessionStore
	llmModelName string
	slmModelName string
}

func NewChatHandler(
	queryRouter *router.QueryRouter,
	slmEngine models.SLMInferencer,
	llmClient models.LLMInferencer,
	cache models.CacheStore,
	sessionStore *chat.SessionStore,
) *ChatHandler {
	return &ChatHandler{
		queryRouter:  queryRouter,
		slmEngine:    slmEngine,
		llmClient:    llmClient,
		cache:        cache,
		sessionStore: sessionStore,
		llmModelName: "gpt-3.5-turbo",
		slmModelName: "llama-3.1-8b-instant",
	}
}

func (h *ChatHandler) SetModelNames(llmModel, slmModel string) {
	h.llmModelName = llmModel
	h.slmModelName = slmModel
}

// HandleChat handles conversational chat requests with session management
func (h *ChatHandler) HandleChat(c *gin.Context) {
	startTime := time.Now()

	var req models.ChatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx := context.Background()

	// Get or create session
	var session *models.ChatSession
	var err error

	if req.SessionID != "" {
		// Try to retrieve existing session
		session, err = h.sessionStore.GetSession(ctx, req.SessionID)
		if err != nil {
			log.Printf("Failed to get session %s: %v, creating new session", req.SessionID, err)
			session, err = h.sessionStore.CreateSession(ctx)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create session"})
				return
			}
		}
	} else {
		// Create new session
		session, err = h.sessionStore.CreateSession(ctx)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create session"})
			return
		}
		log.Printf("Created new chat session: %s", session.SessionID)
	}

	// Build conversation context from session history
	conversationContext := h.sessionStore.BuildConversationContext(session)

	// Create inference request with conversation history
	inferenceReq := &models.InferenceRequest{
		Query:       req.Message,
		Context:     conversationContext,
		MaxTokens:   req.MaxTokens,
		Temperature: req.Temperature,
	}

	// Check cache (with conversation context included in cache key)
	cacheKey := h.queryRouter.GenerateCacheKey(inferenceReq)
	cachedResponse, err := h.cache.Get(ctx, cacheKey)
	if err == nil && cachedResponse != nil {
		// Cache hit - return cached response
		latency := time.Since(startTime)

		// Still add to session history
		inputTokens := utils.EstimateTokenCount(req.Message + conversationContext)
		outputTokens := utils.EstimateTokenCount(cachedResponse.Response)
		h.sessionStore.AddMessage(ctx, session.SessionID, "user", req.Message, inputTokens)
		h.sessionStore.AddMessage(ctx, session.SessionID, "assistant", cachedResponse.Response, outputTokens)

		c.JSON(http.StatusOK, models.ChatResponse{
			SessionID:      session.SessionID,
			Response:       cachedResponse.Response,
			ModelUsed:      cachedResponse.ModelUsed,
			RoutingReason:  "Cache hit (exact match)",
			Latency:        latency,
			CacheHit:       true,
			Timestamp:      time.Now(),
			MessageCount:   session.MessageCount + 1,
			CostMetrics:    cachedResponse.CostMetrics,
		})
		return
	}

	// Route the query
	decision, err := h.queryRouter.Route(ctx, inferenceReq)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Routing failed: %v", err)})
		return
	}

	var response string
	var modelUsed string
	var costMetrics *models.CostMetrics

	if decision.UseLLM {
		// Use LLM (cloud)
		response, err = h.llmClient.Infer(ctx, inferenceReq)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("LLM inference failed: %v", err)})
			return
		}
		modelUsed = h.llmModelName

		// Calculate cost metrics
		costMetrics = utils.CalculateCostMetrics(
			inferenceReq.Query+inferenceReq.Context,
			response,
			"cloud-llm",
			modelUsed,
			false,
			false,
		)
	} else {
		// Use SLM (edge)
		response, err = h.slmEngine.Infer(ctx, inferenceReq)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("SLM inference failed: %v", err)})
			return
		}
		modelUsed = h.slmModelName

		// Calculate cost metrics with savings
		costMetrics = utils.CalculateCostMetrics(
			inferenceReq.Query+inferenceReq.Context,
			response,
			"edge-slm",
			modelUsed,
			false,
			false,
		)
	}

	latency := time.Since(startTime)

	// Store in cache
	inferenceResponse := &models.InferenceResponse{
		Response:      response,
		ModelUsed:     modelUsed,
		RoutingReason: decision.Reason,
		Latency:       latency,
		CacheHit:      false,
		Timestamp:     time.Now(),
		CostMetrics:   costMetrics,
	}

	if err := h.cache.Set(ctx, cacheKey, inferenceResponse); err != nil {
		log.Printf("Failed to cache response: %v", err)
	}

	// Add messages to session history
	inputTokens := utils.EstimateTokenCount(req.Message + conversationContext)
	outputTokens := utils.EstimateTokenCount(response)

	if err := h.sessionStore.AddMessage(ctx, session.SessionID, "user", req.Message, inputTokens); err != nil {
		log.Printf("Failed to add user message to session: %v", err)
	}
	if err := h.sessionStore.AddMessage(ctx, session.SessionID, "assistant", response, outputTokens); err != nil {
		log.Printf("Failed to add assistant message to session: %v", err)
	}

	// Update session
	updatedSession, _ := h.sessionStore.GetSession(ctx, session.SessionID)
	messageCount := 0
	if updatedSession != nil {
		messageCount = updatedSession.MessageCount
	}

	c.JSON(http.StatusOK, models.ChatResponse{
		SessionID:      session.SessionID,
		Response:       response,
		ModelUsed:      modelUsed,
		RoutingReason:  decision.Reason,
		Latency:        latency,
		CacheHit:       false,
		Timestamp:      time.Now(),
		MessageCount:   messageCount,
		CostMetrics:    costMetrics,
	})
}

// GetSession returns session details
func (h *ChatHandler) GetSession(c *gin.Context) {
	sessionID := c.Param("session_id")

	ctx := context.Background()
	session, err := h.sessionStore.GetSession(ctx, sessionID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Session not found"})
		return
	}

	c.JSON(http.StatusOK, session)
}

// DeleteSession deletes a session
func (h *ChatHandler) DeleteSession(c *gin.Context) {
	sessionID := c.Param("session_id")

	ctx := context.Background()
	if err := h.sessionStore.DeleteSession(ctx, sessionID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete session"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Session deleted successfully"})
}

// ListSessions returns all active session IDs
func (h *ChatHandler) ListSessions(c *gin.Context) {
	ctx := context.Background()
	sessionIDs, err := h.sessionStore.GetRecentSessions(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list sessions"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"sessions": sessionIDs,
		"count":    len(sessionIDs),
	})
}
