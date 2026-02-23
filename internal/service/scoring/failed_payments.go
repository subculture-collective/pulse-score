package scoring

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/onnwee/pulse-score/internal/repository"
	"github.com/onnwee/pulse-score/internal/service"
)

// FailedPaymentsFactor wraps the existing PaymentHealthService and adds refinements.
type FailedPaymentsFactor struct {
	healthSvc *service.PaymentHealthService
	payments  *repository.StripePaymentRepository
}

// NewFailedPaymentsFactor creates a new FailedPaymentsFactor.
func NewFailedPaymentsFactor(
	healthSvc *service.PaymentHealthService,
	payments *repository.StripePaymentRepository,
) *FailedPaymentsFactor {
	return &FailedPaymentsFactor{
		healthSvc: healthSvc,
		payments:  payments,
	}
}

// Name returns the factor name.
func (f *FailedPaymentsFactor) Name() string {
	return "failed_payments"
}

// Calculate computes the failed payment score normalized to 0.0-1.0.
func (f *FailedPaymentsFactor) Calculate(ctx context.Context, customerID, orgID uuid.UUID) (*FactorResult, error) {
	healthResult, err := f.healthSvc.Calculate(ctx, customerID)
	if err != nil {
		return nil, fmt.Errorf("payment health calculate: %w", err)
	}

	// Check for recent failures in 90d
	now := time.Now()
	failed90d, err := f.payments.CountFailedByCustomerInWindow(ctx, customerID, now.AddDate(0, 0, -90))
	if err != nil {
		return nil, fmt.Errorf("count failed 90d: %w", err)
	}
	total90d, err := f.payments.CountByCustomerInWindow(ctx, customerID, now.AddDate(0, 0, -90))
	if err != nil {
		return nil, fmt.Errorf("count total 90d: %w", err)
	}

	// No payment data at all: cannot evaluate, use base health score
	if total90d == 0 {
		score := float64(healthResult.Score) / 100.0
		return &FactorResult{Name: f.Name(), Score: &score}, nil
	}

	var score float64

	switch {
	case failed90d == 0:
		// No failures in 90d: perfect score
		score = 1.0

	case failed90d == 1 && healthResult.ConsecutiveFailures == 0:
		// Single failure, resolved: minor penalty
		score = 0.75

	case healthResult.ConsecutiveFailures > 0:
		// Unresolved/consecutive failures: significant penalty
		switch {
		case healthResult.ConsecutiveFailures >= 3:
			score = 0.0
		case healthResult.ConsecutiveFailures == 2:
			score = 0.15
		default:
			score = 0.25
		}

	default:
		// Multiple failures but resolved: proportional penalty
		failRate := float64(failed90d) / float64(total90d)
		score = 1.0 - failRate*1.5
		if score < 0.1 {
			score = 0.1
		}
	}

	// Apply recency weighting: recent failures (7d) penalized more
	failed7d, err := f.payments.CountFailedByCustomerInWindow(ctx, customerID, now.AddDate(0, 0, -7))
	if err == nil && failed7d > 0 {
		penalty := float64(failed7d) * 0.1
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
