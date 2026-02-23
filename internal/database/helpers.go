package database

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// WithTx executes fn inside a database transaction. If fn returns an error
// the transaction is rolled back; otherwise it is committed.
func WithTx(ctx context.Context, pool *pgxpool.Pool, fn func(tx pgx.Tx) error) error {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}

	if err := fn(tx); err != nil {
		if rbErr := tx.Rollback(ctx); rbErr != nil {
			return fmt.Errorf("rollback failed: %v (original: %w)", rbErr, err)
		}
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit tx: %w", err)
	}
	return nil
}

// CollectRows is a thin wrapper around pgx.CollectRows for convenience.
func CollectRows[T any](rows pgx.Rows, fn pgx.RowToFunc[T]) ([]T, error) {
	return pgx.CollectRows(rows, fn)
}

// CollectOneRow is a thin wrapper around pgx.CollectOneRow for convenience.
func CollectOneRow[T any](rows pgx.Rows, fn pgx.RowToFunc[T]) (T, error) {
	return pgx.CollectOneRow(rows, fn)
}
