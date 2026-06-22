package service

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/Temych228/DocflowWeb/services/mail-service/internal/domain"
	"github.com/Temych228/DocflowWeb/services/mail-service/internal/messaging"
	"github.com/Temych228/DocflowWeb/services/mail-service/internal/observability"
	"github.com/Temych228/DocflowWeb/services/mail-service/internal/repository"
	"github.com/Temych228/DocflowWeb/services/mail-service/internal/sender"
	"github.com/google/uuid"
)

type MailService struct {
	repo        repository.MailRepository
	publisher   messaging.Publisher
	mailer      sender.Mailer
	metrics     *observability.Metrics
	serviceName string
	dedupTTL    time.Duration
	from        string
}

func New(repo repository.MailRepository, publisher messaging.Publisher, mailer sender.Mailer, metrics *observability.Metrics, from string, dedupTTL time.Duration) *MailService {
	if publisher == nil {
		publisher = &messaging.NoopPublisher{}
	}
	if dedupTTL <= 0 {
		dedupTTL = 24 * time.Hour
	}
	return &MailService{
		repo:        repo,
		publisher:   publisher,
		mailer:      mailer,
		metrics:     metrics,
		serviceName: "mail-service",
		dedupTTL:    dedupTTL,
		from:        from,
	}
}

func (s *MailService) SeedTemplates(ctx context.Context) error {
	return s.repo.SeedDefaultTemplates(ctx, defaultTemplates)
}

func (s *MailService) HandleQueuedPayload(ctx context.Context, payload []byte) (*domain.MailJob, error) {
	var incoming domain.MailJob
	if err := json.Unmarshal(payload, &incoming); err != nil {
		return nil, err
	}
	if incoming.JobID == "" {
		return nil, domain.ErrInvalidInput
	}
	if incoming.Category == "" {
		incoming.Category = domain.CategorySystem
	}
	return s.HandleQueuedJob(ctx, &incoming)
}

func (s *MailService) HandleQueuedJob(ctx context.Context, incoming *domain.MailJob) (*domain.MailJob, error) {
	if incoming == nil {
		return nil, domain.ErrInvalidInput
	}
	if len(incoming.Recipient) == 0 {
		return nil, domain.ErrInvalidInput
	}
	if incoming.JobID == "" {
		return nil, domain.ErrInvalidInput
	}

	startedAt := time.Now()
	ok, err := s.repo.MarkDedup(ctx, incoming.JobID, s.dedupTTL)
	if err != nil {
		_ = s.log(ctx, "error", "HandleQueuedJob", "dedup check failed", map[string]any{"job_id": incoming.JobID, "error": err.Error()})
		return nil, err
	}
	if !ok {
		return s.repo.GetJobByJobID(ctx, incoming.JobID)
	}

	job, err := s.repo.CreateJob(ctx, incoming)
	if err != nil {
		_ = s.log(ctx, "error", "HandleQueuedJob", "failed to store job", map[string]any{"job_id": incoming.JobID, "error": err.Error()})
		return nil, err
	}

	_ = s.repo.TrackBucket(ctx, job.Category.String(), domain.StatusQueued, job.QueuedAt)
	if s.metrics != nil {
		s.metrics.IncQueued(job.Category.String(), job.TemplateID)
		s.metrics.IncCategory(job.Category.String())
		s.metrics.IncHour(job.QueuedAt.UTC().Format("2006010215"))
	}

	processingAt := time.Now().UTC()
	job, err = s.repo.UpdateJobStatus(ctx, job.ID, domain.StatusProcessing, "", job.Attempts+1, &processingAt, nil, nil)
	if err != nil {
		_ = s.log(ctx, "error", "HandleQueuedJob", "failed to mark processing", map[string]any{"job_id": job.JobID, "error": err.Error()})
		return nil, err
	}

	if s.metrics != nil {
		s.metrics.IncProcessing(job.Category.String(), job.TemplateID)
		s.metrics.ObserveLatency(job.Category.String(), startedAt)
	}

	template := s.templateForJob(ctx, job)
	if template != nil && template.IsActive {
		if job.TemplateID == "" {
			job.TemplateID = template.TemplateID
		}
		if job.Subject == "" {
			job.Subject = template.Subject
		}
		if job.Body == "" {
			rendered, renderErr := renderTemplate(template.BodyTemplate, mergedVars(job))
			if renderErr != nil {
				job, err = s.repo.UpdateJobStatus(ctx, job.ID, domain.StatusFailed, renderErr.Error(), job.Attempts+1, nil, nil, timePtr(time.Now().UTC()))
				_ = s.repo.TrackBucket(ctx, job.Category.String(), domain.StatusFailed, time.Now().UTC())
				if s.metrics != nil {
					s.metrics.IncFailed(job.Category.String(), job.TemplateID)
				}
				_ = s.log(ctx, "error", "HandleQueuedJob", "template render failed", map[string]any{"job_id": job.JobID, "error": renderErr.Error()})
				return job, renderErr
			}
			job.Body = rendered
		}
	}

	if s.mailer == nil {
		err = errors.New("mailer is not configured")
		job, _ = s.repo.UpdateJobStatus(ctx, job.ID, domain.StatusFailed, err.Error(), job.Attempts+1, nil, nil, timePtr(time.Now().UTC()))
		_ = s.repo.TrackBucket(ctx, job.Category.String(), domain.StatusFailed, time.Now().UTC())
		if s.metrics != nil {
			s.metrics.IncFailed(job.Category.String(), job.TemplateID)
		}
		_ = s.log(ctx, "error", "HandleQueuedJob", "mailer not configured", map[string]any{"job_id": job.JobID})
		return job, err
	}

	sendAt := time.Now().UTC()
	err = s.mailer.Send(ctx, domain.SMTPMessage{
		From:    s.from,
		To:      job.Recipient,
		Subject: job.Subject,
		HTML:    job.Body,
		Text:    stripHTML(job.Body),
	})
	if err != nil {
		job, _ = s.repo.UpdateJobStatus(ctx, job.ID, domain.StatusFailed, err.Error(), job.Attempts+1, nil, nil, timePtr(sendAt))
		_ = s.repo.TrackBucket(ctx, job.Category.String(), domain.StatusFailed, sendAt)
		if s.metrics != nil {
			s.metrics.IncFailed(job.Category.String(), job.TemplateID)
		}
		_ = s.log(ctx, "error", "HandleQueuedJob", "mail send failed", map[string]any{"job_id": job.JobID, "error": err.Error()})
		return job, err
	}

	job, err = s.repo.UpdateJobStatus(ctx, job.ID, domain.StatusSent, "", job.Attempts+1, timePtr(sendAt), timePtr(sendAt), nil)
	if err != nil {
		_ = s.log(ctx, "error", "HandleQueuedJob", "failed to update sent status", map[string]any{"job_id": job.JobID, "error": err.Error()})
		return nil, err
	}

	_ = s.repo.TrackBucket(ctx, job.Category.String(), domain.StatusSent, sendAt)
	if s.metrics != nil {
		s.metrics.IncSent(job.Category.String(), job.TemplateID)
	}
	_ = s.log(ctx, "info", "HandleQueuedJob", "mail sent", map[string]any{"job_id": job.JobID, "category": job.Category.String(), "template_id": job.TemplateID})
	return job, nil
}

