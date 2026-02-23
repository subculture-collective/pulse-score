package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/onnwee/pulse-score/internal/repository"
)

// CreateOrgRequest holds the input for creating an organization.
type CreateOrgRequest struct {
	Name string `json:"name"`
}

// OrgResponse is the response for organization operations.
type OrgResponse struct {
	ID   uuid.UUID `json:"id"`
	Name string    `json:"name"`
	Slug string    `json:"slug"`
	Plan string    `json:"plan"`
}

// OrganizationService handles organization logic.
type OrganizationService struct {
	pool *pgxpool.Pool
	orgs *repository.OrganizationRepository
}

// NewOrganizationService creates a new OrganizationService.
func NewOrganizationService(pool *pgxpool.Pool, orgs *repository.OrganizationRepository) *OrganizationService {
	return &OrganizationService{pool: pool, orgs: orgs}
}

// Create creates a new organization and assigns the caller as owner.
func (s *OrganizationService) Create(ctx context.Context, userID uuid.UUID, req CreateOrgRequest) (*OrgResponse, error) {
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return nil, &ValidationError{Field: "name", Message: "organization name is required"}
	}

	slug := generateSlug(name)
	baseSlug := slug
	for i := 1; ; i++ {
		exists, err := s.orgs.SlugExists(ctx, slug)
		if err != nil {
			return nil, fmt.Errorf("check slug: %w", err)
		}
		if !exists {
			break
		}
		slug = fmt.Sprintf("%s-%d", baseSlug, i)
	}

	org := &repository.Organization{
		Name: name,
		Slug: slug,
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	if err := s.orgs.Create(ctx, tx, org); err != nil {
		return nil, fmt.Errorf("create org: %w", err)
	}

	if err := s.orgs.AddMember(ctx, tx, userID, org.ID, "owner"); err != nil {
		return nil, fmt.Errorf("add owner: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit: %w", err)
	}

	return &OrgResponse{
		ID:   org.ID,
		Name: org.Name,
		Slug: org.Slug,
		Plan: "free",
	}, nil
}
