package service

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/google/uuid"

	"github.com/onnwee/pulse-score/internal/repository"
)

// NotificationService handles in-app notification business logic.
type NotificationService struct {
	notifRepo *repository.NotificationRepository
	userRepo  *repository.UserRepository
	prefSvc   *NotificationPreferenceService
}

// NewNotificationService creates a new NotificationService.
func NewNotificationService(
	notifRepo *repository.NotificationRepository,
	userRepo *repository.UserRepository,
	prefSvc *NotificationPreferenceService,
) *NotificationService {
	return &NotificationService{
		notifRepo: notifRepo,
		userRepo:  userRepo,
		prefSvc:   prefSvc,
	}
}

// CreateForAlert creates in-app notifications for all recipients of an alert match.
func (s *NotificationService) CreateForAlert(ctx context.Context, match AlertMatch) {
	for _, email := range match.Rule.Recipients {
		user, err := s.userRepo.GetByEmail(ctx, email)
		if err != nil {
			slog.Debug("notification: recipient not a user", "email", email)
			continue
		}

		// Check if user has in-app notifications enabled
		pref, err := s.prefSvc.Get(ctx, user.ID, match.Rule.OrgID)
		if err == nil && !pref.InAppEnabled {
			continue
		}

		// Check if rule is muted
		if err == nil {
			muted := false
			for _, id := range pref.MutedRuleIDs {
				if id == match.Rule.ID {
					muted = true
					break
				}
			}
			if muted {
				continue
			}
		}

		notif := &repository.Notification{
			UserID:  user.ID,
			OrgID:   match.Rule.OrgID,
			Type:    match.Rule.TriggerType,
			Title:   fmt.Sprintf("Alert: %s", match.Rule.Name),
			Message: s.buildMessage(match),
			Data: map[string]any{
				"alert_rule_id": match.Rule.ID,
				"customer_id":   match.Customer.ID,
				"customer_name": match.Customer.Name,
				"trigger_data":  match.TriggerData,
			},
		}

		if err := s.notifRepo.Create(ctx, notif); err != nil {
			slog.Error("notification: create failed", "user_id", user.ID, "error", err)
		}
	}
}

func (s *NotificationService) buildMessage(match AlertMatch) string {
	switch match.Rule.TriggerType {
	case "score_below":
		score, _ := match.TriggerData["score"].(int)
		threshold, _ := match.TriggerData["threshold"].(int)
		return fmt.Sprintf("%s health score (%d) dropped below threshold (%d)", match.Customer.Name, score, threshold)
	case "score_drop":
		delta, _ := match.TriggerData["delta"].(int)
		return fmt.Sprintf("%s health score dropped by %d points", match.Customer.Name, delta)
	case "risk_change":
		newLevel, _ := match.TriggerData["new_risk_level"].(string)
		return fmt.Sprintf("%s risk level changed to %s", match.Customer.Name, newLevel)
	default:
		return fmt.Sprintf("Alert triggered for %s", match.Customer.Name)
	}
}

// List returns notifications for the current user.
func (s *NotificationService) List(ctx context.Context, userID, orgID uuid.UUID, limit, offset int) ([]*repository.Notification, int, error) {
	return s.notifRepo.ListByUser(ctx, userID, orgID, limit, offset)
}

// CountUnread returns unread notification count for the current user.
func (s *NotificationService) CountUnread(ctx context.Context, userID, orgID uuid.UUID) (int, error) {
	return s.notifRepo.CountUnread(ctx, userID, orgID)
}

// MarkRead marks a notification as read.
func (s *NotificationService) MarkRead(ctx context.Context, id, userID uuid.UUID) error {
	return s.notifRepo.MarkRead(ctx, id, userID)
}

// MarkAllRead marks all notifications as read for the current user.
func (s *NotificationService) MarkAllRead(ctx context.Context, userID, orgID uuid.UUID) error {
	return s.notifRepo.MarkAllRead(ctx, userID, orgID)
}
