package repository

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// OnboardingEvent represents an onboarding_events row.
type OnboardingEvent struct {
	ID         uuid.UUID
	OrgID      uuid.UUID
	StepID     string
	EventType  string
	OccurredAt time.Time
	DurationMS *int64
	Metadata   map[string]any
	CreatedAt  time.Time
}

// OnboardingStepMetrics represents rollups for a single step.
type OnboardingStepMetrics struct {
	StepID            string  `json:"step_id"`
	StartedCount      int     `json:"started_count"`
	CompletedCount    int     `json:"completed_count"`
	SkippedCount      int     `json:"skipped_count"`
	CompletionRate    float64 `json:"completion_rate"`
	SkipRate          float64 `json:"skip_rate"`
	AverageDurationMS float64 `json:"average_duration_ms"`
}

// OnboardingAnalyticsSummary represents org-level onboarding funnel metrics.
type OnboardingAnalyticsSummary struct {
	OverallCompletionRate float64                 `json:"overall_completion_rate"`
	AverageStepDurationMS float64                 `json:"average_step_duration_ms"`
	StepMetrics           []OnboardingStepMetrics `json:"step_metrics"`
}

// OnboardingEventRepository handles onboarding_events database operations.
type OnboardingEventRepository struct {
	pool *pgxpool.Pool
}

// NewOnboardingEventRepository creates a new OnboardingEventRepository.
func NewOnboardingEventRepository(pool *pgxpool.Pool) *OnboardingEventRepository {
	return &OnboardingEventRepository{pool: pool}
}

// Create stores a new onboarding event.
func (r *OnboardingEventRepository) Create(ctx context.Context, e *OnboardingEvent) error {
	query := `
		INSERT INTO onboarding_events (org_id, step_id, event_type, occurred_at, duration_ms, metadata)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, created_at`

	if e.Metadata == nil {
		e.Metadata = map[string]any{}
	}
	if e.OccurredAt.IsZero() {
		e.OccurredAt = time.Now()
	}

	err := r.pool.QueryRow(
		ctx,
		query,
		e.OrgID,
		e.StepID,
		e.EventType,
		e.OccurredAt,
		e.DurationMS,
		e.Metadata,
	).Scan(&e.ID, &e.CreatedAt)
	if err != nil {
		return fmt.Errorf("create onboarding event: %w", err)
	}

	return nil
}

// GetAnalyticsSummary computes onboarding funnel metrics for an organization.
func (r *OnboardingEventRepository) GetAnalyticsSummary(ctx context.Context, orgID uuid.UUID) (*OnboardingAnalyticsSummary, error) {
	stepQuery := `
		SELECT
			step_id,
			COUNT(*) FILTER (WHERE event_type = 'step_started') AS started_count,
			COUNT(*) FILTER (WHERE event_type = 'step_completed') AS completed_count,
			COUNT(*) FILTER (WHERE event_type = 'step_skipped') AS skipped_count,
			COALESCE(AVG(duration_ms) FILTER (WHERE event_type IN ('step_completed', 'step_skipped') AND duration_ms IS NOT NULL), 0) AS avg_duration_ms
		FROM onboarding_events
		WHERE org_id = $1
		GROUP BY step_id`

	rows, err := r.pool.Query(ctx, stepQuery, orgID)
	if err != nil {
		return nil, fmt.Errorf("query onboarding step metrics: %w", err)
	}
	defer rows.Close()

	metrics := make([]OnboardingStepMetrics, 0)
	for rows.Next() {
		var m OnboardingStepMetrics
		if err := rows.Scan(
			&m.StepID,
			&m.StartedCount,
			&m.CompletedCount,
			&m.SkippedCount,
			&m.AverageDurationMS,
		); err != nil {
			return nil, fmt.Errorf("scan onboarding step metrics: %w", err)
		}

		if m.StartedCount > 0 {
			m.CompletionRate = float64(m.CompletedCount) / float64(m.StartedCount)
		}
		totalTerminal := m.CompletedCount + m.SkippedCount
		if totalTerminal > 0 {
			m.SkipRate = float64(m.SkippedCount) / float64(totalTerminal)
		}

		metrics = append(metrics, m)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("onboarding step metrics rows error: %w", err)
	}

	sort.Slice(metrics, func(i, j int) bool {
		return metrics[i].StepID < metrics[j].StepID
	})

	overallQuery := `
		SELECT
			COUNT(*) FILTER (WHERE event_type = 'step_started') AS started_count,
			COUNT(*) FILTER (WHERE event_type = 'onboarding_completed') AS completed_count,
			COALESCE(AVG(duration_ms) FILTER (WHERE event_type IN ('step_completed', 'step_skipped') AND duration_ms IS NOT NULL), 0) AS avg_step_duration_ms
		FROM onboarding_events
		WHERE org_id = $1`

	var startedCount int
	var completedCount int
	var avgStepDuration float64
	if err := r.pool.QueryRow(ctx, overallQuery, orgID).Scan(&startedCount, &completedCount, &avgStepDuration); err != nil {
		return nil, fmt.Errorf("query onboarding overall metrics: %w", err)
	}

	overallCompletionRate := 0.0
	if startedCount > 0 {
		overallCompletionRate = float64(completedCount) / float64(startedCount)
	}

	return &OnboardingAnalyticsSummary{
		OverallCompletionRate: overallCompletionRate,
		AverageStepDurationMS: avgStepDuration,
		StepMetrics:           metrics,
	}, nil
}
