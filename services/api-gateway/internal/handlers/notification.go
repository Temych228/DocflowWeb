package handlers

import (
	"github.com/Temych228/DocflowWeb/api-gateway/internal/clients"
	"github.com/Temych228/DocflowWeb/api-gateway/internal/middleware"
	"github.com/Temych228/DocflowWeb/api-gateway/pkg/dto"
	notifpb "github.com/Temych228/docflow-protos-final/notification/v1"
	"github.com/gin-gonic/gin"
)

type NotificationHandler struct {
	notifClient *clients.NotificationClient
}

func NewNotificationHandler(notifClient *clients.NotificationClient) *NotificationHandler {
	return &NotificationHandler{notifClient: notifClient}
}

func (h *NotificationHandler) SendEmail(c *gin.Context) {
	var req SendEmailRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, dto.NewErrorResponse(dto.ErrorValidation, err.Error(), nil))
		return
	}

	_, err := h.notifClient.Client.SendEmail(c.Request.Context(), &notifpb.SendEmailRequest{
		To:      req.To,
		Subject: req.Subject,
		Body:    req.Body,
	})
	if err != nil {
		c.JSON(500, dto.NewErrorResponse(dto.ErrorInternal, grpcErrMessage(err), nil))
		return
	}

	c.JSON(200, dto.NewSuccessResponse(gin.H{"sent": true}))
}

func (h *NotificationHandler) SendBulkEmail(c *gin.Context) {
	var req SendBulkEmailRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, dto.NewErrorResponse(dto.ErrorValidation, err.Error(), nil))
		return
	}

	_, err := h.notifClient.Client.SendBulkEmail(c.Request.Context(), &notifpb.SendBulkEmailRequest{
		To:      req.To,
		Subject: req.Subject,
		Body:    req.Body,
	})
	if err != nil {
		c.JSON(500, dto.NewErrorResponse(dto.ErrorInternal, grpcErrMessage(err), nil))
		return
	}

	c.JSON(200, dto.NewSuccessResponse(gin.H{"sent": len(req.To)}))
}

func (h *NotificationHandler) CreateNotification(c *gin.Context) {
	userID := middleware.ExtractUserID(c)

	var req CreateNotifRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, dto.NewErrorResponse(dto.ErrorValidation, err.Error(), nil))
		return
	}

	_, err := h.notifClient.Client.CreateNotification(c.Request.Context(), &notifpb.CreateNotificationRequest{
		UserId:  userID,
		Title:   req.Title,
		Body:    req.Body,
		RefId:   req.RefID,
		RefType: req.RefType,
	})
	if err != nil {
		c.JSON(500, dto.NewErrorResponse(dto.ErrorInternal, grpcErrMessage(err), nil))
		return
	}

	c.JSON(201, dto.NewSuccessResponse(gin.H{"created": true}))
}

func (h *NotificationHandler) GetNotificationHistory(c *gin.Context) {
	userID := middleware.ExtractUserID(c)
	if qID := c.Query("user_id"); qID != "" && middleware.ExtractRole(c) == "admin" {
		userID = qID
	}
	page := parseQueryInt(c, "page", 1)
	pageSize := parseQueryInt(c, "page_size", 10)

	resp, err := h.notifClient.Client.GetNotificationHistory(c.Request.Context(), &notifpb.GetNotificationHistoryRequest{
		UserId:   userID,
		Page:     int32(page),
		PageSize: int32(pageSize),
	})
	if err != nil {
		c.JSON(500, dto.NewErrorResponse(dto.ErrorInternal, grpcErrMessage(err), nil))
		return
	}

	c.JSON(200, dto.NewSuccessResponse(gin.H{"notifications": resp.Notifications, "total": resp.Total}))
}

func (h *NotificationHandler) MarkRead(c *gin.Context) {
	notifID := c.Param("id")
	userID := middleware.ExtractUserID(c)

	_, err := h.notifClient.Client.MarkNotificationRead(c.Request.Context(), &notifpb.MarkNotificationReadRequest{
		NotificationId: notifID,
		UserId:         userID,
	})
	if err != nil {
		c.JSON(500, dto.NewErrorResponse(dto.ErrorInternal, grpcErrMessage(err), nil))
		return
	}

	c.JSON(200, dto.NewSuccessResponse(gin.H{"marked": true}))
}

