package database

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// PoolConfig holds connection-pool tuning parameters.
type PoolConfig struct {
	URL                string
	MaxConns           int32
	MinConns           int32
	MaxConnLifetime    time.Duration
	HealthCheckPeriod  time.Duration
	QueryLogEnabled    bool
}

// DefaultPoolConfig returns production-safe defaults.
func DefaultPoolConfig(url string) PoolConfig {
	return PoolConfig{
		URL:               url,
		MaxConns:          10,
		MinConns:          2,
		MaxConnLifetime:   time.Hour,
		HealthCheckPeriod: 30 * time.Second,
	}
}

// NewPool creates a pgxpool.Pool from the given configuration.
func NewPool(ctx context.Context, cfg PoolConfig) (*pgxpool.Pool, error) {
	poolCfg, err := pgxpool.ParseConfig(cfg.URL)
	if err != nil {
		return nil, fmt.Errorf("parse database URL: %w", err)
	}

	poolCfg.MaxConns = cfg.MaxConns
	poolCfg.MinConns = cfg.MinConns
	poolCfg.MaxConnLifetime = cfg.MaxConnLifetime
	poolCfg.HealthCheckPeriod = cfg.HealthCheckPeriod

	if cfg.QueryLogEnabled {
		poolCfg.ConnConfig.Tracer = &queryLogger{}
	}

	pool, err := pgxpool.NewWithConfig(ctx, poolCfg)
	if err != nil {
		return nil, fmt.Errorf("create connection pool: %w", err)
	}

	// Verify connectivity
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping database: %w", err)
	}

	slog.Info("database pool connected",
		"max_conns", cfg.MaxConns,
		"min_conns", cfg.MinConns,
	)

	return pool, nil
}

// Close drains the pool gracefully.
func Close(pool *pgxpool.Pool) {
	if pool != nil {
		pool.Close()
		slog.Info("database pool closed")
	}
}
