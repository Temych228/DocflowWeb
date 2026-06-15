package domain

import (
	"errors"
	"strings"
	"time"
)

var (
	ErrDocumentNotFound  = errors.New("document not found")
	ErrInvalidInput      = errors.New("invalid input")
	ErrInvalidTransition = errors.New("invalid status transition")
	ErrForbidden         = errors.New("forbidden")
)

type DocumentStatus string

const (
	StatusDraft      DocumentStatus = "draft"
	StatusAssigned   DocumentStatus = "assigned"
	StatusInProgress DocumentStatus = "in_progress"
	StatusCompleted  DocumentStatus = "completed"
	StatusOverdue    DocumentStatus = "overdue"
	StatusArchived   DocumentStatus = "archived"
)

func (s DocumentStatus) Valid() bool {
	switch s {
	case StatusDraft, StatusAssigned, StatusInProgress, StatusCompleted, StatusOverdue, StatusArchived:
		return true
	default:
		return false
	}
}

type DocumentType string

const (
	TypeContract DocumentType = "contract"
	TypeInvoice  DocumentType = "invoice"
	TypeReport   DocumentType = "report"
	TypeMemo     DocumentType = "memo"
	TypeOrder    DocumentType = "order"
	TypeOther    DocumentType = "other"
)

func (t DocumentType) Valid() bool {
	switch t {
	case TypeContract, TypeInvoice, TypeReport, TypeMemo, TypeOrder, TypeOther:
		return true
	default:
		return false
	}
}

var allowedTransitions = map[DocumentStatus][]DocumentStatus{
	StatusDraft:      {StatusAssigned, StatusArchived},
	StatusAssigned:   {StatusInProgress, StatusOverdue, StatusArchived},
	StatusInProgress: {StatusCompleted, StatusOverdue, StatusArchived},
	StatusCompleted:  {StatusArchived},
	StatusOverdue:    {StatusInProgress, StatusArchived},
	StatusArchived:   {},
}

func CanTransition(from, to DocumentStatus) bool {
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

type Document struct {
	ID            string
	Title         string
	Description   string
	Type          DocumentType
	Status        DocumentStatus
	CreatorID     string
	ResponsibleID *string
	Deadline      *time.Time
	FileURL       string
	Tags          []string
	IsOverdue     bool
	CreatedAt     time.Time
	UpdatedAt     time.Time
	ArchivedAt    *time.Time
}

type HistoryEntry struct {
	ID         string
	DocumentID string
	ChangedBy  string
	Field      string
	OldValue   string
	NewValue   string
	ChangedAt  time.Time
}

type CreateDocumentInput struct {
	Title         string
	Description   string
	Type          DocumentType
	CreatorID     string
	ResponsibleID *string
	Deadline      *time.Time
	FileURL       string
	Tags          []string
}

func (i *CreateDocumentInput) Validate() error {
	i.Title = strings.TrimSpace(i.Title)
	i.Description = strings.TrimSpace(i.Description)
	i.FileURL = strings.TrimSpace(i.FileURL)

	if i.Title == "" {
		return ErrInvalidInput
	}
	if i.CreatorID == "" {
		return ErrInvalidInput
	}
	if i.Type == "" {
		i.Type = TypeOther
	}
	if !i.Type.Valid() {
		return ErrInvalidInput
	}
	return nil
}

type UpdateDocumentInput struct {
	ID          string
	Title       *string
	Description *string
	Type        *DocumentType
	Deadline    *time.Time
	FileURL     *string
	Tags        []string
	ActorID     string
}

func (i *UpdateDocumentInput) Validate() error {
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
	if i.Type != nil && !i.Type.Valid() {
		return ErrInvalidInput
	}
	return nil
}

type ListFilter struct {
	Page          int
	PageSize      int
	CreatorID     string
	ResponsibleID string
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
	Status        DocumentStatus
	Type          DocumentType
	ResponsibleID string
	CreatorID     string
	DeadlineFrom  *time.Time
	DeadlineTo    *time.Time
	Page          int
	PageSize      int
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

type DocumentStats struct {
	Total      int
	Draft      int
	Assigned   int
	InProgress int
	Completed  int
	Overdue    int
	Archived   int
}

type HistoryFilter struct {
	DocumentID string
	Page       int
	PageSize   int
}

func (f *HistoryFilter) Normalize() {
	if f.Page <= 0 {
		f.Page = 1
	}
	if f.PageSize <= 0 || f.PageSize > 100 {
		f.PageSize = 20
	}
}

func (f *HistoryFilter) Offset() int {
	return (f.Page - 1) * f.PageSize
}
