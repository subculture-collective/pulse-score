package service

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/google/uuid"
	"golang.org/x/sync/errgroup"

	"github.com/onnwee/pulse-score/internal/repository"
)

// DashboardService handles dashboard analytics.
type DashboardService struct {
	customerRepo    *repository.CustomerRepository
	healthScoreRepo *repository.HealthScoreRepository
}

// NewDashboardService creates a new DashboardService.
func NewDashboardService(
	cr *repository.CustomerRepository,
	hsr *repository.HealthScoreRepository,
) *DashboardService {
	return &DashboardService{
		customerRepo:    cr,
		healthScoreRepo: hsr,
	}
}

// DashboardSummary is the response for the dashboard summary endpoint.
type DashboardSummary struct {
	TotalCustomers    int          `json:"total_customers"`
	RiskDistribution  RiskDist     `json:"risk_distribution"`
	TotalMRRCents     int64        `json:"total_mrr_cents"`
	MRRChange30DCents int64        `json:"mrr_change_30d_cents"`
	AtRiskCount       int          `json:"at_risk_count"`
	AtRiskChange7D    int          `json:"at_risk_change_7d"`
	AvgHealthScore    float64      `json:"avg_health_score"`
	ScoreChange7D     float64      `json:"score_change_7d"`
}

// RiskDist holds risk distribution counts.
type RiskDist struct {
	Green  int `json:"green"`
	Yellow int `json:"yellow"`
	Red    int `json:"red"`
}

// GetSummary returns dashboard summary stats for an org.
func (s *DashboardService) GetSummary(ctx context.Context, orgID uuid.UUID) (*DashboardSummary, error) {
	var (
		totalCustomers int
		totalMRR       int64
		riskDist       *repository.RiskDistribution
		avgScore       float64
		atRiskCount    int
		avgScore7DAgo  float64
		atRisk7DAgo    int
	)

	now := time.Now()
	sevenDaysAgo := now.AddDate(0, 0, -7)

	g, gctx := errgroup.WithContext(ctx)

	g.Go(func() error {
		var err error
		totalCustomers, err = s.customerRepo.CountByOrg(gctx, orgID)
		return err
	})

	g.Go(func() error {
		var err error
		totalMRR, err = s.customerRepo.TotalMRRByOrg(gctx, orgID)
		return err
	})

	g.Go(func() error {
		var err error
		riskDist, err = s.healthScoreRepo.GetRiskDistribution(gctx, orgID)
		return err
	})

	g.Go(func() error {
		var err error
		avgScore, err = s.healthScoreRepo.GetAverageScore(gctx, orgID)
		return err
	})

	g.Go(func() error {
		var err error
		atRiskCount, err = s.healthScoreRepo.CountAtRisk(gctx, orgID)
		return err
	})

	g.Go(func() error {
		var err error
		avgScore7DAgo, err = s.healthScoreRepo.GetAverageScoreAt(gctx, orgID, sevenDaysAgo)
		return err
	})

	g.Go(func() error {
		var err error
		atRisk7DAgo, err = s.healthScoreRepo.CountAtRiskAt(gctx, orgID, sevenDaysAgo)
		return err
	})

	if err := g.Wait(); err != nil {
		return nil, fmt.Errorf("dashboard summary: %w", err)
	}

	dist := RiskDist{}
	if riskDist != nil {
		dist = RiskDist{
			Green:  riskDist.Green,
			Yellow: riskDist.Yellow,
			Red:    riskDist.Red,
		}
	}

	return &DashboardSummary{
		TotalCustomers:    totalCustomers,
		RiskDistribution:  dist,
		TotalMRRCents:     totalMRR,
		MRRChange30DCents: 0, // MRR historical tracking not yet available
		AtRiskCount:       atRiskCount,
		AtRiskChange7D:    atRiskCount - atRisk7DAgo,
		AvgHealthScore:    math.Round(avgScore*100) / 100,
		ScoreChange7D:     math.Round((avgScore-avgScore7DAgo)*100) / 100,
	}, nil
}

// ScoreDistributionResponse is the response for the score distribution endpoint.
type ScoreDistributionResponse struct {
	Buckets       []repository.ScoreBucket `json:"buckets"`
	RiskBreakdown RiskBreakdownResponse    `json:"risk_breakdown"`
	AverageScore  float64                  `json:"average_score"`
	MedianScore   float64                  `json:"median_score"`
}

// RiskBreakdownEntry holds count and percentage for a risk level.
type RiskBreakdownEntry struct {
	Count   int     `json:"count"`
	Percent float64 `json:"pct"`
}

// RiskBreakdownResponse holds the risk breakdown by level.
type RiskBreakdownResponse struct {
	Green  RiskBreakdownEntry `json:"green"`
	Yellow RiskBreakdownEntry `json:"yellow"`
	Red    RiskBreakdownEntry `json:"red"`
}

// GetScoreDistribution returns score distribution data for an org.
func (s *DashboardService) GetScoreDistribution(ctx context.Context, orgID uuid.UUID) (*ScoreDistributionResponse, error) {
	var (
		buckets  []repository.ScoreBucket
		riskDist *repository.RiskDistribution
		avgScore float64
		median   float64
	)

	g, gctx := errgroup.WithContext(ctx)

	g.Go(func() error {
		var err error
		buckets, err = s.healthScoreRepo.GetScoreBuckets(gctx, orgID)
		return err
	})

	g.Go(func() error {
		var err error
		riskDist, err = s.healthScoreRepo.GetRiskDistribution(gctx, orgID)
		return err
	})

	g.Go(func() error {
		var err error
		avgScore, err = s.healthScoreRepo.GetAverageScore(gctx, orgID)
		return err
	})

	g.Go(func() error {
		var err error
		median, err = s.healthScoreRepo.GetMedianScore(gctx, orgID)
		return err
	})

	if err := g.Wait(); err != nil {
		return nil, fmt.Errorf("score distribution: %w", err)
	}

	total := 0
	if riskDist != nil {
		total = riskDist.Green + riskDist.Yellow + riskDist.Red
	}

	breakdown := RiskBreakdownResponse{}
	if riskDist != nil && total > 0 {
		breakdown = RiskBreakdownResponse{
			Green:  RiskBreakdownEntry{Count: riskDist.Green, Percent: math.Round(float64(riskDist.Green)/float64(total)*10000) / 100},
			Yellow: RiskBreakdownEntry{Count: riskDist.Yellow, Percent: math.Round(float64(riskDist.Yellow)/float64(total)*10000) / 100},
			Red:    RiskBreakdownEntry{Count: riskDist.Red, Percent: math.Round(float64(riskDist.Red)/float64(total)*10000) / 100},
		}
	}

	return &ScoreDistributionResponse{
		Buckets:       buckets,
		RiskBreakdown: breakdown,
		AverageScore:  math.Round(avgScore*100) / 100,
		MedianScore:   math.Round(median*100) / 100,
	}, nil
}
