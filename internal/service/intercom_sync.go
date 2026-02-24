package service

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"github.com/onnwee/pulse-score/internal/repository"
)

// IntercomSyncService handles syncing data from Intercom to local database.
type IntercomSyncService struct {
	oauthSvc      *IntercomOAuthService
	client        *IntercomClient
	contacts      *repository.IntercomContactRepository
	conversations *repository.IntercomConversationRepository
	customers     *repository.CustomerRepository
	events        *repository.CustomerEventRepository
}

// NewIntercomSyncService creates a new IntercomSyncService.
func NewIntercomSyncService(
	oauthSvc *IntercomOAuthService,
	client *IntercomClient,
	contacts *repository.IntercomContactRepository,
	conversations *repository.IntercomConversationRepository,
	customers *repository.CustomerRepository,
	events *repository.CustomerEventRepository,
) *IntercomSyncService {
	return &IntercomSyncService{
		oauthSvc:      oauthSvc,
		client:        client,
		contacts:      contacts,
		conversations: conversations,
		customers:     customers,
		events:        events,
	}
}

// SyncContacts fetches all contacts from Intercom and upserts them locally.
func (s *IntercomSyncService) SyncContacts(ctx context.Context, orgID uuid.UUID) (*SyncProgress, error) {
	accessToken, err := s.oauthSvc.GetAccessToken(ctx, orgID)
	if err != nil {
		return nil, fmt.Errorf("get access token: %w", err)
	}

	progress := &SyncProgress{Step: "intercom_contacts"}
	cursor := ""

	for {
		resp, err := s.client.ListContacts(ctx, accessToken, cursor)
		if err != nil {
			return progress, fmt.Errorf("list contacts: %w", err)
		}

		for _, c := range resp.Data {
			progress.Total++

			icContact := &repository.IntercomContact{
				OrgID:             orgID,
				IntercomContactID: c.ID,
				Email:             c.Email,
				Name:              c.Name,
				Role:              c.Role,
				IntercomCompanyID: c.CompanyID,
				Metadata:          map[string]any{},
			}

			if err := s.contacts.Upsert(ctx, icContact); err != nil {
				slog.Error("failed to upsert intercom contact", "intercom_id", c.ID, "error", err)
				progress.Errors++
				continue
			}

			// Upsert into customers table
			now := time.Now()
			localCustomer := &repository.Customer{
				OrgID:       orgID,
				ExternalID:  c.ID,
				Source:      "intercom",
				Email:       c.Email,
				Name:        c.Name,
				FirstSeenAt: &now,
				LastSeenAt:  &now,
				Metadata: map[string]any{
					"intercom": map[string]any{
						"role":       c.Role,
						"company_id": c.CompanyID,
					},
				},
			}

			if err := s.customers.UpsertByExternal(ctx, localCustomer); err != nil {
				slog.Error("failed to upsert customer from intercom", "intercom_id", c.ID, "error", err)
				progress.Errors++
				continue
			}

			// Link the Intercom contact to the local customer
			if err := s.contacts.LinkCustomer(ctx, icContact.ID, localCustomer.ID); err != nil {
				slog.Error("failed to link intercom contact to customer", "error", err)
			}

			progress.Current++
		}

		if resp.Pages == nil || resp.Pages.Next == nil || resp.Pages.Next.StartingAfter == "" {
			break
		}
		cursor = resp.Pages.Next.StartingAfter
	}

	slog.Info("intercom contact sync complete",
		"org_id", orgID,
		"total", progress.Total,
		"synced", progress.Current,
		"errors", progress.Errors,
	)

	return progress, nil
}

