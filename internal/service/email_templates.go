package service

import (
	"bytes"
	"embed"
	"fmt"
	"html/template"
	"strings"
)

//go:embed templates/emails/*.html
var emailTemplatesFS embed.FS

// EmailTemplateService renders email templates.
type EmailTemplateService struct {
	scoreDrop     *template.Template
	riskChange    *template.Template
	paymentFailed *template.Template
	weeklyDigest  *template.Template
}

// NewEmailTemplateService creates a new EmailTemplateService loading embedded templates.
func NewEmailTemplateService() (*EmailTemplateService, error) {
	parse := func(name string) (*template.Template, error) {
		base, err := emailTemplatesFS.ReadFile("templates/emails/base.html")
		if err != nil {
			return nil, fmt.Errorf("read base template: %w", err)
		}
		content, err := emailTemplatesFS.ReadFile("templates/emails/" + name)
		if err != nil {
			return nil, fmt.Errorf("read %s template: %w", name, err)
		}
		// Parse base first, then the content template which overrides the "content" block
		t, err := template.New(name).Parse(string(base))
		if err != nil {
			return nil, fmt.Errorf("parse base for %s: %w", name, err)
		}
		t, err = t.Parse(string(content))
		if err != nil {
			return nil, fmt.Errorf("parse %s: %w", name, err)
		}
		return t, nil
	}

	scoreDrop, err := parse("score_drop.html")
	if err != nil {
		return nil, err
	}
	riskChange, err := parse("at_risk.html")
	if err != nil {
		return nil, err
	}
	paymentFailed, err := parse("payment_failed.html")
	if err != nil {
		return nil, err
	}
	weeklyDigest, err := parse("weekly_digest.html")
	if err != nil {
		return nil, err
	}

	return &EmailTemplateService{
		scoreDrop:     scoreDrop,
		riskChange:    riskChange,
		paymentFailed: paymentFailed,
		weeklyDigest:  weeklyDigest,
	}, nil
}

// ScoreDropEmailData holds data for score drop email template.
type ScoreDropEmailData struct {
	CustomerName      string
	CompanyName       string
	OldScore          int
	NewScore          int
	Delta             int
	TopNegativeFactor string
	CustomerDetailURL string
	UnsubscribeURL    string
}

// RiskChangeEmailData holds data for risk change email template.
type RiskChangeEmailData struct {
	CustomerName      string
	CompanyName       string
	PreviousLevel     string
	NewLevel          string
	Score             int
	CustomerDetailURL string
	UnsubscribeURL    string
}

// PaymentFailedEmailData holds data for payment failed email template.
type PaymentFailedEmailData struct {
	CustomerName      string
	CompanyName       string
	Amount            string
	FailureReason     string
	CustomerDetailURL string
	UnsubscribeURL    string
}

// CustomerScoreChange represents a score change for the weekly digest.
type CustomerScoreChange struct {
	Name     string
	OldScore int
	NewScore int
	Delta    int
}

// WeeklyDigestEmailData holds data for weekly digest email template.
type WeeklyDigestEmailData struct {
	OrgName        string
	TotalCustomers int
	AtRiskCount    int
	ImprovedCount  int
	DeclinedCount  int
	TopMovers      []CustomerScoreChange
	DashboardURL   string
	UnsubscribeURL string
}

// RenderScoreDrop renders the score drop email template.
func (s *EmailTemplateService) RenderScoreDrop(data ScoreDropEmailData) (html string, text string, err error) {
	html, err = renderTemplate(s.scoreDrop, data)
	if err != nil {
		return "", "", err
	}
	text = fmt.Sprintf(
		"Health Score Drop Alert\n\n%s's health score dropped from %d to %d (-%d points).\n\nBiggest factor: %s\n\nView details: %s",
		data.CustomerName, data.OldScore, data.NewScore, data.Delta, data.TopNegativeFactor, data.CustomerDetailURL,
	)
	return html, text, nil
}

// RenderRiskChange renders the risk change email template.
func (s *EmailTemplateService) RenderRiskChange(data RiskChangeEmailData) (html string, text string, err error) {
	html, err = renderTemplate(s.riskChange, data)
	if err != nil {
		return "", "", err
	}
	text = fmt.Sprintf(
		"Risk Level Change Alert\n\n%s's risk level changed from %s to %s.\nCurrent score: %d\n\nView details: %s",
		data.CustomerName, data.PreviousLevel, data.NewLevel, data.Score, data.CustomerDetailURL,
	)
	return html, text, nil
}

// RenderPaymentFailed renders the payment failed email template.
func (s *EmailTemplateService) RenderPaymentFailed(data PaymentFailedEmailData) (html string, text string, err error) {
	html, err = renderTemplate(s.paymentFailed, data)
	if err != nil {
		return "", "", err
	}
	text = fmt.Sprintf(
		"Payment Failed Alert\n\nPayment failed for %s.\nAmount: %s\nReason: %s\n\nView details: %s",
		data.CustomerName, data.Amount, data.FailureReason, data.CustomerDetailURL,
	)
	return html, text, nil
}

// RenderWeeklyDigest renders the weekly digest email template.
func (s *EmailTemplateService) RenderWeeklyDigest(data WeeklyDigestEmailData) (html string, text string, err error) {
	html, err = renderTemplate(s.weeklyDigest, data)
	if err != nil {
		return "", "", err
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Weekly Health Digest — %s\n\n", data.OrgName))
	sb.WriteString(fmt.Sprintf("Total Customers: %d\n", data.TotalCustomers))
	sb.WriteString(fmt.Sprintf("At Risk: %d\n", data.AtRiskCount))
	sb.WriteString(fmt.Sprintf("Improved: %d\n", data.ImprovedCount))
	sb.WriteString(fmt.Sprintf("Declined: %d\n\n", data.DeclinedCount))
	if len(data.TopMovers) > 0 {
		sb.WriteString("Top Movers:\n")
		for _, m := range data.TopMovers {
			sb.WriteString(fmt.Sprintf("  %s: %d → %d (%+d)\n", m.Name, m.OldScore, m.NewScore, m.Delta))
		}
	}
	sb.WriteString(fmt.Sprintf("\nView dashboard: %s", data.DashboardURL))
	text = sb.String()
	return html, text, nil
}

func renderTemplate(t *template.Template, data any) (string, error) {
	var buf bytes.Buffer
	if err := t.ExecuteTemplate(&buf, "base", data); err != nil {
		return "", fmt.Errorf("render template: %w", err)
	}
	return buf.String(), nil
}
