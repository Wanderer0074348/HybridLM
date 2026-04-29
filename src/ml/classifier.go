package ml

import (
	"fmt"
	"math"

	"www.github.com/Wanderer0074348/HybridLM/src/models"

	"gonum.org/v1/gonum/mat"
)

type LogisticRegressionClassifier struct {
	Weights      []float64
	Intercept    float64
	FeatureNames []string
}

func NewLogisticRegressionClassifier() *LogisticRegressionClassifier {
	return &LogisticRegressionClassifier{
		Weights:      make([]float64, 0),
		Intercept:    0.0,
		FeatureNames: make([]string, 0),
	}
}

func (lr *LogisticRegressionClassifier) LoadFromModel(model *models.MLModel) {
	lr.Weights = model.Weights
	lr.Intercept = model.Intercept
	lr.FeatureNames = model.FeatureNames
}

func (lr *LogisticRegressionClassifier) Predict(features []float64) (bool, float64) {
	if len(features) != len(lr.Weights) {
		return false, 0.0
	}

	z := lr.Intercept
	for i, w := range lr.Weights {
		z += w * features[i]
	}

	probability := sigmoid(z)

	return probability >= 0.35, probability
}

func (lr *LogisticRegressionClassifier) Train(X [][]float64, y []bool, learningRate float64, epochs int) error {
	if len(X) == 0 || len(X) != len(y) {
		return fmt.Errorf("invalid training data")
	}

	numFeatures := len(X[0])
	lr.Weights = make([]float64, numFeatures)
	lr.Intercept = 0.0

	for epoch := 0; epoch < epochs; epoch++ {
		for i := 0; i < len(X); i++ {
			z := lr.Intercept
			for j := 0; j < numFeatures; j++ {
				z += lr.Weights[j] * X[i][j]
			}

			prediction := sigmoid(z)

			var yVal float64
			if y[i] {
				yVal = 1.0
			} else {
				yVal = 0.0
			}

			error := prediction - yVal

			lr.Intercept -= learningRate * error

			for j := 0; j < numFeatures; j++ {
				lr.Weights[j] -= learningRate * error * X[i][j]
			}
		}
	}

	return nil
}

func (lr *LogisticRegressionClassifier) Evaluate(X [][]float64, y []bool) (*models.MLModelMetrics, error) {
	if len(X) == 0 || len(X) != len(y) {
		return nil, fmt.Errorf("invalid evaluation data")
	}

	truePositive := 0
	trueNegative := 0
	falsePositive := 0
	falseNegative := 0

	for i := 0; i < len(X); i++ {
		prediction, _ := lr.Predict(X[i])

		if prediction && y[i] {
			truePositive++
		} else if !prediction && !y[i] {
			trueNegative++
		} else if prediction && !y[i] {
			falsePositive++
		} else {
			falseNegative++
		}
	}

	accuracy := float64(truePositive+trueNegative) / float64(len(y))

	precision := 0.0
	if (truePositive + falsePositive) > 0 {
		precision = float64(truePositive) / float64(truePositive+falsePositive)
	}

	recall := 0.0
	if (truePositive + falseNegative) > 0 {
		recall = float64(truePositive) / float64(truePositive+falseNegative)
	}

	f1Score := 0.0
	if (precision + recall) > 0 {
		f1Score = 2 * (precision * recall) / (precision + recall)
	}

	return &models.MLModelMetrics{
		Accuracy:          accuracy,
		Precision:         precision,
		Recall:            recall,
		F1Score:           f1Score,
		TrainingSamples:   len(y),
		ValidationSamples: 0,
	}, nil
}

func (lr *LogisticRegressionClassifier) CalculateFeatureImportance() []models.FeatureImportance {
	importance := make([]models.FeatureImportance, len(lr.Weights))

	for i, w := range lr.Weights {
		name := fmt.Sprintf("feature_%d", i)
		if i < len(lr.FeatureNames) && lr.FeatureNames[i] != "" {
			name = lr.FeatureNames[i]
		}

		importance[i] = models.FeatureImportance{
			Name:       name,
			Importance: math.Abs(w),
		}
	}

	return importance
}

func sigmoid(z float64) float64 {
	return 1.0 / (1.0 + math.Exp(-z))
}

func normalizeFeatures(X [][]float64) ([][]float64, []float64, []float64) {
	if len(X) == 0 {
		return X, nil, nil
	}

	numFeatures := len(X[0])
	means := make([]float64, numFeatures)
	stds := make([]float64, numFeatures)

	for j := 0; j < numFeatures; j++ {
		sum := 0.0
		for i := 0; i < len(X); i++ {
			sum += X[i][j]
		}
		means[j] = sum / float64(len(X))
	}

	for j := 0; j < numFeatures; j++ {
		variance := 0.0
		for i := 0; i < len(X); i++ {
			diff := X[i][j] - means[j]
			variance += diff * diff
		}
		stds[j] = math.Sqrt(variance / float64(len(X)))
		if stds[j] == 0 {
			stds[j] = 1.0
		}
	}

	normalized := make([][]float64, len(X))
	for i := 0; i < len(X); i++ {
		normalized[i] = make([]float64, numFeatures)
		for j := 0; j < numFeatures; j++ {
			normalized[i][j] = (X[i][j] - means[j]) / stds[j]
		}
	}

	return normalized, means, stds
}

