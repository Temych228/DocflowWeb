package service

import (
	"context"
	"fmt"

	"github.com/Temych228/DocflowWeb/services/task-service/internal/clients"
	"github.com/Temych228/DocflowWeb/services/task-service/internal/domain"
	"github.com/Temych228/DocflowWeb/services/task-service/internal/repository"
)

type taskService struct {
	tasks        repository.TaskRepository
	history      repository.HistoryRepository
	notification clients.NotificationClient
	calendar     clients.CalendarClient
}

func New(
	tasks repository.TaskRepository,
	history repository.HistoryRepository,
	notification clients.NotificationClient,
	calendar clients.CalendarClient,
) TaskService {
	return &taskService{
		tasks:        tasks,
		history:      history,
		notification: notification,
		calendar:     calendar,
	}
}

func (s *taskService) CreateTask(ctx context.Context, input domain.CreateTaskInput) (*domain.Task, error) {
	if err := input.Validate(); err != nil {
		return nil, err
	}

	task := &domain.Task{
		Title:       input.Title,
		Description: input.Description,
		DocumentID:  input.DocumentID,
		CreatorID:   input.CreatorID,
		Status:      domain.StatusOpen,
		Priority:    input.Priority,
		Deadline:    input.Deadline,
	}

	if err := s.tasks.Create(ctx, task); err != nil {
		return nil, fmt.Errorf("create task: %w", err)
	}

	entry := &domain.HistoryEntry{
		TaskID:    task.ID,
		ChangedBy: input.CreatorID,
		Field:     "status",
		OldValue:  "",
		NewValue:  string(task.Status),
	}
	if err := s.history.Create(ctx, entry); err != nil {
		return nil, fmt.Errorf("record history: %w", err)
	}

	if task.Deadline != nil {
		s.createCalendarEvent(ctx, task, "task")
	}

	return task, nil
}

func (s *taskService) GetTask(ctx context.Context, id string) (*domain.Task, error) {
	if id == "" {
		return nil, domain.ErrInvalidInput
	}
	return s.tasks.GetByID(ctx, id)
}

func (s *taskService) UpdateTask(ctx context.Context, input domain.UpdateTaskInput) (*domain.Task, error) {
	if err := input.Validate(); err != nil {
		return nil, err
	}

	task, err := s.tasks.GetByID(ctx, input.ID)
	if err != nil {
		return nil, err
	}

	if task.Status == domain.StatusDone {
		return nil, domain.ErrForbidden
	}

	if input.Title != nil {
		task.Title = *input.Title
	}
	if input.Description != nil {
		task.Description = *input.Description
	}
	if input.Priority != nil {
		task.Priority = *input.Priority
	}
	if input.Deadline != nil {
		task.Deadline = input.Deadline
	}

	if err := s.tasks.Update(ctx, task); err != nil {
		return nil, fmt.Errorf("update task: %w", err)
	}

	entry := &domain.HistoryEntry{
		TaskID:    task.ID,
		ChangedBy: input.ActorID,
		Field:     "task",
		OldValue:  "",
		NewValue:  "fields updated",
	}
	if err := s.history.Create(ctx, entry); err != nil {
		return nil, fmt.Errorf("record history: %w", err)
	}

	if input.Deadline != nil {
		s.createCalendarEvent(ctx, task, "task")
	}

	return task, nil
}

func (s *taskService) DeleteTask(ctx context.Context, id, actorID string) error {
	if id == "" || actorID == "" {
		return domain.ErrInvalidInput
	}

	task, err := s.tasks.GetByID(ctx, id)
	if err != nil {
		return err
	}

	if task.Status != domain.StatusOpen {
		return domain.ErrForbidden
	}

	return s.tasks.Delete(ctx, id)
}

func (s *taskService) ListTasksByDocument(ctx context.Context, documentID string, filter domain.ListFilter) ([]*domain.Task, int, error) {
	if documentID == "" {
		return nil, 0, domain.ErrInvalidInput
	}
	filter.Normalize()
	return s.tasks.ListByDocument(ctx, documentID, filter)
}

func (s *taskService) ListTasksByAssignee(ctx context.Context, assigneeID string, filter domain.ListFilter) ([]*domain.Task, int, error) {
	if assigneeID == "" {
		return nil, 0, domain.ErrInvalidInput
	}
	filter.Normalize()
	return s.tasks.ListByAssignee(ctx, assigneeID, filter)
}

