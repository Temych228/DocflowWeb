package repository

import (
	"context"
	"time"

	"github.com/Temych228/DocflowWeb/services/notification-service/internal/domain"
)

type NotificationRepository interface {
	CreateNotification(ctx context.Context, n *domain.Notification) (*domain.Notification, error)
	GetHistory(ctx context.Context, userID string, page, pageSize int) ([]*domain.Notification, int, error)
	MarkRead(ctx context.Context, notificationID, userID string) (*domain.Notification, error)
	MarkAllRead(ctx context.Context, userID string) (int32, error)
	Delete(ctx context.Context, notificationID, userID string) error
	GetUnreadCount(ctx context.Context, userID string) (int32, error)
	SetUnreadCountCache(ctx context.Context, userID string, count int32, ttl time.Duration) error
	InvalidateUnreadCountCache(ctx context.Context, userID string) error
	GetPreferences(ctx context.Context, userID string) (*domain.Preferences, error)
	UpsertPreferences(ctx context.Context, p *domain.Preferences) (*domain.Preferences, error)
	GetTemplate(ctx context.Context, templateID string) (*domain.Template, error)
	UpsertTemplate(ctx context.Context, t *domain.Template) (*domain.Template, error)
	SetEmailQueued(ctx context.Context, notificationID string) error
	MarkEventProcessed(ctx context.Context, eventKey string, ttl time.Duration) (bool, error)
	SeedDefaultTemplates(ctx context.Context, templates []*domain.Template) error
}
