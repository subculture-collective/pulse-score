package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// AlertHistory represents an alert_history row.
type AlertHistory struct {
	ID               uuid.UUID      `json:"id"`
	OrgID            uuid.UUID      `json:"org_id"`
	AlertRuleID      uuid.UUID      `json:"alert_rule_id"`
	CustomerID       *uuid.UUID     `json:"customer_id,omitempty"`
	TriggerData      map[string]any `json:"trigger_data"`
	Channel          string         `json:"channel"`
	Status           string         `json:"status"` // sent, failed, pending
	SentAt           *time.Time     `json:"sent_at,omitempty"`
	ErrorMessage     string         `json:"error_message,omitempty"`
	SendGridMsgID    string         `json:"sendgrid_message_id,omitempty"`
	DeliveredAt      *time.Time     `json:"delivered_at,omitempty"`
	OpenedAt         *time.Time     `json:"opened_at,omitempty"`
	ClickedAt        *time.Time     `json:"clicked_at,omitempty"`
	BouncedAt        *time.Time     `json:"bounced_at,omitempty"`
	CreatedAt        time.Time      `json:"created_at"`
}

// AlertHistoryRepository handles alert_history database operations.
type AlertHistoryRepository struct {
	pool *pgxpool.Pool
}

// NewAlertHistoryRepository creates a new AlertHistoryRepository.
func NewAlertHistoryRepository(pool *pgxpool.Pool) *AlertHistoryRepository {
	return &AlertHistoryRepository{pool: pool}
}

// Create inserts a new alert history record.
func (r *AlertHistoryRepository) Create(ctx context.Context, h *AlertHistory) error {
	return r.pool.QueryRow(ctx, `
		INSERT INTO alert_history (org_id, alert_rule_id, customer_id, trigger_data, channel, status, sent_at, error_message, sendgrid_message_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id, created_at
	`, h.OrgID, h.AlertRuleID, h.CustomerID, h.TriggerData, h.Channel,
		h.Status, h.SentAt, h.ErrorMessage, h.SendGridMsgID,
	).Scan(&h.ID, &h.CreatedAt)
}

