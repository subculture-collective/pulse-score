package service

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/google/uuid"

	"github.com/onnwee/pulse-score/internal/repository"
)

// PaymentRecencyResult holds payment recency metrics for a customer.
type PaymentRecencyResult struct {
	CustomerID           uuid.UUID  `json:"customer_id"`
	LastPaymentAt        *time.Time `json:"last_payment_at,omitempty"`
	DaysSinceLastPayment int        `json:"days_since_last_payment"`
	BillingIntervalDays  int        `json:"billing_interval_days"`
	Score                int        `json:"score"` // 0-100 (100 = paid recently)
}

// PaymentRecencyService calculates payment recency scores.
type PaymentRecencyService struct {
	payments *repository.StripePaymentRepository
	subs     *repository.StripeSubscriptionRepository
}

// NewPaymentRecencyService creates a new PaymentRecencyService.
func NewPaymentRecencyService(
	payments *repository.StripePaymentRepository,
	subs *repository.StripeSubscriptionRepository,
) *PaymentRecencyService {
	return &PaymentRecencyService{
		payments: payments,
		subs:     subs,
	}
}

// Calculate computes payment recency score for a customer.
func (s *PaymentRecencyService) Calculate(ctx context.Context, customerID uuid.UUID) (*PaymentRecencyResult, error) {
	lastPayment, err := s.payments.GetLastSuccessfulPayment(ctx, customerID)
	if err != nil {
		return nil, fmt.Errorf("get last payment: %w", err)
	}

	result := &PaymentRecencyResult{CustomerID: customerID}

	if lastPayment == nil || lastPayment.PaidAt == nil {
		// No successful payments — lowest score
		result.DaysSinceLastPayment = -1
		result.Score = 0
		return result, nil
	}

	result.LastPaymentAt = lastPayment.PaidAt
	result.DaysSinceLastPayment = int(time.Since(*lastPayment.PaidAt).Hours() / 24)

	// Determine expected billing interval from active subscriptions
	billingDays, err := s.expectedBillingInterval(ctx, customerID)
	if err != nil {
		return nil, fmt.Errorf("get billing interval: %w", err)
	}
	result.BillingIntervalDays = billingDays

	result.Score = recencyScore(result.DaysSinceLastPayment, billingDays)

	return result, nil
}

// expectedBillingInterval returns the shortest billing interval in days among active subscriptions.
func (s *PaymentRecencyService) expectedBillingInterval(ctx context.Context, customerID uuid.UUID) (int, error) {
	activeSubs, err := s.subs.ListActiveByCustomer(ctx, customerID)
	if err != nil {
		return 30, err // default to monthly
	}

	if len(activeSubs) == 0 {
		return 30, nil // default to monthly
	}

	minDays := math.MaxInt32
	for _, sub := range activeSubs {
		d := intervalToDays(sub.Interval)
		if d < minDays {
			minDays = d
		}
	}

	return minDays, nil
}

func intervalToDays(interval string) int {
	switch interval {
	case "day":
		return 1
	case "week":
		return 7
	case "month":
		return 30
	case "year":
		return 365
	default:
		return 30
	}
}

// recencyScore returns 0-100 normalized by billing interval.
// 100 = paid within expected interval, decays toward 0 at 3x the interval.
func recencyScore(daysSince, billingIntervalDays int) int {
	if daysSince < 0 {
		return 0 // no payments
	}

	if billingIntervalDays <= 0 {
		billingIntervalDays = 30
	}

	ratio := float64(daysSince) / float64(billingIntervalDays)

	// Within expected interval: 80-100
	// 1x-2x interval: 40-80
	// 2x-3x interval: 0-40
	// Beyond 3x: 0
	var score float64
	switch {
	case ratio <= 1.0:
		score = 100 - (ratio * 20) // 100 → 80
	case ratio <= 2.0:
		score = 80 - ((ratio - 1.0) * 40) // 80 → 40
	case ratio <= 3.0:
		score = 40 - ((ratio - 2.0) * 40) // 40 → 0
	default:
		score = 0
	}

	s := int(score)
	if s < 0 {
		return 0
	}
	if s > 100 {
		return 100
	}
	return s
}
