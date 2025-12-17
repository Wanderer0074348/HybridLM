package ml

import (
	"context"
	"fmt"
	"time"

	"www.github.com/Wanderer0074348/HybridLM/src/features"
	"www.github.com/Wanderer0074348/HybridLM/src/models"
	"www.github.com/Wanderer0074348/HybridLM/src/repository"
)

type ModelTrainer struct {
	routingRepo       *repository.RoutingRepository
	modelRepo         *repository.MLModelRepository
	featureExtractor  *features.FeatureExtractor
	minSamples        int
	learningRate      float64
	epochs            int
	testRatio         float64
}

func NewModelTrainer(
	routingRepo *repository.RoutingRepository,
	modelRepo *repository.MLModelRepository,
	featureExtractor *features.FeatureExtractor,
) *ModelTrainer {
	return &ModelTrainer{
		routingRepo:      routingRepo,
		modelRepo:        modelRepo,
		featureExtractor: featureExtractor,
		minSamples:       100,
		learningRate:     0.01,
		epochs:           1000,
		testRatio:        0.2,
	}
}

func (t *ModelTrainer) Train(ctx context.Context) (*models.MLModel, error) {
	decisions, err := t.routingRepo.GetTrainingData(ctx, t.minSamples)
	if err != nil {
		return nil, fmt.Errorf("failed to get training data: %w", err)
	}

	if len(decisions) < t.minSamples {
		return nil, fmt.Errorf("insufficient training data: got %d, need %d", len(decisions), t.minSamples)
	}

	X, y, err := t.prepareTrainingData(decisions)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare training data: %w", err)
	}

	scaler := NewFeatureScaler()
	scaler.Fit(X)
	XScaled := scaler.Transform(X)

	XTrain, yTrain, XTest, yTest := SplitTrainTest(XScaled, y, t.testRatio)

	classifier := NewLogisticRegressionClassifier()
	classifier.FeatureNames = []string{
		"token_count",
		"char_count",
		"word_count",
		"unique_word_ratio",
		"keyword_score",
		"punctuation_density",
		"complexity_score",
		"sentence_count",
		"avg_sentence_length",
		"has_context",
		"has_code_block",
		"question_type",
		"slm_confidence",
		"slm_is_complex",
	}

	err = classifier.Train(XTrain, yTrain, t.learningRate, t.epochs)
	if err != nil {
		return nil, fmt.Errorf("training failed: %w", err)
	}

	_, err = classifier.Evaluate(XTrain, yTrain)
	if err != nil {
		return nil, fmt.Errorf("training evaluation failed: %w", err)
	}

	testMetrics, err := classifier.Evaluate(XTest, yTest)
	if err != nil {
		return nil, fmt.Errorf("test evaluation failed: %w", err)
	}

	testMetrics.TrainingSamples = len(yTrain)
	testMetrics.ValidationSamples = len(yTest)

	featureImportance := classifier.CalculateFeatureImportance()

	version := fmt.Sprintf("v%d", time.Now().Unix())

	mlModel := &models.MLModel{
		Version:           version,
		ModelType:         "logistic_regression",
		Weights:           classifier.Weights,
		Intercept:         classifier.Intercept,
		FeatureNames:      classifier.FeatureNames,
		TrainingMetrics:   *testMetrics,
		FeatureImportance: featureImportance,
		IsActive:          false,
	}

	err = t.modelRepo.SaveModel(ctx, mlModel)
	if err != nil {
		return nil, fmt.Errorf("failed to save model: %w", err)
	}

	return mlModel, nil
}

func (t *ModelTrainer) prepareTrainingData(decisions []models.RoutingDecisionRecord) ([][]float64, []bool, error) {
	X := make([][]float64, 0, len(decisions))
	y := make([]bool, 0, len(decisions))

	for _, decision := range decisions {
		if decision.Feedback == nil || decision.Feedback.Rating == nil {
			continue
		}

		rating := *decision.Feedback.Rating

		usedLLM := decision.Routing.Decision == "llm"

		shouldUseLLM := false
		if rating <= 2 && !usedLLM {
			shouldUseLLM = true
		} else if rating >= 4 && usedLLM {
			shouldUseLLM = true
		} else if rating >= 3 && !usedLLM {
			shouldUseLLM = false
		} else {
			shouldUseLLM = usedLLM
		}

		featureVector := t.featureExtractor.ToVector(decision.Features)
		X = append(X, featureVector)
		y = append(y, shouldUseLLM)
	}

	if len(X) == 0 {
		return nil, nil, fmt.Errorf("no valid training samples")
	}

	return X, y, nil
}

func (t *ModelTrainer) SetHyperparameters(learningRate float64, epochs int, testRatio float64) {
	t.learningRate = learningRate
	t.epochs = epochs
	t.testRatio = testRatio
}

func (t *ModelTrainer) SetMinSamples(minSamples int) {
	t.minSamples = minSamples
}
