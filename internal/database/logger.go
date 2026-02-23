package database

import (
	"context"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5"
)

// queryLogger implements pgx.QueryTracer for development logging.
type queryLogger struct{}

func (q *queryLogger) TraceQueryStart(ctx context.Context, _ *pgx.Conn, data pgx.TraceQueryStartData) context.Context {
	return context.WithValue(ctx, queryStartKey{}, time.Now())
}

func (q *queryLogger) TraceQueryEnd(ctx context.Context, _ *pgx.Conn, data pgx.TraceQueryEndData) {
	start, _ := ctx.Value(queryStartKey{}).(time.Time)
	duration := time.Since(start)

	if data.Err != nil {
		slog.Debug("query error",
"sql", data.CommandTag.String(),
			"duration", duration,
			"error", data.Err,
		)
		return
	}

	slog.Debug("query",
"sql", data.CommandTag.String(),
		"duration", duration,
	)
}

type queryStartKey struct{}
