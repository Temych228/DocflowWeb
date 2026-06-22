package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/Temych228/DocflowWeb/services/notification-service/internal/clients"
	"github.com/Temych228/DocflowWeb/services/notification-service/internal/domain"
	"github.com/Temych228/DocflowWeb/services/notification-service/internal/messaging"
	"github.com/Temych228/DocflowWeb/services/notification-service/internal/observability"
	"github.com/Temych228/DocflowWeb/services/notification-service/internal/repository"
)

type NotificationService struct {
	repo        repository.NotificationRepository
	publisher   messaging.Publisher
	userLookup  clients.UserLookup
	hub         *Hub
	metrics     *observability.Metrics
	serviceName string
	dedupTTL    time.Duration
}

func New(repo repository.NotificationRepository, publisher messaging.Publisher, userLookup clients.UserLookup, hub *Hub, metrics *observability.Metrics, dedupTTL time.Duration) *NotificationService {
	if publisher == nil {
		publisher = &messaging.NoopPublisher{}
	}
	if hub == nil {
		hub = NewHub()
	}
	if dedupTTL <= 0 {
		dedupTTL = 24 * time.Hour
	}
	return &NotificationService{
		repo:        repo,
		publisher:   publisher,
		userLookup:  userLookup,
		hub:         hub,
		metrics:     metrics,
		serviceName: "notification-service",
		dedupTTL:    dedupTTL,
	}
}

func (s *NotificationService) SendEmail(ctx context.Context, to, subject, body, userID, templateID string, vars map[string]string) (string, error) {
	to = strings.TrimSpace(to)
	subject = strings.TrimSpace(subject)
	body = strings.TrimSpace(body)
	userID = strings.TrimSpace(userID)
	templateID = strings.TrimSpace(templateID)

	if to == "" || (subject == "" && templateID == "") || (body == "" && templateID == "") {
		return "", domain.ErrInvalidInput
	}
	if vars == nil {
		vars = map[string]string{}
	}

	jobID, err := s.publisher.PublishEmailJob(ctx, domain.EmailJob{
		Recipient:  []string{to},
		Subject:    subject,
		Body:       body,
		TemplateID: templateID,
		Variables:  vars,
		UserID:     userID,
		Category:   domain.CategorySystem,
	})
	if err != nil {
		_ = s.log(ctx, "error", "SendEmail", "failed to enqueue email job", map[string]any{"error": err.Error(), "user_id": userID})
		return "", err
	}

	_ = s.log(ctx, "info", "SendEmail", "email job enqueued", map[string]any{"user_id": userID, "to": to, "subject": subject, "job_id": jobID})
	if s.metrics != nil {
		s.metrics.IncEmailJob(templateID, domain.CategorySystem.String())
	}
	return jobID, nil
}

func (s *NotificationService) SendBulkEmail(ctx context.Context, to []string, subject, body, templateID string, vars map[string]string) (int32, int32, error) {
	subject = strings.TrimSpace(subject)
	body = strings.TrimSpace(body)
	templateID = strings.TrimSpace(templateID)

	if len(to) == 0 || (subject == "" && templateID == "") || (body == "" && templateID == "") {
		return 0, 0, domain.ErrInvalidInput
	}
	if vars == nil {
		vars = map[string]string{}
	}

	var sent, failed int32
	for _, addr := range to {
		addr = strings.TrimSpace(addr)
		if addr == "" {
			failed++
			continue
		}
		if _, err := s.publisher.PublishEmailJob(ctx, domain.EmailJob{
			Recipient:  []string{addr},
			Subject:    subject,
			Body:       body,
			TemplateID: templateID,
			Variables:  vars,
			Category:   domain.CategorySystem,
		}); err != nil {
			failed++
			continue
		}
		sent++
	}

	_ = s.log(ctx, "info", "SendBulkEmail", "bulk email jobs enqueued", map[string]any{"sent": sent, "failed": failed, "subject": subject})
	if s.metrics != nil {
		s.metrics.IncEmailJob(templateID, domain.CategorySystem.String())
	}
	return sent, failed, nil
}

