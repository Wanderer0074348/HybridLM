package router

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"time"

	"www.github.com/Wanderer0074348/HybridLM/src/config"
	"www.github.com/Wanderer0074348/HybridLM/src/features"
	"www.github.com/Wanderer0074348/HybridLM/src/ml"
	"www.github.com/Wanderer0074348/HybridLM/src/models"
	"www.github.com/Wanderer0074348/HybridLM/src/repository"
)

type RLRoutingStrategy struct {
	config            *config.RouterConfig
	classifier        *ml.LogisticRegressionClassifier
	featureExtractor  *features.FeatureExtractor
	slmAssessor       *features.SLMComplexityAssessor
	abTestRepo        *repository.ABTestRepository
	useMLClassifier   bool
	useSLMAssessment  bool
	explorationRate   float64
	rand              *rand.Rand
}

func NewRLRoutingStrategy(
	cfg *config.RouterConfig,
	featureExtractor *features.FeatureExtractor,
	slmAssessor *features.SLMComplexityAssessor,
	abTestRepo *repository.ABTestRepository,
	explorationRate float64,
) *RLRoutingStrategy {
	return &RLRoutingStrategy{
		config:           cfg,
		classifier:       ml.NewLogisticRegressionClassifier(),
		featureExtractor: featureExtractor,
		slmAssessor:      slmAssessor,
		abTestRepo:       abTestRepo,
		useMLClassifier:  false,
		useSLMAssessment: true,
		explorationRate:  explorationRate,
		rand:             rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

func (s *RLRoutingStrategy) LoadModel(model *models.MLModel) {
	s.classifier.LoadFromModel(model)
	s.useMLClassifier = true
	log.Printf("RL routing strategy loaded model: %s", model.Version)
}

func (s *RLRoutingStrategy) DecideWithFeatures(ctx context.Context, query string, contextStr string) (*models.RoutingDecision, *models.QueryFeatures, string) {
	features := s.featureExtractor.Extract(query, contextStr)

	if s.useSLMAssessment && s.slmAssessor != nil {
		assessmentCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
		defer cancel()

		assessment, err := s.slmAssessor.AssessComplexity(assessmentCtx, query)
		if err == nil {
			features.SLMAssessment = "simple"
			if assessment.IsComplex {
				features.SLMAssessment = "complex"
			}
			features.SLMConfidence = assessment.Confidence
		} else {
			log.Printf("SLM assessment failed: %v", err)
		}
	}

	abTestGroup := "rl_control"
	if s.abTestRepo != nil {
		activeTest, err := s.abTestRepo.GetActiveTest(ctx)
		if err == nil && activeTest != nil {
			abTestGroup = s.assignABTestGroup(query, activeTest.TrafficSplit)
		}
	}

	decision := s.makeDecision(features, abTestGroup)

	return decision, &features, abTestGroup
}

func (s *RLRoutingStrategy) makeDecision(features models.QueryFeatures, abTestGroup string) *models.RoutingDecision {
	decision := &models.RoutingDecision{
		ComplexityScore: features.ComplexityScore,
	}

	if !s.useMLClassifier {
		decision.UseLLM = true
		decision.Reason = "No trained model available, routing to LLM for safety"
		decision.Confidence = 0.5
		return decision
	}

	shouldExplore := s.rand.Float64() < s.explorationRate
	if shouldExplore {
		decision.UseLLM = s.rand.Float64() < 0.5
		decision.Reason = fmt.Sprintf("Exploration: random routing (ε=%.2f)", s.explorationRate)
		decision.Confidence = 0.3
		return decision
	}

	featureVector := s.featureExtractor.ToVector(features)
	useLLM, confidence := s.classifier.Predict(featureVector)

	decision.UseLLM = useLLM
	decision.Confidence = confidence

	if useLLM {
		decision.Reason = fmt.Sprintf("ML model predicts LLM needed (confidence: %.2f)", confidence)
	} else {
		decision.Reason = fmt.Sprintf("ML model predicts SLM sufficient (confidence: %.2f)", confidence)
	}

	if features.SLMAssessment == "complex" && features.SLMConfidence > 0.8 {
		decision.UseLLM = true
		decision.Reason = fmt.Sprintf("SLM assessor strongly indicates complexity (%.2f)", features.SLMConfidence)
		decision.Confidence = features.SLMConfidence
	}

	return decision
}

func (s *RLRoutingStrategy) assignABTestGroup(query string, trafficSplit float64) string {
	hashValue := s.rand.Float64()
	if hashValue < trafficSplit {
		return "rl_treatment"
	}
	return "rl_control"
}

func (s *RLRoutingStrategy) SetExplorationRate(rate float64) {
	s.explorationRate = rate
	log.Printf("RL exploration rate updated: %.2f", rate)
}

func (s *RLRoutingStrategy) SetSLMAssessment(enabled bool) {
	s.useSLMAssessment = enabled
}
