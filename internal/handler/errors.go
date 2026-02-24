package handler

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/onnwee/pulse-score/internal/service"
)

// handleServiceError maps service-layer errors to HTTP responses.
func handleServiceError(w http.ResponseWriter, err error) {
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
