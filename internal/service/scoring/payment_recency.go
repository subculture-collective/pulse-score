package scoring

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/onnwee/pulse-score/internal/service"
)

// PaymentRecencyFactor wraps the existing PaymentRecencyService to produce a 0.0-1.0 score.
type PaymentRecencyFactor struct {
	recencySvc *service.PaymentRecencyService
}

// NewPaymentRecencyFactor creates a new PaymentRecencyFactor.
func NewPaymentRecencyFactor(recencySvc *service.PaymentRecencyService) *PaymentRecencyFactor {
	return &PaymentRecencyFactor{recencySvc: recencySvc}
}

// Name returns the factor name.
func (f *PaymentRecencyFactor) Name() string {
	return "payment_recency"
}

// Calculate computes the payment recency score normalized to 0.0-1.0.
func (f *PaymentRecencyFactor) Calculate(ctx context.Context, customerID, orgID uuid.UUID) (*FactorResult, error) {
	result, err := f.recencySvc.Calculate(ctx, customerID)
	if err != nil {
		return nil, fmt.Errorf("payment recency calculate: %w", err)
	}

	// No payment history: return neutral score
	if result.DaysSinceLastPayment < 0 {
		score := 0.5
		return &FactorResult{Name: f.Name(), Score: &score}, nil
	}

	// Normalize 0-100 → 0.0-1.0
	score := float64(result.Score) / 100.0
	if score < 0 {
		score = 0
	}
	if score > 1 {
		score = 1
	}

	return &FactorResult{Name: f.Name(), Score: &score}, nil
}
