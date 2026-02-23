package repository

import "github.com/jackc/pgx/v5/pgxpool"

// Base provides shared database access for all repositories.
type Base struct {
	pool *pgxpool.Pool
}

// NewBase creates a Base repository with the given pool.
func NewBase(pool *pgxpool.Pool) Base {
	return Base{pool: pool}
}

// Pool returns the underlying connection pool.
func (b *Base) Pool() *pgxpool.Pool {
	return b.pool
}
