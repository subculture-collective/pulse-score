package service

import (
	"context"

	"github.com/google/uuid"
)

// SendInvitationParams holds parameters for sending an invitation email.
type SendInvitationParams struct {
	ToEmail   string
	Token     string
	OrgID     uuid.UUID
	InviterID uuid.UUID
	Role      string
}

// SendPasswordResetParams holds parameters for sending a password reset email.
type SendPasswordResetParams struct {
	ToEmail string
	Token   string
}

// SendEmailParams holds parameters for sending a generic email.
type SendEmailParams struct {
	To       string
	Subject  string
	HTMLBody string
	TextBody string
}

// AlertEmailParams holds parameters for sending an alert email.
type AlertEmailParams struct {
	To          string
	Subject     string
	HTMLBody    string
	TextBody    string
	AlertRuleID uuid.UUID
	CustomerID  *uuid.UUID
}

// EmailService defines the interface for sending emails.
type EmailService interface {
	SendInvitation(ctx context.Context, params SendInvitationParams) error
	SendPasswordReset(ctx context.Context, params SendPasswordResetParams) error
	SendEmail(ctx context.Context, params SendEmailParams) (messageID string, err error)
}
