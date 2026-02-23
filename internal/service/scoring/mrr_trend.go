package scoring

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/onnwee/pulse-score/internal/repository"
)

// MRRTrendFactor calculates a score based on MRR trend over 30/60/90 day windows.
type MRRTrendFactor struct {
	customers *repository.CustomerRepository
	events    *repository.CustomerEventRepository
}

// NewMRRTrendFactor creates a new MRRTrendFactor.
func NewMRRTrendFactor(
	customers *repository.CustomerRepository,
	events *repository.CustomerEventRepository,
) *MRRTrendFactor {
	return &MRRTrendFactor{
		customers: customers,
		events:    events,
	}
}

// Name returns the factor name.
func (f *MRRTrendFactor) Name() string {
	return "mrr_trend"
}

// Calculate computes MRR trend score by comparing current MRR to historical values.
func (f *MRRTrendFactor) Calculate(ctx context.Context, customerID, orgID uuid.UUID) (*FactorResult, error) {
	customer, err := f.customers.GetByID(ctx, customerID)
	if err != nil {
		return nil, fmt.Errorf("get customer: %w", err)
	}
	if customer == nil {
		return &FactorResult{Name: f.Name(), Score: nil}, nil
	}

	now := time.Now()
	currentMRR := customer.MRRCents

	// Get MRR change events over the last 90 days
	events, err := f.events.ListByCustomerAndType(ctx, customerID, "mrr.changed", now.AddDate(0, 0, -90))
	if err != nil {
		return nil, fmt.Errorf("get mrr events: %w", err)
	}

	// No historical data: return neutral
	if len(events) == 0 {
		score := 0.5
		return &FactorResult{Name: f.Name(), Score: &score}, nil
	}

	// Calculate trends for each window with time weighting
	// 30d trend gets 50% weight, 60d trend 30%, 90d trend 20%
	trend30d := f.trendForWindow(events, currentMRR, now.AddDate(0, 0, -30))
	trend60d := f.trendForWindow(events, currentMRR, now.AddDate(0, 0, -60))
	trend90d := f.trendForWindow(events, currentMRR, now.AddDate(0, 0, -90))

	weightedTrend := trend30d*0.50 + trend60d*0.30 + trend90d*0.20

	// Convert trend percentage to 0.0-1.0 score
	score := f.trendToScore(weightedTrend)

	return &FactorResult{Name: f.Name(), Score: &score}, nil
}

// trendForWindow calculates the percentage change from the oldest MRR event in a window to current.
func (f *MRRTrendFactor) trendForWindow(events []*repository.CustomerEvent, currentMRR int, since time.Time) float64 {
	// Find the oldest event in this window to get historical MRR
	var oldestMRR *int
	for i := len(events) - 1; i >= 0; i-- {
		if events[i].OccurredAt.After(since) || events[i].OccurredAt.Equal(since) {
			if oldCents, ok := events[i].Data["old_mrr_cents"]; ok {
				switch v := oldCents.(type) {
				case float64:
					val := int(v)
					oldestMRR = &val
				case int:
					oldestMRR = &v
				}
			}
			break
		}
	}

	if oldestMRR == nil || *oldestMRR == 0 {
		if currentMRR > 0 {
			return 100.0 // New revenue = maximum growth
		}
		return 0.0 // No change
	}

	return float64(currentMRR-*oldestMRR) / float64(*oldestMRR) * 100.0
}

// trendToScore converts a percentage change to a 0.0-1.0 score.
func (f *MRRTrendFactor) trendToScore(trendPercent float64) float64 {
	var score float64
	switch {
	case trendPercent > 5:
		// Growing: 0.8-1.0
		score = 0.8 + min(trendPercent/50.0, 0.2)
	case trendPercent >= -5:
		// Stable: 0.5-0.7
		score = 0.5 + (trendPercent+5.0)/10.0*0.2
	case trendPercent >= -50:
		// Declining: 0.1-0.4
		score = 0.4 + (trendPercent+50.0)/45.0*0.3
	default:
		// Severe decline
		score = 0.0
	}

	if score < 0 {
		score = 0
	}
	if score > 1 {
		score = 1
	}
	return score
}
