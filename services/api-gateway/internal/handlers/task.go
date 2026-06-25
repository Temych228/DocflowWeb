package handlers

import (
	"github.com/Temych228/DocflowWeb/api-gateway/internal/clients"
	"github.com/Temych228/DocflowWeb/api-gateway/internal/middleware"
	"github.com/Temych228/DocflowWeb/api-gateway/pkg/dto"
	taskpb "github.com/Temych228/docflow-protos-final/task/v1"
	"github.com/gin-gonic/gin"
)

type TaskHandler struct {
	taskClient *clients.TaskClient
}

func NewTaskHandler(taskClient *clients.TaskClient) *TaskHandler {
	return &TaskHandler{taskClient: taskClient}
}

func (h *TaskHandler) ListTasksByDocument(c *gin.Context) {
	docID := c.Param("document_id")
	page := parseQueryInt(c, "page", 1)
	pageSize := parseQueryInt(c, "page_size", 10)

	resp, err := h.taskClient.Client.ListTasksByDocument(c.Request.Context(), &taskpb.ListTasksByDocumentRequest{
		DocumentId: docID,
		Page:       int32(page),
		PageSize:   int32(pageSize),
	})
	if err != nil {
		c.JSON(500, dto.NewErrorResponse(dto.ErrorInternal, grpcErrMessage(err), nil))
		return
	}

	c.JSON(200, dto.NewSuccessResponse(gin.H{"tasks": resp.Tasks, "total": resp.Total}))
}

func (h *TaskHandler) ListTasksByAssignee(c *gin.Context) {
	assigneeID := c.Param("assignee_id")
	page := parseQueryInt(c, "page", 1)
	pageSize := parseQueryInt(c, "page_size", 10)

	resp, err := h.taskClient.Client.ListTasksByAssignee(c.Request.Context(), &taskpb.ListTasksByAssigneeRequest{
		AssigneeId: assigneeID,
		Page:       int32(page),
		PageSize:   int32(pageSize),
	})
	if err != nil {
		c.JSON(500, dto.NewErrorResponse(dto.ErrorInternal, grpcErrMessage(err), nil))
		return
	}

	c.JSON(200, dto.NewSuccessResponse(gin.H{"tasks": resp.Tasks, "total": resp.Total}))
}

func (h *TaskHandler) GetTask(c *gin.Context) {
	taskID := c.Param("id")

	resp, err := h.taskClient.Client.GetTask(c.Request.Context(), &taskpb.GetTaskRequest{Id: taskID})
	if err != nil {
		c.JSON(404, dto.NewErrorResponse(dto.ErrorNotFound, "Task not found", nil))
		return
	}

	c.JSON(200, dto.NewSuccessResponse(resp.Task))
}

func (h *TaskHandler) CreateTask(c *gin.Context) {
	creatorID := middleware.ExtractUserID(c)

	var req CreateTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, dto.NewErrorResponse(dto.ErrorValidation, err.Error(), nil))
		return
	}

	resp, err := h.taskClient.Client.CreateTask(c.Request.Context(), &taskpb.CreateTaskRequest{
		DocumentId:  req.DocumentID,
		Title:       req.Title,
		Description: req.Description,
		CreatorId:   creatorID,
	})
	if err != nil {
		c.JSON(500, dto.NewErrorResponse(dto.ErrorInternal, grpcErrMessage(err), nil))
		return
	}

	c.JSON(201, dto.NewSuccessResponse(resp.Task))
}

func (h *TaskHandler) UpdateTask(c *gin.Context) {
	taskID := c.Param("id")
	userID := middleware.ExtractUserID(c)

	var req UpdateTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, dto.NewErrorResponse(dto.ErrorValidation, err.Error(), nil))
		return
	}

	resp, err := h.taskClient.Client.UpdateTask(c.Request.Context(), &taskpb.UpdateTaskRequest{
		Id:          taskID,
		Title:       req.Title,
		Description: req.Description,
		UpdatedBy:   userID,
	})
	if err != nil {
		c.JSON(500, dto.NewErrorResponse(dto.ErrorInternal, grpcErrMessage(err), nil))
		return
	}

	c.JSON(200, dto.NewSuccessResponse(resp.Task))
}

func (h *TaskHandler) DeleteTask(c *gin.Context) {
	taskID := c.Param("id")
	userID := middleware.ExtractUserID(c)

	_, err := h.taskClient.Client.DeleteTask(c.Request.Context(), &taskpb.DeleteTaskRequest{
		Id:        taskID,
		DeletedBy: userID,
	})
	if err != nil {
		c.JSON(500, dto.NewErrorResponse(dto.ErrorInternal, grpcErrMessage(err), nil))
		return
	}

	c.JSON(200, dto.NewSuccessResponse(gin.H{"deleted": true}))
}

func (h *TaskHandler) AssignTask(c *gin.Context) {
	taskID := c.Param("id")
	userID := middleware.ExtractUserID(c)

	var req AssignTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, dto.NewErrorResponse(dto.ErrorValidation, err.Error(), nil))
		return
	}

	resp, err := h.taskClient.Client.AssignTask(c.Request.Context(), &taskpb.AssignTaskRequest{
		TaskId:     taskID,
		AssigneeId: req.AssigneeID,
		AssignedBy: userID,
	})
	if err != nil {
		c.JSON(500, dto.NewErrorResponse(dto.ErrorInternal, grpcErrMessage(err), nil))
		return
	}

	c.JSON(200, dto.NewSuccessResponse(resp.Task))
}

