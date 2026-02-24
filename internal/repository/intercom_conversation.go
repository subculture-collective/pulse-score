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

// IntercomConversation represents an intercom_conversations row.
type IntercomConversation struct {
	ID                      uuid.UUID
	OrgID                   uuid.UUID
	CustomerID              *uuid.UUID
	IntercomConversationID  string
	IntercomContactID       string
	State                   string
	Rating                  int
	RatingRemark            string
	Open                    bool
	Read                    bool
	Priority                string
	Subject                 string
	CreatedAtRemote         *time.Time
	UpdatedAtRemote         *time.Time
	ClosedAt                *time.Time
	FirstResponseAt         *time.Time
	Metadata                map[string]any
	CreatedAt               time.Time
	UpdatedAt               time.Time
}

// IntercomConversationRepository handles intercom_conversations database operations.
type IntercomConversationRepository struct {
	pool *pgxpool.Pool
}

// NewIntercomConversationRepository creates a new IntercomConversationRepository.
func NewIntercomConversationRepository(pool *pgxpool.Pool) *IntercomConversationRepository {
	return &IntercomConversationRepository{pool: pool}
}

// Upsert creates or updates an Intercom conversation by (org_id, intercom_conversation_id).
func (r *IntercomConversationRepository) Upsert(ctx context.Context, c *IntercomConversation) error {
	query := `
		INSERT INTO intercom_conversations (org_id, customer_id, intercom_conversation_id, intercom_contact_id,
			state, rating, rating_remark, open, read, priority, subject,
			created_at_remote, updated_at_remote, closed_at, first_response_at, metadata)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16)
		ON CONFLICT (org_id, intercom_conversation_id) DO UPDATE SET
			customer_id = COALESCE(EXCLUDED.customer_id, intercom_conversations.customer_id),
			intercom_contact_id = EXCLUDED.intercom_contact_id,
			state = EXCLUDED.state,
			rating = EXCLUDED.rating,
			rating_remark = EXCLUDED.rating_remark,
			open = EXCLUDED.open,
			read = EXCLUDED.read,
			priority = EXCLUDED.priority,
			subject = EXCLUDED.subject,
			updated_at_remote = EXCLUDED.updated_at_remote,
			closed_at = EXCLUDED.closed_at,
			first_response_at = EXCLUDED.first_response_at,
			metadata = EXCLUDED.metadata,
			updated_at = NOW()
		RETURNING id, created_at, updated_at`

	return r.pool.QueryRow(ctx, query,
		c.OrgID, c.CustomerID, c.IntercomConversationID, c.IntercomContactID,
		c.State, c.Rating, c.RatingRemark, c.Open, c.Read, c.Priority, c.Subject,
		c.CreatedAtRemote, c.UpdatedAtRemote, c.ClosedAt, c.FirstResponseAt, c.Metadata,
	).Scan(&c.ID, &c.CreatedAt, &c.UpdatedAt)
}

// GetByOrgID returns all Intercom conversations for an org.
func (r *IntercomConversationRepository) GetByOrgID(ctx context.Context, orgID uuid.UUID) ([]IntercomConversation, error) {
	query := `
		SELECT id, org_id, customer_id, intercom_conversation_id, COALESCE(intercom_contact_id, ''),
			COALESCE(state, ''), COALESCE(rating, 0), COALESCE(rating_remark, ''),
			COALESCE(open, true), COALESCE(read, false), COALESCE(priority, ''), COALESCE(subject, ''),
			created_at_remote, updated_at_remote, closed_at, first_response_at,
			COALESCE(metadata, '{}'), created_at, updated_at
		FROM intercom_conversations
		WHERE org_id = $1
		ORDER BY created_at_remote DESC NULLS LAST`

	rows, err := r.pool.Query(ctx, query, orgID)
	if err != nil {
		return nil, fmt.Errorf("list intercom conversations: %w", err)
	}
	defer rows.Close()

	var convos []IntercomConversation
	for rows.Next() {
		c := IntercomConversation{}
		if err := rows.Scan(
			&c.ID, &c.OrgID, &c.CustomerID, &c.IntercomConversationID, &c.IntercomContactID,
			&c.State, &c.Rating, &c.RatingRemark,
			&c.Open, &c.Read, &c.Priority, &c.Subject,
			&c.CreatedAtRemote, &c.UpdatedAtRemote, &c.ClosedAt, &c.FirstResponseAt,
			&c.Metadata, &c.CreatedAt, &c.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan intercom conversation: %w", err)
		}
		convos = append(convos, c)
	}
	return convos, rows.Err()
}

