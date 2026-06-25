package repository

import (
	"context"

	"github.com/Temych228/DocflowWeb/services/task-service/internal/domain"
)

type TaskRepository interface {
	Create(ctx context.Context, task *domain.Task) error
	GetByID(ctx context.Context, id string) (*domain.Task, error)
	Update(ctx context.Context, task *domain.Task) error
	Delete(ctx context.Context, id string) error
	ListByDocument(ctx context.Context, documentID string, filter domain.ListFilter) ([]*domain.Task, int, error)
	ListByAssignee(ctx context.Context, assigneeID string, filter domain.ListFilter) ([]*domain.Task, int, error)
	Filter(ctx context.Context, filter domain.AdvancedFilter) ([]*domain.Task, int, error)
	SetAssignee(ctx context.Context, id, assigneeID string) error
	UpdateStatus(ctx context.Context, id string, status domain.TaskStatus) error
	FindOverdue(ctx context.Context) ([]*domain.Task, error)
	MarkOverdueByIDs(ctx context.Context, ids []string) error
	Stats(ctx context.Context) (*domain.TaskStats, error)
}

type HistoryRepository interface {
	Create(ctx context.Context, entry *domain.HistoryEntry) error
	List(ctx context.Context, taskID string) ([]*domain.HistoryEntry, error)
}
