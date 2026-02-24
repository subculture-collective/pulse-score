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

// HubSpotOAuthConfig holds HubSpot OAuth settings.
type HubSpotOAuthConfig struct {
	ClientID         string
	ClientSecret     string
	OAuthRedirectURL string
	EncryptionKey    string // 32-byte hex-encoded AES key
}

// HubSpotOAuthService handles HubSpot OAuth connect flow.
type HubSpotOAuthService struct {
	cfg      HubSpotOAuthConfig
	connRepo *repository.IntegrationConnectionRepository
}

// NewHubSpotOAuthService creates a new HubSpotOAuthService.
func NewHubSpotOAuthService(cfg HubSpotOAuthConfig, connRepo *repository.IntegrationConnectionRepository) *HubSpotOAuthService {
	return &HubSpotOAuthService{cfg: cfg, connRepo: connRepo}
}

// ConnectURL generates the HubSpot OAuth authorization URL.
func (s *HubSpotOAuthService) ConnectURL(orgID uuid.UUID) (string, error) {
	if s.cfg.ClientID == "" {
		return "", &ValidationError{Field: "hubspot", Message: "HubSpot integration is not configured"}
	}

	state := fmt.Sprintf("%s:%d", orgID.String(), time.Now().UnixNano())

	params := url.Values{
		"client_id":    {s.cfg.ClientID},
		"redirect_uri": {s.cfg.OAuthRedirectURL},
		"scope":        {"crm.objects.contacts.read crm.objects.deals.read crm.objects.companies.read"},
		"state":        {state},
	}

	return "https://app.hubspot.com/oauth/authorize?" + params.Encode(), nil
}

// ExchangeCode exchanges the OAuth code for tokens and stores the connection.
func (s *HubSpotOAuthService) ExchangeCode(ctx context.Context, orgID uuid.UUID, code, state string) error {
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

	tokenResp, err := s.exchangeCodeWithHubSpot(code)
	if err != nil {
		return fmt.Errorf("exchange code with hubspot: %w", err)
	}

	encrypted, err := encryptToken(tokenResp.AccessToken, s.cfg.EncryptionKey)
	if err != nil {
		return fmt.Errorf("encrypt access token: %w", err)
	}

	refreshEncrypted, err := encryptToken(tokenResp.RefreshToken, s.cfg.EncryptionKey)
	if err != nil {
		return fmt.Errorf("encrypt refresh token: %w", err)
	}

	expiresAt := time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second)

	conn := &repository.IntegrationConnection{
		OrgID:                 orgID,
		Provider:              "hubspot",
		Status:                "active",
		AccessTokenEncrypted:  encrypted,
		RefreshTokenEncrypted: refreshEncrypted,
		TokenExpiresAt:        &expiresAt,
		Scopes:                []string{"crm.objects.contacts.read", "crm.objects.deals.read", "crm.objects.companies.read"},
		Metadata:              map[string]any{},
	}

	if err := s.connRepo.Upsert(ctx, conn); err != nil {
		return fmt.Errorf("store connection: %w", err)
	}

	slog.Info("hubspot connection established", "org_id", orgID)
	return nil
}

// RefreshToken refreshes the access token using the refresh token.
func (s *HubSpotOAuthService) RefreshToken(ctx context.Context, orgID uuid.UUID) error {
	conn, err := s.connRepo.GetByOrgAndProvider(ctx, orgID, "hubspot")
	if err != nil {
		return fmt.Errorf("get connection: %w", err)
	}
	if conn == nil {
		return &NotFoundError{Resource: "hubspot_connection", Message: "no HubSpot connection found"}
	}

	refreshToken, err := decryptToken(conn.RefreshTokenEncrypted, s.cfg.EncryptionKey)
	if err != nil {
		return fmt.Errorf("decrypt refresh token: %w", err)
	}

	tokenResp, err := s.refreshTokenWithHubSpot(refreshToken)
	if err != nil {
		return fmt.Errorf("refresh token with hubspot: %w", err)
	}

	encrypted, err := encryptToken(tokenResp.AccessToken, s.cfg.EncryptionKey)
	if err != nil {
		return fmt.Errorf("encrypt access token: %w", err)
	}

	refreshEncrypted, err := encryptToken(tokenResp.RefreshToken, s.cfg.EncryptionKey)
	if err != nil {
		return fmt.Errorf("encrypt refresh token: %w", err)
	}

	expiresAt := time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second)

	conn.AccessTokenEncrypted = encrypted
	conn.RefreshTokenEncrypted = refreshEncrypted
	conn.TokenExpiresAt = &expiresAt

	if err := s.connRepo.Upsert(ctx, conn); err != nil {
		return fmt.Errorf("update connection: %w", err)
	}

	slog.Info("hubspot token refreshed", "org_id", orgID)
	return nil
}

