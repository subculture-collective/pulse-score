package handler

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/onnwee/pulse-score/internal/auth"
	"github.com/onnwee/pulse-score/internal/service"
)

// IntegrationStripeHandler provides Stripe integration HTTP endpoints.
type IntegrationStripeHandler struct {
	oauthSvc      *service.StripeOAuthService
	orchestrator  *service.SyncOrchestratorService
}

// NewIntegrationStripeHandler creates a new IntegrationStripeHandler.
func NewIntegrationStripeHandler(oauthSvc *service.StripeOAuthService, orchestrator *service.SyncOrchestratorService) *IntegrationStripeHandler {
	return &IntegrationStripeHandler{
		oauthSvc:     oauthSvc,
		orchestrator: orchestrator,
	}
}

// Connect handles GET /api/v1/integrations/stripe/connect.
// Returns the OAuth URL to redirect the user to Stripe.
func (h *IntegrationStripeHandler) Connect(w http.ResponseWriter, r *http.Request) {
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

// Callback handles GET /api/v1/integrations/stripe/callback.
// Exchanges the code for tokens and initiates a full sync.
func (h *IntegrationStripeHandler) Callback(w http.ResponseWriter, r *http.Request) {
	orgID, ok := auth.GetOrgID(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, errorResponse("unauthorized"))
		return
	}

	code := r.URL.Query().Get("code")
	state := r.URL.Query().Get("state")

	// Check for Stripe error
	if errMsg := r.URL.Query().Get("error"); errMsg != "" {
		errDesc := r.URL.Query().Get("error_description")
		slog.Warn("stripe oauth error", "error", errMsg, "description", errDesc)
		writeJSON(w, http.StatusBadRequest, errorResponse("Stripe connection failed: "+errDesc))
		return
	}

	if err := h.oauthSvc.ExchangeCode(r.Context(), orgID, code, state); err != nil {
		handleServiceError(w, err)
		return
	}

	// Trigger initial full sync in background
	go h.orchestrator.RunFullSync(r.Context(), orgID)

	writeJSON(w, http.StatusOK, map[string]string{"message": "Stripe connected successfully. Initial sync started."})
}

// Status handles GET /api/v1/integrations/stripe/status.
func (h *IntegrationStripeHandler) Status(w http.ResponseWriter, r *http.Request) {
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

// Disconnect handles DELETE /api/v1/integrations/stripe.
func (h *IntegrationStripeHandler) Disconnect(w http.ResponseWriter, r *http.Request) {
	orgID, ok := auth.GetOrgID(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, errorResponse("unauthorized"))
		return
	}

	if err := h.oauthSvc.Disconnect(r.Context(), orgID); err != nil {
		handleServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "Stripe disconnected"})
}

// TriggerSync handles POST /api/v1/integrations/stripe/sync.
func (h *IntegrationStripeHandler) TriggerSync(w http.ResponseWriter, r *http.Request) {
	orgID, ok := auth.GetOrgID(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, errorResponse("unauthorized"))
		return
	}

	go h.orchestrator.RunFullSync(r.Context(), orgID)

	writeJSON(w, http.StatusAccepted, map[string]string{"message": "sync started"})
}

// handleServiceError writes service errors to HTTP response.
func handleServiceError(w http.ResponseWriter, err error) {
	handler := &AuthHandler{}
	handler.handleServiceError(w, err)
}

// WebhookStripeHandler provides Stripe webhook HTTP endpoints.
type WebhookStripeHandler struct {
	webhookSvc *service.StripeWebhookService
}

// NewWebhookStripeHandler creates a new WebhookStripeHandler.
func NewWebhookStripeHandler(webhookSvc *service.StripeWebhookService) *WebhookStripeHandler {
	return &WebhookStripeHandler{webhookSvc: webhookSvc}
}

// HandleWebhook handles POST /api/v1/webhooks/stripe.
func (h *WebhookStripeHandler) HandleWebhook(w http.ResponseWriter, r *http.Request) {
	const maxBodySize = 65536
	r.Body = http.MaxBytesReader(w, r.Body, maxBodySize)

	payload, err := readBody(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse("invalid request body"))
		return
	}

	sigHeader := r.Header.Get("Stripe-Signature")
	if sigHeader == "" {
		writeJSON(w, http.StatusBadRequest, errorResponse("missing Stripe-Signature header"))
		return
	}

	if err := h.webhookSvc.HandleEvent(r.Context(), payload, sigHeader); err != nil {
		slog.Error("webhook processing error", "error", err)
		// Still return 200 to prevent Stripe from retrying
		writeJSON(w, http.StatusOK, map[string]string{"status": "error logged"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func readBody(r *http.Request) ([]byte, error) {
	var buf []byte
	tmp := make([]byte, 1024)
	for {
		n, err := r.Body.Read(tmp)
		if n > 0 {
			buf = append(buf, tmp[:n]...)
		}
		if err != nil {
			break
		}
	}
	return buf, nil
}

// StripeStatusResponse is returned by GET /api/v1/integrations/stripe/status.
type StripeStatusResponse struct {
	Status        string `json:"status"`
	AccountID     string `json:"account_id,omitempty"`
	LastSyncAt    string `json:"last_sync_at,omitempty"`
	LastSyncError string `json:"last_sync_error,omitempty"`
	CustomerCount int    `json:"customer_count,omitempty"`
}

// marshalJSON is a convenience wrapper to avoid importing json in callers.
func marshalJSON(v any) ([]byte, error) {
	return json.Marshal(v)
}
