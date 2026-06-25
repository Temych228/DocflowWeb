package service

import (
	"context"

	"github.com/Temych228/DocflowWeb/services/task-service/internal/domain"
)

type TaskService interface {
	CreateTask(ctx context.Context, input domain.CreateTaskInput) (*domain.Task, error)
	GetTask(ctx context.Context, id string) (*domain.Task, error)
	UpdateTask(ctx context.Context, input domain.UpdateTaskInput) (*domain.Task, error)
	DeleteTask(ctx context.Context, id, actorID string) error
	ListTasksByDocument(ctx context.Context, documentID string, filter domain.ListFilter) ([]*domain.Task, int, error)
	ListTasksByAssignee(ctx context.Context, assigneeID string, filter domain.ListFilter) ([]*domain.Task, int, error)
	AssignTask(ctx context.Context, id, assigneeID, actorID string) (*domain.Task, error)
	ChangeTaskStatus(ctx context.Context, id string, newStatus domain.TaskStatus, actorID string) (*domain.Task, error)
	GetTaskHistory(ctx context.Context, taskID string) ([]*domain.HistoryEntry, error)
	FilterTasks(ctx context.Context, filter domain.AdvancedFilter) ([]*domain.Task, int, error)
	MarkTasksOverdue(ctx context.Context) ([]string, error)
	GetTaskStats(ctx context.Context) (*domain.TaskStats, error)
}
