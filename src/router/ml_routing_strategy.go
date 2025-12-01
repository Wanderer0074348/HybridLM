package router

import (
	"context"
	"crypto/md5"
	"fmt"
	"math/rand"

	"www.github.com/Wanderer0074348/HybridLM/src/config"
	"www.github.com/Wanderer0074348/HybridLM/src/features"
	"www.github.com/Wanderer0074348/HybridLM/src/ml"
	"www.github.com/Wanderer0074348/HybridLM/src/models"
	"www.github.com/Wanderer0074348/HybridLM/src/repository"
)

type MLRoutingStrategy struct {
	config            *config.RouterConfig
	classifier        *ml.LogisticRegressionClassifier
	featureExtractor  *features.FeatureExtractor
	slmAssessor       *features.SLMComplexityAssessor
	abTestRepo        *repository.ABTestRepository
	useSLMAssessment  bool
	useMLClassifier   bool
}

func NewMLRoutingStrategy(
	cfg *config.RouterConfig,
	featureExtractor *features.FeatureExtractor,
	slmAssessor *features.SLMComplexityAssessor,
	abTestRepo *repository.ABTestRepository,
) *MLRoutingStrategy {
	return &MLRoutingStrategy{
		config:           cfg,
		classifier:       ml.NewLogisticRegressionClassifier(),
		featureExtractor: featureExtractor,
		slmAssessor:      slmAssessor,
		abTestRepo:       abTestRepo,
		useSLMAssessment: true,
		useMLClassifier:  false,
	}
}

func (s *MLRoutingStrategy) LoadModel(model *models.MLModel) {
	s.classifier.LoadFromModel(model)
	s.useMLClassifier = true
}

func (s *MLRoutingStrategy) DecideWithFeatures(ctx context.Context, query string, context string) (*models.RoutingDecision, *models.QueryFeatures, string) {
	features := s.featureExtractor.Extract(query, context)

	if s.useSLMAssessment {
		assessment, err := s.slmAssessor.AssessComplexity(ctx, query)
		if err == nil {
			features.SLMAssessment = "simple"
			if assessment.IsComplex {
				features.SLMAssessment = "complex"
			}
			features.SLMConfidence = assessment.Confidence
		}
	}

	abTestGroup := "control"
	if s.abTestRepo != nil {
		activeTest, err := s.abTestRepo.GetActiveTest(ctx)
		if err == nil && activeTest != nil {
			abTestGroup = s.assignABTestGroup(query, activeTest.TrafficSplit)
			if abTestGroup == activeTest.TreatmentGroup {
				return s.decideTreatment(features, abTestGroup)
			}
		}
	}

	return s.decideControl(features, abTestGroup)
}

func (s *MLRoutingStrategy) decideControl(features models.QueryFeatures, abTestGroup string) (*models.RoutingDecision, *models.QueryFeatures, string) {
	decision := &models.RoutingDecision{
		ComplexityScore: features.ComplexityScore,
	}

	if features.SLMAssessment == "complex" && features.SLMConfidence > 0.7 {
		decision.UseLLM = true
		decision.Reason = "SLM assessment indicates complex query"
		decision.Confidence = features.SLMConfidence
		return decision, &features, abTestGroup
	}

	if features.ComplexityScore > s.config.ComplexityThreshold {
		decision.UseLLM = true
		decision.Reason = "High complexity score from rule-based analysis"
		decision.Confidence = 0.9
		return decision, &features, abTestGroup
	}

	if features.TokenCount > 100 {
		decision.UseLLM = true
		decision.Reason = "Long query requires cloud LLM processing"
		decision.Confidence = 0.85
		return decision, &features, abTestGroup
	}

	if features.HasContext {
		decision.UseLLM = true
		decision.Reason = "Context-aware query routed to LLM"
		decision.Confidence = 0.8
		return decision, &features, abTestGroup
	}

	if s.useMLClassifier && features.ComplexityScore >= 0.45 && features.ComplexityScore <= 0.75 {
		featureVector := s.featureExtractor.ToVector(features)
		useLLM, confidence := s.classifier.Predict(featureVector)

		decision.UseLLM = useLLM
		if useLLM {
			decision.Reason = "ML classifier predicts LLM needed (borderline complexity)"
		} else {
			decision.Reason = "ML classifier predicts SLM sufficient (borderline complexity)"
		}
		decision.Confidence = confidence
		return decision, &features, abTestGroup
	}

	decision.UseLLM = false
	decision.Reason = "Simple query suitable for edge SLM"
	decision.Confidence = 0.95
	return decision, &features, abTestGroup
}

