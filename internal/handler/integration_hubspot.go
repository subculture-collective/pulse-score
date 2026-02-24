package handler

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/onnwee/pulse-score/internal/auth"
	"github.com/onnwee/pulse-score/internal/service"
)

// IntegrationHubSpotHandler provides HubSpot integration HTTP endpoints.
type IntegrationHubSpotHandler struct {
	oauthSvc     *service.HubSpotOAuthService
	orchestrator *service.HubSpotSyncOrchestratorService
}

// NewIntegrationHubSpotHandler creates a new IntegrationHubSpotHandler.
func NewIntegrationHubSpotHandler(oauthSvc *service.HubSpotOAuthService, orchestrator *service.HubSpotSyncOrchestratorService) *IntegrationHubSpotHandler {
	return &IntegrationHubSpotHandler{
		oauthSvc:     oauthSvc,
		orchestrator: orchestrator,
	}
}

// Connect handles GET /api/v1/integrations/hubspot/connect.
func (h *IntegrationHubSpotHandler) Connect(w http.ResponseWriter, r *http.Request) {
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

// Callback handles GET /api/v1/integrations/hubspot/callback.
func (h *IntegrationHubSpotHandler) Callback(w http.ResponseWriter, r *http.Request) {
	orgID, ok := auth.GetOrgID(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, errorResponse("unauthorized"))
		return
	}

	code := r.URL.Query().Get("code")
	state := r.URL.Query().Get("state")

	if errMsg := r.URL.Query().Get("error"); errMsg != "" {
		errDesc := r.URL.Query().Get("error_description")
		slog.Warn("hubspot oauth error", "error", errMsg, "description", errDesc)
		writeJSON(w, http.StatusBadRequest, errorResponse("HubSpot connection failed: "+errDesc))
		return
	}

	if err := h.oauthSvc.ExchangeCode(r.Context(), orgID, code, state); err != nil {
		handleServiceError(w, err)
		return
	}

	// Trigger initial full sync in background
	go h.orchestrator.RunFullSync(r.Context(), orgID)

	writeJSON(w, http.StatusOK, map[string]string{"message": "HubSpot connected successfully. Initial sync started."})
}

// Status handles GET /api/v1/integrations/hubspot/status.
func (h *IntegrationHubSpotHandler) Status(w http.ResponseWriter, r *http.Request) {
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

// Disconnect handles DELETE /api/v1/integrations/hubspot.
func (h *IntegrationHubSpotHandler) Disconnect(w http.ResponseWriter, r *http.Request) {
	orgID, ok := auth.GetOrgID(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, errorResponse("unauthorized"))
		return
	}

	if err := h.oauthSvc.Disconnect(r.Context(), orgID); err != nil {
		handleServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "HubSpot disconnected"})
}

// TriggerSync handles POST /api/v1/integrations/hubspot/sync.
func (h *IntegrationHubSpotHandler) TriggerSync(w http.ResponseWriter, r *http.Request) {
	orgID, ok := auth.GetOrgID(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, errorResponse("unauthorized"))
		return
	}

	go h.orchestrator.RunFullSync(r.Context(), orgID)

	writeJSON(w, http.StatusAccepted, map[string]string{"message": "HubSpot sync started"})
}

// WebhookHubSpotHandler provides HubSpot webhook HTTP endpoints.
type WebhookHubSpotHandler struct {
	webhookSvc *service.HubSpotWebhookService
}

// NewWebhookHubSpotHandler creates a new WebhookHubSpotHandler.
func NewWebhookHubSpotHandler(webhookSvc *service.HubSpotWebhookService) *WebhookHubSpotHandler {
	return &WebhookHubSpotHandler{webhookSvc: webhookSvc}
}

// HandleWebhook handles POST /api/v1/webhooks/hubspot.
func (h *WebhookHubSpotHandler) HandleWebhook(w http.ResponseWriter, r *http.Request) {
	const maxBodySize = 65536
	r.Body = http.MaxBytesReader(w, r.Body, maxBodySize)

	payload, err := readBody(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse("invalid request body"))
		return
	}

	sigHeader := r.Header.Get("X-HubSpot-Signature-v3")
	timestamp := r.Header.Get("X-HubSpot-Request-Timestamp")

	if err := h.webhookSvc.VerifySignature(payload, sigHeader, timestamp, r.Method, r.URL.RequestURI()); err != nil {
		slog.Warn("hubspot webhook signature verification failed", "error", err)
		writeJSON(w, http.StatusUnauthorized, errorResponse("invalid signature"))
		return
	}

	var events []service.HubSpotWebhookEvent
	if err := json.Unmarshal(payload, &events); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse("invalid request body"))
		return
	}

	if err := h.webhookSvc.ProcessEvents(r.Context(), events); err != nil {
		slog.Error("hubspot webhook processing error", "error", err)
	}

	// Always return 200 to prevent HubSpot retries
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
