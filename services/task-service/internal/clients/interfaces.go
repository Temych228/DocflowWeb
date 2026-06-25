package clients

import (
	"context"
	"time"
)

type CreateNotificationInput struct {
	UserID        string
	TaskID        string
	NotifCategory string
	Title         string
	Body          string
	RefID         string
	RefType       string
}

type NotificationClient interface {
	CreateNotification(ctx context.Context, input CreateNotificationInput) error
}

type CreateEventInput struct {
	UserID      string
	Title       string
	Description string
	EventType   string
	StartTime   time.Time
	EndTime     time.Time
	RefID       string
	RefType     string
}

type CalendarClient interface {
	CreateEvent(ctx context.Context, input CreateEventInput) error
}
