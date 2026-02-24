package billing

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stripe/stripe-go/v81"
	"github.com/stripe/stripe-go/v81/webhook"

	planmodel "github.com/onnwee/pulse-score/internal/billing"
	"github.com/onnwee/pulse-score/internal/repository"
	core "github.com/onnwee/pulse-score/internal/service"
)

// WebhookService handles PulseScore billing Stripe webhook events.
type WebhookService struct {
	webhookSecret string
	pool          *pgxpool.Pool
	orgs          *repository.OrganizationRepository
	subscriptions *repository.OrgSubscriptionRepository
	processed     *repository.BillingWebhookEventRepository
	catalog       *planmodel.Catalog
}

func NewWebhookService(
	webhookSecret string,
	pool *pgxpool.Pool,
	orgs *repository.OrganizationRepository,
	subscriptions *repository.OrgSubscriptionRepository,
	processed *repository.BillingWebhookEventRepository,
	catalog *planmodel.Catalog,
) *WebhookService {
	return &WebhookService{
		webhookSecret: strings.TrimSpace(webhookSecret),
		pool:          pool,
		orgs:          orgs,
		subscriptions: subscriptions,
		processed:     processed,
		catalog:       catalog,
	}
}

func (s *WebhookService) HandleEvent(ctx context.Context, payload []byte, sigHeader string) error {
	if s.webhookSecret == "" {
		return &core.ValidationError{Field: "billing", Message: "stripe billing webhook secret is not configured"}
	}

	event, err := webhook.ConstructEvent(payload, sigHeader, s.webhookSecret)
	if err != nil {
		return &core.ValidationError{Field: "signature", Message: "invalid webhook signature"}
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin webhook tx: %w", err)
	}
	defer tx.Rollback(ctx)

	inserted, err := s.processed.MarkProcessedTx(ctx, tx, event.ID, string(event.Type))
	if err != nil {
		return fmt.Errorf("mark webhook event processed: %w", err)
	}
	if !inserted {
		return tx.Commit(ctx)
	}

	switch event.Type {
	case "checkout.session.completed":
		err = s.handleCheckoutSessionCompleted(ctx, tx, event)
	case "customer.subscription.created", "customer.subscription.updated":
		err = s.handleSubscriptionUpsert(ctx, tx, event, false)
	case "customer.subscription.deleted":
		err = s.handleSubscriptionUpsert(ctx, tx, event, true)
	case "invoice.payment_succeeded":
		err = s.handleInvoiceStatusUpdate(ctx, tx, event, "active")
	case "invoice.payment_failed":
		err = s.handleInvoiceStatusUpdate(ctx, tx, event, "past_due")
	default:
		// Ignore unsupported events while still recording idempotency.
		err = nil
	}
	if err != nil {
		return err
	}

	return tx.Commit(ctx)
}

func (s *WebhookService) handleCheckoutSessionCompleted(ctx context.Context, tx pgx.Tx, event stripe.Event) error {
	var session stripe.CheckoutSession
	if err := json.Unmarshal(event.Data.Raw, &session); err != nil {
		return fmt.Errorf("unmarshal checkout session: %w", err)
	}

	orgID, err := resolveOrgIDFromCheckoutSession(&session)
	if err != nil {
		return nil // allow replay/reconciliation by subsequent subscription events
	}

	if session.Customer != nil && session.Customer.ID != "" {
		if err := s.orgs.UpdateStripeCustomerIDTx(ctx, tx, orgID, session.Customer.ID); err != nil {
			return fmt.Errorf("update org stripe customer id: %w", err)
		}
	}

	if tier := strings.TrimSpace(session.Metadata["tier"]); tier != "" {
		if err := s.orgs.UpdatePlanTx(ctx, tx, orgID, string(planmodel.NormalizeTier(tier))); err != nil {
			return fmt.Errorf("update org plan from checkout metadata: %w", err)
		}
	}

	return nil
}

func (s *WebhookService) handleSubscriptionUpsert(ctx context.Context, tx pgx.Tx, event stripe.Event, forceFreePlan bool) error {
	var stripeSub stripe.Subscription
	if err := json.Unmarshal(event.Data.Raw, &stripeSub); err != nil {
		return fmt.Errorf("unmarshal subscription: %w", err)
	}

	orgID, err := s.resolveOrgIDForSubscription(ctx, &stripeSub)
	if err != nil {
		return nil // defer to replay once org can be resolved
	}

	tier, cycle := s.resolveTierAndCycleForSubscription(&stripeSub)
	if forceFreePlan {
		tier = planmodel.TierFree
		cycle = planmodel.BillingCycleMonthly
	}

	localSub := &repository.OrgSubscription{
		OrgID:                orgID,
		StripeSubscriptionID: stripeSub.ID,
		PlanTier:             string(tier),
		BillingCycle:         string(cycle),
		Status:               string(stripeSub.Status),
		CancelAtPeriodEnd:    stripeSub.CancelAtPeriodEnd,
	}
	if stripeSub.Customer != nil {
		localSub.StripeCustomerID = stripeSub.Customer.ID
	}
	if stripeSub.CurrentPeriodStart > 0 {
		t := time.Unix(stripeSub.CurrentPeriodStart, 0)
		localSub.CurrentPeriodStart = &t
	}
	if stripeSub.CurrentPeriodEnd > 0 {
		t := time.Unix(stripeSub.CurrentPeriodEnd, 0)
		localSub.CurrentPeriodEnd = &t
	}

	if err := s.subscriptions.UpsertByOrgTx(ctx, tx, localSub); err != nil {
		return fmt.Errorf("upsert org subscription: %w", err)
	}

	if localSub.StripeCustomerID != "" {
		if err := s.orgs.UpdateStripeCustomerIDTx(ctx, tx, orgID, localSub.StripeCustomerID); err != nil {
			return fmt.Errorf("update org stripe customer id: %w", err)
		}
	}

	if err := s.orgs.UpdatePlanTx(ctx, tx, orgID, string(tier)); err != nil {
		return fmt.Errorf("update org plan from subscription: %w", err)
	}

	return nil
}

