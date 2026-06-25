package httptransport

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/Temych228/DocflowWeb/services/mail-service/internal/domain"
	"github.com/Temych228/DocflowWeb/services/mail-service/internal/service"
	"github.com/gin-gonic/gin"
)

type Handler struct {
	svc *service.MailService
}

func New(svc *service.MailService) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) Register(r *gin.Engine) {
	r.GET("/health", h.health)
	r.GET("/ready", h.ready)
	r.POST("/send-email", h.sendEmail)
	r.POST("/send-bulk", h.sendBulkEmail)

	jobs := r.Group("/jobs")
	{
		jobs.GET("", h.listJobs)
		jobs.GET("/:id", h.getJob)
	}

	templates := r.Group("/templates")
	{
		templates.GET("", h.listTemplates)
		templates.GET("/:id", h.getTemplate)
		templates.PUT("/:id", h.updateTemplate)
	}

	r.GET("/stats", h.stats)
}

func (h *Handler) health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func (h *Handler) ready(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ready"})
}

type sendEmailRequest struct {
	To           string            `json:"to"`
	Subject      string            `json:"subject"`
	Body         string            `json:"body"`
	UserID       string            `json:"user_id"`
	TemplateID   string            `json:"template_id"`
	TemplateVars map[string]string `json:"template_vars"`
	Category     string            `json:"category"`
}

type sendBulkEmailRequest struct {
	To           []string          `json:"to"`
	Subject      string            `json:"subject"`
	Body         string            `json:"body"`
	TemplateID   string            `json:"template_id"`
	TemplateVars map[string]string `json:"template_vars"`
	Category     string            `json:"category"`
}

func (h *Handler) sendEmail(c *gin.Context) {
	var req sendEmailRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		return
	}

	id, err := h.svc.SendEmail(c.Request.Context(), req.To, req.Subject, req.Body, req.UserID, req.TemplateID, req.TemplateVars, req.Category)
	if err != nil {
		handleErr(c, err)
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

	sent, failed, err := h.svc.SendBulkEmail(c.Request.Context(), req.To, req.Subject, req.Body, req.TemplateID, req.TemplateVars, req.Category)
	if err != nil {
		handleErr(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "sent": sent, "failed": failed})
}

func (h *Handler) listJobs(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	status := strings.TrimSpace(c.Query("status"))
	category := strings.TrimSpace(c.Query("category"))

	items, total, err := h.svc.ListJobs(c.Request.Context(), page, pageSize, status, category)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"items":     items,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}

func (h *Handler) getJob(c *gin.Context) {
	job, err := h.svc.GetJob(c.Request.Context(), c.Param("id"))
	if err != nil {
		handleErr(c, err)
		return
	}
	c.JSON(http.StatusOK, job)
}

func (h *Handler) listTemplates(c *gin.Context) {
	items, err := h.svc.ListTemplates(c.Request.Context())
	if err != nil {
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": items})
}

func (h *Handler) getTemplate(c *gin.Context) {
	t, err := h.svc.GetTemplate(c.Request.Context(), c.Param("id"))
	if err != nil {
		handleErr(c, err)
		return
	}
	c.JSON(http.StatusOK, t)
}

type templateRequest struct {
	Subject      string `json:"subject"`
	BodyTemplate string `json:"body_template"`
	Channel      string `json:"channel"`
	IsActive     bool   `json:"is_active"`
}

func (h *Handler) updateTemplate(c *gin.Context) {
	var req templateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		return
	}

	t, err := h.svc.UpdateTemplate(c.Request.Context(), &domain.MailTemplate{
		TemplateID:   c.Param("id"),
		Subject:      req.Subject,
		BodyTemplate: req.BodyTemplate,
		Channel:      req.Channel,
		IsActive:     req.IsActive,
	})
	if err != nil {
		handleErr(c, err)
		return
	}
	c.JSON(http.StatusOK, t)
}

func (h *Handler) stats(c *gin.Context) {
	from := parseTime(c.Query("from"))
	to := parseTime(c.Query("to"))
	if to.IsZero() {
		to = time.Now().UTC()
	}
	if from.IsZero() {
		from = to.Add(-24 * time.Hour)
	}

	s, err := h.svc.GetStats(c.Request.Context(), from, to)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}

	c.JSON(http.StatusOK, s)
}

func parseTime(raw string) time.Time {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return time.Time{}
	}
	t, _ := time.Parse(time.RFC3339, raw)
	return t.UTC()
}

func writeError(c *gin.Context, statusCode int, code, message string) {
	c.JSON(statusCode, gin.H{
		"error": gin.H{
			"code":    code,
			"message": message,
		},
	})
}

func handleErr(c *gin.Context, err error) {
	switch err {
	case domain.ErrNotFound:
		writeError(c, http.StatusNotFound, "NOT_FOUND", err.Error())
	case domain.ErrInvalidInput:
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
	default:
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
	}
}
