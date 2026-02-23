package scoring

import (
	"context"
	"fmt"
	"math"

	"github.com/google/uuid"

	"github.com/onnwee/pulse-score/internal/repository"
)

// RiskDistribution holds the count of customers per risk level.
type RiskDistribution struct {
	Green  int `json:"green"`
	Yellow int `json:"yellow"`
	Red    int `json:"red"`
	Total  int `json:"total"`
}

// ScoreHistogramBucket holds a score range and its count.
type ScoreHistogramBucket struct {
	Min   int `json:"min"`
	Max   int `json:"max"`
	Count int `json:"count"`
}

// RiskCategorizer provides dashboard analytics on health scores.
type RiskCategorizer struct {
	healthScores *repository.HealthScoreRepository
}

// NewRiskCategorizer creates a new RiskCategorizer.
func NewRiskCategorizer(healthScores *repository.HealthScoreRepository) *RiskCategorizer {
	return &RiskCategorizer{healthScores: healthScores}
}

// GetRiskDistribution returns the count of customers per risk level for an org.
func (c *RiskCategorizer) GetRiskDistribution(ctx context.Context, orgID uuid.UUID) (*RiskDistribution, error) {
	counts, err := c.healthScores.CountByRiskLevel(ctx, orgID)
	if err != nil {
		return nil, fmt.Errorf("get risk distribution: %w", err)
	}

	dist := &RiskDistribution{}
	for level, count := range counts {
		switch level {
		case "green":
			dist.Green = count
		case "yellow":
			dist.Yellow = count
		case "red":
			dist.Red = count
		}
		dist.Total += count
	}
	return dist, nil
}

// GetScoreHistogram returns a histogram of scores in 10-point buckets.
func (c *RiskCategorizer) GetScoreHistogram(ctx context.Context, orgID uuid.UUID) ([]ScoreHistogramBucket, error) {
	scores, err := c.healthScores.ScoreDistribution(ctx, orgID)
	if err != nil {
		return nil, fmt.Errorf("get score distribution: %w", err)
	}

	// Create 10 buckets: 0-9, 10-19, ..., 90-100
	buckets := make([]ScoreHistogramBucket, 10)
	for i := range buckets {
		buckets[i] = ScoreHistogramBucket{
			Min: i * 10,
			Max: i*10 + 9,
		}
	}
	// Last bucket includes 100
	buckets[9].Max = 100

	for _, score := range scores {
		idx := int(math.Min(float64(score/10), 9))
		buckets[idx].Count++
	}
	return buckets, nil
}
