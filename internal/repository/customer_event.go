package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// CustomerEvent represents a customer_events row.
type CustomerEvent struct {
	ID              uuid.UUID
	OrgID           uuid.UUID
	CustomerID      uuid.UUID
	EventType       string
	Source          string
	ExternalEventID string
	OccurredAt      time.Time
	Data            map[string]any
	CreatedAt       time.Time
}

// CustomerEventRepository handles customer_events database operations.
type CustomerEventRepository struct {
	pool *pgxpool.Pool
}

// NewCustomerEventRepository creates a new CustomerEventRepository.
func NewCustomerEventRepository(pool *pgxpool.Pool) *CustomerEventRepository {
	return &CustomerEventRepository{pool: pool}
}

// Upsert creates a customer event (idempotent by org_id, source, external_event_id).
func (r *CustomerEventRepository) Upsert(ctx context.Context, e *CustomerEvent) error {
	query := `
		INSERT INTO customer_events (org_id, customer_id, event_type, source, external_event_id, occurred_at, data)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (org_id, source, external_event_id) DO NOTHING
		RETURNING id, created_at`

	err := r.pool.QueryRow(ctx, query,
		e.OrgID, e.CustomerID, e.EventType, e.Source, e.ExternalEventID, e.OccurredAt, e.Data,
	).Scan(&e.ID, &e.CreatedAt)
	// ON CONFLICT DO NOTHING returns no rows — that's fine
	if err != nil && err.Error() == "no rows in result set" {
		return nil
	}
	return err
}

// ListByCustomer returns events for a customer ordered by occurred_at desc.
func (r *CustomerEventRepository) ListByCustomer(ctx context.Context, customerID uuid.UUID, limit int) ([]*CustomerEvent, error) {
	query := `
		SELECT id, org_id, customer_id, event_type, source, COALESCE(external_event_id, ''),
			occurred_at, COALESCE(data, '{}'), created_at
		FROM customer_events
		WHERE customer_id = $1
		ORDER BY occurred_at DESC
		LIMIT $2`

	rows, err := r.pool.Query(ctx, query, customerID, limit)
	if err != nil {
		return nil, fmt.Errorf("list customer events: %w", err)
	}
	defer rows.Close()

	var events []*CustomerEvent
	for rows.Next() {
		e := &CustomerEvent{}
		if err := rows.Scan(
			&e.ID, &e.OrgID, &e.CustomerID, &e.EventType, &e.Source, &e.ExternalEventID,
			&e.OccurredAt, &e.Data, &e.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan customer event: %w", err)
		}
		events = append(events, e)
	}
	return events, rows.Err()
}
