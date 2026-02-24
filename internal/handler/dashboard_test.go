package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"

	"github.com/onnwee/pulse-score/internal/auth"
	"github.com/onnwee/pulse-score/internal/service"
)

type mockDashboardService struct {
	getSummaryFn            func(ctx context.Context, orgID uuid.UUID) (*service.DashboardSummary, error)
	getScoreDistributionFn func(ctx context.Context, orgID uuid.UUID) (*service.ScoreDistributionResponse, error)
}

func (m *mockDashboardService) GetSummary(ctx context.Context, orgID uuid.UUID) (*service.DashboardSummary, error) {
	return m.getSummaryFn(ctx, orgID)
}

func (m *mockDashboardService) GetScoreDistribution(ctx context.Context, orgID uuid.UUID) (*service.ScoreDistributionResponse, error) {
	return m.getScoreDistributionFn(ctx, orgID)
}

func TestDashboardGetSummary_Unauthorized(t *testing.T) {
	h := NewDashboardHandler(&mockDashboardService{})
	req := httptest.NewRequest(http.MethodGet, "/api/v1/dashboard/summary", nil)
	rr := httptest.NewRecorder()

	h.GetSummary(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rr.Code)
	}
}

func TestDashboardGetSummary_Success(t *testing.T) {
	orgID := uuid.New()
	mock := &mockDashboardService{
		getSummaryFn: func(ctx context.Context, oID uuid.UUID) (*service.DashboardSummary, error) {
			if oID != orgID {
				t.Errorf("expected orgID %s, got %s", orgID, oID)
			}
			return &service.DashboardSummary{
				TotalCustomers: 42,
				TotalMRRCents:  100000,
				AvgHealthScore: 75.5,
			}, nil
		},
	}

	h := NewDashboardHandler(mock)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/dashboard/summary", nil)
	req = req.WithContext(auth.WithOrgID(req.Context(), orgID))
	rr := httptest.NewRecorder()

	h.GetSummary(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}

	var resp service.DashboardSummary
	json.NewDecoder(rr.Body).Decode(&resp)
	if resp.TotalCustomers != 42 {
		t.Errorf("expected 42 customers, got %d", resp.TotalCustomers)
	}
}

func TestDashboardGetSummary_ServiceError(t *testing.T) {
	orgID := uuid.New()
	mock := &mockDashboardService{
		getSummaryFn: func(ctx context.Context, oID uuid.UUID) (*service.DashboardSummary, error) {
			return nil, fmt.Errorf("db connection failed")
		},
	}

	h := NewDashboardHandler(mock)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/dashboard/summary", nil)
	req = req.WithContext(auth.WithOrgID(req.Context(), orgID))
	rr := httptest.NewRecorder()

	h.GetSummary(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", rr.Code)
	}
}

func TestDashboardGetScoreDistribution_Unauthorized(t *testing.T) {
	h := NewDashboardHandler(&mockDashboardService{})
	req := httptest.NewRequest(http.MethodGet, "/api/v1/dashboard/score-distribution", nil)
	rr := httptest.NewRecorder()

	h.GetScoreDistribution(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rr.Code)
	}
}

func TestDashboardGetScoreDistribution_Success(t *testing.T) {
	orgID := uuid.New()
	mock := &mockDashboardService{
		getScoreDistributionFn: func(ctx context.Context, oID uuid.UUID) (*service.ScoreDistributionResponse, error) {
			if oID != orgID {
				t.Errorf("expected orgID %s, got %s", orgID, oID)
			}
			return &service.ScoreDistributionResponse{
				AverageScore: 72,
				MedianScore:  75,
			}, nil
		},
	}

	h := NewDashboardHandler(mock)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/dashboard/score-distribution", nil)
	req = req.WithContext(auth.WithOrgID(req.Context(), orgID))
	rr := httptest.NewRecorder()

	h.GetScoreDistribution(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
}
