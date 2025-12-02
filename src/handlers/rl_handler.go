package handlers

import (
	"context"
	"log"
	"time"

	"www.github.com/Wanderer0074348/HybridLM/src/models"
	"www.github.com/Wanderer0074348/HybridLM/src/repository"
	"www.github.com/Wanderer0074348/HybridLM/src/rl"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type RLFeedbackCollector struct {
	routingRepo *repository.RoutingRepository
	rewardModel *rl.RewardModel
	enabled     bool
}

func NewRLFeedbackCollector(
	routingRepo *repository.RoutingRepository,
	rewardModel *rl.RewardModel,
) *RLFeedbackCollector {
	return &RLFeedbackCollector{
		routingRepo: routingRepo,
		rewardModel: rewardModel,
		enabled:     true,
	}
}

func (r *RLFeedbackCollector) CollectFeedback(
	ctx context.Context,
	decisionID string,
	query string,
	response string,
	latencyMs int,
	costUSD float64,
	usedLLM bool,
) error {
	if !r.enabled || r.rewardModel == nil {
		return nil
	}

	go r.evaluateAsync(decisionID, query, response, latencyMs, costUSD, usedLLM)

	return nil
}

func (r *RLFeedbackCollector) evaluateAsync(
	decisionID string,
	query string,
	response string,
	latencyMs int,
	costUSD float64,
	usedLLM bool,
) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	rewardScore, err := r.rewardModel.EvaluateResponse(ctx, query, response)
	if err != nil {
		log.Printf("Failed to evaluate response for RL: %v", err)
		return
	}

	finalReward := r.rewardModel.CalculateReward(
		rewardScore.Score,
		latencyMs,
		costUSD,
		usedLLM,
	)

	rating := r.scoreToRating(rewardScore.Score)

	objID, err := primitive.ObjectIDFromHex(decisionID)
	if err != nil {
		log.Printf("Invalid decision ID for RL feedback: %v", err)
		return
	}

	feedback := &models.UserFeedback{
		Rating:  &rating,
		Comment: rewardScore.Reasoning,
	}

	if rewardScore.Score >= 0.7 {
		feedback.Thumbs = "up"
	} else if rewardScore.Score < 0.5 {
		feedback.Thumbs = "down"
	}

	err = r.routingRepo.UpdateFeedback(ctx, objID, feedback)
	if err != nil {
		log.Printf("Failed to save RL feedback: %v", err)
		return
	}

	log.Printf("RL feedback collected: decision=%s, quality=%.2f, reward=%.2f, rating=%d",
		decisionID, rewardScore.Score, finalReward, rating)
}

func (r *RLFeedbackCollector) scoreToRating(score float64) int {
	if score >= 0.9 {
		return 5
	} else if score >= 0.7 {
		return 4
	} else if score >= 0.5 {
		return 3
	} else if score >= 0.3 {
		return 2
	}
	return 1
}

func (r *RLFeedbackCollector) SetEnabled(enabled bool) {
	r.enabled = enabled
	if enabled {
		log.Printf("RL feedback collection enabled")
	} else {
		log.Printf("RL feedback collection disabled")
	}
}
