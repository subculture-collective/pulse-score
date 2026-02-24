package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// HubSpotDeal represents a hubspot_deals row.
type HubSpotDeal struct {
	ID               uuid.UUID
	OrgID            uuid.UUID
	CustomerID       *uuid.UUID
	HubSpotDealID    string
	HubSpotContactID string
	DealName         string
	Stage            string
	AmountCents      int64
	Currency         string
	CloseDate        *time.Time
	Pipeline         string
	Metadata         map[string]any
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

// HubSpotDealRepository handles hubspot_deals database operations.
type HubSpotDealRepository struct {
	pool *pgxpool.Pool
}

// NewHubSpotDealRepository creates a new HubSpotDealRepository.
func NewHubSpotDealRepository(pool *pgxpool.Pool) *HubSpotDealRepository {
	return &HubSpotDealRepository{pool: pool}
}

// Upsert creates or updates a HubSpot deal by (org_id, hubspot_deal_id).
func (r *HubSpotDealRepository) Upsert(ctx context.Context, d *HubSpotDeal) error {
	query := `
		INSERT INTO hubspot_deals (org_id, customer_id, hubspot_deal_id, hubspot_contact_id,
			deal_name, stage, amount_cents, currency, close_date, pipeline, metadata)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		ON CONFLICT (org_id, hubspot_deal_id) DO UPDATE SET
			customer_id = COALESCE(EXCLUDED.customer_id, hubspot_deals.customer_id),
			hubspot_contact_id = EXCLUDED.hubspot_contact_id,
			deal_name = EXCLUDED.deal_name,
			stage = EXCLUDED.stage,
			amount_cents = EXCLUDED.amount_cents,
			currency = EXCLUDED.currency,
			close_date = EXCLUDED.close_date,
			pipeline = EXCLUDED.pipeline,
			metadata = EXCLUDED.metadata,
			updated_at = NOW()
		RETURNING id, created_at, updated_at`

	return r.pool.QueryRow(ctx, query,
		d.OrgID, d.CustomerID, d.HubSpotDealID, d.HubSpotContactID,
		d.DealName, d.Stage, d.AmountCents, d.Currency, d.CloseDate, d.Pipeline, d.Metadata,
	).Scan(&d.ID, &d.CreatedAt, &d.UpdatedAt)
}

// GetByOrgID returns all HubSpot deals for an org.
func (r *HubSpotDealRepository) GetByOrgID(ctx context.Context, orgID uuid.UUID) ([]HubSpotDeal, error) {
	query := `
		SELECT id, org_id, customer_id, hubspot_deal_id, COALESCE(hubspot_contact_id, ''),
			COALESCE(deal_name, ''), COALESCE(stage, ''), amount_cents,
			COALESCE(currency, 'USD'), close_date, COALESCE(pipeline, ''),
			COALESCE(metadata, '{}'), created_at, updated_at
		FROM hubspot_deals
		WHERE org_id = $1
		ORDER BY updated_at DESC`

	rows, err := r.pool.Query(ctx, query, orgID)
	if err != nil {
		return nil, fmt.Errorf("list hubspot deals: %w", err)
	}
	defer rows.Close()

	var deals []HubSpotDeal
	for rows.Next() {
		d := HubSpotDeal{}
		if err := rows.Scan(
			&d.ID, &d.OrgID, &d.CustomerID, &d.HubSpotDealID, &d.HubSpotContactID,
			&d.DealName, &d.Stage, &d.AmountCents,
			&d.Currency, &d.CloseDate, &d.Pipeline,
			&d.Metadata, &d.CreatedAt, &d.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan hubspot deal: %w", err)
		}
		deals = append(deals, d)
	}
	return deals, rows.Err()
}

// GetByContactID returns all deals for a specific HubSpot contact.
func (r *HubSpotDealRepository) GetByContactID(ctx context.Context, orgID uuid.UUID, hubspotContactID string) ([]HubSpotDeal, error) {
	query := `
		SELECT id, org_id, customer_id, hubspot_deal_id, COALESCE(hubspot_contact_id, ''),
			COALESCE(deal_name, ''), COALESCE(stage, ''), amount_cents,
			COALESCE(currency, 'USD'), close_date, COALESCE(pipeline, ''),
			COALESCE(metadata, '{}'), created_at, updated_at
		FROM hubspot_deals
		WHERE org_id = $1 AND hubspot_contact_id = $2
		ORDER BY updated_at DESC`

	rows, err := r.pool.Query(ctx, query, orgID, hubspotContactID)
	if err != nil {
		return nil, fmt.Errorf("list hubspot deals by contact: %w", err)
	}
	defer rows.Close()

	var deals []HubSpotDeal
	for rows.Next() {
		d := HubSpotDeal{}
		if err := rows.Scan(
			&d.ID, &d.OrgID, &d.CustomerID, &d.HubSpotDealID, &d.HubSpotContactID,
			&d.DealName, &d.Stage, &d.AmountCents,
			&d.Currency, &d.CloseDate, &d.Pipeline,
			&d.Metadata, &d.CreatedAt, &d.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan hubspot deal: %w", err)
		}
		deals = append(deals, d)
	}
	return deals, rows.Err()
}

// CountByOrgID returns the number of HubSpot deals for an org.
func (r *HubSpotDealRepository) CountByOrgID(ctx context.Context, orgID uuid.UUID) (int, error) {
	query := `SELECT COUNT(*) FROM hubspot_deals WHERE org_id = $1`
	var count int
	err := r.pool.QueryRow(ctx, query, orgID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count hubspot deals: %w", err)
	}
	return count, nil
}
