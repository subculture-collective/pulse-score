package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// AlertRule represents an alert rule.
type AlertRule struct {
	ID          uuid.UUID      `json:"id"`
	OrgID       uuid.UUID      `json:"org_id"`
	Name        string         `json:"name"`
	Description string         `json:"description,omitempty"`
	TriggerType string         `json:"trigger_type"`
	Conditions  map[string]any `json:"conditions"`
	Channel     string         `json:"channel"`
	Recipients  []string       `json:"recipients"`
	IsActive    bool           `json:"is_active"`
	CreatedBy   *uuid.UUID     `json:"created_by,omitempty"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
}

// AlertRuleRepository handles alert_rules database operations.
type AlertRuleRepository struct {
	pool *pgxpool.Pool
}

// NewAlertRuleRepository creates a new AlertRuleRepository.
func NewAlertRuleRepository(pool *pgxpool.Pool) *AlertRuleRepository {
	return &AlertRuleRepository{pool: pool}
}

// List returns all alert rules for an organization.
func (r *AlertRuleRepository) List(ctx context.Context, orgID uuid.UUID) ([]*AlertRule, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, org_id, name, description, trigger_type, conditions, channel, recipients, is_active, created_by, created_at, updated_at
		FROM alert_rules
		WHERE org_id = $1
		ORDER BY created_at DESC
	`, orgID)
	if err != nil {
		return nil, fmt.Errorf("query alert rules: %w", err)
	}
	defer rows.Close()

	var rules []*AlertRule
	for rows.Next() {
		rule := &AlertRule{}
		if err := rows.Scan(
			&rule.ID, &rule.OrgID, &rule.Name, &rule.Description,
			&rule.TriggerType, &rule.Conditions, &rule.Channel,
			&rule.Recipients, &rule.IsActive, &rule.CreatedBy,
			&rule.CreatedAt, &rule.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan alert rule: %w", err)
		}
		rules = append(rules, rule)
	}

	return rules, nil
}

// GetByID returns a single alert rule by ID and org.
func (r *AlertRuleRepository) GetByID(ctx context.Context, id, orgID uuid.UUID) (*AlertRule, error) {
	rule := &AlertRule{}
	err := r.pool.QueryRow(ctx, `
		SELECT id, org_id, name, description, trigger_type, conditions, channel, recipients, is_active, created_by, created_at, updated_at
		FROM alert_rules
		WHERE id = $1 AND org_id = $2
	`, id, orgID).Scan(
		&rule.ID, &rule.OrgID, &rule.Name, &rule.Description,
		&rule.TriggerType, &rule.Conditions, &rule.Channel,
		&rule.Recipients, &rule.IsActive, &rule.CreatedBy,
		&rule.CreatedAt, &rule.UpdatedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("query alert rule: %w", err)
	}
	return rule, nil
}

// Create inserts a new alert rule.
func (r *AlertRuleRepository) Create(ctx context.Context, rule *AlertRule) error {
	return r.pool.QueryRow(ctx, `
		INSERT INTO alert_rules (org_id, name, description, trigger_type, conditions, channel, recipients, is_active, created_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id, created_at, updated_at
	`, rule.OrgID, rule.Name, rule.Description, rule.TriggerType,
		rule.Conditions, rule.Channel, rule.Recipients,
		rule.IsActive, rule.CreatedBy,
	).Scan(&rule.ID, &rule.CreatedAt, &rule.UpdatedAt)
}

// Update updates an existing alert rule.
func (r *AlertRuleRepository) Update(ctx context.Context, rule *AlertRule) error {
	ct, err := r.pool.Exec(ctx, `
		UPDATE alert_rules
		SET name = $1, description = $2, trigger_type = $3, conditions = $4, channel = $5, recipients = $6, is_active = $7, updated_at = NOW()
		WHERE id = $8 AND org_id = $9
	`, rule.Name, rule.Description, rule.TriggerType, rule.Conditions,
		rule.Channel, rule.Recipients, rule.IsActive,
		rule.ID, rule.OrgID,
	)
	if err != nil {
		return fmt.Errorf("update alert rule: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

// Delete deletes an alert rule.
func (r *AlertRuleRepository) Delete(ctx context.Context, id, orgID uuid.UUID) error {
	ct, err := r.pool.Exec(ctx, `
		DELETE FROM alert_rules WHERE id = $1 AND org_id = $2
	`, id, orgID)
	if err != nil {
		return fmt.Errorf("delete alert rule: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}
