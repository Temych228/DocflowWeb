package repository

import (
	"context"
	"time"

	"github.com/Temych228/DocflowWeb/services/mail-service/internal/domain"
)

type MailRepository interface {
	CreateJob(ctx context.Context, job *domain.MailJob) (*domain.MailJob, error)
	GetJobByID(ctx context.Context, id string) (*domain.MailJob, error)
	GetJobByJobID(ctx context.Context, jobID string) (*domain.MailJob, error)
	ListJobs(ctx context.Context, page, pageSize int, status, category string) ([]*domain.MailJob, int, error)
	UpdateJobStatus(ctx context.Context, id string, status domain.Status, lastError string, attempts int32, processedAt, sentAt, failedAt *time.Time) (*domain.MailJob, error)
	GetTemplate(ctx context.Context, templateID string) (*domain.MailTemplate, error)
	ListTemplates(ctx context.Context) ([]*domain.MailTemplate, error)
	UpsertTemplate(ctx context.Context, template *domain.MailTemplate) (*domain.MailTemplate, error)
	SeedDefaultTemplates(ctx context.Context, templates []*domain.MailTemplate) error
	MarkDedup(ctx context.Context, jobID string, ttl time.Duration) (bool, error)
	TrackBucket(ctx context.Context, category string, status domain.Status, occurredAt time.Time) error
}
