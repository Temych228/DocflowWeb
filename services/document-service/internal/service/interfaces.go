package service

import (
	"context"

	"github.com/Temych228/DocflowWeb/services/document-service/internal/domain"
)

type DocumentService interface {
	CreateDocument(ctx context.Context, input domain.CreateDocumentInput) (*domain.Document, error)
	GetDocument(ctx context.Context, id string) (*domain.Document, error)
	UpdateDocument(ctx context.Context, input domain.UpdateDocumentInput) (*domain.Document, error)
	DeleteDocument(ctx context.Context, id, actorID string) error
	ListDocuments(ctx context.Context, filter domain.ListFilter) ([]*domain.Document, int, error)
	AssignResponsible(ctx context.Context, id, responsibleID, actorID string) (*domain.Document, error)
	ChangeStatus(ctx context.Context, id string, newStatus domain.DocumentStatus, actorID, comment string) (*domain.Document, error)
	ArchiveDocument(ctx context.Context, id, actorID string) (*domain.Document, error)
	GetDocumentHistory(ctx context.Context, filter domain.HistoryFilter) ([]*domain.HistoryEntry, int, error)
	FilterDocuments(ctx context.Context, filter domain.AdvancedFilter) ([]*domain.Document, int, error)
	MarkOverdue(ctx context.Context) ([]string, error)
	ExportDocumentsCSV(ctx context.Context, filter domain.AdvancedFilter) ([]byte, int, error)
	GetDocumentStats(ctx context.Context) (*domain.DocumentStats, error)
}
