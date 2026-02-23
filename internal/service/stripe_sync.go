package service

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/stripe/stripe-go/v81"
	stripecharge "github.com/stripe/stripe-go/v81/charge"
	stripecustomer "github.com/stripe/stripe-go/v81/customer"
	stripesub "github.com/stripe/stripe-go/v81/subscription"

	"github.com/onnwee/pulse-score/internal/repository"
)

// StripeSyncService handles syncing data from Stripe to local database.
type StripeSyncService struct {
	customers    *repository.CustomerRepository
	subs         *repository.StripeSubscriptionRepository
	payments     *repository.StripePaymentRepository
	events       *repository.CustomerEventRepository
	oauthSvc     *StripeOAuthService
	paymentDays  int
}

// NewStripeSyncService creates a new StripeSyncService.
func NewStripeSyncService(
	customers *repository.CustomerRepository,
	subs *repository.StripeSubscriptionRepository,
	payments *repository.StripePaymentRepository,
	events *repository.CustomerEventRepository,
	oauthSvc *StripeOAuthService,
	paymentDays int,
) *StripeSyncService {
	return &StripeSyncService{
		customers:   customers,
		subs:        subs,
		payments:    payments,
		events:      events,
		oauthSvc:    oauthSvc,
		paymentDays: paymentDays,
	}
}

// SyncProgress tracks the progress of a sync operation.
type SyncProgress struct {
	Step     string `json:"step"`
	Total    int    `json:"total"`
	Current  int    `json:"current"`
	Errors   int    `json:"errors"`
}

// SyncCustomers fetches all customers from Stripe and upserts them locally.
func (s *StripeSyncService) SyncCustomers(ctx context.Context, orgID uuid.UUID) (*SyncProgress, error) {
	accessToken, err := s.oauthSvc.GetAccessToken(ctx, orgID)
	if err != nil {
		return nil, fmt.Errorf("get access token: %w", err)
	}

	progress := &SyncProgress{Step: "customers"}

	params := &stripe.CustomerListParams{}
	params.Limit = stripe.Int64(100)

	client := newStripeCustomerClient(accessToken)
	iter := client.List(params)

	for iter.Next() {
		c := iter.Customer()
		progress.Total++

		now := time.Now()
		created := time.Unix(c.Created, 0)
		localCustomer := &repository.Customer{
			OrgID:       orgID,
			ExternalID:  c.ID,
			Source:      "stripe",
			Email:       c.Email,
			Name:        c.Name,
			Currency:    string(c.Currency),
			FirstSeenAt: &created,
			LastSeenAt:  &now,
			Metadata:    stripeMetadataToMap(c.Metadata),
		}

		if err := s.customers.UpsertByExternal(ctx, localCustomer); err != nil {
			slog.Error("failed to upsert customer", "stripe_id", c.ID, "error", err)
			progress.Errors++
			continue
		}
		progress.Current++
	}

	if err := iter.Err(); err != nil {
		return progress, fmt.Errorf("iterate customers: %w", err)
	}

	slog.Info("customer sync complete",
		"org_id", orgID,
		"total", progress.Total,
		"synced", progress.Current,
		"errors", progress.Errors,
	)

	return progress, nil
}

// SyncCustomersSince fetches customers modified since the given time (incremental sync).
func (s *StripeSyncService) SyncCustomersSince(ctx context.Context, orgID uuid.UUID, since time.Time) (*SyncProgress, error) {
	accessToken, err := s.oauthSvc.GetAccessToken(ctx, orgID)
	if err != nil {
		return nil, fmt.Errorf("get access token: %w", err)
	}

	progress := &SyncProgress{Step: "customers_incremental"}

	params := &stripe.CustomerListParams{}
	params.Limit = stripe.Int64(100)
	params.CreatedRange = &stripe.RangeQueryParams{GreaterThanOrEqual: since.Unix()}

	client := newStripeCustomerClient(accessToken)
	iter := client.List(params)

	for iter.Next() {
		c := iter.Customer()
		progress.Total++

		now := time.Now()
		created := time.Unix(c.Created, 0)
		localCustomer := &repository.Customer{
			OrgID:       orgID,
			ExternalID:  c.ID,
			Source:      "stripe",
			Email:       c.Email,
			Name:        c.Name,
			Currency:    string(c.Currency),
			FirstSeenAt: &created,
			LastSeenAt:  &now,
			Metadata:    stripeMetadataToMap(c.Metadata),
		}

		if err := s.customers.UpsertByExternal(ctx, localCustomer); err != nil {
			slog.Error("failed to upsert customer", "stripe_id", c.ID, "error", err)
			progress.Errors++
			continue
		}
		progress.Current++
	}

	if err := iter.Err(); err != nil {
		return progress, fmt.Errorf("iterate customers: %w", err)
	}

	return progress, nil
}

