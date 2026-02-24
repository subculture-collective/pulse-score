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

// IntegrationConnection represents an integration_connections row.
type IntegrationConnection struct {
	ID                    uuid.UUID
	OrgID                 uuid.UUID
	Provider              string
	Status                string
	AccessTokenEncrypted  []byte
	RefreshTokenEncrypted []byte
	TokenExpiresAt        *time.Time
	ExternalAccountID     string
	Scopes                []string
	Metadata              map[string]any
	LastSyncAt            *time.Time
	LastSyncError         string
	CreatedAt             time.Time
	UpdatedAt             time.Time
}

// IntegrationConnectionRepository handles integration_connections database operations.
type IntegrationConnectionRepository struct {
	pool *pgxpool.Pool
}

// NewIntegrationConnectionRepository creates a new IntegrationConnectionRepository.
func NewIntegrationConnectionRepository(pool *pgxpool.Pool) *IntegrationConnectionRepository {
	return &IntegrationConnectionRepository{pool: pool}
}

// Upsert creates or updates an integration connection for a given org+provider.
func (r *IntegrationConnectionRepository) Upsert(ctx context.Context, conn *IntegrationConnection) error {
	query := `
		INSERT INTO integration_connections (org_id, provider, status, access_token_encrypted, refresh_token_encrypted,
			token_expires_at, external_account_id, scopes, metadata)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		ON CONFLICT (org_id, provider) DO UPDATE SET
			status = EXCLUDED.status,
			access_token_encrypted = EXCLUDED.access_token_encrypted,
			refresh_token_encrypted = EXCLUDED.refresh_token_encrypted,
			token_expires_at = EXCLUDED.token_expires_at,
			external_account_id = EXCLUDED.external_account_id,
			scopes = EXCLUDED.scopes,
			metadata = EXCLUDED.metadata
		RETURNING id, created_at, updated_at`

	return r.pool.QueryRow(ctx, query,
		conn.OrgID, conn.Provider, conn.Status, conn.AccessTokenEncrypted, conn.RefreshTokenEncrypted,
		conn.TokenExpiresAt, conn.ExternalAccountID, conn.Scopes, conn.Metadata,
	).Scan(&conn.ID, &conn.CreatedAt, &conn.UpdatedAt)
}

// GetByOrgAndProvider retrieves a connection by org ID and provider.
func (r *IntegrationConnectionRepository) GetByOrgAndProvider(ctx context.Context, orgID uuid.UUID, provider string) (*IntegrationConnection, error) {
	query := `
		SELECT id, org_id, provider, status, access_token_encrypted, refresh_token_encrypted,
			token_expires_at, external_account_id, scopes, COALESCE(metadata, '{}'),
			last_sync_at, COALESCE(last_sync_error, ''), created_at, updated_at
		FROM integration_connections
		WHERE org_id = $1 AND provider = $2`

	c := &IntegrationConnection{}
	err := r.pool.QueryRow(ctx, query, orgID, provider).Scan(
		&c.ID, &c.OrgID, &c.Provider, &c.Status, &c.AccessTokenEncrypted, &c.RefreshTokenEncrypted,
		&c.TokenExpiresAt, &c.ExternalAccountID, &c.Scopes, &c.Metadata,
		&c.LastSyncAt, &c.LastSyncError, &c.CreatedAt, &c.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get connection by org+provider: %w", err)
	}
	return c, nil
}

// UpdateStatus updates the status of a connection.
func (r *IntegrationConnectionRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status string) error {
	query := `UPDATE integration_connections SET status = $2 WHERE id = $1`
	_, err := r.pool.Exec(ctx, query, id, status)
	if err != nil {
		return fmt.Errorf("update connection status: %w", err)
	}
	return nil
}

// UpdateSyncStatus updates the status and optionally the last sync time for a connection identified by org+provider.
func (r *IntegrationConnectionRepository) UpdateSyncStatus(ctx context.Context, orgID uuid.UUID, provider, status string, syncAt *time.Time) error {
	if syncAt != nil {
		query := `UPDATE integration_connections SET status = $3, last_sync_at = $4, last_sync_error = '' WHERE org_id = $1 AND provider = $2`
		_, err := r.pool.Exec(ctx, query, orgID, provider, status, *syncAt)
		if err != nil {
			return fmt.Errorf("update sync status: %w", err)
		}
	} else {
		query := `UPDATE integration_connections SET status = $3 WHERE org_id = $1 AND provider = $2`
		_, err := r.pool.Exec(ctx, query, orgID, provider, status)
		if err != nil {
			return fmt.Errorf("update sync status: %w", err)
		}
	}
	return nil
}

