package handler

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/onnwee/pulse-score/internal/auth"
	"github.com/onnwee/pulse-score/internal/service"
)

// AlertRuleHandler provides alert rule HTTP endpoints.
type AlertRuleHandler struct {
	alertService *service.AlertRuleService
}

// NewAlertRuleHandler creates a new AlertRuleHandler.
func NewAlertRuleHandler(alertService *service.AlertRuleService) *AlertRuleHandler {
	return &AlertRuleHandler{alertService: alertService}
}

// List handles GET /api/v1/alerts/rules.
func (h *AlertRuleHandler) List(w http.ResponseWriter, r *http.Request) {
	orgID, ok := auth.GetOrgID(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, errorResponse("unauthorized"))
		return
	}

	rules, err := h.alertService.List(r.Context(), orgID)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"rules": rules})
}

// Get handles GET /api/v1/alerts/rules/{id}.
func (h *AlertRuleHandler) Get(w http.ResponseWriter, r *http.Request) {
	orgID, ok := auth.GetOrgID(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, errorResponse("unauthorized"))
		return
	}

	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse("invalid alert rule ID"))
		return
	}

	rule, err := h.alertService.GetByID(r.Context(), id, orgID)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, rule)
}

// Create handles POST /api/v1/alerts/rules.
func (h *AlertRuleHandler) Create(w http.ResponseWriter, r *http.Request) {
	orgID, ok := auth.GetOrgID(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, errorResponse("unauthorized"))
		return
	}

	userID, ok := auth.GetUserID(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, errorResponse("unauthorized"))
		return
	}

	var req service.CreateAlertRuleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse("invalid request body"))
		return
	}

	rule, err := h.alertService.Create(r.Context(), orgID, userID, req)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, rule)
}

// Update handles PATCH /api/v1/alerts/rules/{id}.
func (h *AlertRuleHandler) Update(w http.ResponseWriter, r *http.Request) {
	orgID, ok := auth.GetOrgID(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, errorResponse("unauthorized"))
		return
	}

	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse("invalid alert rule ID"))
		return
	}

	var req service.UpdateAlertRuleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse("invalid request body"))
		return
	}

	rule, err := h.alertService.Update(r.Context(), id, orgID, req)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, rule)
}

// Delete handles DELETE /api/v1/alerts/rules/{id}.
func (h *AlertRuleHandler) Delete(w http.ResponseWriter, r *http.Request) {
	orgID, ok := auth.GetOrgID(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, errorResponse("unauthorized"))
		return
	}

	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse("invalid alert rule ID"))
		return
	}

	if err := h.alertService.Delete(r.Context(), id, orgID); err != nil {
		handleServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusNoContent, nil)
}
