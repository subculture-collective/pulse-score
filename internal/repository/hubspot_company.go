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

// HubSpotCompany represents a hubspot_companies row.
type HubSpotCompany struct {
	ID                  uuid.UUID
	OrgID               uuid.UUID
	HubSpotCompanyID    string
	Name                string
	Domain              string
	Industry            string
	NumberOfEmployees   int
	AnnualRevenueCents  int64
	Metadata            map[string]any
	CreatedAt           time.Time
	UpdatedAt           time.Time
}

// HubSpotCompanyRepository handles hubspot_companies database operations.
type HubSpotCompanyRepository struct {
	pool *pgxpool.Pool
}

// NewHubSpotCompanyRepository creates a new HubSpotCompanyRepository.
func NewHubSpotCompanyRepository(pool *pgxpool.Pool) *HubSpotCompanyRepository {
	return &HubSpotCompanyRepository{pool: pool}
}

// Upsert creates or updates a HubSpot company by (org_id, hubspot_company_id).
func (r *HubSpotCompanyRepository) Upsert(ctx context.Context, c *HubSpotCompany) error {
	query := `
		INSERT INTO hubspot_companies (org_id, hubspot_company_id, name, domain, industry,
			number_of_employees, annual_revenue_cents, metadata)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (org_id, hubspot_company_id) DO UPDATE SET
			name = EXCLUDED.name,
			domain = EXCLUDED.domain,
			industry = EXCLUDED.industry,
			number_of_employees = EXCLUDED.number_of_employees,
			annual_revenue_cents = EXCLUDED.annual_revenue_cents,
			metadata = EXCLUDED.metadata,
			updated_at = NOW()
		RETURNING id, created_at, updated_at`

	return r.pool.QueryRow(ctx, query,
		c.OrgID, c.HubSpotCompanyID, c.Name, c.Domain, c.Industry,
		c.NumberOfEmployees, c.AnnualRevenueCents, c.Metadata,
	).Scan(&c.ID, &c.CreatedAt, &c.UpdatedAt)
}

// GetByOrgID returns all HubSpot companies for an org.
func (r *HubSpotCompanyRepository) GetByOrgID(ctx context.Context, orgID uuid.UUID) ([]HubSpotCompany, error) {
	query := `
		SELECT id, org_id, hubspot_company_id, COALESCE(name, ''), COALESCE(domain, ''),
			COALESCE(industry, ''), COALESCE(number_of_employees, 0), annual_revenue_cents,
			COALESCE(metadata, '{}'), created_at, updated_at
		FROM hubspot_companies
		WHERE org_id = $1
		ORDER BY name`

	rows, err := r.pool.Query(ctx, query, orgID)
	if err != nil {
		return nil, fmt.Errorf("list hubspot companies: %w", err)
	}
	defer rows.Close()

	var companies []HubSpotCompany
	for rows.Next() {
		c := HubSpotCompany{}
		if err := rows.Scan(
			&c.ID, &c.OrgID, &c.HubSpotCompanyID, &c.Name, &c.Domain,
			&c.Industry, &c.NumberOfEmployees, &c.AnnualRevenueCents,
			&c.Metadata, &c.CreatedAt, &c.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan hubspot company: %w", err)
		}
		companies = append(companies, c)
	}
	return companies, rows.Err()
}

// GetByHubSpotID returns a HubSpot company by its HubSpot ID within an org.
func (r *HubSpotCompanyRepository) GetByHubSpotID(ctx context.Context, orgID uuid.UUID, hubspotCompanyID string) (*HubSpotCompany, error) {
	query := `
		SELECT id, org_id, hubspot_company_id, COALESCE(name, ''), COALESCE(domain, ''),
			COALESCE(industry, ''), COALESCE(number_of_employees, 0), annual_revenue_cents,
			COALESCE(metadata, '{}'), created_at, updated_at
		FROM hubspot_companies
		WHERE org_id = $1 AND hubspot_company_id = $2`

	c := &HubSpotCompany{}
	err := r.pool.QueryRow(ctx, query, orgID, hubspotCompanyID).Scan(
		&c.ID, &c.OrgID, &c.HubSpotCompanyID, &c.Name, &c.Domain,
		&c.Industry, &c.NumberOfEmployees, &c.AnnualRevenueCents,
		&c.Metadata, &c.CreatedAt, &c.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get hubspot company by hubspot id: %w", err)
	}
	return c, nil
}
