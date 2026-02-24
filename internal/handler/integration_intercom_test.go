package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"

	"github.com/onnwee/pulse-score/internal/auth"
	"github.com/onnwee/pulse-score/internal/service"
)

func TestIntercomConnect_Unauthorized(t *testing.T) {
	h := NewIntegrationIntercomHandler(
		service.NewIntercomOAuthService(service.IntercomOAuthConfig{}, nil),
		nil,
	)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/integrations/intercom/connect", nil)
	rr := httptest.NewRecorder()

	h.Connect(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rr.Code)
	}
}

func TestIntercomConnect_NotConfigured(t *testing.T) {
	orgID := uuid.New()
	h := NewIntegrationIntercomHandler(
		service.NewIntercomOAuthService(service.IntercomOAuthConfig{}, nil),
		nil,
	)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/integrations/intercom/connect", nil)
	req = req.WithContext(auth.WithOrgID(req.Context(), orgID))
	rr := httptest.NewRecorder()

	h.Connect(rr, req)

	if rr.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422, got %d", rr.Code)
	}
}

func TestIntercomConnect_Success(t *testing.T) {
	orgID := uuid.New()
	h := NewIntegrationIntercomHandler(
		service.NewIntercomOAuthService(service.IntercomOAuthConfig{
			ClientID:         "test-client-id",
			ClientSecret:     "test-client-secret",
			OAuthRedirectURL: "http://localhost/callback",
		}, nil),
		nil,
	)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/integrations/intercom/connect", nil)
	req = req.WithContext(auth.WithOrgID(req.Context(), orgID))
	rr := httptest.NewRecorder()

	h.Connect(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
}

func TestIntercomStatus_Unauthorized(t *testing.T) {
	h := NewIntegrationIntercomHandler(nil, nil)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/integrations/intercom/status", nil)
	rr := httptest.NewRecorder()

	h.Status(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rr.Code)
	}
}

func TestIntercomDisconnect_Unauthorized(t *testing.T) {
	h := NewIntegrationIntercomHandler(nil, nil)
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/integrations/intercom", nil)
	rr := httptest.NewRecorder()

	h.Disconnect(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rr.Code)
	}
}

func TestIntercomTriggerSync_Unauthorized(t *testing.T) {
	h := NewIntegrationIntercomHandler(nil, nil)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/integrations/intercom/sync", nil)
	rr := httptest.NewRecorder()

	h.TriggerSync(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rr.Code)
	}
}

func TestIntercomCallback_Unauthorized(t *testing.T) {
	h := NewIntegrationIntercomHandler(nil, nil)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/integrations/intercom/callback?code=test", nil)
	rr := httptest.NewRecorder()

	h.Callback(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rr.Code)
	}
}

func TestIntercomCallback_OAuthError(t *testing.T) {
	orgID := uuid.New()
	h := NewIntegrationIntercomHandler(nil, nil)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/integrations/intercom/callback?error=access_denied&error_description=User+denied+access", nil)
	req = req.WithContext(auth.WithOrgID(req.Context(), orgID))
	rr := httptest.NewRecorder()

	h.Callback(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
}

func TestWebhookIntercomHandler_MissingSignature(t *testing.T) {
	h := NewWebhookIntercomHandler(
		service.NewIntercomWebhookService("test-secret", nil, nil, nil, nil, nil, nil),
	)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/webhooks/intercom", nil)
	rr := httptest.NewRecorder()

	h.HandleWebhook(rr, req)

	// Should fail on body read or signature verification
	if rr.Code != http.StatusBadRequest && rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 400 or 401, got %d", rr.Code)
	}
}