// UpdateStatus updates the delivery status of an alert history record.
func (r *AlertHistoryRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status string, errorMsg string) error {
	var sentAt *time.Time
	if status == "sent" {
		now := time.Now()
		sentAt = &now
	}

	ct, err := r.pool.Exec(ctx, `
		UPDATE alert_history
		SET status = $1, error_message = $2, sent_at = COALESCE($3, sent_at)
		WHERE id = $4
	`, status, errorMsg, sentAt, id)
	if err != nil {
		return fmt.Errorf("update alert history status: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

// UpdateSendGridMessageID sets the sendgrid message ID after sending.
func (r *AlertHistoryRepository) UpdateSendGridMessageID(ctx context.Context, id uuid.UUID, msgID string) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE alert_history SET sendgrid_message_id = $1 WHERE id = $2
	`, msgID, id)
	if err != nil {
		return fmt.Errorf("update sendgrid message id: %w", err)
	}
	return nil
}

// UpdateDeliveryStatus updates delivery tracking columns based on SendGrid webhook events.
func (r *AlertHistoryRepository) UpdateDeliveryStatus(ctx context.Context, sendgridMsgID string, event string, timestamp time.Time) error {
	var query string
	switch event {
	case "delivered":
		query = `UPDATE alert_history SET delivered_at = $1 WHERE sendgrid_message_id = $2`
	case "bounce":
		query = `UPDATE alert_history SET bounced_at = $1, status = 'failed' WHERE sendgrid_message_id = $2`
	case "open":
		query = `UPDATE alert_history SET opened_at = $1 WHERE sendgrid_message_id = $2`
	case "click":
		query = `UPDATE alert_history SET clicked_at = $1 WHERE sendgrid_message_id = $2`
	case "spamreport":
		query = `UPDATE alert_history SET status = 'failed' WHERE sendgrid_message_id = $2`
	default:
		return nil // ignore unknown events
	}

	_, err := r.pool.Exec(ctx, query, timestamp, sendgridMsgID)
	if err != nil {
		return fmt.Errorf("update delivery status: %w", err)
	}
	return nil
}

// ListByOrg returns a paginated list of alert history for an org.
func (r *AlertHistoryRepository) ListByOrg(ctx context.Context, orgID uuid.UUID, status string, limit, offset int) ([]*AlertHistory, int, error) {
	where := "org_id = $1"
	args := []any{orgID}
	argIdx := 2

	if status != "" {
		where += fmt.Sprintf(" AND status = $%d", argIdx)
		args = append(args, status)
		argIdx++
	}

	// Count
	var total int
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM alert_history WHERE %s", where)
	if err := r.pool.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count alert history: %w", err)
	}

	// Data
	dataQuery := fmt.Sprintf(`
		SELECT id, org_id, alert_rule_id, customer_id, trigger_data, channel, status, sent_at, error_message,
			COALESCE(sendgrid_message_id, ''), delivered_at, opened_at, clicked_at, bounced_at, created_at
		FROM alert_history
		WHERE %s
		ORDER BY created_at DESC
		LIMIT $%d OFFSET $%d
	`, where, argIdx, argIdx+1)
	args = append(args, limit, offset)

	rows, err := r.pool.Query(ctx, dataQuery, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list alert history: %w", err)
	}
	defer rows.Close()

	var items []*AlertHistory
	for rows.Next() {
		h := &AlertHistory{}
		if err := rows.Scan(
			&h.ID, &h.OrgID, &h.AlertRuleID, &h.CustomerID, &h.TriggerData,
			&h.Channel, &h.Status, &h.SentAt, &h.ErrorMessage,
			&h.SendGridMsgID, &h.DeliveredAt, &h.OpenedAt, &h.ClickedAt, &h.BouncedAt,
			&h.CreatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("scan alert history: %w", err)
		}
		items = append(items, h)
	}
	return items, total, rows.Err()
}

// ListByRule returns alert history records for a specific rule.
func (r *AlertHistoryRepository) ListByRule(ctx context.Context, ruleID uuid.UUID, limit, offset int) ([]*AlertHistory, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, org_id, alert_rule_id, customer_id, trigger_data, channel, status, sent_at, error_message,
			COALESCE(sendgrid_message_id, ''), delivered_at, opened_at, clicked_at, bounced_at, created_at
		FROM alert_history
		WHERE alert_rule_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`, ruleID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("list alert history by rule: %w", err)
	}
	defer rows.Close()

	var items []*AlertHistory
	for rows.Next() {
		h := &AlertHistory{}
		if err := rows.Scan(
			&h.ID, &h.OrgID, &h.AlertRuleID, &h.CustomerID, &h.TriggerData,
			&h.Channel, &h.Status, &h.SentAt, &h.ErrorMessage,
			&h.SendGridMsgID, &h.DeliveredAt, &h.OpenedAt, &h.ClickedAt, &h.BouncedAt,
			&h.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan alert history: %w", err)
		}
		items = append(items, h)
	}
	return items, rows.Err()
}

// GetLastAlertForRule returns the most recent alert history for a rule+customer combo (for deduplication/cooldown).
func (r *AlertHistoryRepository) GetLastAlertForRule(ctx context.Context, ruleID, customerID uuid.UUID) (*AlertHistory, error) {
	h := &AlertHistory{}
	err := r.pool.QueryRow(ctx, `
		SELECT id, org_id, alert_rule_id, customer_id, trigger_data, channel, status, sent_at, error_message,
			COALESCE(sendgrid_message_id, ''), delivered_at, opened_at, clicked_at, bounced_at, created_at
		FROM alert_history
		WHERE alert_rule_id = $1 AND customer_id = $2
		ORDER BY created_at DESC
		LIMIT 1
	`, ruleID, customerID).Scan(
		&h.ID, &h.OrgID, &h.AlertRuleID, &h.CustomerID, &h.TriggerData,
		&h.Channel, &h.Status, &h.SentAt, &h.ErrorMessage,
		&h.SendGridMsgID, &h.DeliveredAt, &h.OpenedAt, &h.ClickedAt, &h.BouncedAt,
		&h.CreatedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get last alert for rule: %w", err)
	}
	return h, nil
}

// CountByStatus returns counts grouped by status for an org.
func (r *AlertHistoryRepository) CountByStatus(ctx context.Context, orgID uuid.UUID) (map[string]int, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT status, COUNT(*)
		FROM alert_history
		WHERE org_id = $1
		GROUP BY status
	`, orgID)
	if err != nil {
		return nil, fmt.Errorf("count alert history by status: %w", err)
	}
	defer rows.Close()

	counts := make(map[string]int)
	for rows.Next() {
		var status string
		var count int
		if err := rows.Scan(&status, &count); err != nil {
			return nil, fmt.Errorf("scan status count: %w", err)
		}
		counts[status] = count
	}
	return counts, rows.Err()
}

// ListActiveRulesByOrg returns all active alert rules for an org.
func (r *AlertHistoryRepository) ListActiveRulesByOrg(ctx context.Context, orgID uuid.UUID) ([]*AlertRule, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, org_id, name, description, trigger_type, conditions, channel, recipients, is_active, created_by, created_at, updated_at
		FROM alert_rules
		WHERE org_id = $1 AND is_active = true
		ORDER BY created_at DESC
	`, orgID)
	if err != nil {
		return nil, fmt.Errorf("list active alert rules: %w", err)
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
	return rules, rows.Err()
}

// ListOrgsWithActiveRules returns distinct org IDs that have at least one active alert rule.
func (r *AlertHistoryRepository) ListOrgsWithActiveRules(ctx context.Context) ([]uuid.UUID, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT DISTINCT org_id FROM alert_rules WHERE is_active = true
	`)
	if err != nil {
		return nil, fmt.Errorf("list orgs with active rules: %w", err)
	}
	defer rows.Close()

	var orgIDs []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scan org id: %w", err)
		}
		orgIDs = append(orgIDs, id)
	}
	return orgIDs, rows.Err()
}
