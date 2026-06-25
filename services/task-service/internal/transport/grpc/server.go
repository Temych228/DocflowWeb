package grpcserver

import (
	"context"
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"

	taskv1 "github.com/Temych228/docflow-protos-final/task/v1"

	"github.com/Temych228/DocflowWeb/services/task-service/internal/domain"
	"github.com/Temych228/DocflowWeb/services/task-service/internal/service"
	apperrors "github.com/Temych228/DocflowWeb/services/task-service/pkg/errors"
)

type Server struct {
	taskv1.UnimplementedTaskServiceServer
	svc service.TaskService
}

func New(svc service.TaskService) *Server {
	return &Server{svc: svc}
}

var statusToProto = map[domain.TaskStatus]taskv1.TaskStatus{
	domain.StatusOpen:       taskv1.TaskStatus_TASK_STATUS_OPEN,
	domain.StatusInProgress: taskv1.TaskStatus_TASK_STATUS_IN_PROGRESS,
	domain.StatusDone:       taskv1.TaskStatus_TASK_STATUS_DONE,
	domain.StatusOverdue:    taskv1.TaskStatus_TASK_STATUS_OVERDUE,
}

var statusFromProto = map[taskv1.TaskStatus]domain.TaskStatus{
	taskv1.TaskStatus_TASK_STATUS_OPEN:        domain.StatusOpen,
	taskv1.TaskStatus_TASK_STATUS_IN_PROGRESS: domain.StatusInProgress,
	taskv1.TaskStatus_TASK_STATUS_DONE:        domain.StatusDone,
	taskv1.TaskStatus_TASK_STATUS_OVERDUE:     domain.StatusOverdue,
}

func taskStatusToProto(s domain.TaskStatus) taskv1.TaskStatus {
	if v, ok := statusToProto[s]; ok {
		return v
	}
	return taskv1.TaskStatus_TASK_STATUS_UNSPECIFIED
}

func taskStatusFromProto(s taskv1.TaskStatus) domain.TaskStatus {
	if v, ok := statusFromProto[s]; ok {
		return v
	}
	return ""
}

var priorityToProto = map[domain.TaskPriority]taskv1.TaskPriority{
	domain.PriorityLow:      taskv1.TaskPriority_TASK_PRIORITY_LOW,
	domain.PriorityMedium:   taskv1.TaskPriority_TASK_PRIORITY_MEDIUM,
	domain.PriorityHigh:     taskv1.TaskPriority_TASK_PRIORITY_HIGH,
	domain.PriorityCritical: taskv1.TaskPriority_TASK_PRIORITY_CRITICAL,
}

var priorityFromProto = map[taskv1.TaskPriority]domain.TaskPriority{
	taskv1.TaskPriority_TASK_PRIORITY_LOW:      domain.PriorityLow,
	taskv1.TaskPriority_TASK_PRIORITY_MEDIUM:   domain.PriorityMedium,
	taskv1.TaskPriority_TASK_PRIORITY_HIGH:     domain.PriorityHigh,
	taskv1.TaskPriority_TASK_PRIORITY_CRITICAL: domain.PriorityCritical,
}

func taskPriorityToProto(p domain.TaskPriority) taskv1.TaskPriority {
	if v, ok := priorityToProto[p]; ok {
		return v
	}
	return taskv1.TaskPriority_TASK_PRIORITY_UNSPECIFIED
}

func taskPriorityFromProto(p taskv1.TaskPriority) domain.TaskPriority {
	if v, ok := priorityFromProto[p]; ok {
		return v
	}
	return ""
}

func timeToProto(t *time.Time) *timestamppb.Timestamp {
	if t == nil {
		return nil
	}
	return timestamppb.New(*t)
}

func timeFromProto(ts *timestamppb.Timestamp) *time.Time {
	if ts == nil {
		return nil
	}
	t := ts.AsTime()
	return &t
}

func stringFromPtr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func taskToProto(task *domain.Task) *taskv1.Task {
	if task == nil {
		return nil
	}
	return &taskv1.Task{
		Id:          task.ID,
		Title:       task.Title,
		Description: task.Description,
		DocumentId:  task.DocumentID,
		AssigneeId:  stringFromPtr(task.AssigneeID),
		CreatorId:   task.CreatorID,
		Status:      taskStatusToProto(task.Status),
		Priority:    taskPriorityToProto(task.Priority),
		Deadline:    timeToProto(task.Deadline),
		CreatedAt:   timeToProto(&task.CreatedAt),
		UpdatedAt:   timeToProto(&task.UpdatedAt),
	}
}

