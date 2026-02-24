package service

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/onnwee/pulse-score/internal/repository"
)

// IntercomOAuthConfig holds Intercom OAuth settings.
type IntercomOAuthConfig struct {
	ClientID         string
	ClientSecret     string
	OAuthRedirectURL string
	EncryptionKey    string // 32-byte hex-encoded AES key
}

// IntercomOAuthService handles Intercom OAuth connect flow.
type IntercomOAuthService struct {
	cfg      IntercomOAuthConfig
	connRepo *repository.IntegrationConnectionRepository
}

// NewIntercomOAuthService creates a new IntercomOAuthService.
func NewIntercomOAuthService(cfg IntercomOAuthConfig, connRepo *repository.IntegrationConnectionRepository) *IntercomOAuthService {
	return &IntercomOAuthService{cfg: cfg, connRepo: connRepo}
}

// ConnectURL generates the Intercom OAuth authorization URL.
func (s *IntercomOAuthService) ConnectURL(orgID uuid.UUID) (string, error) {
	if s.cfg.ClientID == "" {
		return "", &ValidationError{Field: "intercom", Message: "Intercom integration is not configured"}
	}

	state := fmt.Sprintf("%s:%d", orgID.String(), time.Now().UnixNano())

	params := url.Values{
		"client_id":    {s.cfg.ClientID},
		"redirect_uri": {s.cfg.OAuthRedirectURL},
		"state":        {state},
	}

	return "https://app.intercom.com/oauth?" + params.Encode(), nil
}

// ExchangeCode exchanges the OAuth code for an access token and stores the connection.
// Intercom provides a non-expiring access token (no refresh needed).
func (s *IntercomOAuthService) ExchangeCode(ctx context.Context, orgID uuid.UUID, code, state string) error {
	if code == "" {
		return &ValidationError{Field: "code", Message: "authorization code is required"}
	}

	// Validate state parameter contains the correct org ID
	parts := strings.SplitN(state, ":", 2)
	if len(parts) != 2 {
		return &ValidationError{Field: "state", Message: "invalid state parameter"}
	}
	stateOrgID, err := uuid.Parse(parts[0])
	if err != nil || stateOrgID != orgID {
		return &ValidationError{Field: "state", Message: "invalid state parameter"}
	}

	tokenResp, err := s.exchangeCodeWithIntercom(code)
	if err != nil {
		return fmt.Errorf("exchange code with intercom: %w", err)
	}

	encrypted, err := encryptToken(tokenResp.AccessToken, s.cfg.EncryptionKey)
	if err != nil {
		return fmt.Errorf("encrypt access token: %w", err)
	}

	conn := &repository.IntegrationConnection{
		OrgID:                orgID,
		Provider:             "intercom",
		Status:               "active",
		AccessTokenEncrypted: encrypted,
		Scopes:               []string{"read_conversations", "read_contacts"},
		Metadata: map[string]any{
			"token_type": tokenResp.TokenType,
		},
	}

	if err := s.connRepo.Upsert(ctx, conn); err != nil {
		return fmt.Errorf("store connection: %w", err)
	}

	slog.Info("intercom connection established", "org_id", orgID)
	return nil
}

// GetAccessToken retrieves and decrypts the access token.
// Intercom tokens don't expire, so no refresh is needed.
func (s *IntercomOAuthService) GetAccessToken(ctx context.Context, orgID uuid.UUID) (string, error) {
	conn, err := s.connRepo.GetByOrgAndProvider(ctx, orgID, "intercom")
	if err != nil {
		return "", fmt.Errorf("get connection: %w", err)
	}
	if conn == nil {
		return "", &NotFoundError{Resource: "intercom_connection", Message: "no Intercom connection found"}
	}
	if conn.Status != "active" && conn.Status != "syncing" {
		return "", &ValidationError{Field: "intercom", Message: "Intercom connection is not active"}
	}

	token, err := decryptToken(conn.AccessTokenEncrypted, s.cfg.EncryptionKey)
	if err != nil {
		return "", fmt.Errorf("decrypt access token: %w", err)
	}
	return token, nil
}

// GetStatus returns the current Intercom connection status for an org.
func (s *IntercomOAuthService) GetStatus(ctx context.Context, orgID uuid.UUID) (*IntercomConnectionStatus, error) {
	conn, err := s.connRepo.GetByOrgAndProvider(ctx, orgID, "intercom")
	if err != nil {
		return nil, fmt.Errorf("get connection: %w", err)
	}

	if conn == nil {
		return &IntercomConnectionStatus{Status: "disconnected"}, nil
	}

	return &IntercomConnectionStatus{
		Status:            conn.Status,
		ExternalAccountID: conn.ExternalAccountID,
		LastSyncAt:        conn.LastSyncAt,
		LastSyncError:     conn.LastSyncError,
		ConnectedAt:       conn.CreatedAt,
	}, nil
}

// Disconnect removes an Intercom connection.
func (s *IntercomOAuthService) Disconnect(ctx context.Context, orgID uuid.UUID) error {
	return s.connRepo.Delete(ctx, orgID, "intercom")
}

// IntercomConnectionStatus holds the status info for frontend display.
type IntercomConnectionStatus struct {
	Status            string     `json:"status"`
	ExternalAccountID string     `json:"external_account_id,omitempty"`
	LastSyncAt        *time.Time `json:"last_sync_at,omitempty"`
	LastSyncError     string     `json:"last_sync_error,omitempty"`
	ConnectedAt       time.Time  `json:"connected_at,omitempty"`
	ConversationCount int        `json:"conversation_count,omitempty"`
	ContactCount      int        `json:"contact_count,omitempty"`
}

// intercomTokenResponse holds the Intercom OAuth token exchange response.
type intercomTokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
}

func (s *IntercomOAuthService) exchangeCodeWithIntercom(code string) (*intercomTokenResponse, error) {
	data := url.Values{
		"code":          {code},
		"client_id":     {s.cfg.ClientID},
		"client_secret": {s.cfg.ClientSecret},
	}

	req, err := http.NewRequest("POST", "https://api.intercom.io/auth/eagle/token", strings.NewReader(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		slog.Error("intercom oauth token exchange failed",
			"status", resp.StatusCode,
			"body", string(body),
		)
		return nil, fmt.Errorf("intercom token exchange failed with status %d", resp.StatusCode)
	}

	var tokenResp intercomTokenResponse
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}

	return &tokenResp, nil
}