// GetByIntercomID returns an Intercom conversation by its Intercom ID within an org.
func (r *IntercomConversationRepository) GetByIntercomID(ctx context.Context, orgID uuid.UUID, intercomConvoID string) (*IntercomConversation, error) {
	query := `
		SELECT id, org_id, customer_id, intercom_conversation_id, COALESCE(intercom_contact_id, ''),
			COALESCE(state, ''), COALESCE(rating, 0), COALESCE(rating_remark, ''),
			COALESCE(open, true), COALESCE(read, false), COALESCE(priority, ''), COALESCE(subject, ''),
			created_at_remote, updated_at_remote, closed_at, first_response_at,
			COALESCE(metadata, '{}'), created_at, updated_at
		FROM intercom_conversations
		WHERE org_id = $1 AND intercom_conversation_id = $2`

	c := &IntercomConversation{}
	err := r.pool.QueryRow(ctx, query, orgID, intercomConvoID).Scan(
		&c.ID, &c.OrgID, &c.CustomerID, &c.IntercomConversationID, &c.IntercomContactID,
		&c.State, &c.Rating, &c.RatingRemark,
		&c.Open, &c.Read, &c.Priority, &c.Subject,
		&c.CreatedAtRemote, &c.UpdatedAtRemote, &c.ClosedAt, &c.FirstResponseAt,
		&c.Metadata, &c.CreatedAt, &c.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get intercom conversation by id: %w", err)
	}
	return c, nil
}

// CountByOrgID returns the number of Intercom conversations for an org.
func (r *IntercomConversationRepository) CountByOrgID(ctx context.Context, orgID uuid.UUID) (int, error) {
	query := `SELECT COUNT(*) FROM intercom_conversations WHERE org_id = $1`
	var count int
	err := r.pool.QueryRow(ctx, query, orgID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count intercom conversations: %w", err)
	}
	return count, nil
}

// CountByCustomerAndState returns conversation counts grouped by state for a specific customer and time window.
func (r *IntercomConversationRepository) CountByCustomerAndState(ctx context.Context, customerID uuid.UUID, since time.Time) (open int, closed int, err error) {
	query := `
		SELECT
			COALESCE(SUM(CASE WHEN state = 'open' OR open = true THEN 1 ELSE 0 END), 0) as open_count,
			COALESCE(SUM(CASE WHEN state = 'closed' THEN 1 ELSE 0 END), 0) as closed_count
		FROM intercom_conversations
		WHERE customer_id = $1 AND created_at_remote >= $2`

	err = r.pool.QueryRow(ctx, query, customerID, since).Scan(&open, &closed)
	if err != nil {
		return 0, 0, fmt.Errorf("count conversations by state: %w", err)
	}
	return open, closed, nil
}

// AvgResolutionHours returns the average time to close conversations for a customer within a time window.
func (r *IntercomConversationRepository) AvgResolutionHours(ctx context.Context, customerID uuid.UUID, since time.Time) (float64, error) {
	query := `
		SELECT COALESCE(AVG(EXTRACT(EPOCH FROM (closed_at - created_at_remote)) / 3600), 0)
		FROM intercom_conversations
		WHERE customer_id = $1 AND closed_at IS NOT NULL AND created_at_remote >= $2`

	var avg float64
	err := r.pool.QueryRow(ctx, query, customerID, since).Scan(&avg)
	if err != nil {
		return 0, fmt.Errorf("avg resolution hours: %w", err)
	}
	return avg, nil
}

// CountOpenByCustomer returns all open conversation counts per customer for an org.
func (r *IntercomConversationRepository) CountOpenByCustomer(ctx context.Context, orgID uuid.UUID) (map[uuid.UUID]int, error) {
	query := `
		SELECT customer_id, COUNT(*)
		FROM intercom_conversations
		WHERE org_id = $1 AND customer_id IS NOT NULL AND (state = 'open' OR open = true)
		GROUP BY customer_id`

	rows, err := r.pool.Query(ctx, query, orgID)
	if err != nil {
		return nil, fmt.Errorf("count open conversations by customer: %w", err)
	}
	defer rows.Close()

	counts := make(map[uuid.UUID]int)
	for rows.Next() {
		var customerID uuid.UUID
		var count int
		if err := rows.Scan(&customerID, &count); err != nil {
			return nil, fmt.Errorf("scan open conversation count: %w", err)
		}
		counts[customerID] = count
	}
	return counts, rows.Err()
}

// LinkCustomer sets the customer_id for an Intercom conversation.
func (r *IntercomConversationRepository) LinkCustomer(ctx context.Context, id, customerID uuid.UUID) error {
	query := `UPDATE intercom_conversations SET customer_id = $2, updated_at = NOW() WHERE id = $1`
	_, err := r.pool.Exec(ctx, query, id, customerID)
	if err != nil {
		return fmt.Errorf("link intercom conversation to customer: %w", err)
	}
	return nil
}
