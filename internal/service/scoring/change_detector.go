package scoring

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"time"

	"github.com/onnwee/pulse-score/internal/repository"
)

// ChangeDetector identifies significant health score changes and records events.
type ChangeDetector struct {
	events           *repository.CustomerEventRepository
	significantDelta float64 // absolute point change threshold (default 10)
}

// NewChangeDetector creates a new ChangeDetector.
func NewChangeDetector(events *repository.CustomerEventRepository, significantDelta float64) *ChangeDetector {
	if significantDelta <= 0 {
		significantDelta = 10.0
	}
	return &ChangeDetector{
		events:           events,
		significantDelta: significantDelta,
	}
}

// DetectAndRecord compares a new score against the previous and records change events.
func (d *ChangeDetector) DetectAndRecord(ctx context.Context, previous *repository.HealthScore, current *HealthScoreResult) error {
	if current == nil {
		return nil
	}
	if previous == nil {
		// First score ever — record initial score event
		return d.recordEvent(ctx, current, "score.initial", map[string]any{
			"score":      current.OverallScore,
			"risk_level": current.RiskLevel,
		})
	}

	delta := float64(current.OverallScore - previous.OverallScore)

	// Check for significant score change
	if math.Abs(delta) >= d.significantDelta {
		direction := "improved"
		if delta < 0 {
			direction = "declined"
		}

		if err := d.recordEvent(ctx, current, "score.changed", map[string]any{
			"previous_score": previous.OverallScore,
			"new_score":      current.OverallScore,
			"delta":          delta,
			"direction":      direction,
		}); err != nil {
			return err
		}

		slog.Info("significant score change detected",
			"customer_id", current.CustomerID,
			"previous", previous.OverallScore,
			"new", current.OverallScore,
			"delta", delta,
		)
	}

	// Check for risk level transition
	if current.RiskLevel != previous.RiskLevel {
		if err := d.recordEvent(ctx, current, "risk_level.changed", map[string]any{
			"previous_level": previous.RiskLevel,
			"new_level":      current.RiskLevel,
			"score":          current.OverallScore,
		}); err != nil {
			return err
		}

		slog.Info("risk level transition detected",
			"customer_id", current.CustomerID,
			"from", previous.RiskLevel,
			"to", current.RiskLevel,
		)
	}

	return nil
}

func (d *ChangeDetector) recordEvent(ctx context.Context, result *HealthScoreResult, eventType string, data map[string]any) error {
	event := &repository.CustomerEvent{
		OrgID:      result.OrgID,
		CustomerID: result.CustomerID,
		EventType:  eventType,
		Source:     "health_scoring",
		OccurredAt: time.Now(),
		Data:       data,
	}

	if err := d.events.Upsert(ctx, event); err != nil {
		return fmt.Errorf("record %s event: %w", eventType, err)
	}
	return nil
}
