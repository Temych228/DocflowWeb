package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/Temych228/DocflowWeb/services/calendar-service/internal/cache"
	"github.com/Temych228/DocflowWeb/services/calendar-service/internal/domain"
	"github.com/Temych228/DocflowWeb/services/calendar-service/internal/repository"
)

const monthCacheTTL = 5 * time.Minute

type calendarService struct {
	events repository.EventRepository
	cache  cache.EventCache
}

func New(events repository.EventRepository) CalendarService {
	return &calendarService{events: events}
}

func NewWithCache(events repository.EventRepository, eventCache cache.EventCache) CalendarService {
	return &calendarService{events: events, cache: eventCache}
}

func (s *calendarService) CreateEvent(ctx context.Context, input domain.CreateEventInput) (*domain.Event, error) {
	if err := input.Validate(); err != nil {
		return nil, err
	}

	event := &domain.Event{
		UserID:      input.UserID,
		Title:       input.Title,
		Description: input.Description,
		EventType:   input.EventType,
		RefID:       input.RefID,
		RefType:     input.RefType,
		StartTime:   input.StartTime,
		EndTime:     input.EndTime,
	}

	if err := s.events.Create(ctx, event); err != nil {
		return nil, fmt.Errorf("create event: %w", err)
	}

	s.invalidateMonth(ctx, event.UserID, event.StartTime)

	return event, nil
}

func (s *calendarService) GetEventsByDay(ctx context.Context, userID string, year, month, day int) ([]*domain.Event, int, error) {
	if userID == "" {
		return nil, 0, domain.ErrInvalidInput
	}

	from := time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.UTC)
	to := from.AddDate(0, 0, 1)

	return s.events.GetByRange(ctx, domain.RangeFilter{
		UserID: userID,
		From:   from,
		To:     to,
	})
}

func (s *calendarService) GetEventsByWeek(ctx context.Context, userID string, year, week int) ([]*domain.Event, int, error) {
	if userID == "" {
		return nil, 0, domain.ErrInvalidInput
	}
	if week < 1 || week > 53 {
		return nil, 0, domain.ErrInvalidInput
	}

	from := weekStart(year, week)
	to := from.AddDate(0, 0, 7)

	return s.events.GetByRange(ctx, domain.RangeFilter{
		UserID: userID,
		From:   from,
		To:     to,
	})
}

func (s *calendarService) GetEventsByMonth(ctx context.Context, userID string, year, month int) ([]*domain.Event, int, error) {
	if userID == "" {
		return nil, 0, domain.ErrInvalidInput
	}
	if month < 1 || month > 12 {
		return nil, 0, domain.ErrInvalidInput
	}

	if s.cache != nil {
		if raw, ok := s.cache.GetMonth(ctx, userID, year, month); ok {
			var cached cachedEvents
			if err := json.Unmarshal(raw, &cached); err == nil {
				return cached.Events, cached.Total, nil
			}
		}
	}

	from := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.UTC)
	to := from.AddDate(0, 1, 0)

	events, total, err := s.events.GetByRange(ctx, domain.RangeFilter{
		UserID: userID,
		From:   from,
		To:     to,
	})
	if err != nil {
		return nil, 0, err
	}

	if s.cache != nil {
		if raw, err := json.Marshal(cachedEvents{Events: events, Total: total}); err == nil {
			s.cache.SetMonth(ctx, userID, year, month, raw, monthCacheTTL)
		}
	}

	return events, total, nil
}

type cachedEvents struct {
	Events []*domain.Event `json:"events"`
	Total  int             `json:"total"`
}

func (s *calendarService) GetEventsByUser(ctx context.Context, filter domain.ListFilter) ([]*domain.Event, int, error) {
	if filter.UserID == "" {
		return nil, 0, domain.ErrInvalidInput
	}
	filter.Normalize()
	return s.events.GetByUser(ctx, filter)
}

func (s *calendarService) GetUpcomingDeadlines(ctx context.Context, userID string, days int) ([]*domain.Event, error) {
	if userID == "" {
		return nil, domain.ErrInvalidInput
	}
	if days <= 0 {
		days = 7
	}
	return s.events.GetUpcomingDeadlines(ctx, userID, days)
}

func (s *calendarService) DeleteEvent(ctx context.Context, id, deletedBy string) error {
	if id == "" || deletedBy == "" {
		return domain.ErrInvalidInput
	}

	event, err := s.events.GetByID(ctx, id)
	if err != nil {
		return err
	}

	if event.UserID != deletedBy {
		return domain.ErrForbidden
	}

	if err := s.events.Delete(ctx, id); err != nil {
		return err
	}

	s.invalidateMonth(ctx, event.UserID, event.StartTime)

	return nil
}

func (s *calendarService) UpdateEvent(ctx context.Context, input domain.UpdateEventInput) (*domain.Event, error) {
	if err := input.Validate(); err != nil {
		return nil, err
	}

	event, err := s.events.GetByID(ctx, input.ID)
	if err != nil {
		return nil, err
	}

	if event.UserID != input.UpdatedBy {
		return nil, domain.ErrForbidden
	}

	oldStartTime := event.StartTime

	if input.Title != nil {
		event.Title = *input.Title
	}
	if input.Description != nil {
		event.Description = *input.Description
	}
	if input.StartTime != nil {
		event.StartTime = *input.StartTime
	}
	if input.EndTime != nil {
		event.EndTime = *input.EndTime
	}

	if err := s.events.Update(ctx, event); err != nil {
		return nil, fmt.Errorf("update event: %w", err)
	}

	s.invalidateMonth(ctx, event.UserID, oldStartTime)
	s.invalidateMonth(ctx, event.UserID, event.StartTime)

	return event, nil
}

func (s *calendarService) invalidateMonth(ctx context.Context, userID string, at time.Time) {
	if s.cache == nil {
		return
	}
	s.cache.InvalidateMonth(ctx, userID, at.Year(), int(at.Month()))
}

func (s *calendarService) FilterEvents(ctx context.Context, filter domain.EventFilter) ([]*domain.Event, int, error) {
	filter.Normalize()
	return s.events.Filter(ctx, filter)
}

func (s *calendarService) GetEventStats(ctx context.Context, userID string) (*domain.EventStats, error) {
	return s.events.Stats(ctx, userID)
}

func weekStart(year, week int) time.Time {
	jan4 := time.Date(year, time.January, 4, 0, 0, 0, 0, time.UTC)
	offset := int(jan4.Weekday() - time.Monday)
	if offset < 0 {
		offset += 7
	}
	week1Monday := jan4.AddDate(0, 0, -offset)
	return week1Monday.AddDate(0, 0, (week-1)*7)
}
