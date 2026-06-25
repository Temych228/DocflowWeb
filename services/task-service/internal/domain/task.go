package domain

import (
	"errors"
	"strings"
	"time"
)

var (
	ErrTaskNotFound      = errors.New("task not found")
	ErrInvalidInput      = errors.New("invalid input")
	ErrInvalidTransition = errors.New("invalid status transition")
	ErrForbidden         = errors.New("forbidden")
)

type TaskStatus string

const (
	StatusOpen       TaskStatus = "open"
	StatusInProgress TaskStatus = "in_progress"
	StatusDone       TaskStatus = "done"
	StatusOverdue    TaskStatus = "overdue"
)

func (s TaskStatus) Valid() bool {
	switch s {
	case StatusOpen, StatusInProgress, StatusDone, StatusOverdue:
		return true
	default:
		return false
	}
}

type TaskPriority string

const (
	PriorityLow      TaskPriority = "low"
	PriorityMedium   TaskPriority = "medium"
	PriorityHigh     TaskPriority = "high"
	PriorityCritical TaskPriority = "critical"
)

func (p TaskPriority) Valid() bool {
	switch p {
	case PriorityLow, PriorityMedium, PriorityHigh, PriorityCritical:
		return true
	default:
		return false
	}
}

var allowedTransitions = map[TaskStatus][]TaskStatus{
	StatusOpen:       {StatusInProgress, StatusOverdue},
	StatusInProgress: {StatusDone, StatusOverdue},
	StatusDone:       {},
	StatusOverdue:    {StatusInProgress, StatusDone},
}

func CanTransition(from, to TaskStatus) bool {
	if from == to {
		return false
	}
	next, ok := allowedTransitions[from]
	if !ok {
		return false
	}
	for _, s := range next {
		if s == to {
			return true
		}
	}
	return false
}

type Task struct {
	ID          string
	Title       string
	Description string
	DocumentID  string
	AssigneeID  *string
	CreatorID   string
	Status      TaskStatus
	Priority    TaskPriority
	Deadline    *time.Time
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type HistoryEntry struct {
	ID        string
	TaskID    string
	ChangedBy string
	Field     string
	OldValue  string
	NewValue  string
	ChangedAt time.Time
}

type CreateTaskInput struct {
	Title       string
	Description string
	DocumentID  string
	CreatorID   string
	Priority    TaskPriority
	Deadline    *time.Time
}

func (i *CreateTaskInput) Validate() error {
	i.Title = strings.TrimSpace(i.Title)
	i.Description = strings.TrimSpace(i.Description)

	if i.Title == "" || i.DocumentID == "" || i.CreatorID == "" {
		return ErrInvalidInput
	}
	if i.Priority == "" {
		i.Priority = PriorityMedium
	}
	if !i.Priority.Valid() {
		return ErrInvalidInput
	}
	return nil
}

type UpdateTaskInput struct {
	ID          string
	Title       *string
	Description *string
	Priority    *TaskPriority
	Deadline    *time.Time
	ActorID     string
}

func (i *UpdateTaskInput) Validate() error {
	if i.ID == "" || i.ActorID == "" {
		return ErrInvalidInput
	}
	if i.Title != nil {
		trimmed := strings.TrimSpace(*i.Title)
		if trimmed == "" {
			return ErrInvalidInput
		}
		i.Title = &trimmed
	}
	if i.Priority != nil && !i.Priority.Valid() {
		return ErrInvalidInput
	}
	return nil
}

type ListFilter struct {
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

type AdvancedFilter struct {
	Status       TaskStatus
	Priority     TaskPriority
	AssigneeID   string
	DocumentID   string
	DeadlineFrom *time.Time
	DeadlineTo   *time.Time
	Page         int
	PageSize     int
}

func (f *AdvancedFilter) Normalize() {
	if f.Page <= 0 {
		f.Page = 1
	}
	if f.PageSize <= 0 || f.PageSize > 100 {
		f.PageSize = 20
	}
}

func (f *AdvancedFilter) Offset() int {
	return (f.Page - 1) * f.PageSize
}

type TaskStats struct {
	Total      int
	Open       int
	InProgress int
	Done       int
	Overdue    int
}
