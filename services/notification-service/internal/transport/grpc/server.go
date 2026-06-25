package grpcserver

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/Temych228/DocflowWeb/services/notification-service/internal/domain"
	"github.com/Temych228/DocflowWeb/services/notification-service/internal/service"
	notifv1 "github.com/Temych228/docflow-protos-final/notification/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type Server struct {
	notifv1.UnimplementedNotificationServiceServer
	svc *service.NotificationService
}

func New(svc *service.NotificationService) *Server {
	return &Server{svc: svc}
}

func (s *Server) SendEmail(ctx context.Context, req *notifv1.SendEmailRequest) (*notifv1.SendEmailResponse, error) {
	id, err := s.svc.SendEmail(ctx, req.GetTo(), req.GetSubject(), req.GetBody(), "", req.GetTemplateId(), req.GetTemplateVars())
	if err != nil {
		return nil, mapErr(err)
	}
	return &notifv1.SendEmailResponse{Success: true, MessageId: id}, nil
}

func (s *Server) SendBulkEmail(ctx context.Context, req *notifv1.SendBulkEmailRequest) (*notifv1.SendBulkEmailResponse, error) {
	sent, failed, err := s.svc.SendBulkEmail(ctx, req.GetTo(), req.GetSubject(), req.GetBody(), req.GetTemplateId(), req.GetTemplateVars())
	if err != nil {
		return nil, mapErr(err)
	}
	return &notifv1.SendBulkEmailResponse{Success: true, Sent: sent, Failed: failed}, nil
}

func (s *Server) CreateNotification(ctx context.Context, req *notifv1.CreateNotificationRequest) (*notifv1.CreateNotificationResponse, error) {
	n, err := s.svc.CreateNotification(
		ctx,
		req.GetUserId(),
		req.GetTitle(),
		req.GetBody(),
		categoryFromProto(req.GetNotifType(), req.GetRefType(), req.GetTitle()),
		req.GetRefId(),
		req.GetRefType(),
		extractDocumentID(req.GetRefType(), req.GetRefId()),
		extractTaskID(req.GetRefType(), req.GetRefId()),
	)
	if err != nil {
		return nil, mapErr(err)
	}
	return &notifv1.CreateNotificationResponse{Notification: toProtoNotification(n)}, nil
}

func (s *Server) GetNotificationHistory(ctx context.Context, req *notifv1.GetNotificationHistoryRequest) (*notifv1.GetNotificationHistoryResponse, error) {
	items, total, err := s.svc.GetNotificationHistory(ctx, req.GetUserId(), int(req.GetPage()), int(req.GetPageSize()))
	if err != nil {
		return nil, mapErr(err)
	}
	out := make([]*notifv1.Notification, 0, len(items))
	for _, n := range items {
		out = append(out, toProtoNotification(n))
	}
	return &notifv1.GetNotificationHistoryResponse{Notifications: out, Total: int32(total)}, nil
}

func (s *Server) MarkNotificationRead(ctx context.Context, req *notifv1.MarkNotificationReadRequest) (*notifv1.MarkNotificationReadResponse, error) {
	if strings.TrimSpace(req.GetNotificationId()) == "" || strings.TrimSpace(req.GetUserId()) == "" {
		return nil, status.Error(codes.InvalidArgument, "notification_id and user_id are required")
	}
	if _, err := s.svc.MarkNotificationRead(ctx, req.GetNotificationId(), req.GetUserId()); err != nil {
		return nil, mapErr(err)
	}
	return &notifv1.MarkNotificationReadResponse{Success: true}, nil
}

func (s *Server) MarkAllRead(ctx context.Context, req *notifv1.MarkAllReadRequest) (*notifv1.MarkAllReadResponse, error) {
	if strings.TrimSpace(req.GetUserId()) == "" {
		return nil, status.Error(codes.InvalidArgument, "user_id is required")
	}
	count, err := s.svc.MarkAllRead(ctx, req.GetUserId())
	if err != nil {
		return nil, mapErr(err)
	}
	return &notifv1.MarkAllReadResponse{Success: true, Count: count}, nil
}