// SyncSubscriptions fetches all subscriptions from Stripe and upserts them locally.
func (s *StripeSyncService) SyncSubscriptions(ctx context.Context, orgID uuid.UUID) (*SyncProgress, error) {
	accessToken, err := s.oauthSvc.GetAccessToken(ctx, orgID)
	if err != nil {
		return nil, fmt.Errorf("get access token: %w", err)
	}

	progress := &SyncProgress{Step: "subscriptions"}

	params := &stripe.SubscriptionListParams{}
	params.Limit = stripe.Int64(100)

	client := newStripeSubClient(accessToken)
	iter := client.List(params)

	for iter.Next() {
		sub := iter.Subscription()
		progress.Total++

		// Find local customer
		localCustomer, err := s.customers.GetByExternalID(ctx, orgID, "stripe", sub.Customer.ID)
		if err != nil {
			slog.Error("failed to lookup customer for subscription",
				"stripe_sub_id", sub.ID,
				"stripe_customer_id", sub.Customer.ID,
				"error", err,
			)
			progress.Errors++
			continue
		}
		if localCustomer == nil {
			slog.Warn("subscription references unknown customer",
				"stripe_sub_id", sub.ID,
				"stripe_customer_id", sub.Customer.ID,
			)
			progress.Errors++
			continue
		}

		// Resolve plan name and amount from subscription items
		planName, amountCents, interval, currency := extractSubscriptionDetails(sub)

		var periodStart, periodEnd *time.Time
		if sub.CurrentPeriodStart > 0 {
			t := time.Unix(sub.CurrentPeriodStart, 0)
			periodStart = &t
		}
		if sub.CurrentPeriodEnd > 0 {
			t := time.Unix(sub.CurrentPeriodEnd, 0)
			periodEnd = &t
		}

		var canceledAt *time.Time
		if sub.CanceledAt > 0 {
			t := time.Unix(sub.CanceledAt, 0)
			canceledAt = &t
		}

		localSub := &repository.StripeSubscription{
			OrgID:                orgID,
			CustomerID:           localCustomer.ID,
			StripeSubscriptionID: sub.ID,
			Status:               string(sub.Status),
			PlanName:             planName,
			AmountCents:          amountCents,
			Currency:             currency,
			Interval:             interval,
			CurrentPeriodStart:   periodStart,
			CurrentPeriodEnd:     periodEnd,
			CanceledAt:           canceledAt,
			Metadata:             stripeMetadataToMap(sub.Metadata),
		}

		if err := s.subs.Upsert(ctx, localSub); err != nil {
			slog.Error("failed to upsert subscription", "stripe_sub_id", sub.ID, "error", err)
			progress.Errors++
			continue
		}
		progress.Current++
	}

	if err := iter.Err(); err != nil {
		return progress, fmt.Errorf("iterate subscriptions: %w", err)
	}

	slog.Info("subscription sync complete",
		"org_id", orgID,
		"total", progress.Total,
		"synced", progress.Current,
		"errors", progress.Errors,
	)

	return progress, nil
}

// SyncPayments fetches charges from Stripe and upserts them locally.
func (s *StripeSyncService) SyncPayments(ctx context.Context, orgID uuid.UUID) (*SyncProgress, error) {
	accessToken, err := s.oauthSvc.GetAccessToken(ctx, orgID)
	if err != nil {
		return nil, fmt.Errorf("get access token: %w", err)
	}

	progress := &SyncProgress{Step: "payments"}
	since := time.Now().AddDate(0, 0, -s.paymentDays)

	params := &stripe.ChargeListParams{}
	params.Limit = stripe.Int64(100)
	params.CreatedRange = &stripe.RangeQueryParams{GreaterThanOrEqual: since.Unix()}

	client := newStripeChargeClient(accessToken)
	iter := client.List(params)

	for iter.Next() {
		ch := iter.Charge()
		progress.Total++

		if ch.Customer == nil {
			continue
		}

		localCustomer, err := s.customers.GetByExternalID(ctx, orgID, "stripe", ch.Customer.ID)
		if err != nil {
			slog.Error("failed to lookup customer for charge",
				"stripe_charge_id", ch.ID,
				"error", err,
			)
			progress.Errors++
			continue
		}
		if localCustomer == nil {
			progress.Errors++
			continue
		}

		status := "succeeded"
		if ch.Status == "failed" {
			status = "failed"
		} else if !ch.Paid {
			status = "pending"
		}

		var paidAt *time.Time
		if ch.Created > 0 {
			t := time.Unix(ch.Created, 0)
			paidAt = &t
		}

		localPayment := &repository.StripePayment{
			OrgID:           orgID,
			CustomerID:      localCustomer.ID,
			StripePaymentID: ch.ID,
			AmountCents:     int(ch.Amount),
			Currency:        string(ch.Currency),
			Status:          status,
			FailureCode:     string(ch.FailureCode),
			FailureMessage:  ch.FailureMessage,
			PaidAt:          paidAt,
		}

		if err := s.payments.Upsert(ctx, localPayment); err != nil {
			slog.Error("failed to upsert payment", "stripe_charge_id", ch.ID, "error", err)
			progress.Errors++
			continue
		}
		progress.Current++

		// Create customer event for failed payments
		if status == "failed" {
			event := &repository.CustomerEvent{
				OrgID:           orgID,
				CustomerID:      localCustomer.ID,
				EventType:       "payment.failed",
				Source:          "stripe",
				ExternalEventID: "charge_failed_" + ch.ID,
				OccurredAt:      time.Unix(ch.Created, 0),
				Data: map[string]any{
					"amount_cents":    ch.Amount,
					"currency":        string(ch.Currency),
					"failure_code":    string(ch.FailureCode),
					"failure_message": ch.FailureMessage,
				},
			}
			if err := s.events.Upsert(ctx, event); err != nil {
				slog.Error("failed to create payment failed event", "error", err)
			}
		}
	}

	if err := iter.Err(); err != nil {
		return progress, fmt.Errorf("iterate charges: %w", err)
	}

	slog.Info("payment sync complete",
		"org_id", orgID,
		"total", progress.Total,
		"synced", progress.Current,
		"errors", progress.Errors,
	)

	return progress, nil
}