// SyncConversations fetches all conversations from Intercom and upserts them locally.
func (s *IntercomSyncService) SyncConversations(ctx context.Context, orgID uuid.UUID) (*SyncProgress, error) {
	accessToken, err := s.oauthSvc.GetAccessToken(ctx, orgID)
	if err != nil {
		return nil, fmt.Errorf("get access token: %w", err)
	}

	progress := &SyncProgress{Step: "intercom_conversations"}
	cursor := ""

	for {
		resp, err := s.client.ListConversations(ctx, accessToken, cursor)
		if err != nil {
			return progress, fmt.Errorf("list conversations: %w", err)
		}

		for _, conv := range resp.Conversations {
			progress.Total++

			contactID := ""
			if conv.Contacts != nil && len(conv.Contacts.Contacts) > 0 {
				contactID = conv.Contacts.Contacts[0].ID
			}

			// Find local customer for this contact
			var customerID *uuid.UUID
			if contactID != "" {
				icContact, err := s.contacts.GetByIntercomID(ctx, orgID, contactID)
				if err == nil && icContact != nil && icContact.CustomerID != nil {
					customerID = icContact.CustomerID
				}
			}

			createdAt := timeFromUnix(conv.CreatedAt)
			updatedAt := timeFromUnix(conv.UpdatedAt)

			var closedAt *time.Time
			var firstResponseAt *time.Time

			if conv.Statistics != nil {
				if conv.Statistics.FirstCloseAt > 0 {
					t := time.Unix(conv.Statistics.FirstCloseAt, 0)
					closedAt = &t
				}
				if conv.Statistics.FirstAdminReplyAt > 0 {
					t := time.Unix(conv.Statistics.FirstAdminReplyAt, 0)
					firstResponseAt = &t
				}
			}

			var rating int
			var ratingRemark string
			if conv.ConversationRating != nil {
				rating = conv.ConversationRating.Rating
				ratingRemark = conv.ConversationRating.Remark
			}

			icConv := &repository.IntercomConversation{
				OrgID:                    orgID,
				CustomerID:               customerID,
				IntercomConversationID:   conv.ID,
				IntercomContactID:        contactID,
				State:                    conv.State,
				Rating:                   rating,
				RatingRemark:             ratingRemark,
				Open:                     conv.Open,
				Read:                     conv.Read,
				Priority:                 conv.Priority,
				Subject:                  conv.Title,
				CreatedAtRemote:          createdAt,
				UpdatedAtRemote:          updatedAt,
				ClosedAt:                 closedAt,
				FirstResponseAt:          firstResponseAt,
				Metadata:                 map[string]any{},
			}

			if err := s.conversations.Upsert(ctx, icConv); err != nil {
				slog.Error("failed to upsert intercom conversation", "intercom_id", conv.ID, "error", err)
				progress.Errors++
				continue
			}

			// Create customer event for conversation
			if customerID != nil {
				eventType := "conversation_" + conv.State
				event := &repository.CustomerEvent{
					OrgID:           orgID,
					CustomerID:      *customerID,
					EventType:       eventType,
					Source:          "intercom",
					ExternalEventID: "conv_" + conv.ID,
					OccurredAt:      time.Now(),
					Data: map[string]any{
						"conversation_id": conv.ID,
						"state":           conv.State,
						"priority":        conv.Priority,
						"subject":         conv.Title,
					},
				}
				if err := s.events.Upsert(ctx, event); err != nil {
					slog.Error("failed to create conversation event", "error", err)
				}
			}

			progress.Current++
		}

		if resp.Pages == nil || resp.Pages.Next == nil || resp.Pages.Next.StartingAfter == "" {
			break
		}
		cursor = resp.Pages.Next.StartingAfter
	}

	slog.Info("intercom conversation sync complete",
		"org_id", orgID,
		"total", progress.Total,
		"synced", progress.Current,
		"errors", progress.Errors,
	)

	return progress, nil
}