func (s *Server) StreamNotifications(req *notifv1.StreamNotificationsRequest, stream notifv1.NotificationService_StreamNotificationsServer) error {
	if strings.TrimSpace(req.GetUserId()) == "" {
		return status.Error(codes.InvalidArgument, "user_id is required")
	}

	ch, unsubscribe := s.svc.StreamNotifications(stream.Context(), req.GetUserId())
	defer unsubscribe()

	for {
		select {
		case <-stream.Context().Done():
			return nil
		case n, ok := <-ch:
			if !ok {
				return nil
			}
			if err := stream.Send(&notifv1.NotificationEvent{
				Notification: toProtoNotification(n),
				EventType:    "notification_created",
				OccurredAt:   timestamppb.New(n.CreatedAt),
			}); err != nil {
				return status.Error(codes.Internal, err.Error())
			}
		}
	}
}

func (s *Server) GetUnreadCount(ctx context.Context, req *notifv1.GetUnreadCountRequest) (*notifv1.GetUnreadCountResponse, error) {
	if strings.TrimSpace(req.GetUserId()) == "" {
		return nil, status.Error(codes.InvalidArgument, "user_id is required")
	}
	count, err := s.svc.GetUnreadCount(ctx, req.GetUserId())
	if err != nil {
		return nil, mapErr(err)
	}
	return &notifv1.GetUnreadCountResponse{Count: count}, nil
}

func (s *Server) DeleteNotification(ctx context.Context, req *notifv1.DeleteNotificationRequest) (*notifv1.DeleteNotificationResponse, error) {
	if strings.TrimSpace(req.GetNotificationId()) == "" || strings.TrimSpace(req.GetUserId()) == "" {
		return nil, status.Error(codes.InvalidArgument, "notification_id and user_id are required")
	}
	if err := s.svc.DeleteNotification(ctx, req.GetNotificationId(), req.GetUserId()); err != nil {
		return nil, mapErr(err)
	}
	return &notifv1.DeleteNotificationResponse{Success: true}, nil
}

func (s *Server) GetTemplate(ctx context.Context, req *notifv1.GetTemplateRequest) (*notifv1.GetTemplateResponse, error) {
	if strings.TrimSpace(req.GetTemplateId()) == "" {
		return nil, status.Error(codes.InvalidArgument, "template_id is required")
	}
	t, err := s.svc.GetTemplate(ctx, req.GetTemplateId())
	if err != nil {
		return nil, mapErr(err)
	}
	return &notifv1.GetTemplateResponse{
		Template: &notifv1.NotificationTemplate{
			Id:      t.TemplateID,
			Name:    t.TemplateID,
			Subject: t.Subject,
			Body:    t.BodyTemplate,
		},
	}, nil
}

func (s *Server) UpdatePreferences(ctx context.Context, req *notifv1.UpdatePreferencesRequest) (*notifv1.UpdatePreferencesResponse, error) {
	p := req.GetPreferences()
	if p == nil {
		return nil, status.Error(codes.InvalidArgument, "preferences is required")
	}
	out, err := s.svc.UpdatePreferences(ctx, &domain.Preferences{
		UserID:        p.GetUserId(),
		EmailEnabled:  p.GetEmailEnabled(),
		PushEnabled:   p.GetPushEnabled(),
		DeadlineNotif: p.GetDeadlineNotif(),
		AssignedNotif: p.GetAssignedNotif(),
		StatusNotif:   p.GetStatusNotif(),
		OverdueNotif:  p.GetOverdueNotif(),
	})
	if err != nil {
		return nil, mapErr(err)
	}
	return &notifv1.UpdatePreferencesResponse{Preferences: &notifv1.NotificationPreferences{
		UserId:        out.UserID,
		EmailEnabled:  out.EmailEnabled,
		PushEnabled:   out.PushEnabled,
		DeadlineNotif: out.DeadlineNotif,
		AssignedNotif: out.AssignedNotif,
		StatusNotif:   out.StatusNotif,
		OverdueNotif:  out.OverdueNotif,
	}}, nil
}