func (s *MLRoutingStrategy) decideTreatment(features models.QueryFeatures, abTestGroup string) (*models.RoutingDecision, *models.QueryFeatures, string) {
	decision := &models.RoutingDecision{
		ComplexityScore: features.ComplexityScore,
	}

	if !s.useMLClassifier {
		return s.decideControl(features, abTestGroup)
	}

	if features.TokenCount > 150 {
		decision.UseLLM = true
		decision.Reason = "Very long query requires LLM"
		decision.Confidence = 0.95
		return decision, &features, abTestGroup
	}

	featureVector := s.featureExtractor.ToVector(features)
	useLLM, confidence := s.classifier.Predict(featureVector)

	decision.UseLLM = useLLM
	if useLLM {
		decision.Reason = "ML classifier predicts LLM needed (treatment group)"
	} else {
		decision.Reason = "ML classifier predicts SLM sufficient (treatment group)"
	}
	decision.Confidence = confidence

	return decision, &features, abTestGroup
}

func (s *MLRoutingStrategy) assignABTestGroup(query string, trafficSplit float64) string {
	hash := md5.Sum([]byte(query))
	hashValue := float64(hash[0]) / 255.0

	if hashValue < trafficSplit {
		return "treatment"
	}
	return "control"
}

func (s *MLRoutingStrategy) SetSLMAssessment(enabled bool) {
	s.useSLMAssessment = enabled
}

type AdaptiveRoutingStrategy struct {
	baseStrategy *MLRoutingStrategy
	config       *config.RouterConfig
}

func NewAdaptiveRoutingStrategy(
	cfg *config.RouterConfig,
	mlStrategy *MLRoutingStrategy,
) *AdaptiveRoutingStrategy {
	return &AdaptiveRoutingStrategy{
		baseStrategy: mlStrategy,
		config:       cfg,
	}
}

func (s *AdaptiveRoutingStrategy) Decide(ctx context.Context, query string, context string) (*models.RoutingDecision, *models.QueryFeatures, string) {
	return s.baseStrategy.DecideWithFeatures(ctx, query, context)
}

type EpsilonGreedyStrategy struct {
	mlStrategy *MLRoutingStrategy
	epsilon    float64
	rand       *rand.Rand
}

func NewEpsilonGreedyStrategy(mlStrategy *MLRoutingStrategy, epsilon float64) *EpsilonGreedyStrategy {
	return &EpsilonGreedyStrategy{
		mlStrategy: mlStrategy,
		epsilon:    epsilon,
		rand:       rand.New(rand.NewSource(42)),
	}
}

func (s *EpsilonGreedyStrategy) Decide(ctx context.Context, query string, context string) (*models.RoutingDecision, *models.QueryFeatures, string) {
	if s.rand.Float64() < s.epsilon {
		decision, features, group := s.mlStrategy.DecideWithFeatures(ctx, query, context)
		decision.UseLLM = !decision.UseLLM
		decision.Reason = fmt.Sprintf("Exploration: flipped decision (%s)", decision.Reason)
		decision.Confidence = 0.5
		return decision, features, group
	}

	return s.mlStrategy.DecideWithFeatures(ctx, query, context)
}
