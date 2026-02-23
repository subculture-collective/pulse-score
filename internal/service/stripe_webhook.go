package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/stripe/stripe-go/v81"
	"github.com/stripe/stripe-go/v81/webhook"

	"github.com/onnwee/pulse-score/internal/repository"
)

// StripeWebhookService handles incoming Stripe webhook events.
type StripeWebhookService struct {
	webhookSecret string
	connRepo      *repository.IntegrationConnectionRepository
	customers     *repository.CustomerRepository
	subs          *repository.StripeSubscriptionRepository
	payments      *repository.StripePaymentRepository
	events        *repository.CustomerEventRepository
	mrrSvc        *MRRService
	paymentHealth *PaymentHealthService

	// processedEvents tracks recently processed event IDs for idempotency
	processedEvents map[string]time.Time
	mu              sync.Mutex
}

// NewStripeWebhookService creates a new StripeWebhookService.
func NewStripeWebhookService(
	webhookSecret string,
	connRepo *repository.IntegrationConnectionRepository,
	customers *repository.CustomerRepository,
	subs *repository.StripeSubscriptionRepository,
	payments *repository.StripePaymentRepository,
	events *repository.CustomerEventRepository,
	mrrSvc *MRRService,
	paymentHealth *PaymentHealthService,
) *StripeWebhookService {
	return &StripeWebhookService{
		webhookSecret:   webhookSecret,
		connRepo:        connRepo,
		customers:       customers,
		subs:            subs,
		payments:        payments,
		events:          events,
		mrrSvc:          mrrSvc,
		paymentHealth:   paymentHealth,
		processedEvents: make(map[string]time.Time),
	}
}

// HandleEvent verifies and processes a Stripe webhook event.
func (s *StripeWebhookService) HandleEvent(ctx context.Context, payload []byte, sigHeader string) error {
	event, err := webhook.ConstructEvent(payload, sigHeader, s.webhookSecret)
	if err != nil {
		return &ValidationError{Field: "signature", Message: "invalid webhook signature"}
	}

	// Idempotency check
	if s.isProcessed(event.ID) {
		slog.Debug("duplicate webhook event skipped", "event_id", event.ID)
		return nil
	}

	slog.Info("processing webhook event",
		"event_id", event.ID,
		"type", event.Type,
	)

	switch event.Type {
	case "customer.created", "customer.updated":
		return s.handleCustomerEvent(ctx, event)
	case "customer.deleted":
		return s.handleCustomerDeleted(ctx, event)
	case "customer.subscription.created", "customer.subscription.updated":
		return s.handleSubscriptionEvent(ctx, event)
	case "customer.subscription.deleted":
		return s.handleSubscriptionDeleted(ctx, event)
	case "invoice.paid":
		return s.handleInvoicePaid(ctx, event)
	case "invoice.payment_failed":
		return s.handleInvoicePaymentFailed(ctx, event)
	default:
		slog.Debug("unhandled webhook event type", "type", event.Type)
	}

	s.markProcessed(event.ID)
	return nil
}

func (s *StripeWebhookService) isProcessed(eventID string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.processedEvents[eventID]; ok {
		return true
	}

	// Clean up old entries (older than 24h)
	cutoff := time.Now().Add(-24 * time.Hour)
	for id, t := range s.processedEvents {
		if t.Before(cutoff) {
			delete(s.processedEvents, id)
		}
	}
	return false
}

func (s *StripeWebhookService) markProcessed(eventID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.processedEvents[eventID] = time.Now()
}

