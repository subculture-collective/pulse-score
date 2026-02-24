package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"
)

// SendGridConfig holds SendGrid configuration.
type SendGridConfig struct {
	APIKey           string
	FromEmail        string
	FrontendURL      string
	DevMode          bool
	WebhookVerifyKey string
}

// SendGridEmailService implements EmailService using SendGrid.
type SendGridEmailService struct {
	cfg    SendGridConfig
	client *http.Client
}

// NewSendGridEmailService creates a new SendGrid email service.
func NewSendGridEmailService(cfg SendGridConfig) *SendGridEmailService {
	return &SendGridEmailService{
		cfg: cfg,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// SendInvitation sends an invitation email.
func (s *SendGridEmailService) SendInvitation(ctx context.Context, params SendInvitationParams) error {
	acceptURL := fmt.Sprintf("%s/invitations/accept?token=%s", s.cfg.FrontendURL, params.Token)

	if s.cfg.DevMode || s.cfg.APIKey == "" {
		slog.Info("invitation email (dev mode)",
			"to", params.ToEmail,
			"role", params.Role,
			"accept_url", acceptURL,
		)
		return nil
	}

	return s.sendEmailLegacy(ctx, params.ToEmail, "You've been invited to PulseScore",
		fmt.Sprintf(`<h2>You've been invited!</h2>
<p>You've been invited to join an organization on PulseScore as a <strong>%s</strong>.</p>
<p><a href="%s">Accept Invitation</a></p>
<p>This invitation expires in 7 days.</p>`, params.Role, acceptURL))
}

// SendPasswordReset sends a password reset email.
func (s *SendGridEmailService) SendPasswordReset(ctx context.Context, params SendPasswordResetParams) error {
	resetURL := fmt.Sprintf("%s/auth/reset-password?token=%s", s.cfg.FrontendURL, params.Token)

	if s.cfg.DevMode || s.cfg.APIKey == "" {
		slog.Info("password reset email (dev mode)",
			"to", params.ToEmail,
			"reset_url", resetURL,
		)
		return nil
	}

	return s.sendEmailLegacy(ctx, params.ToEmail, "Reset your PulseScore password",
		fmt.Sprintf(`<h2>Password Reset</h2>
<p>Click the link below to reset your password:</p>
<p><a href="%s">Reset Password</a></p>
<p>This link expires in 1 hour. If you didn't request this, ignore this email.</p>`, resetURL))
}

// SendEmail sends a generic email with retry logic. Returns the SendGrid message ID.
func (s *SendGridEmailService) SendEmail(ctx context.Context, params SendEmailParams) (string, error) {
	if s.cfg.DevMode || s.cfg.APIKey == "" {
		slog.Info("email (dev mode)",
			"to", params.To,
			"subject", params.Subject,
		)
		return "dev-mode-message-id", nil
	}

	var lastErr error
	backoffs := []time.Duration{1 * time.Second, 2 * time.Second, 4 * time.Second}

	for attempt := 0; attempt <= len(backoffs); attempt++ {
		if attempt > 0 {
			slog.Info("retrying email send",
				"to", params.To,
				"attempt", attempt+1,
			)
			select {
			case <-ctx.Done():
				return "", ctx.Err()
			case <-time.After(backoffs[attempt-1]):
			}
		}

		messageID, err := s.doSendEmail(ctx, params)
		if err == nil {
			slog.Info("email sent successfully",
				"to", params.To,
				"subject", params.Subject,
				"message_id", messageID,
			)
			return messageID, nil
		}

		lastErr = err

		// Only retry on transient errors (5xx or network issues)
		if sendErr, ok := err.(*sendGridError); ok && sendErr.StatusCode < 500 {
			slog.Error("email send failed (non-retryable)",
				"to", params.To,
				"status", sendErr.StatusCode,
				"error", err,
			)
			return "", err
		}

		slog.Warn("email send failed (retryable)",
			"to", params.To,
			"attempt", attempt+1,
			"error", err,
		)
	}

	return "", fmt.Errorf("email send failed after retries: %w", lastErr)
}

type sendGridError struct {
	StatusCode int
	Message    string
}

func (e *sendGridError) Error() string {
	return fmt.Sprintf("sendgrid status %d: %s", e.StatusCode, e.Message)
}

func (s *SendGridEmailService) doSendEmail(ctx context.Context, params SendEmailParams) (string, error) {
	content := []map[string]string{
		{"type": "text/html", "value": params.HTMLBody},
	}
	if params.TextBody != "" {
		content = append([]map[string]string{
			{"type": "text/plain", "value": params.TextBody},
		}, content...)
	}

	payload := map[string]any{
		"personalizations": []map[string]any{
			{"to": []map[string]string{{"email": params.To}}},
		},
		"from":    map[string]string{"email": s.cfg.FromEmail},
		"subject": params.Subject,
		"content": content,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("marshal payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.sendgrid.com/v3/mail/send", bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+s.cfg.APIKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("send email: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		respBody, _ := io.ReadAll(resp.Body)
		return "", &sendGridError{
			StatusCode: resp.StatusCode,
			Message:    string(respBody),
		}
	}

	messageID := resp.Header.Get("X-Message-Id")
	return messageID, nil
}

// sendEmailLegacy is the original simple send method for backward compatibility.
func (s *SendGridEmailService) sendEmailLegacy(ctx context.Context, to, subject, htmlBody string) error {
	payload := fmt.Sprintf(`{
		"personalizations": [{"to": [{"email": %q}]}],
		"from": {"email": %q},
		"subject": %q,
		"content": [{"type": "text/html", "value": %q}]
	}`, to, s.cfg.FromEmail, subject, htmlBody)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.sendgrid.com/v3/mail/send", bytes.NewBufferString(payload))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+s.cfg.APIKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("send email: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("sendgrid returned status %d", resp.StatusCode)
	}

	return nil
}
