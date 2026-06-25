package http

import (
	"net/http"
	"strconv"
	"time"

	"github.com/Temych228/DocflowWeb/services/task-service/internal/domain"
	"github.com/Temych228/DocflowWeb/services/task-service/internal/service"
	"github.com/gin-gonic/gin"
)

type Handler struct {
	taskService service.TaskService
}

func New(taskService service.TaskService) *Handler {
	return &Handler{taskService: taskService}
}

func (h *Handler) Register(r *gin.Engine) {
	tasks := r.Group("/tasks")
	{
		tasks.POST("", h.createTask)
		tasks.GET("/:id", h.getTask)
		tasks.PUT("/:id", h.updateTask)
		tasks.DELETE("/:id", h.deleteTask)
		tasks.GET("/document/:documentID", h.listByDocument)
		tasks.GET("/assignee/:assigneeID", h.listByAssignee)
		tasks.POST("/:id/assign", h.assignTask)
		tasks.PATCH("/:id/status", h.changeStatus)
		tasks.GET("/:id/history", h.getHistory)
		tasks.POST("/filter", h.filterTasks)
		tasks.POST("/mark-overdue", h.markOverdue)
		tasks.GET("/stats", h.getStats)
	}
}

func (h *Handler) createTask(c *gin.Context) {
	var req struct {
		Title       string     `json:"title" binding:"required"`
		Description string     `json:"description"`
		DocumentID  string     `json:"document_id" binding:"required"`
		CreatorID   string     `json:"creator_id" binding:"required"`
		Priority    string     `json:"priority"`
		Deadline    *time.Time `json:"deadline"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	task, err := h.taskService.CreateTask(c.Request.Context(), domain.CreateTaskInput{
		Title:       req.Title,
		Description: req.Description,
		DocumentID:  req.DocumentID,
		CreatorID:   req.CreatorID,
		Priority:    domain.TaskPriority(req.Priority),
		Deadline:    req.Deadline,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, task)
}

func (h *Handler) getTask(c *gin.Context) {
	taskID := c.Param("id")

	task, err := h.taskService.GetTask(c.Request.Context(), taskID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Task not found"})
		return
	}

	c.JSON(http.StatusOK, task)
}

func (h *Handler) updateTask(c *gin.Context) {
	taskID := c.Param("id")

	var req struct {
		Title       *string    `json:"title"`
		Description *string    `json:"description"`
		Priority    *string    `json:"priority"`
		Deadline    *time.Time `json:"deadline"`
		ActorID     string     `json:"actor_id" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var priority *domain.TaskPriority
	if req.Priority != nil {
		p := domain.TaskPriority(*req.Priority)
		priority = &p
	}

	task, err := h.taskService.UpdateTask(c.Request.Context(), domain.UpdateTaskInput{
		ID:          taskID,
		Title:       req.Title,
		Description: req.Description,
		Priority:    priority,
		Deadline:    req.Deadline,
		ActorID:     req.ActorID,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, task)
}

func (h *Handler) deleteTask(c *gin.Context) {
	taskID := c.Param("id")

	var req struct {
		ActorID string `json:"actor_id" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.taskService.DeleteTask(c.Request.Context(), taskID, req.ActorID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Task deleted"})
}

func (h *Handler) listByDocument(c *gin.Context) {
	documentID := c.Param("documentID")
	page := parseQuery(c, "page", 1)
	pageSize := parseQuery(c, "page_size", 10)

	tasks, total, err := h.taskService.ListTasksByDocument(c.Request.Context(), documentID, domain.ListFilter{
		Page:     page,
		PageSize: pageSize,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"tasks": tasks, "total": total})
}

func (h *Handler) listByAssignee(c *gin.Context) {
	assigneeID := c.Param("assigneeID")
	page := parseQuery(c, "page", 1)
	pageSize := parseQuery(c, "page_size", 10)

	tasks, total, err := h.taskService.ListTasksByAssignee(c.Request.Context(), assigneeID, domain.ListFilter{
		Page:     page,
		PageSize: pageSize,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"tasks": tasks, "total": total})
}

func (h *Handler) assignTask(c *gin.Context) {
	taskID := c.Param("id")

	var req struct {
		AssigneeID string `json:"assignee_id" binding:"required"`
		ActorID    string `json:"actor_id" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	task, err := h.taskService.AssignTask(c.Request.Context(), taskID, req.AssigneeID, req.ActorID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, task)
}

func (h *Handler) changeStatus(c *gin.Context) {
	taskID := c.Param("id")

	var req struct {
		NewStatus string `json:"new_status" binding:"required"`
		ActorID   string `json:"actor_id" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	task, err := h.taskService.ChangeTaskStatus(c.Request.Context(), taskID, domain.TaskStatus(req.NewStatus), req.ActorID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, task)
}

func (h *Handler) getHistory(c *gin.Context) {
	taskID := c.Param("id")

	history, err := h.taskService.GetTaskHistory(c.Request.Context(), taskID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"entries": history})
}

func (h *Handler) filterTasks(c *gin.Context) {
	var req struct {
		Status       string     `json:"status"`
		Priority     string     `json:"priority"`
		AssigneeID   string     `json:"assignee_id"`
		DocumentID   string     `json:"document_id"`
		DeadlineFrom *time.Time `json:"deadline_from"`
		DeadlineTo   *time.Time `json:"deadline_to"`
		Page         int        `json:"page"`
		PageSize     int        `json:"page_size"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.Page == 0 {
		req.Page = 1
	}
	if req.PageSize == 0 {
		req.PageSize = 10
	}

	var priority domain.TaskPriority
	if req.Priority != "" {
		priority = domain.TaskPriority(req.Priority)
	}

	tasks, total, err := h.taskService.FilterTasks(c.Request.Context(), domain.AdvancedFilter{
		Status:       domain.TaskStatus(req.Status),
		Priority:     priority,
		AssigneeID:   req.AssigneeID,
		DocumentID:   req.DocumentID,
		DeadlineFrom: req.DeadlineFrom,
		DeadlineTo:   req.DeadlineTo,
		Page:         req.Page,
		PageSize:     req.PageSize,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"tasks": tasks, "total": total})
}

func (h *Handler) markOverdue(c *gin.Context) {
	taskIDs, err := h.taskService.MarkTasksOverdue(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"marked_tasks": taskIDs, "count": len(taskIDs)})
}

func (h *Handler) getStats(c *gin.Context) {
	stats, err := h.taskService.GetTaskStats(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, stats)
}

func parseQuery(c *gin.Context, key string, defaultVal int) int {
	val := c.Query(key)
	if val == "" {
		return defaultVal
	}
	parsed, err := strconv.Atoi(val)
	if err != nil {
		return defaultVal
	}
	return parsed
}
