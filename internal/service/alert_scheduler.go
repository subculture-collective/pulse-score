package service

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"github.com/onnwee/pulse-score/internal/repository"
)

// AlertScheduler runs periodic evaluation of alert rules.
type AlertScheduler struct {
	engine       *AlertEngine
	emailService EmailService
	templates    *EmailTemplateService
	alertHistory *repository.AlertHistoryRepository
	alertRules   *repository.AlertRuleRepository
	userRepo     *repository.UserRepository
	notifPrefSvc *NotificationPreferenceService
	notifService *NotificationService
	interval     time.Duration
	frontendURL  string
}

// AlertSchedulerDeps holds constructor dependencies for AlertScheduler.
type AlertSchedulerDeps struct {
	Engine       *AlertEngine
	EmailService EmailService
	Templates    *EmailTemplateService
	AlertHistory *repository.AlertHistoryRepository
	AlertRules   *repository.AlertRuleRepository
	UserRepo     *repository.UserRepository
	NotifPrefSvc *NotificationPreferenceService
}

// NewAlertScheduler creates a new AlertScheduler.
func NewAlertScheduler(
	deps AlertSchedulerDeps,
	intervalMinutes int,
	frontendURL string,
) *AlertScheduler {
	return &AlertScheduler{
		engine:       deps.Engine,
		emailService: deps.EmailService,
		templates:    deps.Templates,
		alertHistory: deps.AlertHistory,
		alertRules:   deps.AlertRules,
		userRepo:     deps.UserRepo,
		notifPrefSvc: deps.NotifPrefSvc,
		interval:     time.Duration(intervalMinutes) * time.Minute,
		frontendURL:  frontendURL,
	}
}

// SetNotificationService sets the in-app notification service for creating notifications alongside emails.
func (s *AlertScheduler) SetNotificationService(notifSvc *NotificationService) {
	s.notifService = notifSvc
}

// Start begins the periodic alert evaluation loop. Cancel the context to stop.
func (s *AlertScheduler) Start(ctx context.Context) {
	slog.Info("alert scheduler started", "interval", s.interval)

	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			slog.Info("alert scheduler stopped")
			return
		case <-ticker.C:
			s.RunOnce(ctx)
		}
	}
}

// RunOnce performs a single evaluation pass across all orgs with active rules.
func (s *AlertScheduler) RunOnce(ctx context.Context) {
	orgIDs, err := s.alertHistory.ListOrgsWithActiveRules(ctx)
	if err != nil {
		slog.Error("alert scheduler: list orgs", "error", err)
		return
	}

	for _, orgID := range orgIDs {
		if ctx.Err() != nil {
			return
		}
		s.evaluateOrg(ctx, orgID)
	}
}

func (s *AlertScheduler) evaluateOrg(ctx context.Context, orgID uuid.UUID) {
	matches, err := s.engine.EvaluateAll(ctx, orgID)
	if err != nil {
		slog.Error("alert scheduler: evaluate org", "org_id", orgID, "error", err)
		return
	}

	for _, match := range matches {
		if ctx.Err() != nil {
			return
		}
		s.ProcessMatch(ctx, match)
	}

	if len(matches) > 0 {
		slog.Info("alert scheduler: evaluation complete", "org_id", orgID, "matches", len(matches))
	}
}

func (s *AlertScheduler) ProcessMatch(ctx context.Context, match AlertMatch) {
	// Create pending history record
	customerID := &match.Customer.ID
	history := &repository.AlertHistory{
		OrgID:       match.Rule.OrgID,
		AlertRuleID: match.Rule.ID,
		CustomerID:  customerID,
		TriggerData: match.TriggerData,
		Channel:     match.Rule.Channel,
		Status:      "pending",
	}
	if err := s.alertHistory.Create(ctx, history); err != nil {
		slog.Error("alert scheduler: create history", "error", err)
		return
	}

	// Render email
	subject, htmlBody, textBody, err := s.renderEmail(match)
	if err != nil {
		slog.Error("alert scheduler: render email",
			"rule_id", match.Rule.ID,
			"trigger_type", match.Rule.TriggerType,
			"error", err,
		)
		_ = s.alertHistory.UpdateStatus(ctx, history.ID, "failed", err.Error())
		return
	}

	// Send to each recipient (respecting notification preferences)
	for _, recipient := range match.Rule.Recipients {
		if s.userRepo != nil && s.notifPrefSvc != nil {
			user, err := s.userRepo.GetByEmail(ctx, recipient)
			if err == nil && !s.notifPrefSvc.ShouldNotifyEmail(ctx, user.ID, match.Rule.OrgID, match.Rule.ID, time.Now()) {
				slog.Debug("alert scheduler: skipping muted/disabled recipient", "recipient", recipient, "rule_id", match.Rule.ID)
				continue
			}
		}

		msgID, err := s.emailService.SendEmail(ctx, SendEmailParams{
			To:       recipient,
			Subject:  subject,
			HTMLBody: htmlBody,
			TextBody: textBody,
		})
		if err != nil {
			slog.Error("alert scheduler: send email",
				"recipient", recipient,
				"rule_id", match.Rule.ID,
				"error", err,
			)
			_ = s.alertHistory.UpdateStatus(ctx, history.ID, "failed", err.Error())
			return
		}

		if msgID != "" {
			_ = s.alertHistory.UpdateSendGridMessageID(ctx, history.ID, msgID)
		}
	}

	_ = s.alertHistory.UpdateStatus(ctx, history.ID, "sent", "")

	// Create in-app notifications for recipients
	if s.notifService != nil {
		s.notifService.CreateForAlert(ctx, match)
	}
}

