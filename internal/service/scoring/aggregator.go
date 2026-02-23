package scoring

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"time"

	"github.com/google/uuid"

	"github.com/onnwee/pulse-score/internal/repository"
)

// HealthScoreResult holds the result of a full health score calculation.
type HealthScoreResult struct {
	CustomerID   uuid.UUID          `json:"customer_id"`
	OrgID        uuid.UUID          `json:"org_id"`
	OverallScore int                `json:"overall_score"`
	RiskLevel    string             `json:"risk_level"`
	Factors      map[string]float64 `json:"factors"`
	CalculatedAt time.Time          `json:"calculated_at"`
}

// ScoreAggregator computes weighted overall health scores from individual factors.
type ScoreAggregator struct {
	factors    []ScoreFactor
	configRepo *repository.ScoringConfigRepository
}

// NewScoreAggregator creates a new ScoreAggregator.
func NewScoreAggregator(
	factors []ScoreFactor,
	configRepo *repository.ScoringConfigRepository,
) *ScoreAggregator {
	return &ScoreAggregator{
		factors:    factors,
		configRepo: configRepo,
	}
}

// Calculate computes the weighted health score for a customer.
func (a *ScoreAggregator) Calculate(ctx context.Context, customerID, orgID uuid.UUID) (*HealthScoreResult, error) {
	// Load scoring config for org
	config, err := a.configRepo.GetByOrgID(ctx, orgID)
	if err != nil {
		return nil, fmt.Errorf("get scoring config: %w", err)
	}
	if config == nil {
		// Create default config on first access
		config, err = a.configRepo.CreateDefault(ctx, orgID)
		if err != nil {
			return nil, fmt.Errorf("create default scoring config: %w", err)
		}
	}

	// Calculate each factor
	var presentFactors []FactorResult
	var presentWeightSum float64
	factorScores := make(map[string]float64)

	for _, factor := range a.factors {
		result, err := factor.Calculate(ctx, customerID, orgID)
		if err != nil {
			slog.Error("factor calculation error",
				"factor", factor.Name(),
				"customer_id", customerID,
				"error", err,
			)
			continue
		}

		if result.Score != nil {
			presentFactors = append(presentFactors, *result)
			weight := config.Weights[result.Name]
			presentWeightSum += weight
			factorScores[result.Name] = *result.Score
		}
	}

	// Edge case: no factors available
	if len(presentFactors) == 0 {
		return nil, fmt.Errorf("no scoring factors available for customer %s", customerID)
	}

	// Redistribute weights proportionally among present factors
	var weightedSum float64
	for _, f := range presentFactors {
		originalWeight := config.Weights[f.Name]
		adjustedWeight := originalWeight / presentWeightSum
		weightedSum += *f.Score * adjustedWeight
	}

	// Convert to 0-100 integer
	overallScore := int(math.Round(weightedSum * 100))
	if overallScore < 0 {
		overallScore = 0
	}
	if overallScore > 100 {
		overallScore = 100
	}

	// Assign risk level based on thresholds
	riskLevel := assignRiskLevel(overallScore, config.Thresholds)

	return &HealthScoreResult{
		CustomerID:   customerID,
		OrgID:        orgID,
		OverallScore: overallScore,
		RiskLevel:    riskLevel,
		Factors:      factorScores,
		CalculatedAt: time.Now(),
	}, nil
}

// assignRiskLevel determines the risk level based on score and thresholds.
func assignRiskLevel(score int, thresholds map[string]int) string {
	greenThreshold := thresholds["green"]
	yellowThreshold := thresholds["yellow"]

	if greenThreshold == 0 {
		greenThreshold = 70
	}
	if yellowThreshold == 0 {
		yellowThreshold = 40
	}

	switch {
	case score >= greenThreshold:
		return "green"
	case score >= yellowThreshold:
		return "yellow"
	default:
		return "red"
	}
}
