package clients

import (
	"context"
	"log/slog"
)

type NoopNotificationClient struct {
	logger *slog.Logger
}

func NewNoopNotificationClient(logger *slog.Logger) *NoopNotificationClient {
	return &NoopNotificationClient{logger: logger}
}

func (c *NoopNotificationClient) CreateNotification(ctx context.Context, input CreateNotificationInput) error {
	c.logger.Info("notification skipped, client not configured",
		"user_id", input.UserID,
		"category", input.NotifCategory,
		"ref_id", input.RefID,
	)
	return nil
}
