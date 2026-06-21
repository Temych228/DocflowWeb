package repository

import (
	"context"

	"github.com/Temych228/DocflowWeb/services/calendar-service/internal/domain"
)

type EventRepository interface {
	Create(ctx context.Context, event *domain.Event) error
	GetByID(ctx context.Context, id string) (*domain.Event, error)
	Update(ctx context.Context, event *domain.Event) error
	Delete(ctx context.Context, id string) error
	GetByRange(ctx context.Context, filter domain.RangeFilter) ([]*domain.Event, int, error)
	GetByUser(ctx context.Context, filter domain.ListFilter) ([]*domain.Event, int, error)
	GetUpcomingDeadlines(ctx context.Context, userID string, days int) ([]*domain.Event, error)
	Filter(ctx context.Context, filter domain.EventFilter) ([]*domain.Event, int, error)
	Stats(ctx context.Context, userID string) (*domain.EventStats, error)
}
