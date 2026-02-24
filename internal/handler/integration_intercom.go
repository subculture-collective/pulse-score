package handler

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/onnwee/pulse-score/internal/auth"
	"github.com/onnwee/pulse-score/internal/service"
)

// IntegrationIntercomHandler provides Intercom integration HTTP endpoints.
type IntegrationIntercomHandler struct {
	oauthSvc     *service.IntercomOAuthService
	orchestrator *service.IntercomSyncOrchestratorService
}

// NewIntegrationIntercomHandler creates a new IntegrationIntercomHandler.
func NewIntegrationIntercomHandler(oauthSvc *service.IntercomOAuthService, orchestrator *service.IntercomSyncOrchestratorService) *IntegrationIntercomHandler {
	return &IntegrationIntercomHandler{
		oauthSvc:     oauthSvc,
		orchestrator: orchestrator,
	}
}

// Connect handles GET /api/v1/integrations/intercom/connect.
func (h *IntegrationIntercomHandler) Connect(w http.ResponseWriter, r *http.Request) {
	orgID, ok := auth.GetOrgID(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, errorResponse("unauthorized"))
		return
	}

	connectURL, err := h.oauthSvc.ConnectURL(orgID)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"url": connectURL})
}

// Callback handles GET /api/v1/integrations/intercom/callback.
func (h *IntegrationIntercomHandler) Callback(w http.ResponseWriter, r *http.Request) {
	orgID, ok := auth.GetOrgID(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, errorResponse("unauthorized"))
		return
	}

	code := r.URL.Query().Get("code")
	state := r.URL.Query().Get("state")

	if errMsg := r.URL.Query().Get("error"); errMsg != "" {
		errDesc := r.URL.Query().Get("error_description")
		slog.Warn("intercom oauth error", "error", errMsg, "description", errDesc)
		writeJSON(w, http.StatusBadRequest, errorResponse("Intercom connection failed: "+errDesc))
		return
	}

	if err := h.oauthSvc.ExchangeCode(r.Context(), orgID, code, state); err != nil {
		handleServiceError(w, err)
		return
	}

	// Trigger initial full sync in background
	go h.orchestrator.RunFullSync(r.Context(), orgID)

	writeJSON(w, http.StatusOK, map[string]string{"message": "Intercom connected successfully. Initial sync started."})
}

// Status handles GET /api/v1/integrations/intercom/status.
func (h *IntegrationIntercomHandler) Status(w http.ResponseWriter, r *http.Request) {
	orgID, ok := auth.GetOrgID(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, errorResponse("unauthorized"))
		return
	}

	status, err := h.oauthSvc.GetStatus(r.Context(), orgID)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, status)
}

// Disconnect handles DELETE /api/v1/integrations/intercom.
func (h *IntegrationIntercomHandler) Disconnect(w http.ResponseWriter, r *http.Request) {
	orgID, ok := auth.GetOrgID(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, errorResponse("unauthorized"))
		return
	}

	if err := h.oauthSvc.Disconnect(r.Context(), orgID); err != nil {
		handleServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "Intercom disconnected"})
}

// TriggerSync handles POST /api/v1/integrations/intercom/sync.
func (h *IntegrationIntercomHandler) TriggerSync(w http.ResponseWriter, r *http.Request) {
	orgID, ok := auth.GetOrgID(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, errorResponse("unauthorized"))
		return
	}

	go h.orchestrator.RunFullSync(r.Context(), orgID)

	writeJSON(w, http.StatusAccepted, map[string]string{"message": "Intercom sync started"})
}

// WebhookIntercomHandler provides Intercom webhook HTTP endpoints.
type WebhookIntercomHandler struct {
	webhookSvc *service.IntercomWebhookService
}

// NewWebhookIntercomHandler creates a new WebhookIntercomHandler.
func NewWebhookIntercomHandler(webhookSvc *service.IntercomWebhookService) *WebhookIntercomHandler {
	return &WebhookIntercomHandler{webhookSvc: webhookSvc}
}

// HandleWebhook handles POST /api/v1/webhooks/intercom.
func (h *WebhookIntercomHandler) HandleWebhook(w http.ResponseWriter, r *http.Request) {
	const maxBodySize = 65536
	r.Body = http.MaxBytesReader(w, r.Body, maxBodySize)

	payload, err := readBody(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse("invalid request body"))
		return
	}

	sigHeader := r.Header.Get("X-Hub-Signature")

	if err := h.webhookSvc.VerifySignature(payload, sigHeader); err != nil {
		slog.Warn("intercom webhook signature verification failed", "error", err)
		writeJSON(w, http.StatusUnauthorized, errorResponse("invalid signature"))
		return
	}

	var event service.IntercomWebhookEvent
	if err := json.Unmarshal(payload, &event); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse("invalid request body"))
		return
	}

	if err := h.webhookSvc.ProcessEvent(r.Context(), event); err != nil {
		slog.Error("intercom webhook processing error", "error", err)
	}

	// Always return 200 to prevent Intercom retries
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
