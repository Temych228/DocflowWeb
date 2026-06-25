package handlers

import (
	"strings"
	"time"

	"github.com/Temych228/DocflowWeb/api-gateway/internal/clients"
	"github.com/Temych228/DocflowWeb/api-gateway/internal/middleware"
	"github.com/Temych228/DocflowWeb/api-gateway/pkg/dto"
	mailpb "github.com/Temych228/docflow-protos-final/mail/v1"
	"github.com/gin-gonic/gin"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type MailHandler struct {
	mailClient *clients.MailClient
}

func NewMailHandler(mailClient *clients.MailClient) *MailHandler {
	return &MailHandler{mailClient: mailClient}
}

func (h *MailHandler) SendEmail(c *gin.Context) {
	var req sendMailRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, dto.NewErrorResponse(dto.ErrorValidation, err.Error(), nil))
		return
	}

	userID := req.UserID
	if userID == "" {
		userID = middleware.ExtractUserID(c)
	}

	resp, err := h.mailClient.SendEmail(c.Request.Context(), &mailpb.SendEmailRequest{
		To:           req.To,
		Subject:      req.Subject,
		Body:         req.Body,
		TemplateId:   req.TemplateID,
		TemplateVars: req.TemplateVars,
		UserId:       userID,
		Category:     req.Category,
	})
	if err != nil {
		c.JSON(500, dto.NewErrorResponse(dto.ErrorInternal, grpcErrMessage(err), nil))
		return
	}

	c.JSON(200, dto.NewSuccessResponse(gin.H{"success": resp.Success, "message_id": resp.MessageId}))
}

func (h *MailHandler) SendBulkEmail(c *gin.Context) {
	var req sendBulkMailRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, dto.NewErrorResponse(dto.ErrorValidation, err.Error(), nil))
		return
	}

	resp, err := h.mailClient.SendBulkEmail(c.Request.Context(), &mailpb.SendBulkEmailRequest{
		To:           req.To,
		Subject:      req.Subject,
		Body:         req.Body,
		TemplateId:   req.TemplateID,
		TemplateVars: req.TemplateVars,
		Category:     req.Category,
	})
	if err != nil {
		c.JSON(500, dto.NewErrorResponse(dto.ErrorInternal, grpcErrMessage(err), nil))
		return
	}

	c.JSON(200, dto.NewSuccessResponse(gin.H{"success": resp.Success, "sent": resp.Sent, "failed": resp.Failed}))
}

func (h *MailHandler) SubmitMailJob(c *gin.Context) {
	var req submitMailJobRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, dto.NewErrorResponse(dto.ErrorValidation, err.Error(), nil))
		return
	}

	resp, err := h.mailClient.SubmitMailJob(c.Request.Context(), &mailpb.MailJob{
		Id:             req.ID,
		JobId:          req.JobID,
		NotificationId: req.NotificationID,
		UserId:         req.UserID,
		Recipient:      req.Recipient,
		TemplateId:     req.TemplateID,
		Subject:        req.Subject,
		Body:           req.Body,
		Variables:      req.Variables,
		Category:       mailCategory(req.Category),
		Status:         mailStatus(req.Status),
		Attempts:       req.Attempts,
		MaxAttempts:    req.MaxAttempts,
		LastError:      req.LastError,
		QueuedAt:       parseMailTime(req.QueuedAt),
		ProcessedAt:    parseMailTimePtr(req.ProcessedAt),
		SentAt:         parseMailTimePtr(req.SentAt),
		FailedAt:       parseMailTimePtr(req.FailedAt),
		CreatedAt:      parseMailTime(req.CreatedAt),
		UpdatedAt:      parseMailTime(req.UpdatedAt),
	})
	if err != nil {
		c.JSON(500, dto.NewErrorResponse(dto.ErrorInternal, grpcErrMessage(err), nil))
		return
	}

	c.JSON(201, dto.NewSuccessResponse(resp.Job))
}

func (h *MailHandler) GetMailJob(c *gin.Context) {
	resp, err := h.mailClient.GetMailJob(c.Request.Context(), c.Param("id"))
	if err != nil {
		c.JSON(404, dto.NewErrorResponse(dto.ErrorNotFound, "Mail job not found", nil))
		return
	}

	c.JSON(200, dto.NewSuccessResponse(resp.Job))
}

func (h *MailHandler) ListMailJobs(c *gin.Context) {
	page := parseQueryInt(c, "page", 1)
	pageSize := parseQueryInt(c, "page_size", 20)
	status := strings.TrimSpace(c.Query("status"))
	category := strings.TrimSpace(c.Query("category"))

	resp, err := h.mailClient.ListMailJobs(c.Request.Context(), int32(page), int32(pageSize), status, category)
	if err != nil {
		c.JSON(500, dto.NewErrorResponse(dto.ErrorInternal, grpcErrMessage(err), nil))
		return
	}

	c.JSON(200, dto.NewSuccessResponse(gin.H{"jobs": resp.Jobs, "total": resp.Total, "page": page, "page_size": pageSize}))
}

func (h *MailHandler) GetTemplate(c *gin.Context) {
	resp, err := h.mailClient.GetTemplate(c.Request.Context(), c.Param("id"))
	if err != nil {
		c.JSON(404, dto.NewErrorResponse(dto.ErrorNotFound, "Template not found", nil))
		return
	}

	c.JSON(200, dto.NewSuccessResponse(resp.Template))
}

