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

// OnboardingStatus represents an onboarding_status row.
type OnboardingStatus struct {
	ID             uuid.UUID
	OrgID          uuid.UUID
	CurrentStep    string
	CompletedSteps []string
	SkippedSteps   []string
	StepPayloads   map[string]any
	CompletedAt    *time.Time
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// OnboardingStatusRepository handles onboarding_status database operations.
type OnboardingStatusRepository struct {
	pool *pgxpool.Pool
}

// NewOnboardingStatusRepository creates a new OnboardingStatusRepository.
func NewOnboardingStatusRepository(pool *pgxpool.Pool) *OnboardingStatusRepository {
	return &OnboardingStatusRepository{pool: pool}
}

// GetByOrgID retrieves onboarding status by org ID.
func (r *OnboardingStatusRepository) GetByOrgID(ctx context.Context, orgID uuid.UUID) (*OnboardingStatus, error) {
	query := `
		SELECT id, org_id, current_step,
			COALESCE(completed_steps, '{}'),
			COALESCE(skipped_steps, '{}'),
			COALESCE(step_payloads, '{}'),
			completed_at, created_at, updated_at
		FROM onboarding_status
		WHERE org_id = $1`

	status := &OnboardingStatus{}
	err := r.pool.QueryRow(ctx, query, orgID).Scan(
		&status.ID,
		&status.OrgID,
		&status.CurrentStep,
		&status.CompletedSteps,
		&status.SkippedSteps,
		&status.StepPayloads,
		&status.CompletedAt,
		&status.CreatedAt,
		&status.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get onboarding status by org id: %w", err)
	}
	return status, nil
}

// Upsert creates or updates onboarding status for an org.
func (r *OnboardingStatusRepository) Upsert(ctx context.Context, status *OnboardingStatus) error {
	query := `
		INSERT INTO onboarding_status (
			org_id, current_step, completed_steps, skipped_steps, step_payloads, completed_at
		)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (org_id) DO UPDATE SET
			current_step = EXCLUDED.current_step,
			completed_steps = EXCLUDED.completed_steps,
			skipped_steps = EXCLUDED.skipped_steps,
			step_payloads = EXCLUDED.step_payloads,
			completed_at = EXCLUDED.completed_at,
			updated_at = NOW()
		RETURNING id, created_at, updated_at`

	if status.StepPayloads == nil {
		status.StepPayloads = map[string]any{}
	}

	err := r.pool.QueryRow(
		ctx,
		query,
		status.OrgID,
		status.CurrentStep,
		status.CompletedSteps,
		status.SkippedSteps,
		status.StepPayloads,
		status.CompletedAt,
	).Scan(&status.ID, &status.CreatedAt, &status.UpdatedAt)
	if err != nil {
		return fmt.Errorf("upsert onboarding status: %w", err)
	}

	return nil
}

// Reset clears onboarding status for an org.
func (r *OnboardingStatusRepository) Reset(ctx context.Context, orgID uuid.UUID, defaultStep string) error {
	query := `
		INSERT INTO onboarding_status (
			org_id, current_step, completed_steps, skipped_steps, step_payloads, completed_at
		)
		VALUES ($1, $2, '{}', '{}', '{}', NULL)
		ON CONFLICT (org_id) DO UPDATE SET
			current_step = EXCLUDED.current_step,
			completed_steps = '{}',
			skipped_steps = '{}',
			step_payloads = '{}',
			completed_at = NULL,
			updated_at = NOW()`

	_, err := r.pool.Exec(ctx, query, orgID, defaultStep)
	if err != nil {
		return fmt.Errorf("reset onboarding status: %w", err)
	}
	return nil
}