func (s *NotificationService) CreateNotification(ctx context.Context, userID, title, body string, category domain.Category, refID, refType, documentID, taskID string) (*domain.Notification, error) {
	startedAt := time.Now()

	userID = strings.TrimSpace(userID)
	title = strings.TrimSpace(title)
	body = strings.TrimSpace(body)
	refID = strings.TrimSpace(refID)
	refType = strings.TrimSpace(refType)
	documentID = strings.TrimSpace(documentID)
	taskID = strings.TrimSpace(taskID)
	category = domain.NormalizeCategory(category.String())

	if userID == "" || title == "" || body == "" {
		return nil, domain.ErrInvalidInput
	}

	n := &domain.Notification{
		UserID:     userID,
		DocumentID: documentID,
		TaskID:     taskID,
		Category:   category,
		Title:      title,
		Body:       body,
		RefID:      refID,
		RefType:    refType,
		ProtoType:  protoTypeForCategory(category),
		IsRead:     false,
		SentEmail:  false,
	}

	created, err := s.repo.CreateNotification(ctx, n)
	if err != nil {
		_ = s.log(ctx, "error", "CreateNotification", "failed to create notification", map[string]any{"error": err.Error(), "user_id": userID, "category": category.String()})
		return nil, err
	}

	_ = s.publisher.PublishNotificationEvent(ctx, created)
	s.hub.Publish(created)
	_ = s.repo.InvalidateUnreadCountCache(ctx, userID)

	if s.metrics != nil {
		s.metrics.IncCreated(category.String(), "system")
		s.metrics.ObserveProcessing(category.String(), startedAt)
	}

	_ = s.log(ctx, "info", "CreateNotification", "notification created", map[string]any{"notification_id": created.ID, "user_id": userID, "category": category.String()})

	prefs, err := s.repo.GetPreferences(ctx, userID)
	if err == nil && prefs != nil && s.shouldQueueEmail(category, prefs) {
		if recipient, lookupErr := s.resolveRecipient(ctx, userID); lookupErr == nil && recipient != nil && recipient.Email != "" {
			tpl := notificationTemplateForCategory(category)
			vars := buildVars(recipient.Name, title, refID, refType, string(category), "")
			rendered, renderErr := renderTemplate(tpl.BodyTemplate, vars)
			if renderErr == nil {
				jobID, jobErr := s.publisher.PublishEmailJob(ctx, domain.EmailJob{
					Recipient:      []string{recipient.Email},
					Subject:        tpl.Subject,
					Body:           rendered,
					TemplateID:     tpl.TemplateID,
					Variables:      vars,
					UserID:         userID,
					NotificationID: created.ID,
					Category:       category,
				})
				if jobErr == nil {
					_ = s.repo.SetEmailQueued(ctx, created.ID)
					created.SentEmail = true
					if s.metrics != nil {
						s.metrics.IncEmailJob(tpl.TemplateID, category.String())
					}
					_ = s.log(ctx, "info", "CreateNotification", "email job enqueued", map[string]any{"notification_id": created.ID, "user_id": userID, "recipient": recipient.Email, "job_id": jobID, "template_id": tpl.TemplateID})
				} else {
					_ = s.log(ctx, "error", "CreateNotification", "failed to enqueue email job", map[string]any{"notification_id": created.ID, "user_id": userID, "error": jobErr.Error()})
				}
			}
		}
	}

	return created, nil
}

func (s *NotificationService) GetNotificationHistory(ctx context.Context, userID string, page, pageSize int) ([]*domain.Notification, int, error) {
	items, total, err := s.repo.GetHistory(ctx, userID, page, pageSize)
	if err != nil {
		_ = s.log(ctx, "error", "GetNotificationHistory", "failed to load notification history", map[string]any{"error": err.Error(), "user_id": userID})
		return nil, 0, err
	}
	return items, total, nil
}

func (s *NotificationService) MarkNotificationRead(ctx context.Context, notificationID, userID string) (*domain.Notification, error) {
	n, err := s.repo.MarkRead(ctx, notificationID, userID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, err
		}
		_ = s.log(ctx, "error", "MarkNotificationRead", "failed to mark notification as read", map[string]any{"error": err.Error(), "notification_id": notificationID, "user_id": userID})
		return nil, err
	}

	_ = s.publisher.PublishNotificationEvent(ctx, n)
	s.hub.Publish(n)
	if s.metrics != nil {
		s.metrics.IncRead(n.Category.String())
	}
	_ = s.log(ctx, "info", "MarkNotificationRead", "notification marked as read", map[string]any{"notification_id": notificationID, "user_id": userID})
	return n, nil
}

func (s *NotificationService) MarkAllRead(ctx context.Context, userID string) (int32, error) {
	count, err := s.repo.MarkAllRead(ctx, userID)
	if err != nil {
		_ = s.log(ctx, "error", "MarkAllRead", "failed to mark all notifications as read", map[string]any{"error": err.Error(), "user_id": userID})
		return 0, err
	}
	_ = s.log(ctx, "info", "MarkAllRead", "all notifications marked as read", map[string]any{"user_id": userID, "count": count})
	return count, nil
}

