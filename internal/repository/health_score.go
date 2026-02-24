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

// GetScoreAtTime retrieves the closest historical score for a customer at or before the given time.
func (r *HealthScoreRepository) GetScoreAtTime(ctx context.Context, customerID, orgID uuid.UUID, at time.Time) (*HealthScore, error) {
	query := `
		SELECT id, org_id, customer_id, overall_score, risk_level, factors, calculated_at, created_at, created_at
		FROM health_score_history
		WHERE customer_id = $1 AND org_id = $2 AND calculated_at <= $3
		ORDER BY calculated_at DESC
		LIMIT 1`

	hs := &HealthScore{}
	var factorsJSON []byte
	err := r.pool.QueryRow(ctx, query, customerID, orgID, at).Scan(
		&hs.ID, &hs.OrgID, &hs.CustomerID, &hs.OverallScore, &hs.RiskLevel,
		&factorsJSON, &hs.CalculatedAt, &hs.CreatedAt, &hs.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get score at time: %w", err)
	}
	if err := json.Unmarshal(factorsJSON, &hs.Factors); err != nil {
		return nil, fmt.Errorf("unmarshal factors: %w", err)
	}
	return hs, nil
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

// GetAverageScore returns the average overall score for an org.
func (r *HealthScoreRepository) GetAverageScore(ctx context.Context, orgID uuid.UUID) (float64, error) {
	query := `SELECT COALESCE(AVG(overall_score), 0) FROM health_scores WHERE org_id = $1`
	var avg float64
	err := r.pool.QueryRow(ctx, query, orgID).Scan(&avg)
	if err != nil {
		return 0, fmt.Errorf("get average score: %w", err)
	}
	return avg, nil
}

// GetMedianScore returns the median overall score for an org.
func (r *HealthScoreRepository) GetMedianScore(ctx context.Context, orgID uuid.UUID) (float64, error) {
	query := `SELECT COALESCE(PERCENTILE_CONT(0.5) WITHIN GROUP (ORDER BY overall_score), 0) FROM health_scores WHERE org_id = $1`
	var median float64
	err := r.pool.QueryRow(ctx, query, orgID).Scan(&median)
	if err != nil {
		return 0, fmt.Errorf("get median score: %w", err)
	}
	return median, nil
}

// RiskDistribution holds risk level counts.
type RiskDistribution struct {
	Green  int
	Yellow int
	Red    int
}

// GetRiskDistribution returns counts of customers by risk level.
func (r *HealthScoreRepository) GetRiskDistribution(ctx context.Context, orgID uuid.UUID) (*RiskDistribution, error) {
	counts, err := r.CountByRiskLevel(ctx, orgID)
	if err != nil {
		return nil, err
	}
	return &RiskDistribution{
		Green:  counts["low"],
		Yellow: counts["medium"],
		Red:    counts["high"] + counts["critical"],
	}, nil
}

// GetAverageScoreAt returns the average score from history at a given point in time.
func (r *HealthScoreRepository) GetAverageScoreAt(ctx context.Context, orgID uuid.UUID, at time.Time) (float64, error) {
	query := `
		SELECT COALESCE(AVG(overall_score), 0)
		FROM (
			SELECT DISTINCT ON (customer_id) overall_score
			FROM health_score_history
			WHERE org_id = $1 AND calculated_at <= $2
			ORDER BY customer_id, calculated_at DESC
		) sub`
	var avg float64
	err := r.pool.QueryRow(ctx, query, orgID, at).Scan(&avg)
	if err != nil {
		return 0, fmt.Errorf("get average score at: %w", err)
	}
	return avg, nil
}

// CountAtRisk returns the number of customers with high/critical risk level.
func (r *HealthScoreRepository) CountAtRisk(ctx context.Context, orgID uuid.UUID) (int, error) {
	query := `SELECT COUNT(*) FROM health_scores WHERE org_id = $1 AND risk_level IN ('high', 'critical')`
	var count int
	err := r.pool.QueryRow(ctx, query, orgID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count at risk: %w", err)
	}
	return count, nil
}

// CountAtRiskAt returns the at-risk count from history at a given point in time.
func (r *HealthScoreRepository) CountAtRiskAt(ctx context.Context, orgID uuid.UUID, at time.Time) (int, error) {
	query := `
		SELECT COUNT(*)
		FROM (
			SELECT DISTINCT ON (customer_id) risk_level
			FROM health_score_history
			WHERE org_id = $1 AND calculated_at <= $2
			ORDER BY customer_id, calculated_at DESC
		) sub
		WHERE risk_level IN ('high', 'critical')`
	var count int
	err := r.pool.QueryRow(ctx, query, orgID, at).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count at risk at: %w", err)
	}
	return count, nil
}

// ScoreBucket represents a bucket in the score distribution histogram.
type ScoreBucket struct {
	Range string `json:"range"`
	Count int    `json:"count"`
}

// GetScoreBuckets returns score buckets for histogram display.
func (r *HealthScoreRepository) GetScoreBuckets(ctx context.Context, orgID uuid.UUID) ([]ScoreBucket, error) {
	query := `
		SELECT
			CASE
				WHEN overall_score BETWEEN 0 AND 10 THEN '0-10'
				WHEN overall_score BETWEEN 11 AND 20 THEN '11-20'
				WHEN overall_score BETWEEN 21 AND 30 THEN '21-30'
				WHEN overall_score BETWEEN 31 AND 40 THEN '31-40'
				WHEN overall_score BETWEEN 41 AND 50 THEN '41-50'
				WHEN overall_score BETWEEN 51 AND 60 THEN '51-60'
				WHEN overall_score BETWEEN 61 AND 70 THEN '61-70'
				WHEN overall_score BETWEEN 71 AND 80 THEN '71-80'
				WHEN overall_score BETWEEN 81 AND 90 THEN '81-90'
				WHEN overall_score BETWEEN 91 AND 100 THEN '91-100'
			END AS score_range,
			COUNT(*) as count
		FROM health_scores
		WHERE org_id = $1
		GROUP BY score_range
		ORDER BY MIN(overall_score)`

	rows, err := r.pool.Query(ctx, query, orgID)
	if err != nil {
		return nil, fmt.Errorf("get score buckets: %w", err)
	}
	defer rows.Close()

	bucketMap := make(map[string]int)
	for rows.Next() {
		var r string
		var c int
		if err := rows.Scan(&r, &c); err != nil {
			return nil, fmt.Errorf("scan bucket: %w", err)
		}
		bucketMap[r] = c
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}

	// Return all 10 buckets, filling in zeros
	allRanges := []string{"0-10", "11-20", "21-30", "31-40", "41-50", "51-60", "61-70", "71-80", "81-90", "91-100"}
	buckets := make([]ScoreBucket, len(allRanges))
	for i, r := range allRanges {
		buckets[i] = ScoreBucket{Range: r, Count: bucketMap[r]}
	}
	return buckets, nil
}