func (s *Server) GetPreferences(ctx context.Context, req *notifv1.GetPreferencesRequest) (*notifv1.GetPreferencesResponse, error) {
	if strings.TrimSpace(req.GetUserId()) == "" {
		return nil, status.Error(codes.InvalidArgument, "user_id is required")
	}
	p, err := s.svc.GetPreferences(ctx, req.GetUserId())
	if err != nil {
		return nil, mapErr(err)
	}
	return &notifv1.GetPreferencesResponse{Preferences: &notifv1.NotificationPreferences{
		UserId:        p.UserID,
		EmailEnabled:  p.EmailEnabled,
		PushEnabled:   p.PushEnabled,
		DeadlineNotif: p.DeadlineNotif,
		AssignedNotif: p.AssignedNotif,
		StatusNotif:   p.StatusNotif,
		OverdueNotif:  p.OverdueNotif,
	}}, nil
}

func toProtoNotification(n *domain.Notification) *notifv1.Notification {
	if n == nil {
		return nil
	}
	var readAt *timestamppb.Timestamp
	if n.ReadAt != nil {
		readAt = timestamppb.New(*n.ReadAt)
	}
	return &notifv1.Notification{
		Id:        n.ID,
		UserId:    n.UserID,
		Title:     n.Title,
		Body:      n.Body,
		NotifType: protoTypeFromCategory(n.Category),
		RefId:     n.RefID,
		RefType:   n.RefType,
		IsRead:    n.IsRead,
		CreatedAt: timestamppb.New(n.CreatedAt),
		ReadAt:    readAt,
	}
}

func protoTypeFromCategory(category domain.Category) notifv1.NotificationType {
	switch category {
	case domain.CategoryDeadlineReminder:
		return notifv1.NotificationType_NOTIFICATION_TYPE_DEADLINE
	case domain.CategoryDocumentAssigned:
		return notifv1.NotificationType_NOTIFICATION_TYPE_ASSIGNED
	case domain.CategoryTaskAssigned:
		return notifv1.NotificationType_NOTIFICATION_TYPE_ASSIGNED
	case domain.CategoryStatusChanged:
		return notifv1.NotificationType_NOTIFICATION_TYPE_STATUS_CHANGE
	case domain.CategoryOverdue:
		return notifv1.NotificationType_NOTIFICATION_TYPE_OVERDUE
	case domain.CategoryMention:
		return notifv1.NotificationType_NOTIFICATION_TYPE_MENTION
	default:
		return notifv1.NotificationType_NOTIFICATION_TYPE_SYSTEM
	}
}

func categoryFromProto(t notifv1.NotificationType, refType, title string) domain.Category {
	switch t {
	case notifv1.NotificationType_NOTIFICATION_TYPE_DEADLINE:
		return domain.CategoryDeadlineReminder
	case notifv1.NotificationType_NOTIFICATION_TYPE_ASSIGNED:
		if strings.EqualFold(refType, "task") {
			return domain.CategoryTaskAssigned
		}
		return domain.CategoryDocumentAssigned
	case notifv1.NotificationType_NOTIFICATION_TYPE_STATUS_CHANGE:
		return domain.CategoryStatusChanged
	case notifv1.NotificationType_NOTIFICATION_TYPE_OVERDUE:
		return domain.CategoryOverdue
	case notifv1.NotificationType_NOTIFICATION_TYPE_MENTION:
		return domain.CategoryMention
	default:
		l := strings.ToLower(title)
		switch {
		case strings.Contains(l, "task"):
			return domain.CategoryTaskAssigned
		case strings.Contains(l, "document"):
			return domain.CategoryDocumentAssigned
		default:
			return domain.CategorySystem
		}
	}
}

func extractDocumentID(refType, refID string) string {
	if strings.EqualFold(strings.TrimSpace(refType), "document") {
		return refID
	}
	return ""
}

func extractTaskID(refType, refID string) string {
	if strings.EqualFold(strings.TrimSpace(refType), "task") {
		return refID
	}
	return ""
}

func mapErr(err error) error {
	switch {
	case errors.Is(err, domain.ErrNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, domain.ErrInvalidInput):
		return status.Error(codes.InvalidArgument, err.Error())
	default:
		return status.Error(codes.Internal, fmt.Sprintf("%v", err))
	}
}
