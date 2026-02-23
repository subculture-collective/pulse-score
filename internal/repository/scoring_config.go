package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ScoringConfig represents a scoring_configs row.
type ScoringConfig struct {
	ID         uuid.UUID          `json:"id"`
	OrgID      uuid.UUID          `json:"org_id"`
	Weights    map[string]float64 `json:"weights"`
	Thresholds map[string]int     `json:"thresholds"`
	CreatedAt  time.Time          `json:"created_at"`
	UpdatedAt  time.Time          `json:"updated_at"`
}

// DefaultWeights returns the default scoring factor weights.
func DefaultWeights() map[string]float64 {
	return map[string]float64{
		"payment_recency": 0.3,
		"mrr_trend":       0.2,
		"failed_payments": 0.2,
		"support_tickets": 0.15,
		"engagement":      0.15,
	}
}

// DefaultThresholds returns the default risk level thresholds.
func DefaultThresholds() map[string]int {
	return map[string]int{
		"green":  70,
		"yellow": 40,
	}
}

// ValidateWeights checks that weights sum to 1.0 and each is in [0.0, 1.0].
func ValidateWeights(weights map[string]float64) error {
	if len(weights) == 0 {
		return fmt.Errorf("weights cannot be empty")
	}

	var sum float64
	for name, w := range weights {
		if w < 0.0 || w > 1.0 {
			return fmt.Errorf("weight %q must be between 0.0 and 1.0, got %f", name, w)
		}
		sum += w
	}

	if math.Abs(sum-1.0) > 0.001 {
		return fmt.Errorf("weights must sum to 1.0, got %f", sum)
	}

	return nil
}

// ValidateThresholds checks that green > yellow > 0.
func ValidateThresholds(thresholds map[string]int) error {
	green, ok := thresholds["green"]
	if !ok {
		return fmt.Errorf("threshold 'green' is required")
	}
	yellow, ok := thresholds["yellow"]
	if !ok {
		return fmt.Errorf("threshold 'yellow' is required")
	}
	if green <= yellow {
		return fmt.Errorf("green threshold (%d) must be greater than yellow threshold (%d)", green, yellow)
	}
	if yellow <= 0 {
		return fmt.Errorf("yellow threshold must be greater than 0, got %d", yellow)
	}
	if green > 100 {
		return fmt.Errorf("green threshold must be <= 100, got %d", green)
	}
	return nil
}

// ScoringConfigRepository handles scoring_configs database operations.
type ScoringConfigRepository struct {
	pool *pgxpool.Pool
}

// NewScoringConfigRepository creates a new ScoringConfigRepository.
func NewScoringConfigRepository(pool *pgxpool.Pool) *ScoringConfigRepository {
	return &ScoringConfigRepository{pool: pool}
}

// GetByOrgID returns the scoring config for an org, or nil if none exists.
func (r *ScoringConfigRepository) GetByOrgID(ctx context.Context, orgID uuid.UUID) (*ScoringConfig, error) {
	query := `
		SELECT id, org_id, weights, thresholds, created_at, updated_at
		FROM scoring_configs
		WHERE org_id = $1`

	sc := &ScoringConfig{}
	var weightsJSON, thresholdsJSON []byte
	err := r.pool.QueryRow(ctx, query, orgID).Scan(
		&sc.ID, &sc.OrgID, &weightsJSON, &thresholdsJSON, &sc.CreatedAt, &sc.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get scoring config: %w", err)
	}

	if err := json.Unmarshal(weightsJSON, &sc.Weights); err != nil {
		return nil, fmt.Errorf("unmarshal weights: %w", err)
	}
	if err := json.Unmarshal(thresholdsJSON, &sc.Thresholds); err != nil {
		return nil, fmt.Errorf("unmarshal thresholds: %w", err)
	}

	return sc, nil
}

// Upsert creates or updates a scoring config for an org.
func (r *ScoringConfigRepository) Upsert(ctx context.Context, sc *ScoringConfig) error {
	weightsJSON, err := json.Marshal(sc.Weights)
	if err != nil {
		return fmt.Errorf("marshal weights: %w", err)
	}
	thresholdsJSON, err := json.Marshal(sc.Thresholds)
	if err != nil {
		return fmt.Errorf("marshal thresholds: %w", err)
	}

	query := `
		INSERT INTO scoring_configs (org_id, weights, thresholds)
		VALUES ($1, $2, $3)
		ON CONFLICT (org_id) DO UPDATE SET
			weights = EXCLUDED.weights,
			thresholds = EXCLUDED.thresholds,
			updated_at = NOW()
		RETURNING id, created_at, updated_at`

	return r.pool.QueryRow(ctx, query, sc.OrgID, weightsJSON, thresholdsJSON).Scan(
		&sc.ID, &sc.CreatedAt, &sc.UpdatedAt,
	)
}

// CreateDefault creates a scoring config with default weights and thresholds for an org.
func (r *ScoringConfigRepository) CreateDefault(ctx context.Context, orgID uuid.UUID) (*ScoringConfig, error) {
	sc := &ScoringConfig{
		OrgID:      orgID,
		Weights:    DefaultWeights(),
		Thresholds: DefaultThresholds(),
	}
	if err := r.Upsert(ctx, sc); err != nil {
		return nil, fmt.Errorf("create default scoring config: %w", err)
	}
	return sc, nil
}
