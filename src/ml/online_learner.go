package ml

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"www.github.com/Wanderer0074348/HybridLM/src/features"
	"www.github.com/Wanderer0074348/HybridLM/src/repository"
)

type OnlineLearner struct {
	trainer          *ModelTrainer
	routingRepo      *repository.RoutingRepository
	modelRepo        *repository.MLModelRepository
	featureExtractor *features.FeatureExtractor

	retrainThreshold int
	checkInterval    time.Duration
	lastTrainingTime time.Time
	lastDecisionCount int

	mu              sync.Mutex
	isTraining      bool
	stopChan        chan bool
}

func NewOnlineLearner(
	trainer *ModelTrainer,
	routingRepo *repository.RoutingRepository,
	modelRepo *repository.MLModelRepository,
	featureExtractor *features.FeatureExtractor,
	retrainThreshold int,
) *OnlineLearner {
	return &OnlineLearner{
		trainer:          trainer,
		routingRepo:      routingRepo,
		modelRepo:        modelRepo,
		featureExtractor: featureExtractor,
		retrainThreshold: retrainThreshold,
		checkInterval:    5 * time.Minute,
		lastTrainingTime: time.Now(),
		stopChan:         make(chan bool),
	}
}

func (ol *OnlineLearner) Start(ctx context.Context) {
	ticker := time.NewTicker(ol.checkInterval)
	defer ticker.Stop()

	log.Printf("Online learner started (checking every %v, threshold: %d samples)",
		ol.checkInterval, ol.retrainThreshold)

	for {
		select {
		case <-ticker.C:
			ol.checkAndRetrain(ctx)
		case <-ol.stopChan:
			log.Printf("Online learner stopped")
			return
		case <-ctx.Done():
			log.Printf("Online learner stopped (context cancelled)")
			return
		}
	}
}

func (ol *OnlineLearner) Stop() {
	close(ol.stopChan)
}

func (ol *OnlineLearner) checkAndRetrain(ctx context.Context) {
	ol.mu.Lock()
	if ol.isTraining {
		ol.mu.Unlock()
		return
	}
	ol.isTraining = true
	ol.mu.Unlock()

	defer func() {
		ol.mu.Lock()
		ol.isTraining = false
		ol.mu.Unlock()
	}()

	decisions, err := ol.routingRepo.GetTrainingData(ctx, ol.retrainThreshold)
	if err != nil {
		log.Printf("Failed to get training data: %v", err)
		return
	}

	newDecisions := len(decisions) - ol.lastDecisionCount

	if newDecisions >= ol.retrainThreshold {
		log.Printf("Triggering automatic retraining: %d new decisions collected", newDecisions)

		model, err := ol.trainer.Train(ctx)
		if err != nil {
			log.Printf("Online training failed: %v", err)
			return
		}

		log.Printf("Online training successful! New model: %s (accuracy: %.2f%%)",
			model.Version, model.TrainingMetrics.Accuracy*100)

		ol.lastDecisionCount = len(decisions)
		ol.lastTrainingTime = time.Now()
	} else {
		log.Printf("Insufficient new data for retraining: %d/%d decisions",
			newDecisions, ol.retrainThreshold)
	}
}

func (ol *OnlineLearner) ForceRetrain(ctx context.Context) error {
	ol.mu.Lock()
	if ol.isTraining {
		ol.mu.Unlock()
		return fmt.Errorf("training already in progress")
	}
	ol.isTraining = true
	ol.mu.Unlock()

	defer func() {
		ol.mu.Lock()
		ol.isTraining = false
		ol.mu.Unlock()
	}()

	log.Printf("Force retraining triggered")

	model, err := ol.trainer.Train(ctx)
	if err != nil {
		return fmt.Errorf("forced training failed: %w", err)
	}

	log.Printf("Forced training successful! New model: %s (accuracy: %.2f%%)",
		model.Version, model.TrainingMetrics.Accuracy*100)

	ol.lastTrainingTime = time.Now()

	return nil
}

func (ol *OnlineLearner) GetStatus() map[string]interface{} {
	ol.mu.Lock()
	defer ol.mu.Unlock()

	return map[string]interface{}{
		"is_training":         ol.isTraining,
		"last_training_time":  ol.lastTrainingTime,
		"last_decision_count": ol.lastDecisionCount,
		"retrain_threshold":   ol.retrainThreshold,
		"check_interval":      ol.checkInterval.String(),
	}
}
