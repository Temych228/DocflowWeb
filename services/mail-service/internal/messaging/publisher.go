package messaging

import (
	"context"

	"github.com/Temych228/DocflowWeb/services/mail-service/internal/domain"
)

type Publisher interface {
	PublishLog(ctx context.Context, event domain.LogEvent) error
	Close() error
}

type NoopPublisher struct{}

func (n *NoopPublisher) PublishLog(context.Context, domain.LogEvent) error {
	return nil
}

func (n *NoopPublisher) Close() error {
	return nil
}