func (h *NotificationHandler) MarkAllRead(c *gin.Context) {
	userID := middleware.ExtractUserID(c)

	_, err := h.notifClient.Client.MarkAllRead(c.Request.Context(), &notifpb.MarkAllReadRequest{
		UserId: userID,
	})
	if err != nil {
		c.JSON(500, dto.NewErrorResponse(dto.ErrorInternal, grpcErrMessage(err), nil))
		return
	}

	c.JSON(200, dto.NewSuccessResponse(gin.H{"marked": true}))
}

func (h *NotificationHandler) GetUnreadCount(c *gin.Context) {
	userID := middleware.ExtractUserID(c)

	resp, err := h.notifClient.Client.GetUnreadCount(c.Request.Context(), &notifpb.GetUnreadCountRequest{
		UserId: userID,
	})
	if err != nil {
		c.JSON(500, dto.NewErrorResponse(dto.ErrorInternal, grpcErrMessage(err), nil))
		return
	}

	c.JSON(200, dto.NewSuccessResponse(gin.H{"count": resp.Count}))
}

func (h *NotificationHandler) DeleteNotification(c *gin.Context) {
	notifID := c.Param("id")
	userID := middleware.ExtractUserID(c)

	_, err := h.notifClient.Client.DeleteNotification(c.Request.Context(), &notifpb.DeleteNotificationRequest{
		NotificationId: notifID,
		UserId:         userID,
	})
	if err != nil {
		c.JSON(500, dto.NewErrorResponse(dto.ErrorInternal, grpcErrMessage(err), nil))
		return
	}

	c.JSON(200, dto.NewSuccessResponse(gin.H{"deleted": true}))
}

func (h *NotificationHandler) GetTemplate(c *gin.Context) {
	templateID := c.Param("id")

	resp, err := h.notifClient.Client.GetTemplate(c.Request.Context(), &notifpb.GetTemplateRequest{
		TemplateId: templateID,
	})
	if err != nil {
		c.JSON(404, dto.NewErrorResponse(dto.ErrorNotFound, "Template not found", nil))
		return
	}

	c.JSON(200, dto.NewSuccessResponse(resp.Template))
}

func (h *NotificationHandler) UpdatePreferences(c *gin.Context) {
	userID := middleware.ExtractUserID(c)

	var req UpdatePreferencesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, dto.NewErrorResponse(dto.ErrorValidation, err.Error(), nil))
		return
	}

	_, err := h.notifClient.Client.UpdatePreferences(c.Request.Context(), &notifpb.UpdatePreferencesRequest{
		Preferences: &notifpb.NotificationPreferences{
			UserId:        userID,
			EmailEnabled:  req.EmailEnabled,
			PushEnabled:   req.PushEnabled,
			DeadlineNotif: req.DeadlineNotif,
			AssignedNotif: req.AssignedNotif,
			StatusNotif:   req.StatusNotif,
			OverdueNotif:  req.OverdueNotif,
		},
	})
	if err != nil {
		c.JSON(500, dto.NewErrorResponse(dto.ErrorInternal, grpcErrMessage(err), nil))
		return
	}

	c.JSON(200, dto.NewSuccessResponse(gin.H{"updated": true}))
}

func (h *NotificationHandler) GetPreferences(c *gin.Context) {
	userID := middleware.ExtractUserID(c)

	resp, err := h.notifClient.Client.GetPreferences(c.Request.Context(), &notifpb.GetPreferencesRequest{
		UserId: userID,
	})
	if err != nil {
		c.JSON(500, dto.NewErrorResponse(dto.ErrorInternal, grpcErrMessage(err), nil))
		return
	}

	c.JSON(200, dto.NewSuccessResponse(resp.Preferences))
}

type SendEmailRequest struct {
	To      string `json:"to" binding:"required,email"`
	Subject string `json:"subject" binding:"required"`
	Body    string `json:"body" binding:"required"`
}

type SendBulkEmailRequest struct {
	To      []string `json:"to" binding:"required"`
	Subject string   `json:"subject" binding:"required"`
	Body    string   `json:"body" binding:"required"`
}

type CreateNotifRequest struct {
	Title   string `json:"title" binding:"required"`
	Body    string `json:"body" binding:"required"`
	RefID   string `json:"ref_id"`
	RefType string `json:"ref_type"`
}

type UpdatePreferencesRequest struct {
	EmailEnabled  bool `json:"email_enabled"`
	PushEnabled   bool `json:"push_enabled"`
	DeadlineNotif bool `json:"deadline_notif"`
	AssignedNotif bool `json:"assigned_notif"`
	StatusNotif   bool `json:"status_notif"`
	OverdueNotif  bool `json:"overdue_notif"`
}
