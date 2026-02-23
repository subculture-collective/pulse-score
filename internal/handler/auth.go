package handler

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/onnwee/pulse-score/internal/service"
)

// AuthHandler provides auth HTTP endpoints.
type AuthHandler struct {
	authService *service.AuthService
}

// NewAuthHandler creates a new AuthHandler.
func NewAuthHandler(authService *service.AuthService) *AuthHandler {
	return &AuthHandler{authService: authService}
}

// Register handles POST /api/v1/auth/register.
func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req service.RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse("invalid request body"))
		return
	}

	resp, err := h.authService.Register(r.Context(), req)
	if err != nil {
		h.handleServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, resp)
}

// Login handles POST /api/v1/auth/login.
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req service.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse("invalid request body"))
		return
	}

	resp, err := h.authService.Login(r.Context(), req)
	if err != nil {
		h.handleServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, resp)
}

// Refresh handles POST /api/v1/auth/refresh.
func (h *AuthHandler) Refresh(w http.ResponseWriter, r *http.Request) {
	var req service.RefreshRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse("invalid request body"))
		return
	}

	resp, err := h.authService.Refresh(r.Context(), req)
	if err != nil {
		h.handleServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, resp)
}

// RequestPasswordReset handles POST /api/v1/auth/password-reset/request.
func (h *AuthHandler) RequestPasswordReset(w http.ResponseWriter, r *http.Request) {
	var req service.PasswordResetRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse("invalid request body"))
		return
	}

	// Always return 200 to prevent email enumeration
	_ = h.authService.RequestPasswordReset(r.Context(), req)
	writeJSON(w, http.StatusOK, map[string]string{"message": "if an account with that email exists, a password reset link has been sent"})
}

// CompletePasswordReset handles POST /api/v1/auth/password-reset/complete.
func (h *AuthHandler) CompletePasswordReset(w http.ResponseWriter, r *http.Request) {
	var req service.PasswordResetCompleteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse("invalid request body"))
		return
	}

	if err := h.authService.CompletePasswordReset(r.Context(), req); err != nil {
		h.handleServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "password has been reset successfully"})
}

func (h *AuthHandler) handleServiceError(w http.ResponseWriter, err error) {
	var validationErr *service.ValidationError
	var conflictErr *service.ConflictError
	var authErr *service.AuthError
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
	case errors.As(err, &authErr):
		writeJSON(w, http.StatusUnauthorized, errorResponse(authErr.Message))
	case errors.As(err, &notFoundErr):
		writeJSON(w, http.StatusNotFound, errorResponse(notFoundErr.Message))
	case errors.As(err, &forbiddenErr):
		writeJSON(w, http.StatusForbidden, errorResponse(forbiddenErr.Message))
	default:
		slog.Error("internal error", "error", err)
		writeJSON(w, http.StatusInternalServerError, errorResponse("internal server error"))
	}
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func errorResponse(msg string) map[string]string {
	return map[string]string{"error": msg}
}