// SyncContactsSince fetches contacts updated since the given time (incremental sync).
func (s *IntercomSyncService) SyncContactsSince(ctx context.Context, orgID uuid.UUID, since time.Time) (*SyncProgress, error) {
	accessToken, err := s.oauthSvc.GetAccessToken(ctx, orgID)
	if err != nil {
		return nil, fmt.Errorf("get access token: %w", err)
	}

	progress := &SyncProgress{Step: "intercom_contacts_incremental"}
	cursor := ""

	for {
		resp, err := s.client.ListContactsUpdatedSince(ctx, accessToken, since.Unix(), cursor)
		if err != nil {
			return progress, fmt.Errorf("list contacts since: %w", err)
		}

		for _, c := range resp.Data {
			progress.Total++

			icContact := &repository.IntercomContact{
				OrgID:             orgID,
				IntercomContactID: c.ID,
				Email:             c.Email,
				Name:              c.Name,
				Role:              c.Role,
				IntercomCompanyID: c.CompanyID,
				Metadata:          map[string]any{},
			}

			if err := s.contacts.Upsert(ctx, icContact); err != nil {
				progress.Errors++
				continue
			}

			now := time.Now()
			localCustomer := &repository.Customer{
				OrgID:       orgID,
				ExternalID:  c.ID,
				Source:      "intercom",
				Email:       c.Email,
				Name:        c.Name,
				FirstSeenAt: &now,
				LastSeenAt:  &now,
				Metadata: map[string]any{
					"intercom": map[string]any{
						"role":       c.Role,
						"company_id": c.CompanyID,
					},
				},
			}

			if err := s.customers.UpsertByExternal(ctx, localCustomer); err != nil {
				progress.Errors++
				continue
			}

			if err := s.contacts.LinkCustomer(ctx, icContact.ID, localCustomer.ID); err != nil {
				slog.Error("failed to link intercom contact to customer", "error", err)
			}

			progress.Current++
		}

		if resp.Pages == nil || resp.Pages.Next == nil || resp.Pages.Next.StartingAfter == "" {
			break
		}
		cursor = resp.Pages.Next.StartingAfter
	}

	return progress, nil
}

// SyncConversationsSince fetches conversations updated since the given time (incremental sync).
func (s *IntercomSyncService) SyncConversationsSince(ctx context.Context, orgID uuid.UUID, since time.Time) (*SyncProgress, error) {
	accessToken, err := s.oauthSvc.GetAccessToken(ctx, orgID)
	if err != nil {
		return nil, fmt.Errorf("get access token: %w", err)
	}

	progress := &SyncProgress{Step: "intercom_conversations_incremental"}
	cursor := ""

	for {
		resp, err := s.client.ListConversationsUpdatedSince(ctx, accessToken, since.Unix(), cursor)
		if err != nil {
			return progress, fmt.Errorf("list conversations since: %w", err)
		}

		for _, conv := range resp.Conversations {
			progress.Total++

			contactID := ""
			if conv.Contacts != nil && len(conv.Contacts.Contacts) > 0 {
				contactID = conv.Contacts.Contacts[0].ID
			}

			var customerID *uuid.UUID
			if contactID != "" {
				icContact, err := s.contacts.GetByIntercomID(ctx, orgID, contactID)
				if err == nil && icContact != nil && icContact.CustomerID != nil {
					customerID = icContact.CustomerID
				}
			}

			createdAt := timeFromUnix(conv.CreatedAt)
			updatedAt := timeFromUnix(conv.UpdatedAt)

			var closedAt *time.Time
			var firstResponseAt *time.Time

			if conv.Statistics != nil {
				if conv.Statistics.FirstCloseAt > 0 {
					t := time.Unix(conv.Statistics.FirstCloseAt, 0)
					closedAt = &t
				}
				if conv.Statistics.FirstAdminReplyAt > 0 {
					t := time.Unix(conv.Statistics.FirstAdminReplyAt, 0)
					firstResponseAt = &t
				}
			}

			var rating int
			var ratingRemark string
			if conv.ConversationRating != nil {
				rating = conv.ConversationRating.Rating
				ratingRemark = conv.ConversationRating.Remark
			}

			icConv := &repository.IntercomConversation{
				OrgID:                    orgID,
				CustomerID:               customerID,
				IntercomConversationID:   conv.ID,
				IntercomContactID:        contactID,
				State:                    conv.State,
				Rating:                   rating,
				RatingRemark:             ratingRemark,
				Open:                     conv.Open,
				Read:                     conv.Read,
				Priority:                 conv.Priority,
				Subject:                  conv.Title,
				CreatedAtRemote:          createdAt,
				UpdatedAtRemote:          updatedAt,
				ClosedAt:                 closedAt,
				FirstResponseAt:          firstResponseAt,
				Metadata:                 map[string]any{},
			}

			if err := s.conversations.Upsert(ctx, icConv); err != nil {
				progress.Errors++
				continue
			}

			progress.Current++
		}

		if resp.Pages == nil || resp.Pages.Next == nil || resp.Pages.Next.StartingAfter == "" {
			break
		}
		cursor = resp.Pages.Next.StartingAfter
	}

	return progress, nil
}

func timeFromUnix(ts int64) *time.Time {
	if ts == 0 {
		return nil
	}
	t := time.Unix(ts, 0)
	return &t
}