// SyncPaymentsSince fetches charges modified since the given time.
func (s *StripeSyncService) SyncPaymentsSince(ctx context.Context, orgID uuid.UUID, since time.Time) (*SyncProgress, error) {
	accessToken, err := s.oauthSvc.GetAccessToken(ctx, orgID)
	if err != nil {
		return nil, fmt.Errorf("get access token: %w", err)
	}

	progress := &SyncProgress{Step: "payments_incremental"}

	params := &stripe.ChargeListParams{}
	params.Limit = stripe.Int64(100)
	params.CreatedRange = &stripe.RangeQueryParams{GreaterThanOrEqual: since.Unix()}

	client := newStripeChargeClient(accessToken)
	iter := client.List(params)

	for iter.Next() {
		ch := iter.Charge()
		progress.Total++

		if ch.Customer == nil {
			continue
		}

		localCustomer, err := s.customers.GetByExternalID(ctx, orgID, "stripe", ch.Customer.ID)
		if err != nil || localCustomer == nil {
			progress.Errors++
			continue
		}

		status := "succeeded"
		if ch.Status == "failed" {
			status = "failed"
		} else if !ch.Paid {
			status = "pending"
		}

		var paidAt *time.Time
		if ch.Created > 0 {
			t := time.Unix(ch.Created, 0)
			paidAt = &t
		}

		localPayment := &repository.StripePayment{
			OrgID:           orgID,
			CustomerID:      localCustomer.ID,
			StripePaymentID: ch.ID,
			AmountCents:     int(ch.Amount),
			Currency:        string(ch.Currency),
			Status:          status,
			FailureCode:     string(ch.FailureCode),
			FailureMessage:  ch.FailureMessage,
			PaidAt:          paidAt,
		}

		if err := s.payments.Upsert(ctx, localPayment); err != nil {
			progress.Errors++
			continue
		}
		progress.Current++
	}

	if err := iter.Err(); err != nil {
		return progress, fmt.Errorf("iterate charges: %w", err)
	}

	return progress, nil
}

// extractSubscriptionDetails extracts plan name, amount, interval, and currency from a subscription.
func extractSubscriptionDetails(sub *stripe.Subscription) (planName string, amountCents int, interval, currency string) {
	if sub.Items == nil || len(sub.Items.Data) == 0 {
		return "", 0, "", "usd"
	}

	for _, item := range sub.Items.Data {
		if item.Price != nil {
			amountCents += int(item.Price.UnitAmount * item.Quantity)
			currency = string(item.Price.Currency)
			interval = string(item.Price.Recurring.Interval)

			if item.Price.Product != nil {
				planName = item.Price.Product.Name
			}
		}
	}

	return planName, amountCents, interval, currency
}

func stripeMetadataToMap(metadata map[string]string) map[string]any {
	if metadata == nil {
		return map[string]any{}
	}
	result := make(map[string]any, len(metadata))
	for k, v := range metadata {
		result[k] = v
	}
	return result
}

// newStripeCustomerClient creates a Stripe customer client with the given access token.
func newStripeCustomerClient(accessToken string) stripecustomer.Client {
	return stripecustomer.Client{B: stripe.GetBackend(stripe.APIBackend), Key: accessToken}
}

// newStripeSubClient creates a Stripe subscription client with the given access token.
func newStripeSubClient(accessToken string) stripesub.Client {
	return stripesub.Client{B: stripe.GetBackend(stripe.APIBackend), Key: accessToken}
}

// newStripeChargeClient creates a Stripe charge client with the given access token.
func newStripeChargeClient(accessToken string) stripecharge.Client {
	return stripecharge.Client{B: stripe.GetBackend(stripe.APIBackend), Key: accessToken}
}
