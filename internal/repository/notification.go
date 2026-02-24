package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Notification represents an in-app notification.
type Notification struct {
	ID        uuid.UUID      `json:"id"`
	UserID    uuid.UUID      `json:"user_id"`
	OrgID     uuid.UUID      `json:"org_id"`
	Type      string         `json:"type"`
	Title     string         `json:"title"`
	Message   string         `json:"message"`
	Data      map[string]any `json:"data"`
	ReadAt    *time.Time     `json:"read_at"`
	CreatedAt time.Time      `json:"created_at"`
}

// NotificationRepository handles notifications database operations.
type NotificationRepository struct {
	pool *pgxpool.Pool
}

// NewNotificationRepository creates a new NotificationRepository.
func NewNotificationRepository(pool *pgxpool.Pool) *NotificationRepository {
	return &NotificationRepository{pool: pool}
}

// Create inserts a new notification.
func (r *NotificationRepository) Create(ctx context.Context, n *Notification) error {
	query := `
		INSERT INTO notifications (user_id, org_id, type, title, message, data)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, created_at`

	return r.pool.QueryRow(ctx, query,
		n.UserID, n.OrgID, n.Type, n.Title, n.Message, n.Data,
	).Scan(&n.ID, &n.CreatedAt)
}

// ListByUser returns notifications for a user in an org, newest first.
func (r *NotificationRepository) ListByUser(ctx context.Context, userID, orgID uuid.UUID, limit, offset int) ([]*Notification, int, error) {
	countQuery := `SELECT COUNT(*) FROM notifications WHERE user_id = $1 AND org_id = $2`
	var total int
	if err := r.pool.QueryRow(ctx, countQuery, userID, orgID).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count notifications: %w", err)
	}

	query := `
		SELECT id, user_id, org_id, type, title, message, data, read_at, created_at
		FROM notifications
		WHERE user_id = $1 AND org_id = $2
		ORDER BY created_at DESC
		LIMIT $3 OFFSET $4`

	rows, err := r.pool.Query(ctx, query, userID, orgID, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("list notifications: %w", err)
	}
	defer rows.Close()

	var notifications []*Notification
	for rows.Next() {
		n := &Notification{}
		if err := rows.Scan(&n.ID, &n.UserID, &n.OrgID, &n.Type, &n.Title, &n.Message, &n.Data, &n.ReadAt, &n.CreatedAt); err != nil {
			return nil, 0, fmt.Errorf("scan notification: %w", err)
		}
		notifications = append(notifications, n)
	}
	return notifications, total, nil
}

// CountUnread returns the count of unread notifications for a user in an org.
func (r *NotificationRepository) CountUnread(ctx context.Context, userID, orgID uuid.UUID) (int, error) {
	query := `SELECT COUNT(*) FROM notifications WHERE user_id = $1 AND org_id = $2 AND read_at IS NULL`
	var count int
	err := r.pool.QueryRow(ctx, query, userID, orgID).Scan(&count)
	return count, err
}

// MarkRead marks a single notification as read.
func (r *NotificationRepository) MarkRead(ctx context.Context, id, userID uuid.UUID) error {
	query := `UPDATE notifications SET read_at = NOW() WHERE id = $1 AND user_id = $2 AND read_at IS NULL`
	_, err := r.pool.Exec(ctx, query, id, userID)
	return err
}

// MarkAllRead marks all unread notifications as read for a user in an org.
func (r *NotificationRepository) MarkAllRead(ctx context.Context, userID, orgID uuid.UUID) error {
	query := `UPDATE notifications SET read_at = NOW() WHERE user_id = $1 AND org_id = $2 AND read_at IS NULL`
	_, err := r.pool.Exec(ctx, query, userID, orgID)
	return err
}
