package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// HealthScore represents a health_scores row.
type HealthScore struct {
	ID           uuid.UUID          `json:"id"`
	OrgID        uuid.UUID          `json:"org_id"`
	CustomerID   uuid.UUID          `json:"customer_id"`
	OverallScore int                `json:"overall_score"`
	RiskLevel    string             `json:"risk_level"`
	Factors      map[string]float64 `json:"factors"`
	CalculatedAt time.Time          `json:"calculated_at"`
	CreatedAt    time.Time          `json:"created_at"`
	UpdatedAt    time.Time          `json:"updated_at"`
}

// HealthScoreFilters holds filter options for listing health scores.
type HealthScoreFilters struct {
	RiskLevel string
	Limit     int
	Offset    int
}

// HealthScoreRepository handles health_scores database operations.
type HealthScoreRepository struct {
	pool *pgxpool.Pool
}

// NewHealthScoreRepository creates a new HealthScoreRepository.
func NewHealthScoreRepository(pool *pgxpool.Pool) *HealthScoreRepository {
	return &HealthScoreRepository{pool: pool}
}

// UpsertCurrent creates or updates the current health score for a customer.
func (r *HealthScoreRepository) UpsertCurrent(ctx context.Context, score *HealthScore) error {
	factorsJSON, err := json.Marshal(score.Factors)
	if err != nil {
		return fmt.Errorf("marshal factors: %w", err)
	}

	query := `
		INSERT INTO health_scores (org_id, customer_id, overall_score, risk_level, factors, calculated_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (customer_id) DO UPDATE SET
			overall_score = EXCLUDED.overall_score,
			risk_level = EXCLUDED.risk_level,
			factors = EXCLUDED.factors,
			calculated_at = EXCLUDED.calculated_at
		RETURNING id, created_at, updated_at`

	return r.pool.QueryRow(ctx, query,
		score.OrgID, score.CustomerID, score.OverallScore, score.RiskLevel,
		factorsJSON, score.CalculatedAt,
	).Scan(&score.ID, &score.CreatedAt, &score.UpdatedAt)
}

// GetByCustomerID retrieves the current health score for a customer.
func (r *HealthScoreRepository) GetByCustomerID(ctx context.Context, customerID, orgID uuid.UUID) (*HealthScore, error) {
	query := `
		SELECT id, org_id, customer_id, overall_score, risk_level, factors, calculated_at, created_at, updated_at
		FROM health_scores
		WHERE customer_id = $1 AND org_id = $2`

	hs := &HealthScore{}
	var factorsJSON []byte
	err := r.pool.QueryRow(ctx, query, customerID, orgID).Scan(
		&hs.ID, &hs.OrgID, &hs.CustomerID, &hs.OverallScore, &hs.RiskLevel,
		&factorsJSON, &hs.CalculatedAt, &hs.CreatedAt, &hs.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get health score: %w", err)
	}

	if err := json.Unmarshal(factorsJSON, &hs.Factors); err != nil {
		return nil, fmt.Errorf("unmarshal factors: %w", err)
	}
	return hs, nil
}

// ListByOrg retrieves health scores for an org with optional filters.
func (r *HealthScoreRepository) ListByOrg(ctx context.Context, orgID uuid.UUID, filters HealthScoreFilters) ([]*HealthScore, error) {
	query := `
		SELECT id, org_id, customer_id, overall_score, risk_level, factors, calculated_at, created_at, updated_at
		FROM health_scores
		WHERE org_id = $1`
	args := []any{orgID}
	argIdx := 2

	if filters.RiskLevel != "" {
		query += fmt.Sprintf(" AND risk_level = $%d", argIdx)
		args = append(args, filters.RiskLevel)
		argIdx++
	}

	query += " ORDER BY overall_score ASC"

	limit := filters.Limit
	if limit <= 0 {
		limit = 100
	}
	query += fmt.Sprintf(" LIMIT $%d", argIdx)
	args = append(args, limit)
	argIdx++

	if filters.Offset > 0 {
		query += fmt.Sprintf(" OFFSET $%d", argIdx)
		args = append(args, filters.Offset)
	}

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list health scores: %w", err)
	}
	defer rows.Close()

	var scores []*HealthScore
	for rows.Next() {
		hs := &HealthScore{}
		var factorsJSON []byte
		if err := rows.Scan(
			&hs.ID, &hs.OrgID, &hs.CustomerID, &hs.OverallScore, &hs.RiskLevel,
			&factorsJSON, &hs.CalculatedAt, &hs.CreatedAt, &hs.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan health score: %w", err)
		}
		if err := json.Unmarshal(factorsJSON, &hs.Factors); err != nil {
			return nil, fmt.Errorf("unmarshal factors: %w", err)
		}
		scores = append(scores, hs)
	}
	return scores, rows.Err()
}

