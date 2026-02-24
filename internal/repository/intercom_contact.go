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

// IntercomContact represents an intercom_contacts row.
type IntercomContact struct {
	ID                 uuid.UUID
	OrgID              uuid.UUID
	CustomerID         *uuid.UUID
	IntercomContactID  string
	Email              string
	Name               string
	Role               string
	IntercomCompanyID  string
	Metadata           map[string]any
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

// IntercomContactRepository handles intercom_contacts database operations.
type IntercomContactRepository struct {
	pool *pgxpool.Pool
}

// NewIntercomContactRepository creates a new IntercomContactRepository.
func NewIntercomContactRepository(pool *pgxpool.Pool) *IntercomContactRepository {
	return &IntercomContactRepository{pool: pool}
}

// Upsert creates or updates an Intercom contact by (org_id, intercom_contact_id).
func (r *IntercomContactRepository) Upsert(ctx context.Context, c *IntercomContact) error {
	query := `
		INSERT INTO intercom_contacts (org_id, customer_id, intercom_contact_id, email, name, role,
			intercom_company_id, metadata)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (org_id, intercom_contact_id) DO UPDATE SET
			customer_id = COALESCE(EXCLUDED.customer_id, intercom_contacts.customer_id),
			email = EXCLUDED.email,
			name = EXCLUDED.name,
			role = EXCLUDED.role,
			intercom_company_id = EXCLUDED.intercom_company_id,
			metadata = EXCLUDED.metadata,
			updated_at = NOW()
		RETURNING id, created_at, updated_at`

	return r.pool.QueryRow(ctx, query,
		c.OrgID, c.CustomerID, c.IntercomContactID, c.Email, c.Name, c.Role,
		c.IntercomCompanyID, c.Metadata,
	).Scan(&c.ID, &c.CreatedAt, &c.UpdatedAt)
}

// GetByOrgID returns all Intercom contacts for an org.
func (r *IntercomContactRepository) GetByOrgID(ctx context.Context, orgID uuid.UUID) ([]IntercomContact, error) {
	query := `
		SELECT id, org_id, customer_id, intercom_contact_id, COALESCE(email, ''),
			COALESCE(name, ''), COALESCE(role, ''), COALESCE(intercom_company_id, ''),
			COALESCE(metadata, '{}'), created_at, updated_at
		FROM intercom_contacts
		WHERE org_id = $1
		ORDER BY updated_at DESC`

	rows, err := r.pool.Query(ctx, query, orgID)
	if err != nil {
		return nil, fmt.Errorf("list intercom contacts: %w", err)
	}
	defer rows.Close()

	var contacts []IntercomContact
	for rows.Next() {
		c := IntercomContact{}
		if err := rows.Scan(
			&c.ID, &c.OrgID, &c.CustomerID, &c.IntercomContactID, &c.Email,
			&c.Name, &c.Role, &c.IntercomCompanyID,
			&c.Metadata, &c.CreatedAt, &c.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan intercom contact: %w", err)
		}
		contacts = append(contacts, c)
	}
	return contacts, rows.Err()
}

// GetByIntercomID returns an Intercom contact by its Intercom ID within an org.
func (r *IntercomContactRepository) GetByIntercomID(ctx context.Context, orgID uuid.UUID, intercomContactID string) (*IntercomContact, error) {
	query := `
		SELECT id, org_id, customer_id, intercom_contact_id, COALESCE(email, ''),
			COALESCE(name, ''), COALESCE(role, ''), COALESCE(intercom_company_id, ''),
			COALESCE(metadata, '{}'), created_at, updated_at
		FROM intercom_contacts
		WHERE org_id = $1 AND intercom_contact_id = $2`

	c := &IntercomContact{}
	err := r.pool.QueryRow(ctx, query, orgID, intercomContactID).Scan(
		&c.ID, &c.OrgID, &c.CustomerID, &c.IntercomContactID, &c.Email,
		&c.Name, &c.Role, &c.IntercomCompanyID,
		&c.Metadata, &c.CreatedAt, &c.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get intercom contact by intercom id: %w", err)
	}
	return c, nil
}

// GetByEmail returns an Intercom contact by email within an org.
func (r *IntercomContactRepository) GetByEmail(ctx context.Context, orgID uuid.UUID, email string) (*IntercomContact, error) {
	query := `
		SELECT id, org_id, customer_id, intercom_contact_id, COALESCE(email, ''),
			COALESCE(name, ''), COALESCE(role, ''), COALESCE(intercom_company_id, ''),
			COALESCE(metadata, '{}'), created_at, updated_at
		FROM intercom_contacts
		WHERE org_id = $1 AND email = $2`

	c := &IntercomContact{}
	err := r.pool.QueryRow(ctx, query, orgID, email).Scan(
		&c.ID, &c.OrgID, &c.CustomerID, &c.IntercomContactID, &c.Email,
		&c.Name, &c.Role, &c.IntercomCompanyID,
		&c.Metadata, &c.CreatedAt, &c.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get intercom contact by email: %w", err)
	}
	return c, nil
}

// CountByOrgID returns the number of Intercom contacts for an org.
func (r *IntercomContactRepository) CountByOrgID(ctx context.Context, orgID uuid.UUID) (int, error) {
	query := `SELECT COUNT(*) FROM intercom_contacts WHERE org_id = $1`
	var count int
	err := r.pool.QueryRow(ctx, query, orgID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count intercom contacts: %w", err)
	}
	return count, nil
}

// LinkCustomer sets the customer_id for an Intercom contact.
func (r *IntercomContactRepository) LinkCustomer(ctx context.Context, id, customerID uuid.UUID) error {
	query := `UPDATE intercom_contacts SET customer_id = $2, updated_at = NOW() WHERE id = $1`
	_, err := r.pool.Exec(ctx, query, id, customerID)
	if err != nil {
		return fmt.Errorf("link intercom contact to customer: %w", err)
	}
	return nil
}
