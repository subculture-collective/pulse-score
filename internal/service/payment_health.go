package service

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"github.com/onnwee/pulse-score/internal/repository"
)

// PaymentHealthResult holds payment health metrics for a customer.
type PaymentHealthResult struct {
	CustomerID          uuid.UUID `json:"customer_id"`
	FailureRate7d       float64   `json:"failure_rate_7d"`
	FailureRate30d      float64   `json:"failure_rate_30d"`
	FailureRate90d      float64   `json:"failure_rate_90d"`
	ConsecutiveFailures int       `json:"consecutive_failures"`
	Score               int       `json:"score"` // 0-100 (100 = perfectly healthy)
}

// PaymentHealthService calculates payment health metrics.
type PaymentHealthService struct {
	payments  *repository.StripePaymentRepository
	events    *repository.CustomerEventRepository
	customers *repository.CustomerRepository
}

// NewPaymentHealthService creates a new PaymentHealthService.
func NewPaymentHealthService(
	payments *repository.StripePaymentRepository,
	events *repository.CustomerEventRepository,
	customers *repository.CustomerRepository,
) *PaymentHealthService {
	return &PaymentHealthService{
		payments:  payments,
		events:    events,
		customers: customers,
	}
}

// Calculate computes payment health for a customer.
func (s *PaymentHealthService) Calculate(ctx context.Context, customerID uuid.UUID) (*PaymentHealthResult, error) {
	now := time.Now()

	failed7d, err := s.payments.CountFailedByCustomerInWindow(ctx, customerID, now.AddDate(0, 0, -7))
	if err != nil {
		return nil, fmt.Errorf("count failed 7d: %w", err)
	}
	total7d, err := s.payments.CountByCustomerInWindow(ctx, customerID, now.AddDate(0, 0, -7))
	if err != nil {
		return nil, fmt.Errorf("count total 7d: %w", err)
	}

	failed30d, err := s.payments.CountFailedByCustomerInWindow(ctx, customerID, now.AddDate(0, 0, -30))
	if err != nil {
		return nil, fmt.Errorf("count failed 30d: %w", err)
	}
	total30d, err := s.payments.CountByCustomerInWindow(ctx, customerID, now.AddDate(0, 0, -30))
	if err != nil {
		return nil, fmt.Errorf("count total 30d: %w", err)
	}

	failed90d, err := s.payments.CountFailedByCustomerInWindow(ctx, customerID, now.AddDate(0, 0, -90))
	if err != nil {
		return nil, fmt.Errorf("count failed 90d: %w", err)
	}
	total90d, err := s.payments.CountByCustomerInWindow(ctx, customerID, now.AddDate(0, 0, -90))
	if err != nil {
		return nil, fmt.Errorf("count total 90d: %w", err)
	}

	consecutiveFailures, err := s.payments.CountConsecutiveFailures(ctx, customerID)
	if err != nil {
		return nil, fmt.Errorf("count consecutive failures: %w", err)
	}

	result := &PaymentHealthResult{
		CustomerID:          customerID,
		FailureRate7d:       failureRate(failed7d, total7d),
		FailureRate30d:      failureRate(failed30d, total30d),
		FailureRate90d:      failureRate(failed90d, total90d),
		ConsecutiveFailures: consecutiveFailures,
	}

	result.Score = calculatePaymentScore(result)

	return result, nil
}

// TrackFailedPayment records a payment failure and creates an alert event if warranted.
func (s *PaymentHealthService) TrackFailedPayment(ctx context.Context, customerID uuid.UUID) error {
	consecutive, err := s.payments.CountConsecutiveFailures(ctx, customerID)
	if err != nil {
		return fmt.Errorf("count consecutive failures: %w", err)
	}

	// Create an alert event at specific thresholds
	if consecutive == 2 || consecutive == 3 || consecutive == 5 {
		customer, err := s.customers.GetByID(ctx, customerID)
		if err != nil || customer == nil {
			return nil
		}

		event := &repository.CustomerEvent{
			OrgID:           customer.OrgID,
			CustomerID:      customerID,
			EventType:       "payment.consecutive_failures",
			Source:          "system",
			ExternalEventID: fmt.Sprintf("consec_fail_%s_%d_%d", customerID, consecutive, time.Now().Unix()),
			OccurredAt:      time.Now(),
			Data: map[string]any{
				"consecutive_failures": consecutive,
				"severity":             failureSeverity(consecutive),
			},
		}

		if err := s.events.Upsert(ctx, event); err != nil {
			slog.Error("failed to create consecutive failure event", "error", err)
		}
	}

	return nil
}

func failureRate(failed, total int) float64 {
	if total == 0 {
		return 0
	}
	return float64(failed) / float64(total)
}

// calculatePaymentScore returns 0-100 where 100 = perfectly healthy.
// Weighted: 30d failure rate (weight 40), 7d rate (weight 30), consecutive failures (weight 30).
func calculatePaymentScore(r *PaymentHealthResult) int {
	// 7d component (30% weight): 0 failures = 100, 100% failures = 0
	score7d := (1 - r.FailureRate7d) * 100

	// 30d component (40% weight)
	score30d := (1 - r.FailureRate30d) * 100

	// Consecutive failures component (30% weight)
	// 0 consecutive = 100, 3+ consecutive = 0
	var consecutiveScore float64
	switch {
	case r.ConsecutiveFailures == 0:
		consecutiveScore = 100
	case r.ConsecutiveFailures == 1:
		consecutiveScore = 60
	case r.ConsecutiveFailures == 2:
		consecutiveScore = 30
	default:
		consecutiveScore = 0
	}

	total := score7d*0.30 + score30d*0.40 + consecutiveScore*0.30

	score := int(total)
	if score < 0 {
		return 0
	}
	if score > 100 {
		return 100
	}
	return score
}

func failureSeverity(consecutive int) string {
	switch {
	case consecutive >= 5:
		return "critical"
	case consecutive >= 3:
		return "high"
	default:
		return "medium"
	}
}
