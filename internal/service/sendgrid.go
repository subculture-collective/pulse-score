package service

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"
)

// SendGridConfig holds SendGrid configuration.
type SendGridConfig struct {
	APIKey      string
	FromEmail   string
	FrontendURL string
	DevMode     bool
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

	return s.sendEmail(ctx, params.ToEmail, "You've been invited to PulseScore",
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

	return s.sendEmail(ctx, params.ToEmail, "Reset your PulseScore password",
		fmt.Sprintf(`<h2>Password Reset</h2>
<p>Click the link below to reset your password:</p>
<p><a href="%s">Reset Password</a></p>
<p>This link expires in 1 hour. If you didn't request this, ignore this email.</p>`, resetURL))
}

func (s *SendGridEmailService) sendEmail(ctx context.Context, to, subject, htmlBody string) error {
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