func SplitTrainTest(X [][]float64, y []bool, testRatio float64) ([][]float64, []bool, [][]float64, []bool) {
	n := len(X)
	testSize := int(float64(n) * testRatio)
	trainSize := n - testSize

	XTrain := X[:trainSize]
	yTrain := y[:trainSize]
	XTest := X[trainSize:]
	yTest := y[trainSize:]

	return XTrain, yTrain, XTest, yTest
}

type FeatureScaler struct {
	Means []float64
	Stds  []float64
}

func NewFeatureScaler() *FeatureScaler {
	return &FeatureScaler{}
}

func (s *FeatureScaler) Fit(X [][]float64) {
	if len(X) == 0 {
		return
	}

	numFeatures := len(X[0])
	s.Means = make([]float64, numFeatures)
	s.Stds = make([]float64, numFeatures)

	for j := 0; j < numFeatures; j++ {
		sum := 0.0
		for i := 0; i < len(X); i++ {
			sum += X[i][j]
		}
		s.Means[j] = sum / float64(len(X))
	}

	for j := 0; j < numFeatures; j++ {
		variance := 0.0
		for i := 0; i < len(X); i++ {
			diff := X[i][j] - s.Means[j]
			variance += diff * diff
		}
		s.Stds[j] = math.Sqrt(variance / float64(len(X)))
		if s.Stds[j] == 0 {
			s.Stds[j] = 1.0
		}
	}
}

func (s *FeatureScaler) Transform(X [][]float64) [][]float64 {
	if len(X) == 0 || len(s.Means) == 0 {
		return X
	}

	transformed := make([][]float64, len(X))
	for i := 0; i < len(X); i++ {
		transformed[i] = make([]float64, len(X[i]))
		for j := 0; j < len(X[i]); j++ {
			transformed[i][j] = (X[i][j] - s.Means[j]) / s.Stds[j]
		}
	}

	return transformed
}

func (s *FeatureScaler) TransformSingle(x []float64) []float64 {
	if len(x) == 0 || len(s.Means) == 0 {
		return x
	}

	transformed := make([]float64, len(x))
	for i := 0; i < len(x); i++ {
		transformed[i] = (x[i] - s.Means[i]) / s.Stds[i]
	}

	return transformed
}

type CrossValidator struct{}

func NewCrossValidator() *CrossValidator {
	return &CrossValidator{}
}

func (cv *CrossValidator) KFoldSplit(n, k int) [][][]int {
	foldSize := n / k
	folds := make([][][]int, k)

	indices := make([]int, n)
	for i := 0; i < n; i++ {
		indices[i] = i
	}

	for fold := 0; fold < k; fold++ {
		start := fold * foldSize
		end := start + foldSize
		if fold == k-1 {
			end = n
		}

		testIndices := indices[start:end]
		trainIndices := append([]int{}, indices[:start]...)
		trainIndices = append(trainIndices, indices[end:]...)

		folds[fold] = [][]int{trainIndices, testIndices}
	}

	return folds
}

func MatrixMultiply(a, b [][]float64) [][]float64 {
	aRows := len(a)
	aCols := len(a[0])
	bCols := len(b[0])

	result := make([][]float64, aRows)
	for i := range result {
		result[i] = make([]float64, bCols)
	}

	for i := 0; i < aRows; i++ {
		for j := 0; j < bCols; j++ {
			sum := 0.0
			for k := 0; k < aCols; k++ {
				sum += a[i][k] * b[k][j]
			}
			result[i][j] = sum
		}
	}

	return result
}

func DotProduct(a, b []float64) float64 {
	sum := 0.0
	for i := range a {
		sum += a[i] * b[i]
	}
	return sum
}

func VectorAdd(a, b []float64) []float64 {
	result := make([]float64, len(a))
	for i := range a {
		result[i] = a[i] + b[i]
	}
	return result
}

func VectorSubtract(a, b []float64) []float64 {
	result := make([]float64, len(a))
	for i := range a {
		result[i] = a[i] - b[i]
	}
	return result
}

func ScalarMultiply(scalar float64, vec []float64) []float64 {
	result := make([]float64, len(vec))
	for i := range vec {
		result[i] = scalar * vec[i]
	}
	return result
}

func Mean(values []float64) float64 {
	if len(values) == 0 {
		return 0.0
	}
	sum := 0.0
	for _, v := range values {
		sum += v
	}
	return sum / float64(len(values))
}

func StandardDeviation(values []float64) float64 {
	if len(values) == 0 {
		return 0.0
	}
	mean := Mean(values)
	variance := 0.0
	for _, v := range values {
		diff := v - mean
		variance += diff * diff
	}
	return math.Sqrt(variance / float64(len(values)))
}

func MatToSlice(m *mat.Dense) [][]float64 {
	rows, cols := m.Dims()
	result := make([][]float64, rows)
	for i := 0; i < rows; i++ {
		result[i] = make([]float64, cols)
		for j := 0; j < cols; j++ {
			result[i][j] = m.At(i, j)
		}
	}
	return result
}

func SliceToMat(data [][]float64) *mat.Dense {
	if len(data) == 0 {
		return nil
	}
	rows := len(data)
	cols := len(data[0])
	flat := make([]float64, rows*cols)
	for i := 0; i < rows; i++ {
		for j := 0; j < cols; j++ {
			flat[i*cols+j] = data[i][j]
		}
	}
	return mat.NewDense(rows, cols, flat)
}