func (s *AlertScheduler) renderEmail(match AlertMatch) (subject, html, text string, err error) {
	customerURL := fmt.Sprintf("%s/customers/%s", s.frontendURL, match.Customer.ID)
	unsubURL := fmt.Sprintf("%s/settings?tab=notifications", s.frontendURL)

	switch match.Rule.TriggerType {
	case "score_below":
		score, _ := match.TriggerData["score"].(int)
		threshold, _ := match.TriggerData["threshold"].(int)
		riskLevel, _ := match.TriggerData["risk_level"].(string)

		subject = fmt.Sprintf("Alert: %s health score below %d", match.Customer.Name, threshold)
		html, text, err = s.templates.RenderScoreDrop(ScoreDropEmailData{
			CustomerName:      match.Customer.Name,
			CompanyName:       match.Customer.CompanyName,
			OldScore:          threshold,
			NewScore:          score,
			Delta:             threshold - score,
			TopNegativeFactor: riskLevel,
			CustomerDetailURL: customerURL,
			UnsubscribeURL:    unsubURL,
		})

	case "score_drop":
		oldScore, _ := match.TriggerData["old_score"].(int)
		newScore, _ := match.TriggerData["new_score"].(int)
		delta, _ := match.TriggerData["delta"].(int)
		factor, _ := match.TriggerData["biggest_contributing_factor"].(string)

		subject = fmt.Sprintf("Alert: %s health score dropped %d points", match.Customer.Name, -delta)
		html, text, err = s.templates.RenderScoreDrop(ScoreDropEmailData{
			CustomerName:      match.Customer.Name,
			CompanyName:       match.Customer.CompanyName,
			OldScore:          oldScore,
			NewScore:          newScore,
			Delta:             -delta,
			TopNegativeFactor: factor,
			CustomerDetailURL: customerURL,
			UnsubscribeURL:    unsubURL,
		})

	case "risk_change":
		prevLevel, _ := match.TriggerData["previous_level"].(string)
		newLevel, _ := match.TriggerData["new_level"].(string)
		score := extractInt(match.TriggerData, "score")

		subject = fmt.Sprintf("Alert: %s risk level changed to %s", match.Customer.Name, newLevel)
		html, text, err = s.templates.RenderRiskChange(RiskChangeEmailData{
			CustomerName:      match.Customer.Name,
			CompanyName:       match.Customer.CompanyName,
			PreviousLevel:     prevLevel,
			NewLevel:          newLevel,
			Score:             score,
			CustomerDetailURL: customerURL,
			UnsubscribeURL:    unsubURL,
		})

	case "payment_failed":
		amount, _ := match.TriggerData["amount"].(string)
		reason, _ := match.TriggerData["reason"].(string)
		if amount == "" {
			amount = "N/A"
		}
		if reason == "" {
			reason = "Unknown"
		}

		subject = fmt.Sprintf("Alert: Payment failed for %s", match.Customer.Name)
		html, text, err = s.templates.RenderPaymentFailed(PaymentFailedEmailData{
			CustomerName:      match.Customer.Name,
			CompanyName:       match.Customer.CompanyName,
			Amount:            amount,
			FailureReason:     reason,
			CustomerDetailURL: customerURL,
			UnsubscribeURL:    unsubURL,
		})

	default:
		err = fmt.Errorf("unsupported trigger type: %s", match.Rule.TriggerType)
	}

	return subject, html, text, err
}

func extractInt(data map[string]any, key string) int {
	v, ok := data[key]
	if !ok {
		return 0
	}
	switch val := v.(type) {
	case int:
		return val
	case float64:
		return int(val)
	}
	return 0
}
