package handler

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/onnwee/pulse-score/internal/auth"
	"github.com/onnwee/pulse-score/internal/service/scoring"
)

// ScoringHandler provides health scoring HTTP endpoints.
type ScoringHandler struct {
	configSvc   *scoring.ConfigService
	categorizer *scoring.RiskCategorizer
	scheduler   *scoring.ScoreScheduler
}

// NewScoringHandler creates a new ScoringHandler.
func NewScoringHandler(
	configSvc *scoring.ConfigService,
	categorizer *scoring.RiskCategorizer,
	scheduler *scoring.ScoreScheduler,
) *ScoringHandler {
	return &ScoringHandler{
		configSvc:   configSvc,
		categorizer: categorizer,
		scheduler:   scheduler,
	}
}

// GetConfig handles GET /api/v1/scoring/config.
func (h *ScoringHandler) GetConfig(w http.ResponseWriter, r *http.Request) {
	orgID, ok := auth.GetOrgID(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, errorResponse("unauthorized"))
		return
	}

	config, err := h.configSvc.GetConfig(r.Context(), orgID)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, config)
}

// UpdateConfig handles PUT /api/v1/scoring/config.
func (h *ScoringHandler) UpdateConfig(w http.ResponseWriter, r *http.Request) {
	orgID, ok := auth.GetOrgID(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, errorResponse("unauthorized"))
		return
	}

	var req scoring.UpdateConfigRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse("invalid request body"))
		return
	}

	config, err := h.configSvc.UpdateConfig(r.Context(), orgID, req)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, config)
}

// GetRiskDistribution handles GET /api/v1/scoring/risk-distribution.
func (h *ScoringHandler) GetRiskDistribution(w http.ResponseWriter, r *http.Request) {
	orgID, ok := auth.GetOrgID(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, errorResponse("unauthorized"))
		return
	}

	dist, err := h.categorizer.GetRiskDistribution(r.Context(), orgID)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, dist)
}

// GetScoreHistogram handles GET /api/v1/scoring/histogram.
func (h *ScoringHandler) GetScoreHistogram(w http.ResponseWriter, r *http.Request) {
	orgID, ok := auth.GetOrgID(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, errorResponse("unauthorized"))
		return
	}

	histogram, err := h.categorizer.GetScoreHistogram(r.Context(), orgID)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, histogram)
}

// RecalculateCustomer handles POST /api/v1/scoring/customers/{id}/recalculate.
func (h *ScoringHandler) RecalculateCustomer(w http.ResponseWriter, r *http.Request) {
	orgID, ok := auth.GetOrgID(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, errorResponse("unauthorized"))
		return
	}

	customerIDStr := chi.URLParam(r, "id")
	customerID, err := uuid.Parse(customerIDStr)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse("invalid customer ID"))
		return
	}

	if err := h.scheduler.RecalculateCustomer(r.Context(), customerID, orgID); err != nil {
		handleServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusAccepted, map[string]string{"message": "recalculation triggered"})
}