func (s *NotificationService) GetUnreadCount(ctx context.Context, userID string) (int32, error) {
	count, err := s.repo.GetUnreadCount(ctx, userID)
	if err != nil {
		_ = s.log(ctx, "error", "GetUnreadCount", "failed to get unread count", map[string]any{"error": err.Error(), "user_id": userID})
		return 0, err
	}
	return count, nil
}

func (s *NotificationService) DeleteNotification(ctx context.Context, notificationID, userID string) error {
	if err := s.repo.Delete(ctx, notificationID, userID); err != nil {
		_ = s.log(ctx, "error", "DeleteNotification", "failed to delete notification", map[string]any{"error": err.Error(), "notification_id": notificationID, "user_id": userID})
		return err
	}
	_ = s.log(ctx, "info", "DeleteNotification", "notification deleted", map[string]any{"notification_id": notificationID, "user_id": userID})
	return nil
}

func (s *NotificationService) GetTemplate(ctx context.Context, templateID string) (*domain.Template, error) {
	t, err := s.repo.GetTemplate(ctx, templateID)
	if err == nil {
		return t, nil
	}
	if tpl, ok := defaultTemplateByID(templateID); ok {
		return tpl, nil
	}
	return nil, err
}

func (s *NotificationService) UpdatePreferences(ctx context.Context, p *domain.Preferences) (*domain.Preferences, error) {
	if p == nil {
		return nil, domain.ErrInvalidInput
	}
	out, err := s.repo.UpsertPreferences(ctx, p)
	if err != nil {
		_ = s.log(ctx, "error", "UpdatePreferences", "failed to update preferences", map[string]any{"error": err.Error(), "user_id": p.UserID})
		return nil, err
	}
	_ = s.log(ctx, "info", "UpdatePreferences", "preferences updated", map[string]any{"user_id": out.UserID})
	return out, nil
}

func (s *NotificationService) GetPreferences(ctx context.Context, userID string) (*domain.Preferences, error) {
	p, err := s.repo.GetPreferences(ctx, userID)
	if err != nil {
		_ = s.log(ctx, "error", "GetPreferences", "failed to get preferences", map[string]any{"error": err.Error(), "user_id": userID})
		return nil, err
	}
	return p, nil
}

func (s *NotificationService) StreamNotifications(ctx context.Context, userID string) (<-chan *domain.Notification, func()) {
	return s.hub.Subscribe(userID)
}

func (s *NotificationService) SeedTemplates(ctx context.Context) error {
	return s.repo.SeedDefaultTemplates(ctx, defaultTemplates)
}

func (s *NotificationService) RunDailyOverdueScan(ctx context.Context) error {
	return s.log(ctx, "info", "DailyCron", "overdue scan tick", nil)
}

func (s *NotificationService) resolveRecipient(ctx context.Context, userID string) (*clients.UserInfo, error) {
	if s.userLookup == nil {
		return nil, fmt.Errorf("user lookup is not configured")
	}
	return s.userLookup.GetUserByID(ctx, userID)
}

func (s *NotificationService) shouldQueueEmail(category domain.Category, prefs *domain.Preferences) bool {
	if prefs == nil || !prefs.EmailEnabled {
		return false
	}
	switch category {
	case domain.CategoryDocumentAssigned:
		return prefs.AssignedNotif
	case domain.CategoryTaskAssigned:
		return prefs.AssignedNotif
	case domain.CategoryStatusChanged:
		return prefs.StatusNotif
	case domain.CategoryDeadlineReminder:
		return prefs.DeadlineNotif
	case domain.CategoryOverdue:
		return prefs.OverdueNotif
	default:
		return false
	}
}

func (s *NotificationService) log(ctx context.Context, level, action, message string, meta map[string]any) error {
	return s.publisher.PublishLog(ctx, domain.LogEvent{
		Service: s.serviceName,
		Action:  action,
		Level:   level,
		Message: message,
		Meta:    meta,
		At:      time.Now().UTC(),
	})
}

func protoTypeForCategory(category domain.Category) int32 {
	switch category {
	case domain.CategoryDeadlineReminder:
		return 1
	case domain.CategoryDocumentAssigned:
		return 2
	case domain.CategoryTaskAssigned:
		return 2
	case domain.CategoryStatusChanged:
		return 3
	case domain.CategoryOverdue:
		return 4
	case domain.CategoryMention:
		return 5
	default:
		return 6
	}
}
