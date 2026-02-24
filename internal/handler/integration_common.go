package handler

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/google/uuid"

	"github.com/onnwee/pulse-score/internal/auth"
)

func integrationOrgID(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	orgID, ok := auth.GetOrgID(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, errorResponse("unauthorized"))
		return uuid.Nil, false
	}
	return orgID, true
}

func integrationConnect(
	w http.ResponseWriter,
	r *http.Request,
	connectURLFn func(orgID uuid.UUID) (string, error),
) {
	orgID, ok := integrationOrgID(w, r)
	if !ok {
		return
	}

	connectURL, err := connectURLFn(orgID)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"url": connectURL})
}

func integrationStatus(
	w http.ResponseWriter,
	r *http.Request,
	getStatusFn func(ctx context.Context, orgID uuid.UUID) (any, error),
) {
	orgID, ok := integrationOrgID(w, r)
	if !ok {
		return
	}

	status, err := getStatusFn(r.Context(), orgID)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, status)
}

func integrationDisconnect(
	w http.ResponseWriter,
	r *http.Request,
	disconnectFn func(ctx context.Context, orgID uuid.UUID) error,
	disconnectedMessage string,
) {
	orgID, ok := integrationOrgID(w, r)
	if !ok {
		return
	}

	if err := disconnectFn(r.Context(), orgID); err != nil {
		handleServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": disconnectedMessage})
}

func integrationTriggerSync(
	w http.ResponseWriter,
	r *http.Request,
	runFullSyncFn func(ctx context.Context, orgID uuid.UUID),
	startedMessage string,
) {
	orgID, ok := integrationOrgID(w, r)
	if !ok {
		return
	}

	go runFullSyncFn(r.Context(), orgID)

	writeJSON(w, http.StatusAccepted, map[string]string{"message": startedMessage})
}

func integrationCallback(
	w http.ResponseWriter,
	r *http.Request,
	providerLogName string,
	providerDisplayName string,
	connectedMessage string,
	exchangeCodeFn func(ctx context.Context, orgID uuid.UUID, code, state string) error,
	runFullSyncFn func(ctx context.Context, orgID uuid.UUID),
) {
	orgID, ok := integrationOrgID(w, r)
	if !ok {
		return
	}

	code := r.URL.Query().Get("code")
	state := r.URL.Query().Get("state")

	if errMsg := r.URL.Query().Get("error"); errMsg != "" {
		errDesc := r.URL.Query().Get("error_description")
		slog.Warn(providerLogName+" oauth error", "error", errMsg, "description", errDesc)
		writeJSON(w, http.StatusBadRequest, errorResponse(providerDisplayName+" connection failed: "+errDesc))
		return
	}

	if err := exchangeCodeFn(r.Context(), orgID, code, state); err != nil {
		handleServiceError(w, err)
		return
	}

	go runFullSyncFn(r.Context(), orgID)

	writeJSON(w, http.StatusOK, map[string]string{"message": connectedMessage})
}