func (s *WebhookService) handleInvoiceStatusUpdate(ctx context.Context, tx pgx.Tx, event stripe.Event, nextStatus string) error {
	var inv stripe.Invoice
	if err := json.Unmarshal(event.Data.Raw, &inv); err != nil {
		return fmt.Errorf("unmarshal invoice: %w", err)
	}

	var sub *repository.OrgSubscription
	var err error
	if inv.Subscription != nil && inv.Subscription.ID != "" {
		sub, err = s.subscriptions.GetByStripeSubscriptionID(ctx, inv.Subscription.ID)
		if err != nil {
			return fmt.Errorf("get org subscription by stripe subscription id: %w", err)
		}
	}

	if sub == nil && inv.Customer != nil && inv.Customer.ID != "" {
		sub, err = s.subscriptions.GetByStripeCustomerID(ctx, inv.Customer.ID)
		if err != nil {
			return fmt.Errorf("get org subscription by stripe customer id: %w", err)
		}
	}

	if sub == nil {
		return nil
	}

	sub.Status = nextStatus
	if err := s.subscriptions.UpsertByOrgTx(ctx, tx, sub); err != nil {
		return fmt.Errorf("persist invoice status update: %w", err)
	}

	return nil
}

func (s *WebhookService) resolveOrgIDForSubscription(ctx context.Context, stripeSub *stripe.Subscription) (uuid.UUID, error) {
	if stripeSub.Metadata != nil {
		if rawOrgID := strings.TrimSpace(stripeSub.Metadata["org_id"]); rawOrgID != "" {
			orgID, err := uuid.Parse(rawOrgID)
			if err == nil {
				return orgID, nil
			}
		}
	}

	if stripeSub.Customer != nil && stripeSub.Customer.ID != "" {
		org, err := s.orgs.GetByStripeCustomerID(ctx, stripeSub.Customer.ID)
		if err != nil {
			return uuid.Nil, fmt.Errorf("find org by stripe customer id: %w", err)
		}
		if org != nil {
			return org.ID, nil
		}
	}

	return uuid.Nil, fmt.Errorf("could not resolve org for stripe subscription %s", stripeSub.ID)
}

func (s *WebhookService) resolveTierAndCycleForSubscription(stripeSub *stripe.Subscription) (planmodel.Tier, planmodel.BillingCycle) {
	tier := planmodel.TierFree
	cycle := planmodel.BillingCycleMonthly

	if stripeSub.Metadata != nil {
		if rawTier := strings.TrimSpace(stripeSub.Metadata["tier"]); rawTier != "" {
			tier = planmodel.NormalizeTier(rawTier)
		}
		if rawCycle := strings.TrimSpace(stripeSub.Metadata["cycle"]); strings.EqualFold(rawCycle, string(planmodel.BillingCycleAnnual)) {
			cycle = planmodel.BillingCycleAnnual
		}
	}

	if stripeSub.Items != nil {
		for _, item := range stripeSub.Items.Data {
			if item.Price == nil {
				continue
			}

			if mappedTier, mappedCycle, ok := s.catalog.ResolveTierAndCycleByPriceID(item.Price.ID); ok {
				tier = mappedTier
				cycle = mappedCycle
				break
			}

			if item.Price.Recurring != nil && item.Price.Recurring.Interval == "year" {
				cycle = planmodel.BillingCycleAnnual
			}
		}
	}

	return tier, cycle
}

func resolveOrgIDFromCheckoutSession(session *stripe.CheckoutSession) (uuid.UUID, error) {
	if session.Metadata != nil {
		if rawOrgID := strings.TrimSpace(session.Metadata["org_id"]); rawOrgID != "" {
			return uuid.Parse(rawOrgID)
		}
	}
	if strings.TrimSpace(session.ClientReferenceID) != "" {
		return uuid.Parse(session.ClientReferenceID)
	}

	return uuid.Nil, fmt.Errorf("missing org metadata")
}
