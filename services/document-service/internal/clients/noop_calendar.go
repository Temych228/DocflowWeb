package clients

import (
	"context"
	"log/slog"
)

type NoopCalendarClient struct {
	logger *slog.Logger
}

func NewNoopCalendarClient(logger *slog.Logger) *NoopCalendarClient {
	return &NoopCalendarClient{logger: logger}
}

func (c *NoopCalendarClient) CreateEvent(ctx context.Context, input CreateEventInput) error {
	c.logger.Info("calendar event skipped, client not configured",
		"user_id", input.UserID,
		"event_type", input.EventType,
		"ref_id", input.RefID,
	)
	return nil
}
