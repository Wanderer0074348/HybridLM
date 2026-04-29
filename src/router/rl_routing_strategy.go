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

	log.Printf("[RL] useSLMAssessment=%v slmAssessor=%v useMLClassifier=%v", s.useSLMAssessment, s.slmAssessor != nil, s.useMLClassifier)

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
			log.Printf("[RL] SLM assessment: %s (confidence=%.2f)", features.SLMAssessment, features.SLMConfidence)
		} else {
			log.Printf("[RL] SLM assessment failed: %v", err)
		}
	} else {
		log.Printf("[RL] SLM assessment skipped (useSLMAssessment=%v, assessor=%v)", s.useSLMAssessment, s.slmAssessor != nil)
	}

	abTestGroup := "rl_control"
	if s.abTestRepo != nil {
		activeTest, err := s.abTestRepo.GetActiveTest(ctx)
		if err == nil && activeTest != nil {
			abTestGroup = s.assignABTestGroup(query, activeTest.TrafficSplit)
		}
	}

	decision := s.makeDecision(features, abTestGroup)

	log.Printf("[RL] decision: useLLM=%v reason=%q confidence=%.2f", decision.UseLLM, decision.Reason, decision.Confidence)

	return decision, &features, abTestGroup
}

func (s *RLRoutingStrategy) makeDecision(features models.QueryFeatures, abTestGroup string) *models.RoutingDecision {
	decision := &models.RoutingDecision{
		ComplexityScore: features.ComplexityScore,
	}

	// RL exploration: randomly try alternate routes so the online learner
	// can observe outcomes and improve the model over time.
	shouldExplore := s.rand.Float64() < s.explorationRate
	if shouldExplore {
		decision.UseLLM = s.rand.Float64() < 0.5
		decision.Reason = fmt.Sprintf("RL exploration (ε=%.2f)", s.explorationRate)
		decision.Confidence = 0.3
		log.Printf("[RL] exploration triggered → useLLM=%v", decision.UseLLM)
		return decision
	}

	// SLM assessor is the primary semantic signal — it actually reads the query.
	// High confidence SLM decisions are trusted directly.
	if features.SLMAssessment != "" && features.SLMConfidence >= 0.75 {
		decision.UseLLM = features.SLMAssessment == "complex"
		decision.Confidence = features.SLMConfidence
		decision.Reason = fmt.Sprintf("SLM assessor: %s (confidence: %.2f)", features.SLMAssessment, features.SLMConfidence)
		return decision
	}

	// Borderline SLM confidence (0.45–0.75): defer to ML classifier if available.
	if s.useMLClassifier && features.SLMAssessment != "" {
		featureVector := s.featureExtractor.ToVector(features)
		useLLM, confidence := s.classifier.Predict(featureVector)
		decision.UseLLM = useLLM
		decision.Confidence = confidence
		if useLLM {
			decision.Reason = fmt.Sprintf("ML classifier (borderline SLM): LLM (%.2f)", confidence)
		} else {
			decision.Reason = fmt.Sprintf("ML classifier (borderline SLM): SLM (%.2f)", confidence)
		}
		return decision
	}

	// No SLM assessor ran — fall back to ML classifier alone.
	if s.useMLClassifier {
		featureVector := s.featureExtractor.ToVector(features)
		useLLM, confidence := s.classifier.Predict(featureVector)
		decision.UseLLM = useLLM
		decision.Confidence = confidence
		if useLLM {
			decision.Reason = fmt.Sprintf("ML classifier: LLM (%.2f)", confidence)
		} else {
			decision.Reason = fmt.Sprintf("ML classifier: SLM (%.2f)", confidence)
		}
		return decision
	}

	// Last resort: complexity threshold.
	decision.UseLLM = features.ComplexityScore > s.config.ComplexityThreshold
	decision.Confidence = 0.5
	decision.Reason = "Complexity threshold fallback"
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