func (s *StripeWebhookService) handleCustomerEvent(ctx context.Context, event stripe.Event) error {
	var cust stripe.Customer
	if err := json.Unmarshal(event.Data.Raw, &cust); err != nil {
		return fmt.Errorf("unmarshal customer: %w", err)
	}

	// Find the org that has this Stripe account connected
	orgID, err := s.findOrgForStripeAccount(ctx, event.Account)
	if err != nil {
		return err
	}

	now := time.Now()
	created := time.Unix(cust.Created, 0)
	localCust := &repository.Customer{
		OrgID:       orgID,
		ExternalID:  cust.ID,
		Source:      "stripe",
		Email:       cust.Email,
		Name:        cust.Name,
		Currency:    string(cust.Currency),
		FirstSeenAt: &created,
		LastSeenAt:  &now,
		Metadata:    stripeMetadataToMap(cust.Metadata),
	}

	if err := s.customers.UpsertByExternal(ctx, localCust); err != nil {
		return fmt.Errorf("upsert customer: %w", err)
	}

	s.markProcessed(event.ID)
	return nil
}

func (s *StripeWebhookService) handleCustomerDeleted(ctx context.Context, event stripe.Event) error {
	var cust stripe.Customer
	if err := json.Unmarshal(event.Data.Raw, &cust); err != nil {
		return fmt.Errorf("unmarshal customer: %w", err)
	}

	orgID, err := s.findOrgForStripeAccount(ctx, event.Account)
	if err != nil {
		return err
	}

	if err := s.customers.SoftDelete(ctx, orgID, "stripe", cust.ID); err != nil {
		return fmt.Errorf("soft delete customer: %w", err)
	}

	s.markProcessed(event.ID)
	return nil
}

func (s *StripeWebhookService) handleSubscriptionEvent(ctx context.Context, event stripe.Event) error {
	var sub stripe.Subscription
	if err := json.Unmarshal(event.Data.Raw, &sub); err != nil {
		return fmt.Errorf("unmarshal subscription: %w", err)
	}

	orgID, err := s.findOrgForStripeAccount(ctx, event.Account)
	if err != nil {
		return err
	}

	if sub.Customer == nil {
		return fmt.Errorf("subscription has no customer")
	}

	localCustomer, err := s.customers.GetByExternalID(ctx, orgID, "stripe", sub.Customer.ID)
	if err != nil {
		return fmt.Errorf("get customer: %w", err)
	}
	if localCustomer == nil {
		slog.Warn("subscription webhook: customer not found locally",
			"stripe_customer_id", sub.Customer.ID,
			"stripe_sub_id", sub.ID,
		)
		return nil
	}

	planName, amountCents, interval, currency := extractSubscriptionDetails(&sub)

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
		return fmt.Errorf("upsert subscription: %w", err)
	}

	// Recalculate MRR for the customer
	if err := s.mrrSvc.CalculateForCustomer(ctx, localCustomer.ID); err != nil {
		slog.Error("failed to recalculate MRR after subscription webhook", "error", err)
	}

	s.markProcessed(event.ID)
	return nil
}

func (s *StripeWebhookService) handleSubscriptionDeleted(ctx context.Context, event stripe.Event) error {
	var sub stripe.Subscription
	if err := json.Unmarshal(event.Data.Raw, &sub); err != nil {
		return fmt.Errorf("unmarshal subscription: %w", err)
	}

	orgID, err := s.findOrgForStripeAccount(ctx, event.Account)
	if err != nil {
		return err
	}

	// Update subscription status to canceled
	localSub, err := s.subs.GetByStripeID(ctx, sub.ID)
	if err != nil {
		return fmt.Errorf("get subscription: %w", err)
	}
	if localSub != nil {
		now := time.Now()
		localSub.Status = "canceled"
		localSub.CanceledAt = &now
		if err := s.subs.Upsert(ctx, localSub); err != nil {
			return fmt.Errorf("update subscription: %w", err)
		}

		// Find the customer and recalculate MRR
		localCustomer, err := s.customers.GetByExternalID(ctx, orgID, "stripe", sub.Customer.ID)
		if err == nil && localCustomer != nil {
			if err := s.mrrSvc.CalculateForCustomer(ctx, localCustomer.ID); err != nil {
				slog.Error("failed to recalculate MRR after subscription deletion", "error", err)
			}
		}
	}

	s.markProcessed(event.ID)
	return nil
}