// InsertHistory appends a score to the health_score_history table.
func (r *HealthScoreRepository) InsertHistory(ctx context.Context, score *HealthScore) error {
	factorsJSON, err := json.Marshal(score.Factors)
	if err != nil {
		return fmt.Errorf("marshal factors: %w", err)
	}

	query := `
		INSERT INTO health_score_history (org_id, customer_id, overall_score, risk_level, factors, calculated_at)
		VALUES ($1, $2, $3, $4, $5, $6)`

	_, err = r.pool.Exec(ctx, query,
		score.OrgID, score.CustomerID, score.OverallScore, score.RiskLevel,
		factorsJSON, score.CalculatedAt,
	)
	if err != nil {
		return fmt.Errorf("insert history: %w", err)
	}
	return nil
}

// GetHistory retrieves score history for a customer, ordered by calculated_at DESC.
func (r *HealthScoreRepository) GetHistory(ctx context.Context, customerID uuid.UUID, limit int) ([]*HealthScore, error) {
	if limit <= 0 {
		limit = 50
	}

	query := `
		SELECT id, org_id, customer_id, overall_score, risk_level, factors, calculated_at, created_at, created_at
		FROM health_score_history
		WHERE customer_id = $1
		ORDER BY calculated_at DESC
		LIMIT $2`

	rows, err := r.pool.Query(ctx, query, customerID, limit)
	if err != nil {
		return nil, fmt.Errorf("get history: %w", err)
	}
	defer rows.Close()

	var scores []*HealthScore
	for rows.Next() {
		hs := &HealthScore{}
		var factorsJSON []byte
		if err := rows.Scan(
			&hs.ID, &hs.OrgID, &hs.CustomerID, &hs.OverallScore, &hs.RiskLevel,
			&factorsJSON, &hs.CalculatedAt, &hs.CreatedAt, &hs.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan history: %w", err)
		}
		if err := json.Unmarshal(factorsJSON, &hs.Factors); err != nil {
			return nil, fmt.Errorf("unmarshal factors: %w", err)
		}
		scores = append(scores, hs)
	}
	return scores, rows.Err()
}

// CountByRiskLevel returns count of health scores grouped by risk level for an org.
func (r *HealthScoreRepository) CountByRiskLevel(ctx context.Context, orgID uuid.UUID) (map[string]int, error) {
	query := `
		SELECT risk_level, COUNT(*)
		FROM health_scores
		WHERE org_id = $1
		GROUP BY risk_level`

	rows, err := r.pool.Query(ctx, query, orgID)
	if err != nil {
		return nil, fmt.Errorf("count by risk level: %w", err)
	}
	defer rows.Close()

	counts := make(map[string]int)
	for rows.Next() {
		var riskLevel string
		var count int
		if err := rows.Scan(&riskLevel, &count); err != nil {
			return nil, fmt.Errorf("scan risk level count: %w", err)
		}
		counts[riskLevel] = count
	}
	return counts, rows.Err()
}

// ScoreDistribution returns all overall scores for an org (for histogram generation).
func (r *HealthScoreRepository) ScoreDistribution(ctx context.Context, orgID uuid.UUID) ([]int, error) {
	query := `SELECT overall_score FROM health_scores WHERE org_id = $1 ORDER BY overall_score`

	rows, err := r.pool.Query(ctx, query, orgID)
	if err != nil {
		return nil, fmt.Errorf("score distribution: %w", err)
	}
	defer rows.Close()

	var scores []int
	for rows.Next() {
		var s int
		if err := rows.Scan(&s); err != nil {
			return nil, fmt.Errorf("scan score: %w", err)
		}
		scores = append(scores, s)
	}
	return scores, rows.Err()
}