func tasksToProto(tasks []*domain.Task) []*taskv1.Task {
	out := make([]*taskv1.Task, len(tasks))
	for i, task := range tasks {
		out[i] = taskToProto(task)
	}
	return out
}

func historyEntryToProto(entry *domain.HistoryEntry) *taskv1.TaskHistoryEntry {
	if entry == nil {
		return nil
	}
	return &taskv1.TaskHistoryEntry{
		Id:        entry.ID,
		TaskId:    entry.TaskID,
		ChangedBy: entry.ChangedBy,
		Field:     entry.Field,
		OldValue:  entry.OldValue,
		NewValue:  entry.NewValue,
		ChangedAt: timeToProto(&entry.ChangedAt),
	}
}

func historyEntriesToProto(entries []*domain.HistoryEntry) []*taskv1.TaskHistoryEntry {
	out := make([]*taskv1.TaskHistoryEntry, len(entries))
	for i, entry := range entries {
		out[i] = historyEntryToProto(entry)
	}
	return out
}

func (s *Server) CreateTask(ctx context.Context, req *taskv1.CreateTaskRequest) (*taskv1.CreateTaskResponse, error) {
	input := domain.CreateTaskInput{
		Title:       req.GetTitle(),
		Description: req.GetDescription(),
		DocumentID:  req.GetDocumentId(),
		CreatorID:   req.GetCreatorId(),
		Priority:    taskPriorityFromProto(req.GetPriority()),
		Deadline:    timeFromProto(req.GetDeadline()),
	}

	task, err := s.svc.CreateTask(ctx, input)
	if err != nil {
		return nil, apperrors.ToGRPCError(err)
	}

	return &taskv1.CreateTaskResponse{Task: taskToProto(task)}, nil
}

func (s *Server) GetTask(ctx context.Context, req *taskv1.GetTaskRequest) (*taskv1.GetTaskResponse, error) {
	task, err := s.svc.GetTask(ctx, req.GetId())
	if err != nil {
		return nil, apperrors.ToGRPCError(err)
	}

	return &taskv1.GetTaskResponse{Task: taskToProto(task)}, nil
}

func (s *Server) UpdateTask(ctx context.Context, req *taskv1.UpdateTaskRequest) (*taskv1.UpdateTaskResponse, error) {
	input := domain.UpdateTaskInput{
		ID:      req.GetId(),
		ActorID: req.GetUpdatedBy(),
	}

	if req.GetTitle() != "" {
		title := req.GetTitle()
		input.Title = &title
	}
	if req.GetDescription() != "" {
		description := req.GetDescription()
		input.Description = &description
	}
	if req.GetPriority() != taskv1.TaskPriority_TASK_PRIORITY_UNSPECIFIED {
		priority := taskPriorityFromProto(req.GetPriority())
		input.Priority = &priority
	}
	if req.GetDeadline() != nil {
		input.Deadline = timeFromProto(req.GetDeadline())
	}

	task, err := s.svc.UpdateTask(ctx, input)
	if err != nil {
		return nil, apperrors.ToGRPCError(err)
	}

	return &taskv1.UpdateTaskResponse{Task: taskToProto(task)}, nil
}

func (s *Server) DeleteTask(ctx context.Context, req *taskv1.DeleteTaskRequest) (*taskv1.DeleteTaskResponse, error) {
	if err := s.svc.DeleteTask(ctx, req.GetId(), req.GetDeletedBy()); err != nil {
		return nil, apperrors.ToGRPCError(err)
	}

	return &taskv1.DeleteTaskResponse{Success: true}, nil
}

func (s *Server) ListTasksByDocument(ctx context.Context, req *taskv1.ListTasksByDocumentRequest) (*taskv1.ListTasksResponse, error) {
	filter := domain.ListFilter{
		Page:     int(req.GetPage()),
		PageSize: int(req.GetPageSize()),
	}

	tasks, total, err := s.svc.ListTasksByDocument(ctx, req.GetDocumentId(), filter)
	if err != nil {
		return nil, apperrors.ToGRPCError(err)
	}

	return &taskv1.ListTasksResponse{
		Tasks: tasksToProto(tasks),
		Total: int32(total),
	}, nil
}