func (s *StripeWebhookService) handleInvoicePaid(ctx context.Context, event stripe.Event) error {
	var inv stripe.Invoice
	if err := json.Unmarshal(event.Data.Raw, &inv); err != nil {
		return fmt.Errorf("unmarshal invoice: %w", err)
	}

	orgID, err := s.findOrgForStripeAccount(ctx, event.Account)
	if err != nil {
		return err
	}

	if inv.Customer == nil {
		return nil
	}

	localCustomer, err := s.customers.GetByExternalID(ctx, orgID, "stripe", inv.Customer.ID)
	if err != nil || localCustomer == nil {
		return nil
	}

	paidAt := time.Unix(inv.StatusTransitions.PaidAt, 0)
	payment := &repository.StripePayment{
		OrgID:           orgID,
		CustomerID:      localCustomer.ID,
		StripePaymentID: inv.ID,
		AmountCents:     int(inv.AmountPaid),
		Currency:        string(inv.Currency),
		Status:          "succeeded",
		PaidAt:          &paidAt,
	}

	if err := s.payments.Upsert(ctx, payment); err != nil {
		return fmt.Errorf("upsert payment: %w", err)
	}

	s.markProcessed(event.ID)
	return nil
}

func (s *StripeWebhookService) handleInvoicePaymentFailed(ctx context.Context, event stripe.Event) error {
	var inv stripe.Invoice
	if err := json.Unmarshal(event.Data.Raw, &inv); err != nil {
		return fmt.Errorf("unmarshal invoice: %w", err)
	}

	orgID, err := s.findOrgForStripeAccount(ctx, event.Account)
	if err != nil {
		return err
	}

	if inv.Customer == nil {
		return nil
	}

	localCustomer, err := s.customers.GetByExternalID(ctx, orgID, "stripe", inv.Customer.ID)
	if err != nil || localCustomer == nil {
		return nil
	}

	now := time.Now()
	payment := &repository.StripePayment{
		OrgID:           orgID,
		CustomerID:      localCustomer.ID,
		StripePaymentID: inv.ID + "_failed",
		AmountCents:     int(inv.AmountDue),
		Currency:        string(inv.Currency),
		Status:          "failed",
		PaidAt:          &now,
	}

	if err := s.payments.Upsert(ctx, payment); err != nil {
		return fmt.Errorf("upsert failed payment: %w", err)
	}

	// Create customer event
	custEvent := &repository.CustomerEvent{
		OrgID:           orgID,
		CustomerID:      localCustomer.ID,
		EventType:       "payment.failed",
		Source:          "stripe",
		ExternalEventID: "webhook_" + event.ID,
		OccurredAt:      now,
		Data: map[string]any{
			"invoice_id":   inv.ID,
			"amount_cents": inv.AmountDue,
			"currency":     string(inv.Currency),
		},
	}
	if err := s.events.Upsert(ctx, custEvent); err != nil {
		slog.Error("failed to create payment failed event", "error", err)
	}

	// Track failed payment
	if err := s.paymentHealth.TrackFailedPayment(ctx, localCustomer.ID); err != nil {
		slog.Error("failed to track failed payment", "error", err)
	}

	s.markProcessed(event.ID)
	return nil
}

// findOrgForStripeAccount finds the org that has the given Stripe account connected.
func (s *StripeWebhookService) findOrgForStripeAccount(ctx context.Context, stripeAccountID string) (uuid.UUID, error) {
	conns, err := s.connRepo.ListActiveByProvider(ctx, "stripe")
	if err != nil {
		return uuid.Nil, fmt.Errorf("list connections: %w", err)
	}

	for _, conn := range conns {
		if conn.ExternalAccountID == stripeAccountID {
			return conn.OrgID, nil
		}
	}

	// If no account ID match (e.g. direct account, not Connect), 
	// try to find the single active connection
	if stripeAccountID == "" && len(conns) == 1 {
		return conns[0].OrgID, nil
	}

	return uuid.Nil, fmt.Errorf("no org found for Stripe account %s", stripeAccountID)
}
