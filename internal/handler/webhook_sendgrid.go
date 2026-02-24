package handler

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/onnwee/pulse-score/internal/repository"
)

// WebhookSendGridHandler handles SendGrid event webhooks for delivery tracking.
type WebhookSendGridHandler struct {
	alertHistory *repository.AlertHistoryRepository
}

// NewWebhookSendGridHandler creates a new WebhookSendGridHandler.
func NewWebhookSendGridHandler(alertHistory *repository.AlertHistoryRepository) *WebhookSendGridHandler {
	return &WebhookSendGridHandler{alertHistory: alertHistory}
}

// sendgridEvent represents a single event from the SendGrid Event Webhook.
type sendgridEvent struct {
	Email     string `json:"email"`
	Timestamp int64  `json:"timestamp"`
	Event     string `json:"event"` // delivered, open, click, bounce, dropped, deferred, etc.
	SGMsgID   string `json:"sg_message_id"`
}

// HandleWebhook handles POST /api/v1/webhooks/sendgrid.
func (h *WebhookSendGridHandler) HandleWebhook(w http.ResponseWriter, r *http.Request) {
	const maxBodySize = 1 << 20 // 1MB
	r.Body = http.MaxBytesReader(w, r.Body, maxBodySize)

	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse("invalid request body"))
		return
	}

	var events []sendgridEvent
	if err := json.Unmarshal(body, &events); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse("invalid JSON"))
		return
	}

	for _, evt := range events {
		if evt.SGMsgID == "" {
			continue
		}

		ts := time.Unix(evt.Timestamp, 0)

		switch evt.Event {
		case "delivered", "open", "click", "bounce", "dropped":
			if err := h.alertHistory.UpdateDeliveryStatus(r.Context(), evt.SGMsgID, evt.Event, ts); err != nil {
				slog.Error("sendgrid webhook: update delivery status",
					"sg_message_id", evt.SGMsgID,
					"event", evt.Event,
					"error", err,
				)
			}
		default:
			// Ignore deferred, processed, etc.
		}
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
