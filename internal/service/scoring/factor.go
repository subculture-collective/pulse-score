package scoring

import (
	"context"

	"github.com/google/uuid"
)

// FactorResult holds the result of a single scoring factor calculation.
type FactorResult struct {
	Name  string   // Factor name (e.g., "payment_recency")
	Score *float64 // nil = factor unavailable (skip in aggregation)
}

// ScoreFactor is the interface all scoring factors must implement.
type ScoreFactor interface {
	Name() string
	Calculate(ctx context.Context, customerID, orgID uuid.UUID) (*FactorResult, error)
}
