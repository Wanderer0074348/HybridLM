package features

import (
	"regexp"
	"strings"
	"unicode"

	"www.github.com/Wanderer0074348/HybridLM/src/models"
)

type FeatureExtractor struct {
	complexityKeywords []string
}

func NewFeatureExtractor() *FeatureExtractor {
	return &FeatureExtractor{
		complexityKeywords: []string{
			"explain", "analyze", "compare", "evaluate", "why",
			"how does", "what if", "reasoning", "detailed",
			"describe", "elaborate", "discuss", "assess",
			"prove", "demonstrate", "justify", "argue",
			"critique", "synthesize", "contrast", "deduce",
			"debug", "optimize", "refactor", "implement",
			"derive", "calculate", "solve", "compute",
		},
	}
}

func (f *FeatureExtractor) Extract(query string, context string) models.QueryFeatures {
	query = strings.TrimSpace(query)

	charCount := len(query)
	words := strings.Fields(query)
	wordCount := len(words)
	tokenCount := f.estimateTokens(query)

	uniqueWords := make(map[string]bool)
	for _, word := range words {
		uniqueWords[strings.ToLower(word)] = true
	}

	uniqueWordRatio := 0.0
	if wordCount > 0 {
		uniqueWordRatio = float64(len(uniqueWords)) / float64(wordCount)
	}

	keywordScore := f.calculateKeywordScore(query)
	punctuationDensity := f.calculatePunctuationDensity(query)
	questionType := f.detectQuestionType(query)
	sentenceCount := f.countSentences(query)

	avgSentenceLength := 0.0
	if sentenceCount > 0 {
		avgSentenceLength = float64(wordCount) / float64(sentenceCount)
	}

	hasCodeBlock := f.detectCodeBlock(query)
	hasContext := len(context) > 0
	hasMath := f.detectMathematical(query)
	hasLogic := f.detectLogical(query)

	lengthFactor := min(float64(tokenCount)/1000.0, 1.0)
	diversityFactor := uniqueWordRatio
	keywordFactor := keywordScore
	punctuationFactor := min(punctuationDensity, 0.3)

	mathBoost := 0.0
	if hasMath {
		mathBoost = 0.2
	}
	logicBoost := 0.0
	if hasLogic {
		logicBoost = 0.15
	}

	complexityScore := (lengthFactor * 0.25) + (diversityFactor * 0.25) + (keywordFactor * 0.25) + (punctuationFactor * 0.1) + mathBoost + logicBoost
	if complexityScore > 1.0 {
		complexityScore = 1.0
	}

	return models.QueryFeatures{
		TokenCount:         tokenCount,
		CharCount:          charCount,
		WordCount:          wordCount,
		UniqueWordRatio:    uniqueWordRatio,
		KeywordScore:       keywordScore,
		PunctuationDensity: punctuationDensity,
		ComplexityScore:    complexityScore,
		QuestionType:       questionType,
		HasContext:         hasContext,
		SentenceCount:      sentenceCount,
		AvgSentenceLength:  avgSentenceLength,
		HasCodeBlock:       hasCodeBlock,
	}
}

func (f *FeatureExtractor) estimateTokens(text string) int {
	return len(text) / 4
}

func (f *FeatureExtractor) calculateKeywordScore(query string) float64 {
	lowerQuery := strings.ToLower(query)
	score := 0.0

	for _, keyword := range f.complexityKeywords {
		if strings.Contains(lowerQuery, keyword) {
			score += 0.15
		}
	}

	return min(score, 1.0)
}

func (f *FeatureExtractor) calculatePunctuationDensity(text string) float64 {
	if len(text) == 0 {
		return 0.0
	}

	punctCount := 0
	for _, ch := range text {
		if unicode.IsPunct(ch) {
			punctCount++
		}
	}

	return float64(punctCount) / float64(len(text))
}

