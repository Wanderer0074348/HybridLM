package repository

import (
	"context"
	"time"

	"www.github.com/Wanderer0074348/HybridLM/src/models"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type RoutingRepository struct {
	collection *mongo.Collection
}

func NewRoutingRepository(db *mongo.Database) *RoutingRepository {
	return &RoutingRepository{
		collection: db.Collection("routing_decisions"),
	}
}

func (r *RoutingRepository) SaveDecision(ctx context.Context, decision *models.RoutingDecisionRecord) error {
	decision.Timestamp = time.Now()
	result, err := r.collection.InsertOne(ctx, decision)
	if err != nil {
		return err
	}
	decision.ID = result.InsertedID.(primitive.ObjectID)
	return nil
}

func (r *RoutingRepository) UpdateFeedback(ctx context.Context, decisionID primitive.ObjectID, feedback *models.UserFeedback) error {
	now := time.Now()
	feedback.CollectedAt = &now

	update := bson.M{
		"$set": bson.M{
			"feedback": feedback,
		},
	}

	_, err := r.collection.UpdateByID(ctx, decisionID, update)
	return err
}

func (r *RoutingRepository) GetDecisionByID(ctx context.Context, id primitive.ObjectID) (*models.RoutingDecisionRecord, error) {
	var decision models.RoutingDecisionRecord
	err := r.collection.FindOne(ctx, bson.M{"_id": id}).Decode(&decision)
	if err != nil {
		return nil, err
	}
	return &decision, nil
}

func (r *RoutingRepository) GetTrainingData(ctx context.Context, minSamples int) ([]models.RoutingDecisionRecord, error) {
	filter := bson.M{
		"feedback": bson.M{"$exists": true},
	}

	opts := options.Find().SetLimit(int64(minSamples * 2))

	cursor, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var decisions []models.RoutingDecisionRecord
	if err = cursor.All(ctx, &decisions); err != nil {
		return nil, err
	}

	return decisions, nil
}

func (r *RoutingRepository) GetABTestMetrics(ctx context.Context, group string, since time.Time) (*models.ABTestMetrics, error) {
	pipeline := mongo.Pipeline{
		{{Key: "$match", Value: bson.M{
			"routing.ab_test_group": group,
			"timestamp":             bson.M{"$gte": since},
		}}},
		{{Key: "$group", Value: bson.M{
			"_id":               "$routing.ab_test_group",
			"total_requests":    bson.M{"$sum": 1},
			"avg_latency_ms":    bson.M{"$avg": "$performance.latency_ms"},
			"avg_cost":          bson.M{"$avg": "$performance.cost_usd"},
			"avg_rating":        bson.M{"$avg": "$feedback.rating"},
			"thumbs_up_count":   bson.M{"$sum": bson.M{"$cond": bson.A{bson.M{"$eq": bson.A{"$feedback.thumbs", "up"}}, 1, 0}}},
			"thumbs_down_count": bson.M{"$sum": bson.M{"$cond": bson.A{bson.M{"$eq": bson.A{"$feedback.thumbs", "down"}}, 1, 0}}},
		}}},
	}

	cursor, err := r.collection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var results []struct {
		TotalRequests   int     `bson:"total_requests"`
		AvgLatencyMs    float64 `bson:"avg_latency_ms"`
		AvgCost         float64 `bson:"avg_cost"`
		AvgRating       float64 `bson:"avg_rating"`
		ThumbsUpCount   int     `bson:"thumbs_up_count"`
		ThumbsDownCount int     `bson:"thumbs_down_count"`
	}

	if err = cursor.All(ctx, &results); err != nil {
		return nil, err
	}

	if len(results) == 0 {
		return &models.ABTestMetrics{
			Group:     group,
			UpdatedAt: time.Now(),
		}, nil
	}

	return &models.ABTestMetrics{
		Group:           group,
		TotalRequests:   results[0].TotalRequests,
		AvgLatencyMs:    results[0].AvgLatencyMs,
		AvgCost:         results[0].AvgCost,
		AvgRating:       results[0].AvgRating,
		ThumbsUpCount:   results[0].ThumbsUpCount,
		ThumbsDownCount: results[0].ThumbsDownCount,
		UpdatedAt:       time.Now(),
	}, nil
}

type MLModelRepository struct {
	collection *mongo.Collection
}

func NewMLModelRepository(db *mongo.Database) *MLModelRepository {
	return &MLModelRepository{
		collection: db.Collection("ml_models"),
	}
}

func (r *MLModelRepository) SaveModel(ctx context.Context, model *models.MLModel) error {
	filter := bson.M{"is_active": true}
	update := bson.M{"$set": bson.M{"is_active": false}}
	_, err := r.collection.UpdateMany(ctx, filter, update)
	if err != nil {
		return err
	}

	model.TrainedAt = time.Now()
	model.IsActive = true

	result, err := r.collection.InsertOne(ctx, model)
	if err != nil {
		return err
	}
	model.ID = result.InsertedID.(primitive.ObjectID)
	return nil
}

func (r *MLModelRepository) GetActiveModel(ctx context.Context) (*models.MLModel, error) {
	var model models.MLModel
	filter := bson.M{"is_active": true}
	opts := options.FindOne().SetSort(bson.M{"trained_at": -1})

	err := r.collection.FindOne(ctx, filter, opts).Decode(&model)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		return nil, err
	}
	return &model, nil
}

func (r *MLModelRepository) GetModelByVersion(ctx context.Context, version string) (*models.MLModel, error) {
	var model models.MLModel
	filter := bson.M{"version": version}

	err := r.collection.FindOne(ctx, filter).Decode(&model)
	if err != nil {
		return nil, err
	}
	return &model, nil
}

type ABTestRepository struct {
	collection *mongo.Collection
}

func NewABTestRepository(db *mongo.Database) *ABTestRepository {
	return &ABTestRepository{
		collection: db.Collection("ab_tests"),
	}
}

func (r *ABTestRepository) GetActiveTest(ctx context.Context) (*models.ABTestConfig, error) {
	var test models.ABTestConfig
	filter := bson.M{"is_active": true}

	err := r.collection.FindOne(ctx, filter).Decode(&test)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		return nil, err
	}
	return &test, nil
}

func (r *ABTestRepository) CreateTest(ctx context.Context, test *models.ABTestConfig) error {
	test.StartedAt = time.Now()
	test.IsActive = true

	result, err := r.collection.InsertOne(ctx, test)
	if err != nil {
		return err
	}
	test.ID = result.InsertedID.(primitive.ObjectID)
	return nil
}

func (r *ABTestRepository) EndTest(ctx context.Context, testID primitive.ObjectID) error {
	now := time.Now()
	update := bson.M{
		"$set": bson.M{
			"is_active": false,
			"ended_at":  now,
		},
	}

	_, err := r.collection.UpdateByID(ctx, testID, update)
	return err
}
