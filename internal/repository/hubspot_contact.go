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

// HubSpotContact represents a hubspot_contacts row.
type HubSpotContact struct {
	ID               uuid.UUID
	OrgID            uuid.UUID
	CustomerID       *uuid.UUID
	HubSpotContactID string
	Email            string
	FirstName        string
	LastName         string
	HubSpotCompanyID string
	LifecycleStage   string
	LeadStatus       string
	Metadata         map[string]any
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

// HubSpotContactRepository handles hubspot_contacts database operations.
type HubSpotContactRepository struct {
	pool *pgxpool.Pool
}

// NewHubSpotContactRepository creates a new HubSpotContactRepository.
func NewHubSpotContactRepository(pool *pgxpool.Pool) *HubSpotContactRepository {
	return &HubSpotContactRepository{pool: pool}
}

// Upsert creates or updates a HubSpot contact by (org_id, hubspot_contact_id).
func (r *HubSpotContactRepository) Upsert(ctx context.Context, c *HubSpotContact) error {
	query := `
		INSERT INTO hubspot_contacts (org_id, customer_id, hubspot_contact_id, email, first_name, last_name,
			hubspot_company_id, lifecycle_stage, lead_status, metadata)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		ON CONFLICT (org_id, hubspot_contact_id) DO UPDATE SET
			customer_id = COALESCE(EXCLUDED.customer_id, hubspot_contacts.customer_id),
			email = EXCLUDED.email,
			first_name = EXCLUDED.first_name,
			last_name = EXCLUDED.last_name,
			hubspot_company_id = EXCLUDED.hubspot_company_id,
			lifecycle_stage = EXCLUDED.lifecycle_stage,
			lead_status = EXCLUDED.lead_status,
			metadata = EXCLUDED.metadata,
			updated_at = NOW()
		RETURNING id, created_at, updated_at`

	return r.pool.QueryRow(ctx, query,
		c.OrgID, c.CustomerID, c.HubSpotContactID, c.Email, c.FirstName, c.LastName,
		c.HubSpotCompanyID, c.LifecycleStage, c.LeadStatus, c.Metadata,
	).Scan(&c.ID, &c.CreatedAt, &c.UpdatedAt)
}

// GetByOrgID returns all HubSpot contacts for an org.
func (r *HubSpotContactRepository) GetByOrgID(ctx context.Context, orgID uuid.UUID) ([]HubSpotContact, error) {
	query := `
		SELECT id, org_id, customer_id, hubspot_contact_id, COALESCE(email, ''),
			COALESCE(first_name, ''), COALESCE(last_name, ''), COALESCE(hubspot_company_id, ''),
			COALESCE(lifecycle_stage, ''), COALESCE(lead_status, ''),
			COALESCE(metadata, '{}'), created_at, updated_at
		FROM hubspot_contacts
		WHERE org_id = $1
		ORDER BY updated_at DESC`

	rows, err := r.pool.Query(ctx, query, orgID)
	if err != nil {
		return nil, fmt.Errorf("list hubspot contacts: %w", err)
	}
	defer rows.Close()

	var contacts []HubSpotContact
	for rows.Next() {
		c := HubSpotContact{}
		if err := rows.Scan(
			&c.ID, &c.OrgID, &c.CustomerID, &c.HubSpotContactID, &c.Email,
			&c.FirstName, &c.LastName, &c.HubSpotCompanyID,
			&c.LifecycleStage, &c.LeadStatus,
			&c.Metadata, &c.CreatedAt, &c.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan hubspot contact: %w", err)
		}
		contacts = append(contacts, c)
	}
	return contacts, rows.Err()
}

// GetByEmail returns a HubSpot contact by email within an org.
func (r *HubSpotContactRepository) GetByEmail(ctx context.Context, orgID uuid.UUID, email string) (*HubSpotContact, error) {
	query := `
		SELECT id, org_id, customer_id, hubspot_contact_id, COALESCE(email, ''),
			COALESCE(first_name, ''), COALESCE(last_name, ''), COALESCE(hubspot_company_id, ''),
			COALESCE(lifecycle_stage, ''), COALESCE(lead_status, ''),
			COALESCE(metadata, '{}'), created_at, updated_at
		FROM hubspot_contacts
		WHERE org_id = $1 AND email = $2`

	c := &HubSpotContact{}
	err := r.pool.QueryRow(ctx, query, orgID, email).Scan(
		&c.ID, &c.OrgID, &c.CustomerID, &c.HubSpotContactID, &c.Email,
		&c.FirstName, &c.LastName, &c.HubSpotCompanyID,
		&c.LifecycleStage, &c.LeadStatus,
		&c.Metadata, &c.CreatedAt, &c.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get hubspot contact by email: %w", err)
	}
	return c, nil
}

// CountByOrgID returns the number of HubSpot contacts for an org.
func (r *HubSpotContactRepository) CountByOrgID(ctx context.Context, orgID uuid.UUID) (int, error) {
	query := `SELECT COUNT(*) FROM hubspot_contacts WHERE org_id = $1`
	var count int
	err := r.pool.QueryRow(ctx, query, orgID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count hubspot contacts: %w", err)
	}
	return count, nil
}

// LinkCustomer sets the customer_id for a HubSpot contact.
func (r *HubSpotContactRepository) LinkCustomer(ctx context.Context, id, customerID uuid.UUID) error {
	query := `UPDATE hubspot_contacts SET customer_id = $2, updated_at = NOW() WHERE id = $1`
	_, err := r.pool.Exec(ctx, query, id, customerID)
	if err != nil {
		return fmt.Errorf("link hubspot contact to customer: %w", err)
	}
	return nil
}

// GetByHubSpotID returns a HubSpot contact by its HubSpot ID within an org.
func (r *HubSpotContactRepository) GetByHubSpotID(ctx context.Context, orgID uuid.UUID, hubspotContactID string) (*HubSpotContact, error) {
	query := `
		SELECT id, org_id, customer_id, hubspot_contact_id, COALESCE(email, ''),
			COALESCE(first_name, ''), COALESCE(last_name, ''), COALESCE(hubspot_company_id, ''),
			COALESCE(lifecycle_stage, ''), COALESCE(lead_status, ''),
			COALESCE(metadata, '{}'), created_at, updated_at
		FROM hubspot_contacts
		WHERE org_id = $1 AND hubspot_contact_id = $2`

	c := &HubSpotContact{}
	err := r.pool.QueryRow(ctx, query, orgID, hubspotContactID).Scan(
		&c.ID, &c.OrgID, &c.CustomerID, &c.HubSpotContactID, &c.Email,
		&c.FirstName, &c.LastName, &c.HubSpotCompanyID,
		&c.LifecycleStage, &c.LeadStatus,
		&c.Metadata, &c.CreatedAt, &c.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get hubspot contact by hubspot id: %w", err)
	}
	return c, nil
}
