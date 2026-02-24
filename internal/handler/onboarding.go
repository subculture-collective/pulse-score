package handler

import (
	"encoding/json"
	"net/http"

	"github.com/onnwee/pulse-score/internal/auth"
	"github.com/onnwee/pulse-score/internal/service"
)

// OnboardingHandler provides onboarding HTTP endpoints.
type OnboardingHandler struct {
	onboardingService *service.OnboardingService
}

// NewOnboardingHandler creates a new OnboardingHandler.
func NewOnboardingHandler(onboardingService *service.OnboardingService) *OnboardingHandler {
	return &OnboardingHandler{onboardingService: onboardingService}
}

// GetStatus handles GET /api/v1/onboarding/status.
func (h *OnboardingHandler) GetStatus(w http.ResponseWriter, r *http.Request) {
	orgID, ok := auth.GetOrgID(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, errorResponse("unauthorized"))
		return
	}

	status, err := h.onboardingService.GetStatus(r.Context(), orgID)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, status)
}

// UpdateStatus handles PATCH /api/v1/onboarding/status.
func (h *OnboardingHandler) UpdateStatus(w http.ResponseWriter, r *http.Request) {
	orgID, ok := auth.GetOrgID(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, errorResponse("unauthorized"))
		return
	}

	var req service.UpdateOnboardingStatusRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse("invalid request body"))
		return
	}

	status, err := h.onboardingService.UpdateStatus(r.Context(), orgID, req)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, status)
}

// Complete handles POST /api/v1/onboarding/complete.
func (h *OnboardingHandler) Complete(w http.ResponseWriter, r *http.Request) {
	orgID, ok := auth.GetOrgID(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, errorResponse("unauthorized"))
		return
	}

	status, err := h.onboardingService.Complete(r.Context(), orgID)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, status)
}

// Reset handles POST /api/v1/onboarding/reset.
func (h *OnboardingHandler) Reset(w http.ResponseWriter, r *http.Request) {
	orgID, ok := auth.GetOrgID(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, errorResponse("unauthorized"))
		return
	}

	status, err := h.onboardingService.Reset(r.Context(), orgID)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, status)
}

// Analytics handles GET /api/v1/onboarding/analytics.
func (h *OnboardingHandler) Analytics(w http.ResponseWriter, r *http.Request) {
	orgID, ok := auth.GetOrgID(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, errorResponse("unauthorized"))
		return
	}

	analytics, err := h.onboardingService.GetAnalytics(r.Context(), orgID)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, analytics)
}
