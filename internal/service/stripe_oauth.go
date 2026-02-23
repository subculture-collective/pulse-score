package service

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/hex"
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

// StripeOAuthConfig holds Stripe OAuth settings.
type StripeOAuthConfig struct {
	ClientID       string
	SecretKey      string
	OAuthRedirectURL string
	EncryptionKey  string // 32-byte hex-encoded AES key
}

// StripeOAuthService handles Stripe OAuth connect flow.
type StripeOAuthService struct {
	cfg      StripeOAuthConfig
	connRepo *repository.IntegrationConnectionRepository
}

// NewStripeOAuthService creates a new StripeOAuthService.
func NewStripeOAuthService(cfg StripeOAuthConfig, connRepo *repository.IntegrationConnectionRepository) *StripeOAuthService {
	return &StripeOAuthService{cfg: cfg, connRepo: connRepo}
}

// ConnectURL generates the Stripe OAuth authorization URL.
func (s *StripeOAuthService) ConnectURL(orgID uuid.UUID) (string, error) {
	if s.cfg.ClientID == "" {
		return "", &ValidationError{Field: "stripe", Message: "Stripe integration is not configured"}
	}

	state := fmt.Sprintf("%s:%d", orgID.String(), time.Now().UnixNano())

	params := url.Values{
		"response_type": {"code"},
		"client_id":     {s.cfg.ClientID},
		"scope":         {"read_only"},
		"redirect_uri":  {s.cfg.OAuthRedirectURL},
		"state":         {state},
		"stripe_landing": {"login"},
	}

	return "https://connect.stripe.com/oauth/authorize?" + params.Encode(), nil
}

// ExchangeCode exchanges the OAuth code for an access token and stores the connection.
func (s *StripeOAuthService) ExchangeCode(ctx context.Context, orgID uuid.UUID, code, state string) error {
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

	// Exchange code for access token via Stripe API
	tokenResp, err := s.exchangeCodeWithStripe(code)
	if err != nil {
		return fmt.Errorf("exchange code with stripe: %w", err)
	}

	// Encrypt access token
	encrypted, err := encryptToken(tokenResp.AccessToken, s.cfg.EncryptionKey)
	if err != nil {
		return fmt.Errorf("encrypt access token: %w", err)
	}

	var refreshEncrypted []byte
	if tokenResp.RefreshToken != "" {
		refreshEncrypted, err = encryptToken(tokenResp.RefreshToken, s.cfg.EncryptionKey)
		if err != nil {
			return fmt.Errorf("encrypt refresh token: %w", err)
		}
	}

	conn := &repository.IntegrationConnection{
		OrgID:                 orgID,
		Provider:              "stripe",
		Status:                "active",
		AccessTokenEncrypted:  encrypted,
		RefreshTokenEncrypted: refreshEncrypted,
		ExternalAccountID:     tokenResp.StripeUserID,
		Scopes:                []string{"read_only"},
		Metadata:              map[string]any{"livemode": tokenResp.Livemode},
	}

	if err := s.connRepo.Upsert(ctx, conn); err != nil {
		return fmt.Errorf("store connection: %w", err)
	}

	slog.Info("stripe connection established",
		"org_id", orgID,
		"stripe_user_id", tokenResp.StripeUserID,
	)

	return nil
}

// GetStatus returns the current Stripe connection status for an org.
func (s *StripeOAuthService) GetStatus(ctx context.Context, orgID uuid.UUID) (*StripeConnectionStatus, error) {
	conn, err := s.connRepo.GetByOrgAndProvider(ctx, orgID, "stripe")
	if err != nil {
		return nil, fmt.Errorf("get connection: %w", err)
	}

	if conn == nil {
		return &StripeConnectionStatus{Status: "disconnected"}, nil
	}

	return &StripeConnectionStatus{
		Status:            conn.Status,
		ExternalAccountID: conn.ExternalAccountID,
		LastSyncAt:        conn.LastSyncAt,
		LastSyncError:     conn.LastSyncError,
		ConnectedAt:       conn.CreatedAt,
	}, nil
}

// Disconnect removes a Stripe connection.
func (s *StripeOAuthService) Disconnect(ctx context.Context, orgID uuid.UUID) error {
	return s.connRepo.Delete(ctx, orgID, "stripe")
}

// GetAccessToken retrieves and decrypts the access token for an org's Stripe connection.
func (s *StripeOAuthService) GetAccessToken(ctx context.Context, orgID uuid.UUID) (string, error) {
	conn, err := s.connRepo.GetByOrgAndProvider(ctx, orgID, "stripe")
	if err != nil {
		return "", fmt.Errorf("get connection: %w", err)
	}
	if conn == nil {
		return "", &NotFoundError{Resource: "stripe_connection", Message: "no Stripe connection found"}
	}
	if conn.Status != "active" {
		return "", &ValidationError{Field: "stripe", Message: "Stripe connection is not active"}
	}

	token, err := decryptToken(conn.AccessTokenEncrypted, s.cfg.EncryptionKey)
	if err != nil {
		return "", fmt.Errorf("decrypt access token: %w", err)
	}
	return token, nil
}

// StripeConnectionStatus holds the status info for frontend display.
type StripeConnectionStatus struct {
	Status            string     `json:"status"`
	ExternalAccountID string     `json:"external_account_id,omitempty"`
	LastSyncAt        *time.Time `json:"last_sync_at,omitempty"`
	LastSyncError     string     `json:"last_sync_error,omitempty"`
	ConnectedAt       time.Time  `json:"connected_at,omitempty"`
}

// stripeTokenResponse holds the Stripe OAuth token exchange response.
type stripeTokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	StripeUserID string `json:"stripe_user_id"`
	Livemode     bool   `json:"livemode"`
}

func (s *StripeOAuthService) exchangeCodeWithStripe(code string) (*stripeTokenResponse, error) {
	data := url.Values{
		"grant_type": {"authorization_code"},
		"code":       {code},
	}

	req, err := http.NewRequest("POST", "https://connect.stripe.com/oauth/token", strings.NewReader(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetBasicAuth(s.cfg.SecretKey, "")

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
		slog.Error("stripe oauth token exchange failed",
			"status", resp.StatusCode,
			"body", string(body),
		)
		return nil, fmt.Errorf("stripe token exchange failed with status %d", resp.StatusCode)
	}

	var tokenResp stripeTokenResponse
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}

	return &tokenResp, nil
}

// encryptToken encrypts a token string using AES-GCM.
func encryptToken(plaintext, keyHex string) ([]byte, error) {
	if keyHex == "" {
		// In dev mode, just store plaintext as bytes (not for production)
		return []byte(plaintext), nil
	}

	key, err := hex.DecodeString(keyHex)
	if err != nil {
		return nil, fmt.Errorf("decode encryption key: %w", err)
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("create gcm: %w", err)
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, fmt.Errorf("generate nonce: %w", err)
	}

	return gcm.Seal(nonce, nonce, []byte(plaintext), nil), nil
}

// decryptToken decrypts a token using AES-GCM.
func decryptToken(ciphertext []byte, keyHex string) (string, error) {
	if keyHex == "" {
		// In dev mode, just return plaintext
		return string(ciphertext), nil
	}

	key, err := hex.DecodeString(keyHex)
	if err != nil {
		return "", fmt.Errorf("decode encryption key: %w", err)
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("create gcm: %w", err)
	}

	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return "", fmt.Errorf("ciphertext too short")
	}

	nonce, encrypted := ciphertext[:nonceSize], ciphertext[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, encrypted, nil)
	if err != nil {
		return "", fmt.Errorf("decrypt: %w", err)
	}

	return string(plaintext), nil
}
