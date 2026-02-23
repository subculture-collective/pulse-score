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

// Invitation represents an invitations row.
type Invitation struct {
	ID        uuid.UUID
	OrgID     uuid.UUID
	Email     string
	Role      string
	Token     string
	Status    string
	InvitedBy uuid.UUID
	ExpiresAt time.Time
	CreatedAt time.Time
}

// InvitationRepository handles invitation database operations.
type InvitationRepository struct {
	pool *pgxpool.Pool
}

// NewInvitationRepository creates a new InvitationRepository.
func NewInvitationRepository(pool *pgxpool.Pool) *InvitationRepository {
	return &InvitationRepository{pool: pool}
}

// Create inserts a new invitation.
func (r *InvitationRepository) Create(ctx context.Context, inv *Invitation) error {
	query := `
		INSERT INTO invitations (org_id, email, role, token, invited_by, expires_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, status, created_at`

	return r.pool.QueryRow(ctx, query,
		inv.OrgID, inv.Email, inv.Role, inv.Token, inv.InvitedBy, inv.ExpiresAt,
	).Scan(&inv.ID, &inv.Status, &inv.CreatedAt)
}

// GetByToken retrieves an invitation by its token.
func (r *InvitationRepository) GetByToken(ctx context.Context, token string) (*Invitation, error) {
	query := `
		SELECT id, org_id, email, role, token, status, invited_by, expires_at, created_at
		FROM invitations
		WHERE token = $1`

	inv := &Invitation{}
	err := r.pool.QueryRow(ctx, query, token).Scan(
		&inv.ID, &inv.OrgID, &inv.Email, &inv.Role, &inv.Token,
		&inv.Status, &inv.InvitedBy, &inv.ExpiresAt, &inv.CreatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get invitation by token: %w", err)
	}
	return inv, nil
}

// GetByID retrieves an invitation by ID.
func (r *InvitationRepository) GetByID(ctx context.Context, id uuid.UUID) (*Invitation, error) {
	query := `
		SELECT id, org_id, email, role, token, status, invited_by, expires_at, created_at
		FROM invitations
		WHERE id = $1`

	inv := &Invitation{}
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&inv.ID, &inv.OrgID, &inv.Email, &inv.Role, &inv.Token,
		&inv.Status, &inv.InvitedBy, &inv.ExpiresAt, &inv.CreatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get invitation by id: %w", err)
	}
	return inv, nil
}

// ListPendingByOrg returns all pending invitations for an org.
func (r *InvitationRepository) ListPendingByOrg(ctx context.Context, orgID uuid.UUID) ([]Invitation, error) {
	query := `
		SELECT id, org_id, email, role, token, status, invited_by, expires_at, created_at
		FROM invitations
		WHERE org_id = $1 AND status = 'pending' AND expires_at > NOW()
		ORDER BY created_at DESC`

	rows, err := r.pool.Query(ctx, query, orgID)
	if err != nil {
		return nil, fmt.Errorf("list pending invitations: %w", err)
	}
	defer rows.Close()

	var invitations []Invitation
	for rows.Next() {
		var inv Invitation
		if err := rows.Scan(
			&inv.ID, &inv.OrgID, &inv.Email, &inv.Role, &inv.Token,
			&inv.Status, &inv.InvitedBy, &inv.ExpiresAt, &inv.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan invitation: %w", err)
		}
		invitations = append(invitations, inv)
	}
	return invitations, rows.Err()
}

// HasPendingInvitation checks if there's already a pending invitation for an email in an org.
func (r *InvitationRepository) HasPendingInvitation(ctx context.Context, orgID uuid.UUID, email string) (bool, error) {
	query := `SELECT EXISTS(
		SELECT 1 FROM invitations
		WHERE org_id = $1 AND email = $2 AND status = 'pending' AND expires_at > NOW()
	)`
	var exists bool
	err := r.pool.QueryRow(ctx, query, orgID, email).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("check pending invitation: %w", err)
	}
	return exists, nil
}

// MarkAccepted marks an invitation as accepted.
func (r *InvitationRepository) MarkAccepted(ctx context.Context, tx pgx.Tx, id uuid.UUID) error {
	query := `UPDATE invitations SET status = 'accepted' WHERE id = $1`
	_, err := tx.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("mark invitation accepted: %w", err)
	}
	return nil
}

// Delete removes an invitation.
func (r *InvitationRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM invitations WHERE id = $1`
	_, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("delete invitation: %w", err)
	}
	return nil
}
