package middleware

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"time"

	"www.github.com/Wanderer0074348/HybridLM/src/models"
	"www.github.com/Wanderer0074348/HybridLM/src/repository"
)

type RoutingLogMiddleware struct {
	routingRepo *repository.RoutingRepository
}

func NewRoutingLogMiddleware(routingRepo *repository.RoutingRepository) *RoutingLogMiddleware {
	return &RoutingLogMiddleware{
		routingRepo: routingRepo,
	}
}

func (m *RoutingLogMiddleware) LogDecision(
	ctx context.Context,
	query string,
	queryContext string,
	features *models.QueryFeatures,
	decision *models.RoutingDecision,
	abTestGroup string,
	performance *models.PerformanceMetrics,
	userID string,
	sessionID string,
) (string, error) {
	if m.routingRepo == nil || features == nil {
		return "", nil
	}

	queryHash := m.generateHash(query + queryContext)

	decisionStr := "slm"
	if decision.UseLLM {
		decisionStr = "llm"
	}

	record := &models.RoutingDecisionRecord{
		Query:     query,
		QueryHash: queryHash,
		Features:  *features,
		Routing: models.RoutingMetadata{
			Decision:    decisionStr,
			Reason:      decision.Reason,
			Confidence:  decision.Confidence,
			Strategy:    "ml",
			ABTestGroup: abTestGroup,
		},
		Performance: *performance,
		UserID:      userID,
		SessionID:   sessionID,
		Timestamp:   time.Now(),
	}

	err := m.routingRepo.SaveDecision(ctx, record)
	if err != nil {
		return "", err
	}

	return record.ID.Hex(), nil
}

func (m *RoutingLogMiddleware) generateHash(data string) string {
	hash := md5.Sum([]byte(data))
	return hex.EncodeToString(hash[:])
}
