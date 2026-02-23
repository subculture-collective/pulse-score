package scoring

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/google/uuid"

	"github.com/onnwee/pulse-score/internal/repository"
)

// EngagementFactor calculates a score based on customer activity relative to org.
type EngagementFactor struct {
	events *repository.CustomerEventRepository
}

// NewEngagementFactor creates a new EngagementFactor.
func NewEngagementFactor(events *repository.CustomerEventRepository) *EngagementFactor {
	return &EngagementFactor{events: events}
}

// Name returns the factor name.
func (f *EngagementFactor) Name() string {
	return "engagement"
}

// engagementEventTypes are the event types considered as engagement activity.
var engagementEventTypes = []string{"login", "feature_use", "api_call"}

// Calculate computes the engagement score relative to org median.
// Returns nil if no activity data exists (factor skipped in aggregation).
func (f *EngagementFactor) Calculate(ctx context.Context, customerID, orgID uuid.UUID) (*FactorResult, error) {
	now := time.Now()
	since30d := now.AddDate(0, 0, -30)
	since7d := now.AddDate(0, 0, -7)

	// Aggregate activity counts across all engagement event types
	totalCounts := make(map[uuid.UUID]int)
	recentCounts := make(map[uuid.UUID]int)

	for _, eventType := range engagementEventTypes {
		counts, err := f.events.CountEventsByTypeForOrg(ctx, orgID, eventType, since30d)
		if err != nil {
			return nil, fmt.Errorf("count %s events: %w", eventType, err)
		}
		for id, count := range counts {
			totalCounts[id] += count
		}

		recent, err := f.events.CountEventsByTypeForOrg(ctx, orgID, eventType, since7d)
		if err != nil {
			return nil, fmt.Errorf("count recent %s events: %w", eventType, err)
		}
		for id, count := range recent {
			recentCounts[id] += count
		}
	}

	// No activity data: return nil (factor skipped)
	if len(totalCounts) == 0 {
		return &FactorResult{Name: f.Name(), Score: nil}, nil
	}

	customerCount := totalCounts[customerID]
	customerRecent := recentCounts[customerID]

	// Calculate org median
	median := f.orgMedian(totalCounts)

	// Score based on position relative to median
	var score float64
	if median == 0 {
		if customerCount > 0 {
			score = 0.8
		} else {
			score = 0.3
		}
	} else {
		ratio := float64(customerCount) / float64(median)
		switch {
		case ratio >= 1.5:
			// Well above median: 0.8-1.0
			score = 0.8 + min((ratio-1.5)*0.1, 0.2)
		case ratio >= 0.5:
			// Around median: 0.4-0.8
			score = 0.4 + (ratio-0.5)*0.4
		default:
			// Below median: 0.0-0.4
			score = ratio * 0.8
		}
	}

	// Recency bonus: recent activity in last 7d weighted higher
	if customerRecent > 0 {
		bonus := min(float64(customerRecent)*0.02, 0.1)
		score += bonus
	}

	if score < 0 {
		score = 0
	}
	if score > 1 {
		score = 1
	}

	return &FactorResult{Name: f.Name(), Score: &score}, nil
}

// orgMedian calculates the median activity count across all customers in org.
func (f *EngagementFactor) orgMedian(counts map[uuid.UUID]int) int {
	if len(counts) == 0 {
		return 0
	}

	values := make([]int, 0, len(counts))
	for _, v := range counts {
		values = append(values, v)
	}

	for i := 0; i < len(values); i++ {
		for j := i + 1; j < len(values); j++ {
			if values[j] < values[i] {
				values[i], values[j] = values[j], values[i]
			}
		}
	}

	n := len(values)
	if n%2 == 0 {
		return int(math.Round(float64(values[n/2-1]+values[n/2]) / 2.0))
	}
	return values[n/2]
}
