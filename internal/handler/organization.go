package handler

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/onnwee/pulse-score/internal/auth"
	"github.com/onnwee/pulse-score/internal/service"
)

// OrganizationHandler provides organization HTTP endpoints.
type OrganizationHandler struct {
	orgService organizationServicer
}

// NewOrganizationHandler creates a new OrganizationHandler.
func NewOrganizationHandler(orgService organizationServicer) *OrganizationHandler {
	return &OrganizationHandler{orgService: orgService}
}

// GetCurrent handles GET /api/v1/organizations/current.
func (h *OrganizationHandler) GetCurrent(w http.ResponseWriter, r *http.Request) {
	orgID, ok := auth.GetOrgID(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, errorResponse("unauthorized"))
		return
	}

	resp, err := h.orgService.GetCurrent(r.Context(), orgID)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, resp)
}

// UpdateCurrent handles PATCH /api/v1/organizations/current.
func (h *OrganizationHandler) UpdateCurrent(w http.ResponseWriter, r *http.Request) {
	orgID, ok := auth.GetOrgID(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, errorResponse("unauthorized"))
		return
	}

	var req service.UpdateOrgRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse("invalid request body"))
		return
	}

	resp, err := h.orgService.UpdateCurrent(r.Context(), orgID, req)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, resp)
}

// Create handles POST /api/v1/organizations.
func (h *OrganizationHandler) Create(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.GetUserID(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, errorResponse("unauthorized"))
		return
	}

	var req service.CreateOrgRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse("invalid request body"))
		return
	}

	resp, err := h.orgService.Create(r.Context(), userID, req)
	if err != nil {
		var validationErr *service.ValidationError
		if errors.As(err, &validationErr) {
			writeJSON(w, http.StatusUnprocessableEntity, map[string]any{
				"error": validationErr.Message,
				"field": validationErr.Field,
			})
			return
		}
		slog.Error("create org error", "error", err)
		writeJSON(w, http.StatusInternalServerError, errorResponse("internal server error"))
		return
	}

	writeJSON(w, http.StatusCreated, resp)
}
