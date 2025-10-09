package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"www.github.com/Wanderer0074348/HybridLM/src/config"
	"www.github.com/Wanderer0074348/HybridLM/src/mocks"
	"www.github.com/Wanderer0074348/HybridLM/src/models"
	"www.github.com/Wanderer0074348/HybridLM/src/router"
)

func setupTestHandler() (*InferenceHandler, *mocks.MockLLMClient, *mocks.MockSLMEngine, *mocks.MockCache) {
	gin.SetMode(gin.TestMode)

	mockLLM := new(mocks.MockLLMClient)
	mockSLM := new(mocks.MockSLMEngine)
	mockCache := new(mocks.MockCache)

	cfg := &config.RouterConfig{
		ComplexityThreshold: 0.65,
	}
	queryRouter := router.NewQueryRouter(cfg)

	// Now using interfaces - this will work!
	handler := NewInferenceHandler(queryRouter, mockSLM, mockLLM, mockCache)

	return handler, mockLLM, mockSLM, mockCache
}

func TestInferenceHandler_SimpleQuery(t *testing.T) {
	handler, _, mockSLM, mockCache := setupTestHandler()

	mockCache.On("Get", mock.Anything, mock.Anything).Return(nil, nil)
	mockSLM.On("Infer", mock.Anything, mock.Anything).Return("4", nil)
	mockCache.On("Set", mock.Anything, mock.Anything, mock.Anything).Return(nil)

	reqBody := models.InferenceRequest{
		Query:       "What is 2+2?",
		Temperature: 0.7,
	}
	jsonBody, _ := json.Marshal(reqBody)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/api/v1/inference", bytes.NewBuffer(jsonBody))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.HandleInference(c)

	assert.Equal(t, http.StatusOK, w.Code)

	var response models.InferenceResponse
	json.Unmarshal(w.Body.Bytes(), &response)

	assert.Equal(t, "4", response.Response)
	assert.Equal(t, "edge-slm", response.ModelUsed)
	assert.False(t, response.CacheHit)

	mockSLM.AssertExpectations(t)
	mockCache.AssertExpectations(t)
}

func TestInferenceHandler_ComplexQuery(t *testing.T) {
	handler, mockLLM, _, mockCache := setupTestHandler()

	mockCache.On("Get", mock.Anything, mock.Anything).Return(nil, nil)
	mockLLM.On("Infer", mock.Anything, mock.Anything).Return("Detailed explanation", nil)
	mockCache.On("Set", mock.Anything, mock.Anything, mock.Anything).Return(nil)

	reqBody := models.InferenceRequest{
		Query:       "Simple question",
		Context:     "With some context to force LLM routing",
		Temperature: 0.7,
	}
	jsonBody, _ := json.Marshal(reqBody)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/api/v1/inference", bytes.NewBuffer(jsonBody))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.HandleInference(c)

	assert.Equal(t, http.StatusOK, w.Code)

	var response models.InferenceResponse
	json.Unmarshal(w.Body.Bytes(), &response)

	assert.Equal(t, "cloud-llm", response.ModelUsed)

	mockLLM.AssertExpectations(t)
	mockCache.AssertExpectations(t)
}

func TestInferenceHandler_CacheHit(t *testing.T) {
	handler, _, _, mockCache := setupTestHandler()

	cachedResponse := &models.InferenceResponse{
		Response:      "Cached answer",
		ModelUsed:     "edge-slm",
		RoutingReason: "Simple query",
		Latency:       50 * time.Millisecond,
		Timestamp:     time.Now(),
	}

	mockCache.On("Get", mock.Anything, mock.Anything).Return(cachedResponse, nil)

	reqBody := models.InferenceRequest{Query: "What is 2+2?"}
	jsonBody, _ := json.Marshal(reqBody)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/api/v1/inference", bytes.NewBuffer(jsonBody))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.HandleInference(c)

	assert.Equal(t, http.StatusOK, w.Code)

	var response models.InferenceResponse
	json.Unmarshal(w.Body.Bytes(), &response)

	assert.True(t, response.CacheHit)
	assert.Equal(t, "Cached answer", response.Response)
}

func TestInferenceHandler_InvalidRequest(t *testing.T) {
	handler, _, _, _ := setupTestHandler()

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/api/v1/inference", bytes.NewBufferString("invalid json"))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.HandleInference(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestInferenceHandler_HealthCheck(t *testing.T) {
	handler, _, _, _ := setupTestHandler()

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/api/v1/health", nil)

	handler.HealthCheck(c)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)

	assert.Equal(t, "healthy", response["status"])
}
