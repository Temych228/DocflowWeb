package service

import (
	"context"

	"github.com/Temych228/DocflowWeb/services/calendar-service/internal/domain"
)

type CalendarService interface {
	CreateEvent(ctx context.Context, input domain.CreateEventInput) (*domain.Event, error)
	GetEventsByDay(ctx context.Context, userID string, year, month, day int) ([]*domain.Event, int, error)
	GetEventsByWeek(ctx context.Context, userID string, year, week int) ([]*domain.Event, int, error)
	GetEventsByMonth(ctx context.Context, userID string, year, month int) ([]*domain.Event, int, error)
	GetEventsByUser(ctx context.Context, filter domain.ListFilter) ([]*domain.Event, int, error)
	GetUpcomingDeadlines(ctx context.Context, userID string, days int) ([]*domain.Event, error)
	DeleteEvent(ctx context.Context, id, deletedBy string) error
	UpdateEvent(ctx context.Context, input domain.UpdateEventInput) (*domain.Event, error)
	FilterEvents(ctx context.Context, filter domain.EventFilter) ([]*domain.Event, int, error)
	GetEventStats(ctx context.Context, userID string) (*domain.EventStats, error)
}
