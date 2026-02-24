package handler

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/onnwee/pulse-score/internal/auth"
	"github.com/onnwee/pulse-score/internal/service"
)

// IntegrationHandler provides integration management HTTP endpoints.
type IntegrationHandler struct {
	integrationService *service.IntegrationService
}

// NewIntegrationHandler creates a new IntegrationHandler.
func NewIntegrationHandler(integrationService *service.IntegrationService) *IntegrationHandler {
	return &IntegrationHandler{integrationService: integrationService}
}

// List handles GET /api/v1/integrations.
func (h *IntegrationHandler) List(w http.ResponseWriter, r *http.Request) {
	orgID, ok := auth.GetOrgID(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, errorResponse("unauthorized"))
		return
	}

	summaries, err := h.integrationService.List(r.Context(), orgID)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"integrations": summaries})
}

// GetStatus handles GET /api/v1/integrations/{provider}/status.
func (h *IntegrationHandler) GetStatus(w http.ResponseWriter, r *http.Request) {
	orgID, ok := auth.GetOrgID(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, errorResponse("unauthorized"))
		return
	}

	provider := chi.URLParam(r, "provider")
	if provider == "" {
		writeJSON(w, http.StatusBadRequest, errorResponse("provider is required"))
		return
	}

	status, err := h.integrationService.GetStatus(r.Context(), orgID, provider)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, status)
}

// TriggerSync handles POST /api/v1/integrations/{provider}/sync.
func (h *IntegrationHandler) TriggerSync(w http.ResponseWriter, r *http.Request) {
	orgID, ok := auth.GetOrgID(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, errorResponse("unauthorized"))
		return
	}

	provider := chi.URLParam(r, "provider")
	if provider == "" {
		writeJSON(w, http.StatusBadRequest, errorResponse("provider is required"))
		return
	}

	if err := h.integrationService.TriggerSync(r.Context(), orgID, provider); err != nil {
		handleServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusAccepted, map[string]string{"status": "sync_started"})
}

// Disconnect handles DELETE /api/v1/integrations/{provider}.
func (h *IntegrationHandler) Disconnect(w http.ResponseWriter, r *http.Request) {
	orgID, ok := auth.GetOrgID(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, errorResponse("unauthorized"))
		return
	}

	provider := chi.URLParam(r, "provider")
	if provider == "" {
		writeJSON(w, http.StatusBadRequest, errorResponse("provider is required"))
		return
	}

	if err := h.integrationService.Disconnect(r.Context(), orgID, provider); err != nil {
		handleServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusNoContent, nil)
}
