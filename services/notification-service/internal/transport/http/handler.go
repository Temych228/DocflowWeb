package httptransport

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/Temych228/DocflowWeb/services/notification-service/internal/domain"
	"github.com/Temych228/DocflowWeb/services/notification-service/internal/service"
	"github.com/gin-gonic/gin"
)

type Handler struct {
	svc *service.NotificationService
}

func New(svc *service.NotificationService) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) Register(r *gin.Engine) {
	r.GET("/health", h.health)

	notifs := r.Group("/notifications")
	{
		notifs.POST("/send-email", h.sendEmail)
		notifs.POST("/send-bulk", h.sendBulkEmail)
		notifs.POST("", h.createNotification)
		notifs.GET("", h.getHistory)
		notifs.GET("/unread", h.getUnreadCount)
		notifs.PATCH("/:id/read", h.markRead)
		notifs.PATCH("/read-all", h.markAllRead)
		notifs.DELETE("/:id", h.deleteNotification)
	}

	prefs := r.Group("/notification-preferences")
	{
		prefs.GET("/:user_id", h.getPreferences)
		prefs.PUT("/:user_id", h.updatePreferences)
	}

	templates := r.Group("/notification-templates")
	{
		templates.GET("/:template_id", h.getTemplate)
	}
}

func (h *Handler) health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

type createNotificationRequest struct {
	UserID     string `json:"user_id"`
	Title      string `json:"title"`
	Body       string `json:"body"`
	Category   string `json:"category"`
	RefID      string `json:"ref_id"`
	RefType    string `json:"ref_type"`
	DocumentID string `json:"document_id"`
	TaskID     string `json:"task_id"`
}

type sendEmailRequest struct {
	To           string            `json:"to"`
	Subject      string            `json:"subject"`
	Body         string            `json:"body"`
	UserID       string            `json:"user_id"`
	TemplateID   string            `json:"template_id"`
	TemplateVars map[string]string `json:"template_vars"`
}

type sendBulkEmailRequest struct {
	To           []string          `json:"to"`
	Subject      string            `json:"subject"`
	Body         string            `json:"body"`
	TemplateID   string            `json:"template_id"`
	TemplateVars map[string]string `json:"template_vars"`
}

type updatePreferencesRequest struct {
	EmailEnabled  bool `json:"email_enabled"`
	PushEnabled   bool `json:"push_enabled"`
	DeadlineNotif bool `json:"deadline_notif"`
	AssignedNotif bool `json:"assigned_notif"`
	StatusNotif   bool `json:"status_notif"`
	OverdueNotif  bool `json:"overdue_notif"`
}

func (h *Handler) sendEmail(c *gin.Context) {
	var req sendEmailRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		return
	}

	id, err := h.svc.SendEmail(c.Request.Context(), req.To, req.Subject, req.Body, req.UserID, req.TemplateID, req.TemplateVars)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "message_id": id})
}

func (h *Handler) sendBulkEmail(c *gin.Context) {
	var req sendBulkEmailRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		return
	}

	sent, failed, err := h.svc.SendBulkEmail(c.Request.Context(), req.To, req.Subject, req.Body, req.TemplateID, req.TemplateVars)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "sent": sent, "failed": failed})
}

func (h *Handler) createNotification(c *gin.Context) {
	var req createNotificationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		return
	}

	n, err := h.svc.CreateNotification(
		c.Request.Context(),
		req.UserID,
		req.Title,
		req.Body,
		domain.NormalizeCategory(req.Category),
		req.RefID,
		req.RefType,
		req.DocumentID,
		req.TaskID,
	)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusCreated, n)
}

func (h *Handler) getHistory(c *gin.Context) {
	userID := strings.TrimSpace(c.Query("user_id"))
	if userID == "" {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "user_id is required")
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	items, total, err := h.svc.GetNotificationHistory(c.Request.Context(), userID, page, pageSize)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"items":     items,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}

func (h *Handler) getUnreadCount(c *gin.Context) {
	userID := strings.TrimSpace(c.Query("user_id"))
	if userID == "" {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "user_id is required")
		return
	}

	count, err := h.svc.GetUnreadCount(c.Request.Context(), userID)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"count": count})
}

func (h *Handler) markRead(c *gin.Context) {
	userID := strings.TrimSpace(c.Query("user_id"))
	if userID == "" {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "user_id is required")
		return
	}

	if _, err := h.svc.MarkNotificationRead(c.Request.Context(), c.Param("id"), userID); err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}

func (h *Handler) markAllRead(c *gin.Context) {
	userID := strings.TrimSpace(c.Query("user_id"))
	if userID == "" {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "user_id is required")
		return
	}

	count, err := h.svc.MarkAllRead(c.Request.Context(), userID)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "count": count})
}

func (h *Handler) deleteNotification(c *gin.Context) {
	userID := strings.TrimSpace(c.Query("user_id"))
	if userID == "" {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "user_id is required")
		return
	}

	if err := h.svc.DeleteNotification(c.Request.Context(), c.Param("id"), userID); err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}

func (h *Handler) getPreferences(c *gin.Context) {
	p, err := h.svc.GetPreferences(c.Request.Context(), c.Param("user_id"))
	if err != nil {
		handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, p)
}

func (h *Handler) updatePreferences(c *gin.Context) {
	var req updatePreferencesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		return
	}

	out, err := h.svc.UpdatePreferences(c.Request.Context(), &domain.Preferences{
		UserID:        c.Param("user_id"),
		EmailEnabled:  req.EmailEnabled,
		PushEnabled:   req.PushEnabled,
		DeadlineNotif: req.DeadlineNotif,
		AssignedNotif: req.AssignedNotif,
		StatusNotif:   req.StatusNotif,
		OverdueNotif:  req.OverdueNotif,
	})
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, out)
}

func (h *Handler) getTemplate(c *gin.Context) {
	t, err := h.svc.GetTemplate(c.Request.Context(), c.Param("template_id"))
	if err != nil {
		handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, t)
}

func writeError(c *gin.Context, statusCode int, code, message string) {
	c.JSON(statusCode, gin.H{
		"error": gin.H{
			"code":    code,
			"message": message,
		},
	})
}

func handleError(c *gin.Context, err error) {
	switch err {
	case domain.ErrInvalidInput:
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
	case domain.ErrNotFound:
		writeError(c, http.StatusNotFound, "NOT_FOUND", err.Error())
	default:
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
	}
}
