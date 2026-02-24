package service

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"log/slog"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/onnwee/pulse-score/internal/repository"
)

// IntercomWebhookEvent represents an incoming Intercom webhook event.
type IntercomWebhookEvent struct {
	Type      string                 `json:"type"`
	AppID     string                 `json:"app_id"`
	Data      IntercomWebhookData    `json:"data"`
	ID        string                 `json:"id"`
	Topic     string                 `json:"topic"`
	CreatedAt int64                  `json:"created_at"`
	DeliveryAttempts int             `json:"delivery_attempts"`
}

// IntercomWebhookData holds the data section of a webhook event.
type IntercomWebhookData struct {
	Type string                 `json:"type"`
	Item map[string]any         `json:"item"`
}

// IntercomWebhookService handles incoming Intercom webhook events.
type IntercomWebhookService struct {
	webhookSecret string
	syncSvc       *IntercomSyncService
	mergeSvc      *CustomerMergeService
	connRepo      *repository.IntegrationConnectionRepository
	contacts      *repository.IntercomContactRepository
	conversations *repository.IntercomConversationRepository
	events        *repository.CustomerEventRepository

	processedEvents map[string]time.Time
	mu              sync.Mutex
}

// NewIntercomWebhookService creates a new IntercomWebhookService.
func NewIntercomWebhookService(
	webhookSecret string,
	syncSvc *IntercomSyncService,
	mergeSvc *CustomerMergeService,
	connRepo *repository.IntegrationConnectionRepository,
	contacts *repository.IntercomContactRepository,
	conversations *repository.IntercomConversationRepository,
	events *repository.CustomerEventRepository,
) *IntercomWebhookService {
	return &IntercomWebhookService{
		webhookSecret:   webhookSecret,
		syncSvc:         syncSvc,
		mergeSvc:        mergeSvc,
		connRepo:        connRepo,
		contacts:        contacts,
		conversations:   conversations,
		events:          events,
		processedEvents: make(map[string]time.Time),
	}
}

// VerifySignature verifies the Intercom webhook signature.
// Intercom signs webhooks with HMAC-SHA256 using the webhook secret.
func (s *IntercomWebhookService) VerifySignature(requestBody []byte, signatureHeader string) error {
	if signatureHeader == "" {
		return &ValidationError{Field: "signature", Message: "missing signature header"}
	}

	mac := hmac.New(sha256.New, []byte(s.webhookSecret))
	mac.Write(requestBody)
	expectedSig := hex.EncodeToString(mac.Sum(nil))

	if !hmac.Equal([]byte(expectedSig), []byte(signatureHeader)) {
		return &ValidationError{Field: "signature", Message: "invalid webhook signature"}
	}

	return nil
}

// ProcessEvent processes a single Intercom webhook event.
func (s *IntercomWebhookService) ProcessEvent(ctx context.Context, event IntercomWebhookEvent) error {
	if s.isProcessed(event.ID) {
		slog.Debug("duplicate intercom webhook event skipped", "event_id", event.ID)
		return nil
	}

	// Find the org for this app
	conn, err := s.connRepo.GetByProviderAndExternalID(ctx, "intercom", event.AppID)
	if err != nil {
		slog.Error("failed to lookup intercom connection", "app_id", event.AppID, "error", err)
		return err
	}
	if conn == nil {
		slog.Warn("no intercom connection for app", "app_id", event.AppID)
		return nil
	}

	if err := s.processEventForOrg(ctx, conn.OrgID, event); err != nil {
		slog.Error("failed to process intercom event",
			"event_id", event.ID,
			"topic", event.Topic,
			"error", err,
		)
		return err
	}

	s.markProcessed(event.ID)
	return nil
}

func (s *IntercomWebhookService) processEventForOrg(ctx context.Context, orgID uuid.UUID, event IntercomWebhookEvent) error {
	switch event.Topic {
	case "conversation.created":
		return s.handleConversationCreated(ctx, orgID, event)
	case "conversation.closed":
		return s.handleConversationClosed(ctx, orgID, event)
	case "conversation.admin.replied":
		return s.handleConversationAdminReplied(ctx, orgID, event)
	case "conversation.user.replied":
		return s.handleConversationUserReplied(ctx, orgID, event)
	case "conversation.rating.added":
		return s.handleConversationRatingAdded(ctx, orgID, event)
	default:
		slog.Debug("ignoring unknown intercom event topic", "topic", event.Topic)
		return nil
	}
}

func (s *IntercomWebhookService) handleConversationCreated(ctx context.Context, orgID uuid.UUID, event IntercomWebhookEvent) error {
	convID, _ := event.Data.Item["id"].(string)
	if convID == "" {
		return nil
	}

	contactID := extractIntercomContactID(event.Data.Item)
	var customerID *uuid.UUID
	if contactID != "" {
		if icContact, err := s.contacts.GetByIntercomID(ctx, orgID, contactID); err == nil && icContact != nil && icContact.CustomerID != nil {
			customerID = icContact.CustomerID
		}
	}

	title, _ := event.Data.Item["title"].(string)
	priority, _ := event.Data.Item["priority"].(string)

	conv := &repository.IntercomConversation{
		OrgID:                  orgID,
		CustomerID:             customerID,
		IntercomConversationID: convID,
		IntercomContactID:      contactID,
		State:                  "open",
		Open:                   true,
		Priority:               priority,
		Subject:                title,
		CreatedAtRemote:        timeFromUnix(event.CreatedAt),
		UpdatedAtRemote:        timeFromUnix(event.CreatedAt),
		Metadata:               map[string]any{},
	}

	if err := s.conversations.Upsert(ctx, conv); err != nil {
		return err
	}

	if customerID != nil {
		customerEvent := &repository.CustomerEvent{
			OrgID:           orgID,
			CustomerID:      *customerID,
			EventType:       "conversation_created",
			Source:          "intercom",
			ExternalEventID: "conv_created_" + convID,
			OccurredAt:      time.Now(),
			Data: map[string]any{
				"conversation_id": convID,
				"priority":        priority,
				"subject":         title,
			},
		}
		if err := s.events.Upsert(ctx, customerEvent); err != nil {
			slog.Error("failed to create conversation_created event", "error", err)
		}
	}

	slog.Info("intercom conversation created via webhook", "conversation_id", convID)
	return nil
}