func (h *TaskHandler) ChangeTaskStatus(c *gin.Context) {
	taskID := c.Param("id")
	userID := middleware.ExtractUserID(c)

	var req ChangeTaskStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, dto.NewErrorResponse(dto.ErrorValidation, err.Error(), nil))
		return
	}

	statusMap := map[string]taskpb.TaskStatus{
		"open":        taskpb.TaskStatus_TASK_STATUS_OPEN,
		"in_progress": taskpb.TaskStatus_TASK_STATUS_IN_PROGRESS,
		"done":        taskpb.TaskStatus_TASK_STATUS_DONE,
		"overdue":     taskpb.TaskStatus_TASK_STATUS_OVERDUE,
	}
	status, ok := statusMap[req.Status]
	if !ok {
		c.JSON(400, dto.NewErrorResponse(dto.ErrorValidation, "Invalid status value", nil))
		return
	}

	resp, err := h.taskClient.Client.ChangeTaskStatus(c.Request.Context(), &taskpb.ChangeTaskStatusRequest{
		TaskId:    taskID,
		NewStatus: status,
		ChangedBy: userID,
	})
	if err != nil {
		c.JSON(500, dto.NewErrorResponse(dto.ErrorInternal, grpcErrMessage(err), nil))
		return
	}

	c.JSON(200, dto.NewSuccessResponse(resp.Task))
}

func (h *TaskHandler) GetTaskHistory(c *gin.Context) {
	taskID := c.Param("id")

	resp, err := h.taskClient.Client.GetTaskHistory(c.Request.Context(), &taskpb.GetTaskHistoryRequest{
		TaskId: taskID,
	})
	if err != nil {
		c.JSON(500, dto.NewErrorResponse(dto.ErrorInternal, grpcErrMessage(err), nil))
		return
	}

	c.JSON(200, dto.NewSuccessResponse(gin.H{"entries": resp.Entries}))
}

func (h *TaskHandler) GetTaskStats(c *gin.Context) {
	resp, err := h.taskClient.Client.GetTaskStats(c.Request.Context(), &taskpb.GetTaskStatsRequest{})
	if err != nil {
		c.JSON(500, dto.NewErrorResponse(dto.ErrorInternal, grpcErrMessage(err), nil))
		return
	}

	c.JSON(200, dto.NewSuccessResponse(gin.H{"stats": resp.Stats}))
}

func (h *TaskHandler) FilterTasks(c *gin.Context) {
	var req FilterTasksRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, dto.NewErrorResponse(dto.ErrorValidation, err.Error(), nil))
		return
	}

	statusMap := map[string]taskpb.TaskStatus{
		"open":        taskpb.TaskStatus_TASK_STATUS_OPEN,
		"in_progress": taskpb.TaskStatus_TASK_STATUS_IN_PROGRESS,
		"done":        taskpb.TaskStatus_TASK_STATUS_DONE,
		"overdue":     taskpb.TaskStatus_TASK_STATUS_OVERDUE,
	}
	priorityMap := map[string]taskpb.TaskPriority{
		"low":      taskpb.TaskPriority_TASK_PRIORITY_LOW,
		"medium":   taskpb.TaskPriority_TASK_PRIORITY_MEDIUM,
		"high":     taskpb.TaskPriority_TASK_PRIORITY_HIGH,
		"critical": taskpb.TaskPriority_TASK_PRIORITY_CRITICAL,
	}

	pbStatus := taskpb.TaskStatus_TASK_STATUS_UNSPECIFIED
	if req.Status != "" {
		if s, ok := statusMap[req.Status]; ok {
			pbStatus = s
		}
	}

	pbPriority := taskpb.TaskPriority_TASK_PRIORITY_UNSPECIFIED
	if req.Priority != "" {
		if p, ok := priorityMap[req.Priority]; ok {
			pbPriority = p
		}
	}

	resp, err := h.taskClient.Client.FilterTasks(c.Request.Context(), &taskpb.FilterTasksRequest{
		Status:     pbStatus,
		Priority:   pbPriority,
		AssigneeId: req.AssigneeID,
		DocumentId: req.DocumentID,
		Page:       req.Page,
		PageSize:   req.PageSize,
	})
	if err != nil {
		c.JSON(500, dto.NewErrorResponse(dto.ErrorInternal, grpcErrMessage(err), nil))
		return
	}

	c.JSON(200, dto.NewSuccessResponse(gin.H{"tasks": resp.Tasks, "total": resp.Total}))
}

func (h *TaskHandler) MarkTasksOverdue(c *gin.Context) {
	resp, err := h.taskClient.Client.MarkTasksOverdue(c.Request.Context(), &taskpb.MarkTasksOverdueRequest{})
	if err != nil {
		c.JSON(500, dto.NewErrorResponse(dto.ErrorInternal, grpcErrMessage(err), nil))
		return
	}

	c.JSON(200, dto.NewSuccessResponse(gin.H{"marked_count": resp.MarkedCount}))
}

type CreateTaskRequest struct {
	DocumentID  string `json:"document_id" binding:"required"`
	Title       string `json:"title" binding:"required"`
	Description string `json:"description"`
}

type UpdateTaskRequest struct {
	Title       string `json:"title"`
	Description string `json:"description"`
}

type AssignTaskRequest struct {
	AssigneeID string `json:"assignee_id" binding:"required"`
}

type ChangeTaskStatusRequest struct {
	Status string `json:"status" binding:"required"`
}

type FilterTasksRequest struct {
	DocumentID string `json:"document_id,omitempty"`
	AssigneeID string `json:"assignee_id,omitempty"`
	Status     string `json:"status,omitempty"`
	Priority   string `json:"priority,omitempty"`
	Page       int32  `json:"page,omitempty"`
	PageSize   int32  `json:"page_size,omitempty"`
}