// GetAccessToken retrieves and decrypts the access token, auto-refreshing if expired.
func (s *HubSpotOAuthService) GetAccessToken(ctx context.Context, orgID uuid.UUID) (string, error) {
	conn, err := s.connRepo.GetByOrgAndProvider(ctx, orgID, "hubspot")
	if err != nil {
		return "", fmt.Errorf("get connection: %w", err)
	}
	if conn == nil {
		return "", &NotFoundError{Resource: "hubspot_connection", Message: "no HubSpot connection found"}
	}
	if conn.Status != "active" && conn.Status != "syncing" {
		return "", &ValidationError{Field: "hubspot", Message: "HubSpot connection is not active"}
	}

	// Auto-refresh if token is expired or about to expire (within 5 minutes)
	if conn.TokenExpiresAt != nil && time.Now().Add(5*time.Minute).After(*conn.TokenExpiresAt) {
		if err := s.RefreshToken(ctx, orgID); err != nil {
			return "", fmt.Errorf("auto-refresh token: %w", err)
		}
		// Re-fetch the connection with the new token
		conn, err = s.connRepo.GetByOrgAndProvider(ctx, orgID, "hubspot")
		if err != nil {
			return "", fmt.Errorf("get refreshed connection: %w", err)
		}
	}

	token, err := decryptToken(conn.AccessTokenEncrypted, s.cfg.EncryptionKey)
	if err != nil {
		return "", fmt.Errorf("decrypt access token: %w", err)
	}
	return token, nil
}

// GetStatus returns the current HubSpot connection status for an org.
func (s *HubSpotOAuthService) GetStatus(ctx context.Context, orgID uuid.UUID) (*HubSpotConnectionStatus, error) {
	conn, err := s.connRepo.GetByOrgAndProvider(ctx, orgID, "hubspot")
	if err != nil {
		return nil, fmt.Errorf("get connection: %w", err)
	}

	if conn == nil {
		return &HubSpotConnectionStatus{Status: "disconnected"}, nil
	}

	return &HubSpotConnectionStatus{
		Status:            conn.Status,
		ExternalAccountID: conn.ExternalAccountID,
		LastSyncAt:        conn.LastSyncAt,
		LastSyncError:     conn.LastSyncError,
		ConnectedAt:       conn.CreatedAt,
	}, nil
}

// Disconnect removes a HubSpot connection.
func (s *HubSpotOAuthService) Disconnect(ctx context.Context, orgID uuid.UUID) error {
	return s.connRepo.Delete(ctx, orgID, "hubspot")
}

// HubSpotConnectionStatus holds the status info for frontend display.
type HubSpotConnectionStatus struct {
	Status            string     `json:"status"`
	ExternalAccountID string     `json:"external_account_id,omitempty"`
	LastSyncAt        *time.Time `json:"last_sync_at,omitempty"`
	LastSyncError     string     `json:"last_sync_error,omitempty"`
	ConnectedAt       time.Time  `json:"connected_at,omitempty"`
	ContactCount      int        `json:"contact_count,omitempty"`
	DealCount         int        `json:"deal_count,omitempty"`
	CompanyCount      int        `json:"company_count,omitempty"`
}

// hubspotTokenResponse holds the HubSpot OAuth token exchange response.
type hubspotTokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
	TokenType    string `json:"token_type"`
}

func (s *HubSpotOAuthService) exchangeCodeWithHubSpot(code string) (*hubspotTokenResponse, error) {
	data := url.Values{
		"grant_type":    {"authorization_code"},
		"client_id":     {s.cfg.ClientID},
		"client_secret": {s.cfg.ClientSecret},
		"redirect_uri":  {s.cfg.OAuthRedirectURL},
		"code":          {code},
	}

	return s.postTokenRequest(data)
}

func (s *HubSpotOAuthService) refreshTokenWithHubSpot(refreshToken string) (*hubspotTokenResponse, error) {
	data := url.Values{
		"grant_type":    {"refresh_token"},
		"client_id":     {s.cfg.ClientID},
		"client_secret": {s.cfg.ClientSecret},
		"refresh_token": {refreshToken},
	}

	return s.postTokenRequest(data)
}

func (s *HubSpotOAuthService) postTokenRequest(data url.Values) (*hubspotTokenResponse, error) {
	req, err := http.NewRequest("POST", "https://api.hubapi.com/oauth/v1/token", strings.NewReader(data.Encode()))
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
		slog.Error("hubspot oauth token exchange failed",
			"status", resp.StatusCode,
			"body", string(body),
		)
		return nil, fmt.Errorf("hubspot token exchange failed with status %d", resp.StatusCode)
	}

	var tokenResp hubspotTokenResponse
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}

	return &tokenResp, nil
}
