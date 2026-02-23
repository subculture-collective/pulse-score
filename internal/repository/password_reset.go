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

// PasswordReset represents a password_resets row.
type PasswordReset struct {
	ID        uuid.UUID
	UserID    uuid.UUID
	TokenHash string
	ExpiresAt time.Time
	UsedAt    *time.Time
	CreatedAt time.Time
}

// PasswordResetRepository handles password reset database operations.
type PasswordResetRepository struct {
	pool *pgxpool.Pool
}

// NewPasswordResetRepository creates a new PasswordResetRepository.
func NewPasswordResetRepository(pool *pgxpool.Pool) *PasswordResetRepository {
	return &PasswordResetRepository{pool: pool}
}

// Create inserts a new password reset token.
func (r *PasswordResetRepository) Create(ctx context.Context, userID uuid.UUID, tokenHash string, expiresAt time.Time) error {
	query := `INSERT INTO password_resets (user_id, token_hash, expires_at) VALUES ($1, $2, $3)`
	_, err := r.pool.Exec(ctx, query, userID, tokenHash, expiresAt)
	if err != nil {
		return fmt.Errorf("create password reset: %w", err)
	}
	return nil
}

// GetByHash retrieves a password reset by token hash.
func (r *PasswordResetRepository) GetByHash(ctx context.Context, tokenHash string) (*PasswordReset, error) {
	query := `
		SELECT id, user_id, token_hash, expires_at, used_at, created_at
		FROM password_resets
		WHERE token_hash = $1`

	pr := &PasswordReset{}
	err := r.pool.QueryRow(ctx, query, tokenHash).Scan(
		&pr.ID, &pr.UserID, &pr.TokenHash, &pr.ExpiresAt, &pr.UsedAt, &pr.CreatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get password reset by hash: %w", err)
	}
	return pr, nil
}

// MarkUsed marks a password reset token as used.
func (r *PasswordResetRepository) MarkUsed(ctx context.Context, tx pgx.Tx, id uuid.UUID) error {
	query := `UPDATE password_resets SET used_at = NOW() WHERE id = $1`
	_, err := tx.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("mark password reset used: %w", err)
	}
	return nil
}
