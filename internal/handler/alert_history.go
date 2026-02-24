package handler

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/onnwee/pulse-score/internal/auth"
	"github.com/onnwee/pulse-score/internal/repository"
)

// AlertHistoryHandler provides alert history HTTP endpoints.
type AlertHistoryHandler struct {
	alertHistoryRepo *repository.AlertHistoryRepository
}

// NewAlertHistoryHandler creates a new AlertHistoryHandler.
func NewAlertHistoryHandler(alertHistoryRepo *repository.AlertHistoryRepository) *AlertHistoryHandler {
	return &AlertHistoryHandler{alertHistoryRepo: alertHistoryRepo}
}

// List handles GET /api/v1/alerts/history.
func (h *AlertHistoryHandler) List(w http.ResponseWriter, r *http.Request) {
	orgID, ok := auth.GetOrgID(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, errorResponse("unauthorized"))
		return
	}

	status := r.URL.Query().Get("status")
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	if limit <= 0 || limit > 100 {
		limit = 25
	}
	if offset < 0 {
		offset = 0
	}

	history, total, err := h.alertHistoryRepo.ListByOrg(r.Context(), orgID, status, limit, offset)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"history": history,
		"total":   total,
		"limit":   limit,
		"offset":  offset,
	})
}

// Stats handles GET /api/v1/alerts/stats.
func (h *AlertHistoryHandler) Stats(w http.ResponseWriter, r *http.Request) {
	orgID, ok := auth.GetOrgID(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, errorResponse("unauthorized"))
		return
	}

	counts, err := h.alertHistoryRepo.CountByStatus(r.Context(), orgID)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, counts)
}

// ListByRule handles GET /api/v1/alerts/rules/{id}/history.
func (h *AlertHistoryHandler) ListByRule(w http.ResponseWriter, r *http.Request) {
	_, ok := auth.GetOrgID(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, errorResponse("unauthorized"))
		return
	}

	ruleID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse("invalid rule ID"))
		return
	}

	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	if limit <= 0 || limit > 100 {
		limit = 25
	}
	if offset < 0 {
		offset = 0
	}

	history, err := h.alertHistoryRepo.ListByRule(r.Context(), ruleID, limit, offset)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"history": history,
	})
}