func (s *IntercomWebhookService) handleConversationClosed(ctx context.Context, orgID uuid.UUID, event IntercomWebhookEvent) error {
	convID, _ := event.Data.Item["id"].(string)
	if convID == "" {
		return nil
	}

	conv, err := s.conversations.GetByIntercomID(ctx, orgID, convID)
	if err != nil {
		return err
	}
	if conv == nil {
		return nil
	}

	now := time.Now()
	conv.State = "closed"
	conv.Open = false
	conv.ClosedAt = &now
	conv.UpdatedAtRemote = &now

	if err := s.conversations.Upsert(ctx, conv); err != nil {
		return err
	}

	if conv.CustomerID != nil {
		customerEvent := &repository.CustomerEvent{
			OrgID:           orgID,
			CustomerID:      *conv.CustomerID,
			EventType:       "conversation_closed",
			Source:          "intercom",
			ExternalEventID: "conv_closed_" + convID,
			OccurredAt:      now,
			Data: map[string]any{
				"conversation_id": convID,
			},
		}
		if err := s.events.Upsert(ctx, customerEvent); err != nil {
			slog.Error("failed to create conversation_closed event", "error", err)
		}
	}

	slog.Info("intercom conversation closed via webhook", "conversation_id", convID)
	return nil
}

func (s *IntercomWebhookService) handleConversationAdminReplied(ctx context.Context, orgID uuid.UUID, event IntercomWebhookEvent) error {
	convID, _ := event.Data.Item["id"].(string)
	if convID == "" {
		return nil
	}

	conv, err := s.conversations.GetByIntercomID(ctx, orgID, convID)
	if err != nil || conv == nil {
		return err
	}

	now := time.Now()
	conv.UpdatedAtRemote = &now
	if conv.FirstResponseAt == nil {
		conv.FirstResponseAt = &now
	}

	if err := s.conversations.Upsert(ctx, conv); err != nil {
		return err
	}

	slog.Debug("intercom admin replied via webhook", "conversation_id", convID)
	return nil
}

func (s *IntercomWebhookService) handleConversationUserReplied(ctx context.Context, orgID uuid.UUID, event IntercomWebhookEvent) error {
	convID, _ := event.Data.Item["id"].(string)
	if convID == "" {
		return nil
	}

	conv, err := s.conversations.GetByIntercomID(ctx, orgID, convID)
	if err != nil || conv == nil {
		return err
	}

	now := time.Now()
	conv.UpdatedAtRemote = &now

	if err := s.conversations.Upsert(ctx, conv); err != nil {
		return err
	}

	slog.Debug("intercom user replied via webhook", "conversation_id", convID)
	return nil
}

func (s *IntercomWebhookService) handleConversationRatingAdded(ctx context.Context, orgID uuid.UUID, event IntercomWebhookEvent) error {
	convID, _ := event.Data.Item["id"].(string)
	if convID == "" {
		return nil
	}

	conv, err := s.conversations.GetByIntercomID(ctx, orgID, convID)
	if err != nil || conv == nil {
		return err
	}

	if ratingData, ok := event.Data.Item["conversation_rating"].(map[string]any); ok {
		if ratingVal, ok := ratingData["rating"].(float64); ok {
			conv.Rating = int(ratingVal)
		}
		if remark, ok := ratingData["remark"].(string); ok {
			conv.RatingRemark = remark
		}
	}

	now := time.Now()
	conv.UpdatedAtRemote = &now

	if err := s.conversations.Upsert(ctx, conv); err != nil {
		return err
	}

	slog.Info("intercom conversation rating added via webhook", "conversation_id", convID, "rating", conv.Rating)
	return nil
}

func extractIntercomContactID(item map[string]any) string {
	contactsData, ok := item["contacts"].(map[string]any)
	if !ok {
		return ""
	}
	contacts, ok := contactsData["contacts"].([]any)
	if !ok || len(contacts) == 0 {
		return ""
	}
	first, ok := contacts[0].(map[string]any)
	if !ok {
		return ""
	}
	id, _ := first["id"].(string)
	return id
}

func (s *IntercomWebhookService) isProcessed(eventID string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.processedEvents[eventID]; exists {
		return true
	}

	// Clean up old entries (older than 1 hour)
	cutoff := time.Now().Add(-1 * time.Hour)
	for id, t := range s.processedEvents {
		if t.Before(cutoff) {
			delete(s.processedEvents, id)
		}
	}

	return false
}

func (s *IntercomWebhookService) markProcessed(eventID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.processedEvents[eventID] = time.Now()
}
