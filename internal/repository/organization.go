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

// Organization represents an organizations row.
type Organization struct {
	ID               uuid.UUID
	Name             string
	Slug             string
	Plan             string
	StripeCustomerID string
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

// OrganizationRepository handles organization database operations.
type OrganizationRepository struct {
	pool *pgxpool.Pool
}

// NewOrganizationRepository creates a new OrganizationRepository.
func NewOrganizationRepository(pool *pgxpool.Pool) *OrganizationRepository {
	return &OrganizationRepository{pool: pool}
}

// Create inserts a new organization inside a transaction.
func (r *OrganizationRepository) Create(ctx context.Context, tx pgx.Tx, org *Organization) error {
	query := `
		INSERT INTO organizations (id, name, slug)
		VALUES ($1, $2, $3)
		RETURNING created_at, updated_at`

	if org.ID == uuid.Nil {
		org.ID = uuid.New()
	}

	return tx.QueryRow(ctx, query, org.ID, org.Name, org.Slug).Scan(&org.CreatedAt, &org.UpdatedAt)
}

// SlugExists checks if an organization slug is already taken.
func (r *OrganizationRepository) SlugExists(ctx context.Context, slug string) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM organizations WHERE slug = $1 AND deleted_at IS NULL)`
	var exists bool
	err := r.pool.QueryRow(ctx, query, slug).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("check slug exists: %w", err)
	}
	return exists, nil
}

// AddMember adds a user to an organization with a given role inside a transaction.
func (r *OrganizationRepository) AddMember(ctx context.Context, tx pgx.Tx, userID, orgID uuid.UUID, role string) error {
	query := `INSERT INTO user_organizations (user_id, org_id, role) VALUES ($1, $2, $3)`
	_, err := tx.Exec(ctx, query, userID, orgID, role)
	if err != nil {
		return fmt.Errorf("add org member: %w", err)
	}
	return nil
}

// GetByID retrieves an organization by ID.
func (r *OrganizationRepository) GetByID(ctx context.Context, id uuid.UUID) (*Organization, error) {
	query := `
		SELECT id, name, slug, plan, COALESCE(stripe_customer_id, ''),
			   created_at, updated_at
		FROM organizations
		WHERE id = $1 AND deleted_at IS NULL`

	o := &Organization{}
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&o.ID, &o.Name, &o.Slug, &o.Plan, &o.StripeCustomerID,
		&o.CreatedAt, &o.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get org by id: %w", err)
	}
	return o, nil
}

// IsMember checks if a user is a member of an organization.
func (r *OrganizationRepository) IsMember(ctx context.Context, userID, orgID uuid.UUID) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM user_organizations WHERE user_id = $1 AND org_id = $2)`
	var exists bool
	err := r.pool.QueryRow(ctx, query, userID, orgID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("check org membership: %w", err)
	}
	return exists, nil
}
