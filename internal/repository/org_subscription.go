package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// OrgSubscription represents an org_subscriptions row.
type OrgSubscription struct {
	ID                   uuid.UUID
	OrgID                uuid.UUID
	StripeSubscriptionID string
	StripeCustomerID     string
	PlanTier             string
	BillingCycle         string
	Status               string
	CurrentPeriodStart   *time.Time
	CurrentPeriodEnd     *time.Time
	CancelAtPeriodEnd    bool
	CreatedAt            time.Time
	UpdatedAt            time.Time
}

// OrgSubscriptionRepository handles org subscription database operations.
type OrgSubscriptionRepository struct {
	pool *pgxpool.Pool
}

// NewOrgSubscriptionRepository creates a new OrgSubscriptionRepository.
func NewOrgSubscriptionRepository(pool *pgxpool.Pool) *OrgSubscriptionRepository {
	return &OrgSubscriptionRepository{pool: pool}
}

const upsertOrgSubscriptionQuery = `
	INSERT INTO org_subscriptions (
		org_id, stripe_subscription_id, stripe_customer_id, plan_tier, billing_cycle,
		status, current_period_start, current_period_end, cancel_at_period_end
	)
	VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	ON CONFLICT (org_id) DO UPDATE SET
		stripe_subscription_id = EXCLUDED.stripe_subscription_id,
		stripe_customer_id = EXCLUDED.stripe_customer_id,
		plan_tier = EXCLUDED.plan_tier,
		billing_cycle = EXCLUDED.billing_cycle,
		status = EXCLUDED.status,
		current_period_start = EXCLUDED.current_period_start,
		current_period_end = EXCLUDED.current_period_end,
		cancel_at_period_end = EXCLUDED.cancel_at_period_end
	RETURNING id, created_at, updated_at`

// UpsertByOrg creates or updates org subscription state by org ID.
func (r *OrgSubscriptionRepository) UpsertByOrg(ctx context.Context, sub *OrgSubscription) error {
	return r.pool.QueryRow(ctx, upsertOrgSubscriptionQuery,
		sub.OrgID,
		emptyToNil(sub.StripeSubscriptionID),
		emptyToNil(sub.StripeCustomerID),
		sub.PlanTier,
		sub.BillingCycle,
		sub.Status,
		sub.CurrentPeriodStart,
		sub.CurrentPeriodEnd,
		sub.CancelAtPeriodEnd,
	).Scan(&sub.ID, &sub.CreatedAt, &sub.UpdatedAt)
}

// UpsertByOrgTx creates or updates org subscription state by org ID in an existing transaction.
func (r *OrgSubscriptionRepository) UpsertByOrgTx(ctx context.Context, tx pgx.Tx, sub *OrgSubscription) error {
	return tx.QueryRow(ctx, upsertOrgSubscriptionQuery,
		sub.OrgID,
		emptyToNil(sub.StripeSubscriptionID),
		emptyToNil(sub.StripeCustomerID),
		sub.PlanTier,
		sub.BillingCycle,
		sub.Status,
		sub.CurrentPeriodStart,
		sub.CurrentPeriodEnd,
		sub.CancelAtPeriodEnd,
	).Scan(&sub.ID, &sub.CreatedAt, &sub.UpdatedAt)
}

// GetByOrg retrieves org subscription state by organization ID.
func (r *OrgSubscriptionRepository) GetByOrg(ctx context.Context, orgID uuid.UUID) (*OrgSubscription, error) {
	query := `
		SELECT id, org_id, COALESCE(stripe_subscription_id, ''), COALESCE(stripe_customer_id, ''),
			plan_tier, billing_cycle, status, current_period_start, current_period_end,
			cancel_at_period_end, created_at, updated_at
		FROM org_subscriptions
		WHERE org_id = $1`

	sub := &OrgSubscription{}
	err := r.pool.QueryRow(ctx, query, orgID).Scan(
		&sub.ID,
		&sub.OrgID,
		&sub.StripeSubscriptionID,
		&sub.StripeCustomerID,
		&sub.PlanTier,
		&sub.BillingCycle,
		&sub.Status,
		&sub.CurrentPeriodStart,
		&sub.CurrentPeriodEnd,
		&sub.CancelAtPeriodEnd,
		&sub.CreatedAt,
		&sub.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get org subscription by org: %w", err)
	}

	return sub, nil
}

// GetByStripeSubscriptionID retrieves org subscription state by Stripe subscription ID.
func (r *OrgSubscriptionRepository) GetByStripeSubscriptionID(ctx context.Context, stripeSubscriptionID string) (*OrgSubscription, error) {
	query := `
		SELECT id, org_id, COALESCE(stripe_subscription_id, ''), COALESCE(stripe_customer_id, ''),
			plan_tier, billing_cycle, status, current_period_start, current_period_end,
			cancel_at_period_end, created_at, updated_at
		FROM org_subscriptions
		WHERE stripe_subscription_id = $1`

	sub := &OrgSubscription{}
	err := r.pool.QueryRow(ctx, query, stripeSubscriptionID).Scan(
		&sub.ID,
		&sub.OrgID,
		&sub.StripeSubscriptionID,
		&sub.StripeCustomerID,
		&sub.PlanTier,
		&sub.BillingCycle,
		&sub.Status,
		&sub.CurrentPeriodStart,
		&sub.CurrentPeriodEnd,
		&sub.CancelAtPeriodEnd,
		&sub.CreatedAt,
		&sub.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get org subscription by stripe subscription id: %w", err)
	}

	return sub, nil
}

// GetByStripeCustomerID retrieves org subscription state by Stripe customer ID.
func (r *OrgSubscriptionRepository) GetByStripeCustomerID(ctx context.Context, stripeCustomerID string) (*OrgSubscription, error) {
	query := `
		SELECT id, org_id, COALESCE(stripe_subscription_id, ''), COALESCE(stripe_customer_id, ''),
			plan_tier, billing_cycle, status, current_period_start, current_period_end,
			cancel_at_period_end, created_at, updated_at
		FROM org_subscriptions
		WHERE stripe_customer_id = $1`

	sub := &OrgSubscription{}
	err := r.pool.QueryRow(ctx, query, stripeCustomerID).Scan(
		&sub.ID,
		&sub.OrgID,
		&sub.StripeSubscriptionID,
		&sub.StripeCustomerID,
		&sub.PlanTier,
		&sub.BillingCycle,
		&sub.Status,
		&sub.CurrentPeriodStart,
		&sub.CurrentPeriodEnd,
		&sub.CancelAtPeriodEnd,
		&sub.CreatedAt,
		&sub.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get org subscription by stripe customer id: %w", err)
	}

	return sub, nil
}

func emptyToNil(s string) any {
	if s == "" {
		return nil
	}
	return s
}
