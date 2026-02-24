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

// NotificationPreference represents a user's notification preferences for an org.
type NotificationPreference struct {
	ID              uuid.UUID   `json:"id"`
	UserID          uuid.UUID   `json:"user_id"`
	OrgID           uuid.UUID   `json:"org_id"`
	EmailEnabled    bool        `json:"email_enabled"`
	InAppEnabled    bool        `json:"in_app_enabled"`
	DigestEnabled   bool        `json:"digest_enabled"`
	DigestFrequency string      `json:"digest_frequency"`
	MutedRuleIDs    []uuid.UUID `json:"muted_rule_ids"`
	CreatedAt       time.Time   `json:"created_at"`
	UpdatedAt       time.Time   `json:"updated_at"`
}

// NotificationPreferenceRepository handles notification_preferences database operations.
type NotificationPreferenceRepository struct {
	pool *pgxpool.Pool
}

// NewNotificationPreferenceRepository creates a new NotificationPreferenceRepository.
func NewNotificationPreferenceRepository(pool *pgxpool.Pool) *NotificationPreferenceRepository {
	return &NotificationPreferenceRepository{pool: pool}
}

// GetByUserAndOrg retrieves notification preferences for a user in an org.
// Returns a default preference if none exists.
func (r *NotificationPreferenceRepository) GetByUserAndOrg(ctx context.Context, userID, orgID uuid.UUID) (*NotificationPreference, error) {
	query := `
		SELECT id, user_id, org_id, email_enabled, in_app_enabled, digest_enabled, digest_frequency, muted_rule_ids, created_at, updated_at
		FROM notification_preferences
		WHERE user_id = $1 AND org_id = $2`

	pref := &NotificationPreference{}
	err := r.pool.QueryRow(ctx, query, userID, orgID).Scan(
		&pref.ID, &pref.UserID, &pref.OrgID,
		&pref.EmailEnabled, &pref.InAppEnabled,
		&pref.DigestEnabled, &pref.DigestFrequency,
		&pref.MutedRuleIDs,
		&pref.CreatedAt, &pref.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		// Return defaults
		return &NotificationPreference{
			UserID:          userID,
			OrgID:           orgID,
			EmailEnabled:    true,
			InAppEnabled:    true,
			DigestEnabled:   false,
			DigestFrequency: "weekly",
			MutedRuleIDs:    []uuid.UUID{},
		}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get notification preferences: %w", err)
	}
	return pref, nil
}

// Upsert creates or updates notification preferences.
func (r *NotificationPreferenceRepository) Upsert(ctx context.Context, pref *NotificationPreference) error {
	query := `
		INSERT INTO notification_preferences (user_id, org_id, email_enabled, in_app_enabled, digest_enabled, digest_frequency, muted_rule_ids)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (user_id, org_id) DO UPDATE SET
			email_enabled = EXCLUDED.email_enabled,
			in_app_enabled = EXCLUDED.in_app_enabled,
			digest_enabled = EXCLUDED.digest_enabled,
			digest_frequency = EXCLUDED.digest_frequency,
			muted_rule_ids = EXCLUDED.muted_rule_ids
		RETURNING id, created_at, updated_at`

	return r.pool.QueryRow(ctx, query,
		pref.UserID, pref.OrgID,
		pref.EmailEnabled, pref.InAppEnabled,
		pref.DigestEnabled, pref.DigestFrequency,
		pref.MutedRuleIDs,
	).Scan(&pref.ID, &pref.CreatedAt, &pref.UpdatedAt)
}