func (s *MailService) Submit(ctx context.Context, job *domain.MailJob) (*domain.MailJob, error) {
	if job == nil {
		return nil, domain.ErrInvalidInput
	}
	if len(job.Recipient) == 0 {
		return nil, domain.ErrInvalidInput
	}
	if job.JobID == "" {
		job.JobID = uuid.NewString()
	}
	if job.Category == "" {
		job.Category = domain.CategorySystem
	}
	if job.Status == "" {
		job.Status = domain.StatusQueued
	}

	created, err := s.repo.CreateJob(ctx, job)
	if err != nil {
		return nil, err
	}
	if s.metrics != nil {
		s.metrics.IncQueued(created.Category.String(), created.TemplateID)
		s.metrics.IncCategory(created.Category.String())
		s.metrics.IncHour(created.QueuedAt.UTC().Format("2006010215"))
	}
	return created, nil
}

func (s *MailService) SendEmail(ctx context.Context, to, subject, body, userID, templateID string, vars map[string]string, category string) (string, error) {
	if strings.TrimSpace(to) == "" {
		return "", domain.ErrInvalidInput
	}

	job := &domain.MailJob{
		JobID:       uuid.NewString(),
		UserID:      userID,
		Recipient:   []string{strings.TrimSpace(to)},
		TemplateID:  templateID,
		Subject:     subject,
		Body:        body,
		Variables:   vars,
		Category:    domain.NormalizeCategory(category),
		Status:      domain.StatusProcessing,
		Attempts:    1,
		MaxAttempts: 3,
	}

	template := s.templateForJob(ctx, job)
	if template != nil && template.IsActive {
		if job.Subject == "" {
			job.Subject = template.Subject
		}
		if job.Body == "" {
			rendered, err := renderTemplate(template.BodyTemplate, mergedVars(job))
			if err != nil {
				return "", err
			}
			job.Body = rendered
		}
	}

	if s.mailer == nil {
		return "", errors.New("mailer is not configured")
	}

	if err := s.mailer.Send(ctx, domain.SMTPMessage{
		From:    s.from,
		To:      job.Recipient,
		Subject: job.Subject,
		HTML:    job.Body,
		Text:    stripHTML(job.Body),
	}); err != nil {
		return "", err
	}

	return job.JobID, nil
}

