package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type QueryFeatures struct {
	TokenCount         int     `bson:"token_count" json:"token_count"`
	CharCount          int     `bson:"char_count" json:"char_count"`
	WordCount          int     `bson:"word_count" json:"word_count"`
	UniqueWordRatio    float64 `bson:"unique_word_ratio" json:"unique_word_ratio"`
	KeywordScore       float64 `bson:"keyword_score" json:"keyword_score"`
	PunctuationDensity float64 `bson:"punctuation_density" json:"punctuation_density"`
	ComplexityScore    float64 `bson:"complexity_score" json:"complexity_score"`
	QuestionType       string  `bson:"question_type" json:"question_type"`
	HasContext         bool    `bson:"has_context" json:"has_context"`
	SentenceCount      int     `bson:"sentence_count" json:"sentence_count"`
	AvgSentenceLength  float64 `bson:"avg_sentence_length" json:"avg_sentence_length"`
	HasCodeBlock       bool    `bson:"has_code_block" json:"has_code_block"`
	SLMAssessment      string  `bson:"slm_assessment" json:"slm_assessment"`
	SLMConfidence      float64 `bson:"slm_confidence" json:"slm_confidence"`
}

type RoutingMetadata struct {
	Decision   string  `bson:"decision" json:"decision"`
	Reason     string  `bson:"reason" json:"reason"`
	Confidence float64 `bson:"confidence" json:"confidence"`
	Strategy   string  `bson:"strategy" json:"strategy"`
	ABTestGroup string `bson:"ab_test_group" json:"ab_test_group"`
}

type PerformanceMetrics struct {
	LatencyMs int     `bson:"latency_ms" json:"latency_ms"`
	CostUSD   float64 `bson:"cost_usd" json:"cost_usd"`
	CacheHit  bool    `bson:"cache_hit" json:"cache_hit"`
	ModelUsed string  `bson:"model_used" json:"model_used"`
}

type UserFeedback struct {
	Rating      *int       `bson:"rating,omitempty" json:"rating,omitempty"`
	Thumbs      string     `bson:"thumbs,omitempty" json:"thumbs,omitempty"`
	CollectedAt *time.Time `bson:"collected_at,omitempty" json:"collected_at,omitempty"`
	Comment     string     `bson:"comment,omitempty" json:"comment,omitempty"`
}

type RoutingDecisionRecord struct {
	ID          primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Query       string             `bson:"query" json:"query"`
	QueryHash   string             `bson:"query_hash" json:"query_hash"`
	Features    QueryFeatures      `bson:"features" json:"features"`
	Routing     RoutingMetadata    `bson:"routing" json:"routing"`
	Performance PerformanceMetrics `bson:"performance" json:"performance"`
	Feedback    *UserFeedback      `bson:"feedback,omitempty" json:"feedback,omitempty"`
	UserID      string             `bson:"user_id,omitempty" json:"user_id,omitempty"`
	SessionID   string             `bson:"session_id,omitempty" json:"session_id,omitempty"`
	Timestamp   time.Time          `bson:"timestamp" json:"timestamp"`
}

type MLModelMetrics struct {
	Accuracy        float64 `bson:"accuracy" json:"accuracy"`
	Precision       float64 `bson:"precision" json:"precision"`
	Recall          float64 `bson:"recall" json:"recall"`
	F1Score         float64 `bson:"f1_score" json:"f1_score"`
	TrainingSamples int     `bson:"training_samples" json:"training_samples"`
	ValidationSamples int   `bson:"validation_samples" json:"validation_samples"`
}

type FeatureImportance struct {
	Name       string  `bson:"name" json:"name"`
	Importance float64 `bson:"importance" json:"importance"`
}

type MLModel struct {
	ID              primitive.ObjectID  `bson:"_id,omitempty" json:"id"`
	Version         string              `bson:"version" json:"version"`
	ModelType       string              `bson:"model_type" json:"model_type"`
	Weights         []float64           `bson:"weights" json:"weights"`
	Intercept       float64             `bson:"intercept" json:"intercept"`
	FeatureNames    []string            `bson:"feature_names" json:"feature_names"`
	TrainingMetrics MLModelMetrics      `bson:"training_metrics" json:"training_metrics"`
	FeatureImportance []FeatureImportance `bson:"feature_importance" json:"feature_importance"`
	TrainedAt       time.Time           `bson:"trained_at" json:"trained_at"`
	IsActive        bool                `bson:"is_active" json:"is_active"`
}

type ABTestConfig struct {
	ID              primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Name            string             `bson:"name" json:"name"`
	Description     string             `bson:"description" json:"description"`
	ControlGroup    string             `bson:"control_group" json:"control_group"`
	TreatmentGroup  string             `bson:"treatment_group" json:"treatment_group"`
	TrafficSplit    float64            `bson:"traffic_split" json:"traffic_split"`
	IsActive        bool               `bson:"is_active" json:"is_active"`
	StartedAt       time.Time          `bson:"started_at" json:"started_at"`
	EndedAt         *time.Time         `bson:"ended_at,omitempty" json:"ended_at,omitempty"`
}

type ABTestMetrics struct {
	ID              primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	TestID          primitive.ObjectID `bson:"test_id" json:"test_id"`
	Group           string             `bson:"group" json:"group"`
	TotalRequests   int                `bson:"total_requests" json:"total_requests"`
	AvgLatencyMs    float64            `bson:"avg_latency_ms" json:"avg_latency_ms"`
	AvgCost         float64            `bson:"avg_cost" json:"avg_cost"`
	AvgRating       float64            `bson:"avg_rating" json:"avg_rating"`
	ThumbsUpCount   int                `bson:"thumbs_up_count" json:"thumbs_up_count"`
	ThumbsDownCount int                `bson:"thumbs_down_count" json:"thumbs_down_count"`
	UpdatedAt       time.Time          `bson:"updated_at" json:"updated_at"`
}

type FeedbackRequest struct {
	DecisionID string `json:"decision_id" binding:"required"`
	Rating     *int   `json:"rating,omitempty"`
	Thumbs     string `json:"thumbs,omitempty"`
	Comment    string `json:"comment,omitempty"`
}

type SLMComplexityAssessment struct {
	IsComplex  bool    `json:"is_complex"`
	Confidence float64 `json:"confidence"`
	Reasoning  string  `json:"reasoning"`
}