func (f *FeatureExtractor) detectQuestionType(query string) string {
	lowerQuery := strings.ToLower(strings.TrimSpace(query))

	if strings.HasPrefix(lowerQuery, "what") {
		return "what"
	} else if strings.HasPrefix(lowerQuery, "why") {
		return "why"
	} else if strings.HasPrefix(lowerQuery, "how") {
		return "how"
	} else if strings.HasPrefix(lowerQuery, "when") {
		return "when"
	} else if strings.HasPrefix(lowerQuery, "where") {
		return "where"
	} else if strings.HasPrefix(lowerQuery, "who") {
		return "who"
	} else if strings.HasPrefix(lowerQuery, "which") {
		return "which"
	} else if strings.HasSuffix(lowerQuery, "?") {
		return "yes_no"
	}

	return "statement"
}

func (f *FeatureExtractor) countSentences(text string) int {
	re := regexp.MustCompile(`[.!?]+`)
	sentences := re.Split(text, -1)

	count := 0
	for _, s := range sentences {
		if strings.TrimSpace(s) != "" {
			count++
		}
	}

	if count == 0 {
		count = 1
	}

	return count
}

func (f *FeatureExtractor) detectCodeBlock(text string) bool {
	codePatterns := []string{
		"```",
		"def ",
		"function ",
		"class ",
		"import ",
		"const ",
		"let ",
		"var ",
		"#include",
		"public class",
	}

	lowerText := strings.ToLower(text)
	for _, pattern := range codePatterns {
		if strings.Contains(lowerText, pattern) {
			return true
		}
	}

	return false
}

func (f *FeatureExtractor) detectMathematical(text string) bool {
	mathPatterns := []string{
		"+", "-", "×", "÷", "=", "≠", "≤", "≥",
		"∫", "∑", "∏", "√", "^",
		"equation", "formula", "calculate", "solve",
		"derivative", "integral", "theorem", "proof",
		"matrix", "vector", "probability", "statistics",
	}

	lowerText := strings.ToLower(text)

	mathCount := 0
	for _, pattern := range mathPatterns {
		if strings.Contains(lowerText, pattern) {
			mathCount++
		}
	}

	hasNumbers := regexp.MustCompile(`\d+`).MatchString(text)

	return mathCount >= 2 || (mathCount >= 1 && hasNumbers)
}

func (f *FeatureExtractor) detectLogical(text string) bool {
	logicalPatterns := []string{
		"therefore", "thus", "hence", "consequently",
		"because", "since", "as a result",
		"if", "then", "else", "unless",
		"implies", "entails", "follows that",
		"assume", "given that", "suppose",
		"contradict", "paradox", "fallacy",
		"premise", "conclusion", "argument",
	}

	lowerText := strings.ToLower(text)

	logicalCount := 0
	for _, pattern := range logicalPatterns {
		if strings.Contains(lowerText, pattern) {
			logicalCount++
		}
	}

	return logicalCount >= 2
}

func (f *FeatureExtractor) ToVector(features models.QueryFeatures) []float64 {
	vector := []float64{
		float64(features.TokenCount),
		float64(features.CharCount),
		float64(features.WordCount),
		features.UniqueWordRatio,
		features.KeywordScore,
		features.PunctuationDensity,
		features.ComplexityScore,
		float64(features.SentenceCount),
		features.AvgSentenceLength,
	}

	if features.HasContext {
		vector = append(vector, 1.0)
	} else {
		vector = append(vector, 0.0)
	}

	if features.HasCodeBlock {
		vector = append(vector, 1.0)
	} else {
		vector = append(vector, 0.0)
	}

	questionTypeMap := map[string]float64{
		"what":      1.0,
		"why":       2.0,
		"how":       3.0,
		"when":      4.0,
		"where":     5.0,
		"who":       6.0,
		"which":     7.0,
		"yes_no":    8.0,
		"statement": 0.0,
	}

	vector = append(vector, questionTypeMap[features.QuestionType])

	if features.SLMConfidence > 0 {
		vector = append(vector, features.SLMConfidence)
		if features.SLMAssessment == "complex" {
			vector = append(vector, 1.0)
		} else {
			vector = append(vector, 0.0)
		}
	}

	return vector
}

func min(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}