func (s *taskService) AssignTask(ctx context.Context, id, assigneeID, actorID string) (*domain.Task, error) {
	if id == "" || assigneeID == "" || actorID == "" {
		return nil, domain.ErrInvalidInput
	}

	task, err := s.tasks.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if task.Status == domain.StatusDone {
		return nil, domain.ErrForbidden
	}

	oldAssignee := ""
	if task.AssigneeID != nil {
		oldAssignee = *task.AssigneeID
	}

	if err := s.tasks.SetAssignee(ctx, id, assigneeID); err != nil {
		return nil, fmt.Errorf("assign task: %w", err)
	}
	task.AssigneeID = &assigneeID

	entry := &domain.HistoryEntry{
		TaskID:    id,
		ChangedBy: actorID,
		Field:     "assignee_id",
		OldValue:  oldAssignee,
		NewValue:  assigneeID,
	}
	if err := s.history.Create(ctx, entry); err != nil {
		return nil, fmt.Errorf("record history: %w", err)
	}

	if s.notification != nil {
		_ = s.notification.CreateNotification(ctx, clients.CreateNotificationInput{
			UserID:        assigneeID,
			TaskID:        task.ID,
			NotifCategory: "task_assigned",
			Title:         "Task assigned to you",
			Body:          fmt.Sprintf("You were assigned to task %q", task.Title),
			RefID:         task.ID,
			RefType:       "task",
		})
	}

	return task, nil
}

func (s *taskService) ChangeTaskStatus(ctx context.Context, id string, newStatus domain.TaskStatus, actorID string) (*domain.Task, error) {
	if id == "" || actorID == "" {
		return nil, domain.ErrInvalidInput
	}
	if !newStatus.Valid() {
		return nil, domain.ErrInvalidInput
	}

	task, err := s.tasks.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if !domain.CanTransition(task.Status, newStatus) {
		return nil, domain.ErrInvalidTransition
	}

	oldStatus := task.Status

	if err := s.tasks.UpdateStatus(ctx, id, newStatus); err != nil {
		return nil, fmt.Errorf("change task status: %w", err)
	}
	task.Status = newStatus

	entry := &domain.HistoryEntry{
		TaskID:    id,
		ChangedBy: actorID,
		Field:     "status",
		OldValue:  string(oldStatus),
		NewValue:  string(newStatus),
	}
	if err := s.history.Create(ctx, entry); err != nil {
		return nil, fmt.Errorf("record history: %w", err)
	}

	if s.notification != nil && task.AssigneeID != nil {
		_ = s.notification.CreateNotification(ctx, clients.CreateNotificationInput{
			UserID:        *task.AssigneeID,
			TaskID:        task.ID,
			NotifCategory: "task_status_changed",
			Title:         "Task status changed",
			Body:          fmt.Sprintf("Task %q status changed from %s to %s", task.Title, oldStatus, newStatus),
			RefID:         task.ID,
			RefType:       "task",
		})
	}

	return task, nil
}

func (s *taskService) GetTaskHistory(ctx context.Context, taskID string) ([]*domain.HistoryEntry, error) {
	if taskID == "" {
		return nil, domain.ErrInvalidInput
	}
	return s.history.List(ctx, taskID)
}

func (s *taskService) FilterTasks(ctx context.Context, filter domain.AdvancedFilter) ([]*domain.Task, int, error) {
	filter.Normalize()
	return s.tasks.Filter(ctx, filter)
}

func (s *taskService) MarkTasksOverdue(ctx context.Context) ([]string, error) {
	tasks, err := s.tasks.FindOverdue(ctx)
	if err != nil {
		return nil, fmt.Errorf("find overdue tasks: %w", err)
	}
	if len(tasks) == 0 {
		return nil, nil
	}

	ids := make([]string, len(tasks))
	for i, task := range tasks {
		ids[i] = task.ID
	}

	if err := s.tasks.MarkOverdueByIDs(ctx, ids); err != nil {
		return nil, fmt.Errorf("mark tasks overdue: %w", err)
	}

	for _, task := range tasks {
		entry := &domain.HistoryEntry{
			TaskID:    task.ID,
			ChangedBy: "00000000-0000-0000-0000-000000000000",
			Field:     "status",
			OldValue:  string(task.Status),
			NewValue:  string(domain.StatusOverdue),
		}
		_ = s.history.Create(ctx, entry)

		if s.notification != nil && task.AssigneeID != nil {
			_ = s.notification.CreateNotification(ctx, clients.CreateNotificationInput{
				UserID:        *task.AssigneeID,
				TaskID:        task.ID,
				NotifCategory: "task_overdue",
				Title:         "Task is overdue",
				Body:          fmt.Sprintf("Task %q has passed its deadline", task.Title),
				RefID:         task.ID,
				RefType:       "task",
			})
		}
	}

	return ids, nil
}

func (s *taskService) GetTaskStats(ctx context.Context) (*domain.TaskStats, error) {
	return s.tasks.Stats(ctx)
}

func (s *taskService) createCalendarEvent(ctx context.Context, task *domain.Task, eventType string) {
	if s.calendar == nil || task.Deadline == nil {
		return
	}

	recipients := map[string]struct{}{
		task.CreatorID: {},
	}
	if task.AssigneeID != nil && *task.AssigneeID != "" {
		recipients[*task.AssigneeID] = struct{}{}
	}

	for userID := range recipients {
		_ = s.calendar.CreateEvent(ctx, clients.CreateEventInput{
			UserID:      userID,
			Title:       fmt.Sprintf("Deadline: %s", task.Title),
			Description: task.Description,
			EventType:   eventType,
			StartTime:   *task.Deadline,
			EndTime:     *task.Deadline,
			RefID:       task.ID,
			RefType:     "task",
		})
	}
}
