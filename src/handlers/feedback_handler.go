package handlers

import (
	"net/http"

	"www.github.com/Wanderer0074348/HybridLM/src/models"
	"www.github.com/Wanderer0074348/HybridLM/src/repository"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type FeedbackHandler struct {
	routingRepo *repository.RoutingRepository
}

func NewFeedbackHandler(routingRepo *repository.RoutingRepository) *FeedbackHandler {
	return &FeedbackHandler{
		routingRepo: routingRepo,
	}
}

func (h *FeedbackHandler) SubmitFeedback(c *gin.Context) {
	var req models.FeedbackRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.Rating != nil && (*req.Rating < 1 || *req.Rating > 5) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "rating must be between 1 and 5"})
		return
	}

	if req.Thumbs != "" && req.Thumbs != "up" && req.Thumbs != "down" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "thumbs must be 'up' or 'down'"})
		return
	}

	decisionID, err := primitive.ObjectIDFromHex(req.DecisionID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid decision_id"})
		return
	}

	feedback := &models.UserFeedback{
		Rating:  req.Rating,
		Thumbs:  req.Thumbs,
		Comment: req.Comment,
	}

	err = h.routingRepo.UpdateFeedback(c.Request.Context(), decisionID, feedback)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save feedback"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "feedback submitted successfully",
		"decision_id": req.DecisionID,
	})
}

func (h *FeedbackHandler) GetDecision(c *gin.Context) {
	decisionIDStr := c.Param("id")

	decisionID, err := primitive.ObjectIDFromHex(decisionIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid decision_id"})
		return
	}

	decision, err := h.routingRepo.GetDecisionByID(c.Request.Context(), decisionID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "decision not found"})
		return
	}

	c.JSON(http.StatusOK, decision)
}

type ABTestHandler struct {
	abTestRepo    *repository.ABTestRepository
	routingRepo   *repository.RoutingRepository
}

func NewABTestHandler(abTestRepo *repository.ABTestRepository, routingRepo *repository.RoutingRepository) *ABTestHandler {
	return &ABTestHandler{
		abTestRepo:  abTestRepo,
		routingRepo: routingRepo,
	}
}

func (h *ABTestHandler) CreateTest(c *gin.Context) {
	var test models.ABTestConfig
	if err := c.ShouldBindJSON(&test); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if test.TrafficSplit < 0 || test.TrafficSplit > 1 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "traffic_split must be between 0 and 1"})
		return
	}

	err := h.abTestRepo.CreateTest(c.Request.Context(), &test)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create test"})
		return
	}

	c.JSON(http.StatusCreated, test)
}

func (h *ABTestHandler) GetActiveTest(c *gin.Context) {
	test, err := h.abTestRepo.GetActiveTest(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get active test"})
		return
	}

	if test == nil {
		c.JSON(http.StatusNotFound, gin.H{"message": "no active test"})
		return
	}

	c.JSON(http.StatusOK, test)
}

func (h *ABTestHandler) EndTest(c *gin.Context) {
	testIDStr := c.Param("id")

	testID, err := primitive.ObjectIDFromHex(testIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid test_id"})
		return
	}

	err = h.abTestRepo.EndTest(c.Request.Context(), testID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to end test"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "test ended successfully"})
}

func (h *ABTestHandler) GetTestMetrics(c *gin.Context) {
	group := c.Query("group")
	if group == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "group parameter required"})
		return
	}

	metrics, err := h.routingRepo.GetABTestMetrics(c.Request.Context(), group, c.MustGet("test_start_time").(primitive.DateTime).Time())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get metrics"})
		return
	}

	c.JSON(http.StatusOK, metrics)
}

type TrainingHandler struct {
	trainer *repository.RoutingRepository
}

func NewTrainingHandler(trainer *repository.RoutingRepository) *TrainingHandler {
	return &TrainingHandler{
		trainer: trainer,
	}
}
