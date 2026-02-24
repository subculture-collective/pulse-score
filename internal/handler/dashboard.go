package handler

import (
	"net/http"

	"github.com/onnwee/pulse-score/internal/auth"
)

// DashboardHandler provides dashboard HTTP endpoints.
type DashboardHandler struct {
	dashboardService dashboardServicer
}

// NewDashboardHandler creates a new DashboardHandler.
func NewDashboardHandler(ds dashboardServicer) *DashboardHandler {
	return &DashboardHandler{dashboardService: ds}
}

// GetSummary handles GET /api/v1/dashboard/summary.
func (h *DashboardHandler) GetSummary(w http.ResponseWriter, r *http.Request) {
	orgID, ok := auth.GetOrgID(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, errorResponse("unauthorized"))
		return
	}

	summary, err := h.dashboardService.GetSummary(r.Context(), orgID)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, summary)
}

// GetScoreDistribution handles GET /api/v1/dashboard/score-distribution.
func (h *DashboardHandler) GetScoreDistribution(w http.ResponseWriter, r *http.Request) {
	orgID, ok := auth.GetOrgID(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, errorResponse("unauthorized"))
		return
	}

	dist, err := h.dashboardService.GetScoreDistribution(r.Context(), orgID)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, dist)
}
