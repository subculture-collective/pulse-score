package handler

import (
	"encoding/json"
	"net/http"

	"github.com/onnwee/pulse-score/internal/auth"
	"github.com/onnwee/pulse-score/internal/service"
)

// NotificationPreferenceHandler provides notification preference HTTP endpoints.
type NotificationPreferenceHandler struct {
	prefService *service.NotificationPreferenceService
}

// NewNotificationPreferenceHandler creates a new NotificationPreferenceHandler.
func NewNotificationPreferenceHandler(prefService *service.NotificationPreferenceService) *NotificationPreferenceHandler {
	return &NotificationPreferenceHandler{prefService: prefService}
}

// Get handles GET /api/v1/notifications/preferences.
func (h *NotificationPreferenceHandler) Get(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.GetUserID(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, errorResponse("unauthorized"))
		return
	}
	orgID, ok := auth.GetOrgID(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, errorResponse("unauthorized"))
		return
	}

	pref, err := h.prefService.Get(r.Context(), userID, orgID)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, pref)
}

// Update handles PATCH /api/v1/notifications/preferences.
func (h *NotificationPreferenceHandler) Update(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.GetUserID(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, errorResponse("unauthorized"))
		return
	}
	orgID, ok := auth.GetOrgID(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, errorResponse("unauthorized"))
		return
	}

	var req service.UpdatePreferencesRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse("invalid request body"))
		return
	}

	pref, err := h.prefService.Update(r.Context(), userID, orgID, req)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, pref)
}