func (s *MailService) SendBulkEmail(ctx context.Context, to []string, subject, body, templateID string, vars map[string]string, category string) (int32, int32, error) {
	if len(to) == 0 {
		return 0, 0, domain.ErrInvalidInput
	}

	var sent, failed int32
	for _, recipient := range to {
		_, err := s.SendEmail(ctx, recipient, subject, body, "", templateID, vars, category)
		if err != nil {
			failed++
		} else {
			sent++
		}
	}

	return sent, failed, nil
}

func (s *MailService) GetJob(ctx context.Context, id string) (*domain.MailJob, error) {
	return s.repo.GetJobByID(ctx, id)
}

func (s *MailService) ListJobs(ctx context.Context, page, pageSize int, status, category string) ([]*domain.MailJob, int, error) {
	return s.repo.ListJobs(ctx, page, pageSize, status, category)
}

func (s *MailService) GetTemplate(ctx context.Context, templateID string) (*domain.MailTemplate, error) {
	t, err := s.repo.GetTemplate(ctx, templateID)
	if err == nil {
		return t, nil
	}
	if tpl, ok := defaultTemplateByID(templateID); ok {
		return tpl, nil
	}
	return nil, err
}

func (s *MailService) ListTemplates(ctx context.Context) ([]*domain.MailTemplate, error) {
	items, err := s.repo.ListTemplates(ctx)
	if err != nil {
		return nil, err
	}
	if len(items) == 0 {
		return defaultTemplates, nil
	}
	return items, nil
}

func (s *MailService) UpdateTemplate(ctx context.Context, template *domain.MailTemplate) (*domain.MailTemplate, error) {
	return s.repo.UpsertTemplate(ctx, template)
}

func (s *MailService) templateForJob(ctx context.Context, job *domain.MailJob) *domain.MailTemplate {
	if job == nil {
		return nil
	}
	if strings.TrimSpace(job.TemplateID) != "" {
		if template, err := s.repo.GetTemplate(ctx, job.TemplateID); err == nil {
			return template
		}
		if template, ok := defaultTemplateByID(job.TemplateID); ok {
			return template
		}
	}
	return mailTemplateForCategory(job.Category)
}

func (s *MailService) GetStats(ctx context.Context, from, to time.Time) (*domain.MailStats, error) {
	page := 1
	pageSize := 1000
	items, _, err := s.repo.ListJobs(ctx, page, pageSize, "", "")
	if err != nil {
		return nil, err
	}

	stats := &domain.MailStats{
		ByCategory: make(map[string]int64),
		ByHour:     make(map[string]int64),
		From:       from,
		To:         to,
	}

	for _, job := range items {
		if !job.CreatedAt.IsZero() && job.CreatedAt.Before(from) {
			continue
		}
		if !job.CreatedAt.IsZero() && job.CreatedAt.After(to) {
			continue
		}
		stats.Total++
		stats.ByCategory[job.Category.String()]++
		stats.ByHour[job.CreatedAt.UTC().Format("2006010215")]++
		switch job.Status {
		case domain.StatusQueued:
			stats.Queued++
		case domain.StatusProcessing:
			stats.Processing++
		case domain.StatusSent:
			stats.Sent++
		case domain.StatusFailed, domain.StatusBounced, domain.StatusCancelled:
			stats.Failed++
		}
	}

	return stats, nil
}

func (s *MailService) log(ctx context.Context, level, action, message string, meta map[string]any) error {
	return s.publisher.PublishLog(ctx, domain.LogEvent{
		Service: s.serviceName,
		Action:  action,
		Level:   level,
		Message: message,
		Meta:    meta,
		At:      time.Now().UTC(),
	})
}

func mergedVars(job *domain.MailJob) map[string]string {
	out := make(map[string]string)
	for k, v := range job.Variables {
		out[k] = v
	}
	if len(job.Recipient) > 0 {
		out["Recipient"] = job.Recipient[0]
	}
	out["Title"] = job.Subject
	out["Body"] = job.Body
	return out
}

func stripHTML(raw string) string {
	r := strings.NewReplacer("<br>", "\n", "<br/>", "\n", "<br />", "\n", "</p>", "\n", "</div>", "\n")
	return strings.TrimSpace(r.Replace(raw))
}

func timePtr(t time.Time) *time.Time {
	return &t
}
