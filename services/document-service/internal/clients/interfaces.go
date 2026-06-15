package clients

import (
	"context"
	"time"
)

type NotificationClient interface {
	CreateNotification(ctx context.Context, input CreateNotificationInput) error
}

type CreateNotificationInput struct {
	UserID        string
	DocumentID    string
	NotifCategory string
	Title         string
	Body          string
	RefID         string
	RefType       string
}

type CalendarClient interface {
	CreateEvent(ctx context.Context, input CreateEventInput) error
}

type CreateEventInput struct {
	UserID      string
	Title       string
	Description string
	EventType   string
	StartAt     time.Time
	EndAt       time.Time
	AllDay      bool
	RefID       string
	RefType     string
}
