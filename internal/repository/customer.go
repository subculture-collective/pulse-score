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

// Customer represents a customers row.
type Customer struct {
	ID          uuid.UUID
	OrgID       uuid.UUID
	ExternalID  string
	Source      string
	Email       string
	Name        string
	CompanyName string
	MRRCents    int
	Currency    string
	FirstSeenAt *time.Time
	LastSeenAt  *time.Time
	Metadata    map[string]any
	CreatedAt   time.Time
	UpdatedAt   time.Time
	DeletedAt   *time.Time
}

// CustomerRepository handles customer database operations.
type CustomerRepository struct {
	pool *pgxpool.Pool
}

// NewCustomerRepository creates a new CustomerRepository.
func NewCustomerRepository(pool *pgxpool.Pool) *CustomerRepository {
	return &CustomerRepository{pool: pool}
}

// UpsertByExternal creates or updates a customer by (org_id, source, external_id).
func (r *CustomerRepository) UpsertByExternal(ctx context.Context, c *Customer) error {
	query := `
		INSERT INTO customers (org_id, external_id, source, email, name, company_name, currency, first_seen_at, last_seen_at, metadata)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		ON CONFLICT (org_id, source, external_id) DO UPDATE SET
			email = EXCLUDED.email,
			name = EXCLUDED.name,
			company_name = EXCLUDED.company_name,
			currency = EXCLUDED.currency,
			last_seen_at = EXCLUDED.last_seen_at,
			metadata = EXCLUDED.metadata,
			deleted_at = NULL
		RETURNING id, mrr_cents, created_at, updated_at`

	return r.pool.QueryRow(ctx, query,
		c.OrgID, c.ExternalID, c.Source, c.Email, c.Name, c.CompanyName,
		c.Currency, c.FirstSeenAt, c.LastSeenAt, c.Metadata,
	).Scan(&c.ID, &c.MRRCents, &c.CreatedAt, &c.UpdatedAt)
}

// GetByExternalID retrieves a customer by (org_id, source, external_id).
func (r *CustomerRepository) GetByExternalID(ctx context.Context, orgID uuid.UUID, source, externalID string) (*Customer, error) {
	query := `
		SELECT id, org_id, external_id, source, COALESCE(email, ''), COALESCE(name, ''),
			COALESCE(company_name, ''), mrr_cents, currency,
			first_seen_at, last_seen_at, COALESCE(metadata, '{}'), created_at, updated_at, deleted_at
		FROM customers
		WHERE org_id = $1 AND source = $2 AND external_id = $3 AND deleted_at IS NULL`

	c := &Customer{}
	err := r.pool.QueryRow(ctx, query, orgID, source, externalID).Scan(
		&c.ID, &c.OrgID, &c.ExternalID, &c.Source, &c.Email, &c.Name,
		&c.CompanyName, &c.MRRCents, &c.Currency,
		&c.FirstSeenAt, &c.LastSeenAt, &c.Metadata, &c.CreatedAt, &c.UpdatedAt, &c.DeletedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get customer by external id: %w", err)
	}
	return c, nil
}

// GetByID retrieves a customer by ID.
func (r *CustomerRepository) GetByID(ctx context.Context, id uuid.UUID) (*Customer, error) {
	query := `
		SELECT id, org_id, external_id, source, COALESCE(email, ''), COALESCE(name, ''),
			COALESCE(company_name, ''), mrr_cents, currency,
			first_seen_at, last_seen_at, COALESCE(metadata, '{}'), created_at, updated_at, deleted_at
		FROM customers
		WHERE id = $1 AND deleted_at IS NULL`

	c := &Customer{}
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&c.ID, &c.OrgID, &c.ExternalID, &c.Source, &c.Email, &c.Name,
		&c.CompanyName, &c.MRRCents, &c.Currency,
		&c.FirstSeenAt, &c.LastSeenAt, &c.Metadata, &c.CreatedAt, &c.UpdatedAt, &c.DeletedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get customer by id: %w", err)
	}
	return c, nil
}

// SoftDelete marks a customer as deleted.
func (r *CustomerRepository) SoftDelete(ctx context.Context, orgID uuid.UUID, source, externalID string) error {
	query := `UPDATE customers SET deleted_at = NOW() WHERE org_id = $1 AND source = $2 AND external_id = $3 AND deleted_at IS NULL`
	_, err := r.pool.Exec(ctx, query, orgID, source, externalID)
	if err != nil {
		return fmt.Errorf("soft delete customer: %w", err)
	}
	return nil
}

// UpdateMRR updates the mrr_cents for a customer.
func (r *CustomerRepository) UpdateMRR(ctx context.Context, customerID uuid.UUID, mrrCents int) error {
	query := `UPDATE customers SET mrr_cents = $2 WHERE id = $1 AND deleted_at IS NULL`
	_, err := r.pool.Exec(ctx, query, customerID, mrrCents)
	if err != nil {
		return fmt.Errorf("update mrr: %w", err)
	}
	return nil
}

// ListByOrg retrieves all non-deleted customers for an org.
func (r *CustomerRepository) ListByOrg(ctx context.Context, orgID uuid.UUID) ([]*Customer, error) {
	query := `
		SELECT id, org_id, external_id, source, COALESCE(email, ''), COALESCE(name, ''),
			COALESCE(company_name, ''), mrr_cents, currency,
			first_seen_at, last_seen_at, COALESCE(metadata, '{}'), created_at, updated_at, deleted_at
		FROM customers
		WHERE org_id = $1 AND deleted_at IS NULL
		ORDER BY name`

	rows, err := r.pool.Query(ctx, query, orgID)
	if err != nil {
		return nil, fmt.Errorf("list customers: %w", err)
	}
	defer rows.Close()

	var customers []*Customer
	for rows.Next() {
		c := &Customer{}
		if err := rows.Scan(
			&c.ID, &c.OrgID, &c.ExternalID, &c.Source, &c.Email, &c.Name,
			&c.CompanyName, &c.MRRCents, &c.Currency,
			&c.FirstSeenAt, &c.LastSeenAt, &c.Metadata, &c.CreatedAt, &c.UpdatedAt, &c.DeletedAt,
		); err != nil {
			return nil, fmt.Errorf("scan customer: %w", err)
		}
		customers = append(customers, c)
	}
	return customers, rows.Err()
}

// CountByOrg returns the number of active customers for an org.
func (r *CustomerRepository) CountByOrg(ctx context.Context, orgID uuid.UUID) (int, error) {
	query := `SELECT COUNT(*) FROM customers WHERE org_id = $1 AND deleted_at IS NULL`
	var count int
	err := r.pool.QueryRow(ctx, query, orgID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count customers: %w", err)
	}
	return count, nil
}
