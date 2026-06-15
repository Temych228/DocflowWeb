package repository

import (
	"context"

	"github.com/Temych228/DocflowWeb/services/document-service/internal/domain"
)

type DocumentRepository interface {
	Create(ctx context.Context, doc *domain.Document) error
	GetByID(ctx context.Context, id string) (*domain.Document, error)
	Update(ctx context.Context, doc *domain.Document) error
	Delete(ctx context.Context, id string) error
	List(ctx context.Context, filter domain.ListFilter) ([]*domain.Document, int, error)
	Filter(ctx context.Context, filter domain.AdvancedFilter) ([]*domain.Document, int, error)
	UpdateStatus(ctx context.Context, id string, status domain.DocumentStatus) error
	SetResponsible(ctx context.Context, id string, responsibleID string) error
	Archive(ctx context.Context, id string) error
	FindOverdue(ctx context.Context) ([]*domain.Document, error)
	MarkOverdueByIDs(ctx context.Context, ids []string) error
	Stats(ctx context.Context) (*domain.DocumentStats, error)
}

type HistoryRepository interface {
	Create(ctx context.Context, entry *domain.HistoryEntry) error
	List(ctx context.Context, filter domain.HistoryFilter) ([]*domain.HistoryEntry, int, error)
}
