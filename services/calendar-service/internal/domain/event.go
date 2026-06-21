package domain

import (
	"errors"
	"strings"
	"time"
)

var (
	ErrEventNotFound = errors.New("event not found")
	ErrInvalidInput  = errors.New("invalid input")
	ErrForbidden     = errors.New("forbidden")
)

type EventType string

const (
	EventTypeDeadline EventType = "deadline"
	EventTypeTask     EventType = "task"
	EventTypeMeeting  EventType = "meeting"
	EventTypeReminder EventType = "reminder"
)

func (t EventType) Valid() bool {
	switch t {
	case EventTypeDeadline, EventTypeTask, EventTypeMeeting, EventTypeReminder:
		return true
	default:
		return false
	}
}

type Event struct {
	ID          string
	UserID      string
	Title       string
	Description string
	EventType   EventType
	RefID       string
	RefType     string
	StartTime   time.Time
	EndTime     time.Time
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type CreateEventInput struct {
	Title       string
	Description string
	EventType   EventType
	UserID      string
	RefID       string
	RefType     string
	StartTime   time.Time
	EndTime     time.Time
}

func (i *CreateEventInput) Validate() error {
	i.Title = strings.TrimSpace(i.Title)
	i.Description = strings.TrimSpace(i.Description)
	i.RefID = strings.TrimSpace(i.RefID)
	i.RefType = strings.TrimSpace(i.RefType)

	if i.Title == "" || i.UserID == "" {
		return ErrInvalidInput
	}
	if i.EventType == "" {
		i.EventType = EventTypeReminder
	}
	if !i.EventType.Valid() {
		return ErrInvalidInput
	}
	if i.StartTime.IsZero() {
		return ErrInvalidInput
	}
	if i.EndTime.IsZero() {
		i.EndTime = i.StartTime
	}
	if i.EndTime.Before(i.StartTime) {
		return ErrInvalidInput
	}
	return nil
}

type UpdateEventInput struct {
	ID          string
	Title       *string
	Description *string
	StartTime   *time.Time
	EndTime     *time.Time
	UpdatedBy   string
}

func (i *UpdateEventInput) Validate() error {
	if i.ID == "" || i.UpdatedBy == "" {
		return ErrInvalidInput
	}
	if i.Title != nil {
		trimmed := strings.TrimSpace(*i.Title)
		if trimmed == "" {
			return ErrInvalidInput
		}
		i.Title = &trimmed
	}
	if i.StartTime != nil && i.EndTime != nil && i.EndTime.Before(*i.StartTime) {
		return ErrInvalidInput
	}
	return nil
}

type RangeFilter struct {
	UserID string
	From   time.Time
	To     time.Time
}

type ListFilter struct {
	UserID   string
	Page     int
	PageSize int
}

func (f *ListFilter) Normalize() {
	if f.Page <= 0 {
		f.Page = 1
	}
	if f.PageSize <= 0 || f.PageSize > 100 {
		f.PageSize = 20
	}
}

func (f *ListFilter) Offset() int {
	return (f.Page - 1) * f.PageSize
}

type EventFilter struct {
	UserID    string
	EventType *EventType
	RefType   string
	From      *time.Time
	To        *time.Time
	Page      int
	PageSize  int
}

func (f *EventFilter) Normalize() {
	if f.Page <= 0 {
		f.Page = 1
	}
	if f.PageSize <= 0 || f.PageSize > 100 {
		f.PageSize = 20
	}
}

func (f *EventFilter) Offset() int {
	return (f.Page - 1) * f.PageSize
}

type EventStats struct {
	Total     int
	Deadlines int
	Tasks     int
	Meetings  int
	Reminders int
}
