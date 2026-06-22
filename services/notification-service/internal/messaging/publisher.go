package messaging

import (
	"context"

	"github.com/Temych228/DocflowWeb/services/notification-service/internal/domain"
)

type Publisher interface {
	PublishLog(ctx context.Context, event domain.LogEvent) error
	PublishEmailJob(ctx context.Context, job domain.EmailJob) (string, error)
	PublishNotificationEvent(ctx context.Context, payload any) error
	Close() error
}

type NoopPublisher struct{}

func (n *NoopPublisher) PublishLog(context.Context, domain.LogEvent) error {
	return nil
}

func (n *NoopPublisher) PublishEmailJob(context.Context, domain.EmailJob) (string, error) {
	return "", nil
}

func (n *NoopPublisher) PublishNotificationEvent(context.Context, any) error {
	return nil
}

func (n *NoopPublisher) Close() error {
	return nil
}
