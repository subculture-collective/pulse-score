package repository

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// BillingWebhookEventRepository tracks processed billing webhook events for idempotency.
type BillingWebhookEventRepository struct {
	pool *pgxpool.Pool
}

// NewBillingWebhookEventRepository creates a new BillingWebhookEventRepository.
func NewBillingWebhookEventRepository(pool *pgxpool.Pool) *BillingWebhookEventRepository {
	return &BillingWebhookEventRepository{pool: pool}
}

// MarkProcessed inserts an event id and returns true if inserted, false if already processed.
func (r *BillingWebhookEventRepository) MarkProcessed(ctx context.Context, eventID, eventType string) (bool, error) {
	cmdTag, err := r.pool.Exec(ctx, `
		INSERT INTO billing_webhook_events (event_id, event_type)
		VALUES ($1, $2)
		ON CONFLICT (event_id) DO NOTHING`, eventID, eventType)
	if err != nil {
		return false, fmt.Errorf("insert billing webhook event: %w", err)
	}
	return cmdTag.RowsAffected() == 1, nil
}

// MarkProcessedTx inserts an event id in an existing transaction.
func (r *BillingWebhookEventRepository) MarkProcessedTx(ctx context.Context, tx pgx.Tx, eventID, eventType string) (bool, error) {
	cmdTag, err := tx.Exec(ctx, `
		INSERT INTO billing_webhook_events (event_id, event_type)
		VALUES ($1, $2)
		ON CONFLICT (event_id) DO NOTHING`, eventID, eventType)
	if err != nil {
		return false, fmt.Errorf("insert billing webhook event in tx: %w", err)
	}
	return cmdTag.RowsAffected() == 1, nil
}