func (s *Server) ListTasksByAssignee(ctx context.Context, req *taskv1.ListTasksByAssigneeRequest) (*taskv1.ListTasksResponse, error) {
	filter := domain.ListFilter{
		Page:     int(req.GetPage()),
		PageSize: int(req.GetPageSize()),
	}

	tasks, total, err := s.svc.ListTasksByAssignee(ctx, req.GetAssigneeId(), filter)
	if err != nil {
		return nil, apperrors.ToGRPCError(err)
	}

	return &taskv1.ListTasksResponse{
		Tasks: tasksToProto(tasks),
		Total: int32(total),
	}, nil
}

func (s *Server) AssignTask(ctx context.Context, req *taskv1.AssignTaskRequest) (*taskv1.AssignTaskResponse, error) {
	task, err := s.svc.AssignTask(ctx, req.GetTaskId(), req.GetAssigneeId(), req.GetAssignedBy())
	if err != nil {
		return nil, apperrors.ToGRPCError(err)
	}

	return &taskv1.AssignTaskResponse{Task: taskToProto(task)}, nil
}

func (s *Server) ChangeTaskStatus(ctx context.Context, req *taskv1.ChangeTaskStatusRequest) (*taskv1.ChangeTaskStatusResponse, error) {
	newStatus := taskStatusFromProto(req.GetNewStatus())

	task, err := s.svc.ChangeTaskStatus(ctx, req.GetTaskId(), newStatus, req.GetChangedBy())
	if err != nil {
		return nil, apperrors.ToGRPCError(err)
	}

	return &taskv1.ChangeTaskStatusResponse{Task: taskToProto(task)}, nil
}

func (s *Server) GetTaskHistory(ctx context.Context, req *taskv1.GetTaskHistoryRequest) (*taskv1.GetTaskHistoryResponse, error) {
	entries, err := s.svc.GetTaskHistory(ctx, req.GetTaskId())
	if err != nil {
		return nil, apperrors.ToGRPCError(err)
	}

	return &taskv1.GetTaskHistoryResponse{
		Entries: historyEntriesToProto(entries),
	}, nil
}

func (s *Server) FilterTasks(ctx context.Context, req *taskv1.FilterTasksRequest) (*taskv1.FilterTasksResponse, error) {
	filter := domain.AdvancedFilter{
		Status:       taskStatusFromProto(req.GetStatus()),
		Priority:     taskPriorityFromProto(req.GetPriority()),
		AssigneeID:   req.GetAssigneeId(),
		DocumentID:   req.GetDocumentId(),
		DeadlineFrom: timeFromProto(req.GetDeadlineFrom()),
		DeadlineTo:   timeFromProto(req.GetDeadlineTo()),
		Page:         int(req.GetPage()),
		PageSize:     int(req.GetPageSize()),
	}

	tasks, total, err := s.svc.FilterTasks(ctx, filter)
	if err != nil {
		return nil, apperrors.ToGRPCError(err)
	}

	return &taskv1.FilterTasksResponse{
		Tasks: tasksToProto(tasks),
		Total: int32(total),
	}, nil
}

func (s *Server) MarkTasksOverdue(ctx context.Context, _ *taskv1.MarkTasksOverdueRequest) (*taskv1.MarkTasksOverdueResponse, error) {
	ids, err := s.svc.MarkTasksOverdue(ctx)
	if err != nil {
		return nil, apperrors.ToGRPCError(err)
	}

	return &taskv1.MarkTasksOverdueResponse{
		MarkedCount: int32(len(ids)),
	}, nil
}

func (s *Server) GetTaskStats(ctx context.Context, _ *taskv1.GetTaskStatsRequest) (*taskv1.GetTaskStatsResponse, error) {
	stats, err := s.svc.GetTaskStats(ctx)
	if err != nil {
		return nil, apperrors.ToGRPCError(err)
	}

	return &taskv1.GetTaskStatsResponse{
		Stats: &taskv1.TaskStats{
			Total:      int32(stats.Total),
			Open:       int32(stats.Open),
			InProgress: int32(stats.InProgress),
			Done:       int32(stats.Done),
			Overdue:    int32(stats.Overdue),
		},
	}, nil
}