func (h *MailHandler) ListTemplates(c *gin.Context) {
	resp, err := h.mailClient.ListTemplates(c.Request.Context())
	if err != nil {
		c.JSON(500, dto.NewErrorResponse(dto.ErrorInternal, grpcErrMessage(err), nil))
		return
	}

	c.JSON(200, dto.NewSuccessResponse(gin.H{"templates": resp.Templates}))
}

func (h *MailHandler) UpdateTemplate(c *gin.Context) {
	var req updateTemplateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, dto.NewErrorResponse(dto.ErrorValidation, err.Error(), nil))
		return
	}

	resp, err := h.mailClient.UpdateTemplate(c.Request.Context(), &mailpb.MailTemplate{
		Id:       c.Param("id"),
		Name:     req.Name,
		Subject:  req.Subject,
		Body:     req.BodyTemplate,
		Channel:  req.Channel,
		IsActive: req.IsActive,
	})
	if err != nil {
		c.JSON(500, dto.NewErrorResponse(dto.ErrorInternal, grpcErrMessage(err), nil))
		return
	}

	c.JSON(200, dto.NewSuccessResponse(resp.Template))
}

func (h *MailHandler) GetStats(c *gin.Context) {
	resp, err := h.mailClient.GetStats(c.Request.Context(), parseMailTimePtr(c.Query("from")), parseMailTimePtr(c.Query("to")))
	if err != nil {
		c.JSON(500, dto.NewErrorResponse(dto.ErrorInternal, grpcErrMessage(err), nil))
		return
	}

	c.JSON(200, dto.NewSuccessResponse(resp.Stats))
}

type sendMailRequest struct {
	To           string            `json:"to" binding:"required,email"`
	Subject      string            `json:"subject" binding:"required"`
	Body         string            `json:"body" binding:"required"`
	TemplateID   string            `json:"template_id"`
	TemplateVars map[string]string `json:"template_vars"`
	UserID       string            `json:"user_id"`
	Category     string            `json:"category"`
}

type sendBulkMailRequest struct {
	To           []string          `json:"to" binding:"required"`
	Subject      string            `json:"subject" binding:"required"`
	Body         string            `json:"body" binding:"required"`
	TemplateID   string            `json:"template_id"`
	TemplateVars map[string]string `json:"template_vars"`
	Category     string            `json:"category"`
}

type submitMailJobRequest struct {
	ID             string            `json:"id"`
	JobID          string            `json:"job_id"`
	NotificationID string            `json:"notification_id"`
	UserID         string            `json:"user_id"`
	Recipient      []string          `json:"recipient" binding:"required"`
	TemplateID     string            `json:"template_id"`
	Subject        string            `json:"subject"`
	Body           string            `json:"body"`
	Variables      map[string]string `json:"variables"`
	Category       string            `json:"category"`
	Status         string            `json:"status"`
	Attempts       int32             `json:"attempts"`
	MaxAttempts    int32             `json:"max_attempts"`
	LastError      string            `json:"last_error"`
	QueuedAt       string            `json:"queued_at"`
	ProcessedAt    string            `json:"processed_at"`
	SentAt         string            `json:"sent_at"`
	FailedAt       string            `json:"failed_at"`
	CreatedAt      string            `json:"created_at"`
	UpdatedAt      string            `json:"updated_at"`
}

type updateTemplateRequest struct {
	Name         string `json:"name"`
	Subject      string `json:"subject"`
	BodyTemplate string `json:"body_template"`
	Channel      string `json:"channel"`
	IsActive     bool   `json:"is_active"`
}

func mailCategory(raw string) mailpb.MailCategory {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "document_assigned":
		return mailpb.MailCategory_MAIL_CATEGORY_DOCUMENT_ASSIGNED
	case "status_changed":
		return mailpb.MailCategory_MAIL_CATEGORY_STATUS_CHANGED
	case "deadline_reminder":
		return mailpb.MailCategory_MAIL_CATEGORY_DEADLINE_REMINDER
	case "overdue":
		return mailpb.MailCategory_MAIL_CATEGORY_OVERDUE
	case "task_assigned":
		return mailpb.MailCategory_MAIL_CATEGORY_TASK_ASSIGNED
	case "mention":
		return mailpb.MailCategory_MAIL_CATEGORY_MENTION
	case "verification":
		return mailpb.MailCategory_MAIL_CATEGORY_VERIFICATION
	case "password_reset":
		return mailpb.MailCategory_MAIL_CATEGORY_PASSWORD_RESET
	case "system":
		return mailpb.MailCategory_MAIL_CATEGORY_SYSTEM
	default:
		return mailpb.MailCategory_MAIL_CATEGORY_UNSPECIFIED
	}
}

func mailStatus(raw string) mailpb.MailStatus {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "queued":
		return mailpb.MailStatus_MAIL_STATUS_QUEUED
	case "processing":
		return mailpb.MailStatus_MAIL_STATUS_PROCESSING
	case "sent":
		return mailpb.MailStatus_MAIL_STATUS_SENT
	case "failed":
		return mailpb.MailStatus_MAIL_STATUS_FAILED
	case "bounced":
		return mailpb.MailStatus_MAIL_STATUS_BOUNCED
	case "cancelled":
		return mailpb.MailStatus_MAIL_STATUS_CANCELLED
	default:
		return mailpb.MailStatus_MAIL_STATUS_UNSPECIFIED
	}
}

func parseMailTime(raw string) *timestamppb.Timestamp {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	t, err := time.Parse(time.RFC3339, strings.TrimSpace(raw))
	if err != nil {
		return nil
	}
	return timestamppb.New(t)
}

func parseMailTimePtr(raw string) *timestamppb.Timestamp {
	return parseMailTime(raw)
}
