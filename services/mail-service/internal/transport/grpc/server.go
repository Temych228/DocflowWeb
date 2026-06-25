package grpc

import (
	"context"
	"time"

	"github.com/Temych228/DocflowWeb/services/mail-service/internal/domain"
	"github.com/Temych228/DocflowWeb/services/mail-service/internal/service"
	mailv1 "github.com/Temych228/docflow-protos-final/mail/v1"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type Server struct {
	mailv1.UnimplementedMailServiceServer
	svc *service.MailService
}

func New(svc *service.MailService) *Server {
	return &Server{svc: svc}
}

func Register(server mailv1.MailServiceServer, svc *service.MailService) {
	_ = server
	_ = svc
}

func (s *Server) SendEmail(
	ctx context.Context,
	req *mailv1.SendEmailRequest,
) (*mailv1.SendEmailResponse, error) {
	id, err := s.svc.SendEmail(
		ctx,
		req.GetTo(),
		req.GetSubject(),
		req.GetBody(),
		req.GetUserId(),
		req.GetTemplateId(),
		req.GetTemplateVars(),
		req.GetCategory(),
	)
	if err != nil {
		return nil, err
	}

	return &mailv1.SendEmailResponse{
		Success:   true,
		MessageId: id,
	}, nil
}

func (s *Server) SendBulkEmail(
	ctx context.Context,
	req *mailv1.SendBulkEmailRequest,
) (*mailv1.SendBulkEmailResponse, error) {
	sent, failed, err := s.svc.SendBulkEmail(
		ctx,
		req.GetTo(),
		req.GetSubject(),
		req.GetBody(),
		req.GetTemplateId(),
		req.GetTemplateVars(),
		req.GetCategory(),
	)
	if err != nil {
		return nil, err
	}

	return &mailv1.SendBulkEmailResponse{
		Success: true,
		Sent:    sent,
		Failed:  failed,
	}, nil
}

func (s *Server) SubmitMailJob(
	ctx context.Context,
	req *mailv1.SubmitMailJobRequest,
) (*mailv1.SubmitMailJobResponse, error) {
	if req.GetJob() == nil {
		return nil, domain.ErrInvalidInput
	}

	job, err := s.svc.Submit(ctx, protoToDomainJob(req.GetJob()))
	if err != nil {
		return nil, err
	}

	return &mailv1.SubmitMailJobResponse{
		Success: true,
		Job:     toProtoJob(job),
	}, nil
}

func (s *Server) GetMailJob(
	ctx context.Context,
	req *mailv1.GetMailJobRequest,
) (*mailv1.GetMailJobResponse, error) {
	job, err := s.svc.GetJob(ctx, req.GetId())
	if err != nil {
		return nil, err
	}

	return &mailv1.GetMailJobResponse{
		Job: toProtoJob(job),
	}, nil
}

func (s *Server) ListMailJobs(
	ctx context.Context,
	req *mailv1.ListMailJobsRequest,
) (*mailv1.ListMailJobsResponse, error) {
	jobs, total, err := s.svc.ListJobs(
		ctx,
		int(req.GetPage()),
		int(req.GetPageSize()),
		req.GetStatus(),
		req.GetCategory(),
	)
	if err != nil {
		return nil, err
	}

	out := make([]*mailv1.MailJob, 0, len(jobs))
	for _, job := range jobs {
		out = append(out, toProtoJob(job))
	}

	return &mailv1.ListMailJobsResponse{
		Jobs:  out,
		Total: int32(total),
	}, nil
}

func (s *Server) GetTemplate(
	ctx context.Context,
	req *mailv1.GetTemplateRequest,
) (*mailv1.GetTemplateResponse, error) {
	t, err := s.svc.GetTemplate(ctx, req.GetTemplateId())
	if err != nil {
		return nil, err
	}

	return &mailv1.GetTemplateResponse{Template: toProtoTemplate(t)}, nil
}

func (s *Server) ListTemplates(
	ctx context.Context,
	req *mailv1.ListTemplatesRequest,
) (*mailv1.ListTemplatesResponse, error) {
	items, err := s.svc.ListTemplates(ctx)
	if err != nil {
		return nil, err
	}

	out := make([]*mailv1.MailTemplate, 0, len(items))
	for _, item := range items {
		out = append(out, toProtoTemplate(item))
	}

	return &mailv1.ListTemplatesResponse{Templates: out}, nil
}

func (s *Server) UpdateTemplate(
	ctx context.Context,
	req *mailv1.UpdateTemplateRequest,
) (*mailv1.UpdateTemplateResponse, error) {
	if req.GetTemplate() == nil {
		return nil, domain.ErrInvalidInput
	}

	updated, err := s.svc.UpdateTemplate(ctx, protoToDomainTemplate(req.GetTemplate()))
	if err != nil {
		return nil, err
	}

	return &mailv1.UpdateTemplateResponse{Template: toProtoTemplate(updated)}, nil
}

func (s *Server) GetStats(
	ctx context.Context,
	req *mailv1.GetStatsRequest,
) (*mailv1.GetStatsResponse, error) {
	from := fromProtoTimestamp(req.GetFrom())
	to := fromProtoTimestamp(req.GetTo())

	stats, err := s.svc.GetStats(ctx, from, to)
	if err != nil {
		return nil, err
	}

	return &mailv1.GetStatsResponse{Stats: toProtoStats(stats)}, nil
}

func toProtoJob(j *domain.MailJob) *mailv1.MailJob {
	if j == nil {
		return nil
	}

	return &mailv1.MailJob{
		Id:             j.ID,
		JobId:          j.JobID,
		NotificationId: j.NotificationID,
		UserId:         j.UserID,
		Recipient:      j.Recipient,
		TemplateId:     j.TemplateID,
		Subject:        j.Subject,
		Body:           j.Body,
		Variables:      j.Variables,
		Category:       toProtoCategory(j.Category),
		Status:         toProtoStatus(j.Status),
		Attempts:       j.Attempts,
		MaxAttempts:    j.MaxAttempts,
		LastError:      j.LastError,
		QueuedAt:       toProtoTimestamp(j.QueuedAt),
		ProcessedAt:    toProtoTimestampPtr(j.ProcessedAt),
		SentAt:         toProtoTimestampPtr(j.SentAt),
		FailedAt:       toProtoTimestampPtr(j.FailedAt),
		CreatedAt:      toProtoTimestamp(j.CreatedAt),
		UpdatedAt:      toProtoTimestamp(j.UpdatedAt),
	}
}

func protoToDomainJob(j *mailv1.MailJob) *domain.MailJob {
	if j == nil {
		return nil
	}

	return &domain.MailJob{
		ID:             j.GetId(),
		JobID:          j.GetJobId(),
		NotificationID: j.GetNotificationId(),
		UserID:         j.GetUserId(),
		Recipient:      j.GetRecipient(),
		TemplateID:     j.GetTemplateId(),
		Subject:        j.GetSubject(),
		Body:           j.GetBody(),
		Variables:      j.GetVariables(),
		Category:       protoCategoryToDomain(j.GetCategory()),
		Status:         protoStatusToDomain(j.GetStatus()),
		Attempts:       j.GetAttempts(),
		MaxAttempts:    j.GetMaxAttempts(),
		LastError:      j.GetLastError(),
		QueuedAt:       fromProtoTimestamp(j.GetQueuedAt()),
		ProcessedAt:    ptrTime(fromProtoTimestamp(j.GetProcessedAt())),
		SentAt:         ptrTime(fromProtoTimestamp(j.GetSentAt())),
		FailedAt:       ptrTime(fromProtoTimestamp(j.GetFailedAt())),
		CreatedAt:      fromProtoTimestamp(j.GetCreatedAt()),
		UpdatedAt:      fromProtoTimestamp(j.GetUpdatedAt()),
	}
}

func toProtoTemplate(t *domain.MailTemplate) *mailv1.MailTemplate {
	if t == nil {
		return nil
	}

	return &mailv1.MailTemplate{
		Id:       t.TemplateID,
		Name:     t.TemplateID,
		Subject:  t.Subject,
		Body:     t.BodyTemplate,
		Channel:  t.Channel,
		IsActive: t.IsActive,
	}
}

func protoToDomainTemplate(t *mailv1.MailTemplate) *domain.MailTemplate {
	if t == nil {
		return nil
	}

	return &domain.MailTemplate{
		TemplateID:   t.GetId(),
		Subject:      t.GetSubject(),
		BodyTemplate: t.GetBody(),
		Channel:      t.GetChannel(),
		IsActive:     t.GetIsActive(),
	}
}

func toProtoStats(stats *domain.MailStats) *mailv1.MailStats {
	if stats == nil {
		return nil
	}

	return &mailv1.MailStats{
		Total:      stats.Total,
		Queued:     stats.Queued,
		Processing: stats.Processing,
		Sent:       stats.Sent,
		Failed:     stats.Failed,
		ByCategory: stats.ByCategory,
		ByHour:     stats.ByHour,
		From:       toProtoTimestamp(stats.From),
		To:         toProtoTimestamp(stats.To),
	}
}

func toProtoCategory(c domain.Category) mailv1.MailCategory {
	switch c {
	case domain.CategoryDocumentAssigned:
		return mailv1.MailCategory_MAIL_CATEGORY_DOCUMENT_ASSIGNED
	case domain.CategoryStatusChanged:
		return mailv1.MailCategory_MAIL_CATEGORY_STATUS_CHANGED
	case domain.CategoryDeadlineReminder:
		return mailv1.MailCategory_MAIL_CATEGORY_DEADLINE_REMINDER
	case domain.CategoryOverdue:
		return mailv1.MailCategory_MAIL_CATEGORY_OVERDUE
	case domain.CategoryTaskAssigned:
		return mailv1.MailCategory_MAIL_CATEGORY_TASK_ASSIGNED
	case domain.CategoryMention:
		return mailv1.MailCategory_MAIL_CATEGORY_MENTION
	case domain.CategorySystem:
		return mailv1.MailCategory_MAIL_CATEGORY_SYSTEM
	case domain.CategoryVerification:
		return mailv1.MailCategory_MAIL_CATEGORY_VERIFICATION
	case domain.CategoryPasswordReset:
		return mailv1.MailCategory_MAIL_CATEGORY_PASSWORD_RESET
	default:
		return mailv1.MailCategory_MAIL_CATEGORY_UNSPECIFIED
	}
}

func toProtoStatus(s domain.Status) mailv1.MailStatus {
	switch s {
	case domain.StatusQueued:
		return mailv1.MailStatus_MAIL_STATUS_QUEUED
	case domain.StatusProcessing:
		return mailv1.MailStatus_MAIL_STATUS_PROCESSING
	case domain.StatusSent:
		return mailv1.MailStatus_MAIL_STATUS_SENT
	case domain.StatusFailed:
		return mailv1.MailStatus_MAIL_STATUS_FAILED
	case domain.StatusBounced:
		return mailv1.MailStatus_MAIL_STATUS_BOUNCED
	case domain.StatusCancelled:
		return mailv1.MailStatus_MAIL_STATUS_CANCELLED
	default:
		return mailv1.MailStatus_MAIL_STATUS_UNSPECIFIED
	}
}

func protoCategoryToDomain(c mailv1.MailCategory) domain.Category {
	switch c {
	case mailv1.MailCategory_MAIL_CATEGORY_DOCUMENT_ASSIGNED:
		return domain.CategoryDocumentAssigned
	case mailv1.MailCategory_MAIL_CATEGORY_STATUS_CHANGED:
		return domain.CategoryStatusChanged
	case mailv1.MailCategory_MAIL_CATEGORY_DEADLINE_REMINDER:
		return domain.CategoryDeadlineReminder
	case mailv1.MailCategory_MAIL_CATEGORY_OVERDUE:
		return domain.CategoryOverdue
	case mailv1.MailCategory_MAIL_CATEGORY_TASK_ASSIGNED:
		return domain.CategoryTaskAssigned
	case mailv1.MailCategory_MAIL_CATEGORY_MENTION:
		return domain.CategoryMention
	case mailv1.MailCategory_MAIL_CATEGORY_SYSTEM:
		return domain.CategorySystem
	case mailv1.MailCategory_MAIL_CATEGORY_VERIFICATION:
		return domain.CategoryVerification
	case mailv1.MailCategory_MAIL_CATEGORY_PASSWORD_RESET:
		return domain.CategoryPasswordReset
	default:
		return domain.CategorySystem
	}
}

func protoStatusToDomain(s mailv1.MailStatus) domain.Status {
	switch s {
	case mailv1.MailStatus_MAIL_STATUS_QUEUED:
		return domain.StatusQueued
	case mailv1.MailStatus_MAIL_STATUS_PROCESSING:
		return domain.StatusProcessing
	case mailv1.MailStatus_MAIL_STATUS_SENT:
		return domain.StatusSent
	case mailv1.MailStatus_MAIL_STATUS_FAILED:
		return domain.StatusFailed
	case mailv1.MailStatus_MAIL_STATUS_BOUNCED:
		return domain.StatusBounced
	case mailv1.MailStatus_MAIL_STATUS_CANCELLED:
		return domain.StatusCancelled
	default:
		return domain.StatusQueued
	}
}

func toProtoTimestamp(t time.Time) *timestamppb.Timestamp {
	if t.IsZero() {
		return nil
	}
	return timestamppb.New(t)
}

func toProtoTimestampPtr(t *time.Time) *timestamppb.Timestamp {
	if t == nil || t.IsZero() {
		return nil
	}
	return timestamppb.New(*t)
}

func fromProtoTimestamp(ts *timestamppb.Timestamp) time.Time {
	if ts == nil {
		return time.Time{}
	}
	return ts.AsTime()
}

func ptrTime(t time.Time) *time.Time {
	if t.IsZero() {
		return nil
	}
	return &t
}
