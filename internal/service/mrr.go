package service

import (
	"context"
	"fmt"
	"log/slog"
	"math"

	"github.com/google/uuid"

	"github.com/onnwee/pulse-score/internal/repository"
)

// MRRService calculates Monthly Recurring Revenue per customer.
type MRRService struct {
	customers *repository.CustomerRepository
	subs      *repository.StripeSubscriptionRepository
	events    *repository.CustomerEventRepository
}

// NewMRRService creates a new MRRService.
func NewMRRService(
	customers *repository.CustomerRepository,
	subs *repository.StripeSubscriptionRepository,
	events *repository.CustomerEventRepository,
) *MRRService {
	return &MRRService{
		customers: customers,
		subs:      subs,
		events:    events,
	}
}

// CalculateForCustomer calculates and updates MRR for a single customer.
func (s *MRRService) CalculateForCustomer(ctx context.Context, customerID uuid.UUID) error {
	customer, err := s.customers.GetByID(ctx, customerID)
	if err != nil {
		return fmt.Errorf("get customer: %w", err)
	}
	if customer == nil {
		return nil
	}

	activeSubs, err := s.subs.ListActiveByCustomer(ctx, customerID)
	if err != nil {
		return fmt.Errorf("list active subscriptions: %w", err)
	}

	newMRR := 0
	for _, sub := range activeSubs {
		monthlyAmount := normalizeToMonthly(sub.AmountCents, sub.Interval, sub.Status)
		newMRR += monthlyAmount
	}

	oldMRR := customer.MRRCents

	if err := s.customers.UpdateMRR(ctx, customerID, newMRR); err != nil {
		return fmt.Errorf("update mrr: %w", err)
	}

	// Create MRR change event if significant (>10% change)
	if oldMRR > 0 {
		changePercent := math.Abs(float64(newMRR-oldMRR)) / float64(oldMRR) * 100
		if changePercent > 10 {
			event := &repository.CustomerEvent{
				OrgID:           customer.OrgID,
				CustomerID:      customerID,
				EventType:       "mrr.changed",
				Source:          "system",
				ExternalEventID: fmt.Sprintf("mrr_change_%s_%d", customerID, newMRR),
				OccurredAt:      customer.UpdatedAt,
				Data: map[string]any{
					"old_mrr_cents":  oldMRR,
					"new_mrr_cents":  newMRR,
					"change_percent": changePercent,
				},
			}
			if err := s.events.Upsert(ctx, event); err != nil {
				slog.Error("failed to create MRR change event", "error", err)
			}
		}
	}

	return nil
}

// CalculateForOrg recalculates MRR for all customers in an org.
func (s *MRRService) CalculateForOrg(ctx context.Context, orgID uuid.UUID) error {
	customers, err := s.customers.ListByOrg(ctx, orgID)
	if err != nil {
		return fmt.Errorf("list customers: %w", err)
	}

	var errs int
	for _, c := range customers {
		if err := s.CalculateForCustomer(ctx, c.ID); err != nil {
			slog.Error("failed to calculate MRR for customer",
				"customer_id", c.ID,
				"error", err,
			)
			errs++
		}
	}

	slog.Info("MRR calculation complete",
		"org_id", orgID,
		"total_customers", len(customers),
		"errors", errs,
	)

	return nil
}

// normalizeToMonthly converts a subscription amount to monthly equivalent.
func normalizeToMonthly(amountCents int, interval, status string) int {
	// Canceled subscriptions contribute 0 MRR
	if status == "canceled" || status == "incomplete_expired" {
		return 0
	}

	// Trialing and past_due subscriptions count at full price
	switch interval {
	case "month":
		return amountCents
	case "year":
		return amountCents / 12
	case "week":
		// 52 weeks / 12 months ≈ 4.33
		return int(float64(amountCents) * 4.33)
	case "day":
		// ~30.44 days per month
		return int(float64(amountCents) * 30.44)
	default:
		// Default to treating as monthly
		return amountCents
	}
}
