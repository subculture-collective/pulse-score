package handler

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/onnwee/pulse-score/internal/auth"
	"github.com/onnwee/pulse-score/internal/service"
)

// InvitationHandler provides invitation HTTP endpoints.
type InvitationHandler struct {
	invitationService *service.InvitationService
}

// NewInvitationHandler creates a new InvitationHandler.
func NewInvitationHandler(invitationService *service.InvitationService) *InvitationHandler {
	return &InvitationHandler{invitationService: invitationService}
}

// Create handles POST /api/v1/organizations/{org_id}/invitations.
func (h *InvitationHandler) Create(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.GetUserID(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, errorResponse("unauthorized"))
		return
	}

	orgID, ok := auth.GetOrgID(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, errorResponse("missing organization context"))
		return
	}

	var req service.CreateInvitationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse("invalid request body"))
		return
	}

	resp, err := h.invitationService.Create(r.Context(), orgID, userID, req)
	if err != nil {
		h.handleError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, resp)
}

// List handles GET /api/v1/invitations.
func (h *InvitationHandler) List(w http.ResponseWriter, r *http.Request) {
	orgID, ok := auth.GetOrgID(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, errorResponse("missing organization context"))
		return
	}

	resp, err := h.invitationService.ListPending(r.Context(), orgID)
	if err != nil {
		slog.Error("list invitations error", "error", err)
		writeJSON(w, http.StatusInternalServerError, errorResponse("internal server error"))
		return
	}

	writeJSON(w, http.StatusOK, resp)
}

// Revoke handles DELETE /api/v1/invitations/{id}.
func (h *InvitationHandler) Revoke(w http.ResponseWriter, r *http.Request) {
	orgID, ok := auth.GetOrgID(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, errorResponse("missing organization context"))
		return
	}

	invitationID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse("invalid invitation ID"))
		return
	}

	if err := h.invitationService.Revoke(r.Context(), orgID, invitationID); err != nil {
		h.handleError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// Accept handles POST /api/v1/invitations/accept (public endpoint).
func (h *InvitationHandler) Accept(w http.ResponseWriter, r *http.Request) {
	var req service.AcceptInvitationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse("invalid request body"))
		return
	}

	resp, err := h.invitationService.Accept(r.Context(), req)
	if err != nil {
		h.handleError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, resp)
}

func (h *InvitationHandler) handleError(w http.ResponseWriter, err error) {
	var validationErr *service.ValidationError
	var conflictErr *service.ConflictError
	var notFoundErr *service.NotFoundError
	var forbiddenErr *service.ForbiddenError

	switch {
	case errors.As(err, &validationErr):
		writeJSON(w, http.StatusUnprocessableEntity, map[string]any{
			"error": validationErr.Message,
			"field": validationErr.Field,
		})
	case errors.As(err, &conflictErr):
		writeJSON(w, http.StatusConflict, errorResponse(conflictErr.Message))
	case errors.As(err, &notFoundErr):
		writeJSON(w, http.StatusNotFound, errorResponse(notFoundErr.Message))
	case errors.As(err, &forbiddenErr):
		writeJSON(w, http.StatusForbidden, errorResponse(forbiddenErr.Message))
	default:
		slog.Error("invitation error", "error", err)
		writeJSON(w, http.StatusInternalServerError, errorResponse("internal server error"))
	}
}
