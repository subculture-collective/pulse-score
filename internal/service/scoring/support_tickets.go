package scoring

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/google/uuid"

	"github.com/onnwee/pulse-score/internal/repository"
)

// SupportTicketsFactor calculates a score based on support ticket volume relative to org.
type SupportTicketsFactor struct {
	events *repository.CustomerEventRepository
}

// NewSupportTicketsFactor creates a new SupportTicketsFactor.
func NewSupportTicketsFactor(events *repository.CustomerEventRepository) *SupportTicketsFactor {
	return &SupportTicketsFactor{events: events}
}

// Name returns the factor name.
func (f *SupportTicketsFactor) Name() string {
	return "support_tickets"
}

// Calculate computes the support ticket score relative to org median.
// Returns nil if no ticket data exists (factor skipped in aggregation).
func (f *SupportTicketsFactor) Calculate(ctx context.Context, customerID, orgID uuid.UUID) (*FactorResult, error) {
	now := time.Now()
	since := now.AddDate(0, 0, -90)

	// Get ticket counts per customer for the org
	ticketCounts, err := f.events.CountEventsByTypeForOrg(ctx, orgID, "ticket.opened", since)
	if err != nil {
		return nil, fmt.Errorf("count ticket events: %w", err)
	}

	// No ticket data: return nil (factor skipped)
	if len(ticketCounts) == 0 {
		return &FactorResult{Name: f.Name(), Score: nil}, nil
	}

	// Get this customer's count
	customerCount := ticketCounts[customerID]

	// Count unresolved tickets (opened minus resolved)
	resolvedCounts, err := f.events.CountEventsByTypeForOrg(ctx, orgID, "ticket.resolved", since)
	if err != nil {
		return nil, fmt.Errorf("count resolved events: %w", err)
	}

	unresolvedCount := customerCount - resolvedCounts[customerID]
	if unresolvedCount < 0 {
		unresolvedCount = 0
	}

	// Calculate org median ticket count
	median := f.orgMedian(ticketCounts)

	// Score based on position relative to median
	var score float64
	if median == 0 {
		// Very few tickets in org
		if customerCount == 0 {
			score = 1.0
		} else {
			score = 0.5
		}
	} else {
		ratio := float64(customerCount) / float64(median)
		switch {
		case ratio <= 0.5:
			// Well below average: 0.7-1.0
			score = 1.0 - ratio*0.6
		case ratio <= 1.5:
			// Around average: 0.4-0.7
			score = 0.7 - (ratio-0.5)*0.3
		default:
			// Above average: 0.0-0.4
			score = 0.4 - min((ratio-1.5)*0.2, 0.4)
		}
	}

	// Penalize unresolved tickets more
	if unresolvedCount > 0 {
		penalty := float64(unresolvedCount) * 0.1
		score -= penalty
	}

	if score < 0 {
		score = 0
	}
	if score > 1 {
		score = 1
	}

	return &FactorResult{Name: f.Name(), Score: &score}, nil
}

// orgMedian calculates the median ticket count across all customers in org.
func (f *SupportTicketsFactor) orgMedian(counts map[uuid.UUID]int) int {
	if len(counts) == 0 {
		return 0
	}

	values := make([]int, 0, len(counts))
	for _, v := range counts {
		values = append(values, v)
	}

	// Simple sort for median
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
