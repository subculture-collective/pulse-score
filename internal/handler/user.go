package handler

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/onnwee/pulse-score/internal/auth"
	"github.com/onnwee/pulse-score/internal/service"
)

// UserHandler provides user profile HTTP endpoints.
type UserHandler struct {
	userService *service.UserService
}

// NewUserHandler creates a new UserHandler.
func NewUserHandler(userService *service.UserService) *UserHandler {
	return &UserHandler{userService: userService}
}

// GetProfile handles GET /api/v1/users/me.
func (h *UserHandler) GetProfile(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.GetUserID(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, errorResponse("unauthorized"))
		return
	}

	resp, err := h.userService.GetProfile(r.Context(), userID)
	if err != nil {
		var notFoundErr *service.NotFoundError
		if errors.As(err, &notFoundErr) {
			writeJSON(w, http.StatusNotFound, errorResponse(notFoundErr.Message))
			return
		}
		slog.Error("get profile error", "error", err)
		writeJSON(w, http.StatusInternalServerError, errorResponse("internal server error"))
		return
	}

	writeJSON(w, http.StatusOK, resp)
}

// UpdateProfile handles PATCH /api/v1/users/me.
func (h *UserHandler) UpdateProfile(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.GetUserID(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, errorResponse("unauthorized"))
		return
	}

	var req service.UpdateProfileRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse("invalid request body"))
		return
	}

	resp, err := h.userService.UpdateProfile(r.Context(), userID, req)
	if err != nil {
		var notFoundErr *service.NotFoundError
		var validationErr *service.ValidationError
		switch {
		case errors.As(err, &notFoundErr):
			writeJSON(w, http.StatusNotFound, errorResponse(notFoundErr.Message))
		case errors.As(err, &validationErr):
			writeJSON(w, http.StatusUnprocessableEntity, map[string]any{
				"error": validationErr.Message,
				"field": validationErr.Field,
			})
		default:
			slog.Error("update profile error", "error", err)
			writeJSON(w, http.StatusInternalServerError, errorResponse("internal server error"))
		}
		return
	}

	writeJSON(w, http.StatusOK, resp)
}