// ListActiveByProvider returns all active connections for a given provider.
func (r *IntegrationConnectionRepository) ListActiveByProvider(ctx context.Context, provider string) ([]*IntegrationConnection, error) {
	query := `
		SELECT id, org_id, provider, status, access_token_encrypted, refresh_token_encrypted,
			token_expires_at, external_account_id, scopes, COALESCE(metadata, '{}'),
			last_sync_at, COALESCE(last_sync_error, ''), created_at, updated_at
		FROM integration_connections
		WHERE provider = $1 AND status = 'active'`

	rows, err := r.pool.Query(ctx, query, provider)
	if err != nil {
		return nil, fmt.Errorf("list active connections: %w", err)
	}
	defer rows.Close()

	var conns []*IntegrationConnection
	for rows.Next() {
		c := &IntegrationConnection{}
		if err := rows.Scan(
			&c.ID, &c.OrgID, &c.Provider, &c.Status, &c.AccessTokenEncrypted, &c.RefreshTokenEncrypted,
			&c.TokenExpiresAt, &c.ExternalAccountID, &c.Scopes, &c.Metadata,
			&c.LastSyncAt, &c.LastSyncError, &c.CreatedAt, &c.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan connection: %w", err)
		}
		conns = append(conns, c)
	}
	return conns, rows.Err()
}

// Delete removes a connection.
func (r *IntegrationConnectionRepository) Delete(ctx context.Context, orgID uuid.UUID, provider string) error {
	query := `DELETE FROM integration_connections WHERE org_id = $1 AND provider = $2`
	_, err := r.pool.Exec(ctx, query, orgID, provider)
	if err != nil {
		return fmt.Errorf("delete connection: %w", err)
	}
	return nil
}

// UpdateErrorCount increments error tracking for a connection identified by org+provider.
func (r *IntegrationConnectionRepository) UpdateErrorCount(ctx context.Context, orgID uuid.UUID, provider, lastError string) error {
	query := `
		UPDATE integration_connections
		SET metadata = COALESCE(metadata, '{}'::jsonb) || jsonb_build_object(
			'error_count', (COALESCE((metadata->>'error_count')::int, 0) + 1),
			'last_error', $3
		),
		last_sync_error = $3
		WHERE org_id = $1 AND provider = $2`
	_, err := r.pool.Exec(ctx, query, orgID, provider, lastError)
	if err != nil {
		return fmt.Errorf("update error count: %w", err)
	}
	return nil
}

// ListByOrg returns all connections for an org.
func (r *IntegrationConnectionRepository) ListByOrg(ctx context.Context, orgID uuid.UUID) ([]*IntegrationConnection, error) {
	query := `
		SELECT id, org_id, provider, status, access_token_encrypted, refresh_token_encrypted,
			token_expires_at, external_account_id, scopes, COALESCE(metadata, '{}'),
			last_sync_at, COALESCE(last_sync_error, ''), created_at, updated_at
		FROM integration_connections
		WHERE org_id = $1
		ORDER BY created_at`

	rows, err := r.pool.Query(ctx, query, orgID)
	if err != nil {
		return nil, fmt.Errorf("list connections by org: %w", err)
	}
	defer rows.Close()

	var conns []*IntegrationConnection
	for rows.Next() {
		c := &IntegrationConnection{}
		if err := rows.Scan(
			&c.ID, &c.OrgID, &c.Provider, &c.Status, &c.AccessTokenEncrypted, &c.RefreshTokenEncrypted,
			&c.TokenExpiresAt, &c.ExternalAccountID, &c.Scopes, &c.Metadata,
			&c.LastSyncAt, &c.LastSyncError, &c.CreatedAt, &c.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan connection: %w", err)
		}
		conns = append(conns, c)
	}
	return conns, rows.Err()
}

// GetCustomerCountBySource returns the number of customers from a specific source.
func (r *IntegrationConnectionRepository) GetCustomerCountBySource(ctx context.Context, orgID uuid.UUID, source string) (int, error) {
	query := `SELECT COUNT(*) FROM customers WHERE org_id = $1 AND source = $2 AND deleted_at IS NULL`
	var count int
	err := r.pool.QueryRow(ctx, query, orgID, source).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count customers by source: %w", err)
	}
	return count, nil
}

// GetByProviderAndExternalID finds a connection by provider and external account ID.
func (r *IntegrationConnectionRepository) GetByProviderAndExternalID(ctx context.Context, provider, externalAccountID string) (*IntegrationConnection, error) {
	query := `
		SELECT id, org_id, provider, status, access_token_encrypted, refresh_token_encrypted,
			token_expires_at, external_account_id, scopes, COALESCE(metadata, '{}'),
			last_sync_at, COALESCE(last_sync_error, ''), created_at, updated_at
		FROM integration_connections
		WHERE provider = $1 AND external_account_id = $2`

	conn := &IntegrationConnection{}
	err := r.pool.QueryRow(ctx, query, provider, externalAccountID).Scan(
		&conn.ID, &conn.OrgID, &conn.Provider, &conn.Status,
		&conn.AccessTokenEncrypted, &conn.RefreshTokenEncrypted,
		&conn.TokenExpiresAt, &conn.ExternalAccountID, &conn.Scopes, &conn.Metadata,
		&conn.LastSyncAt, &conn.LastSyncError, &conn.CreatedAt, &conn.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get by provider and external id: %w", err)
	}
	return conn, nil
}
