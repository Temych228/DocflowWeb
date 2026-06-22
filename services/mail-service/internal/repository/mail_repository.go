package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/Temych228/DocflowWeb/services/mail-service/internal/cache"
	"github.com/Temych228/DocflowWeb/services/mail-service/internal/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

type postgresRepo struct {
	db       *pgxpool.Pool
	cache    *redis.Client
	cacheTTL time.Duration
	dedupTTL time.Duration
}

func New(db *pgxpool.Pool, cacheClient *redis.Client, cacheTTL, dedupTTL time.Duration) MailRepository {
	return &postgresRepo{
		db:       db,
		cache:    cacheClient,
		cacheTTL: cacheTTL,
		dedupTTL: dedupTTL,
	}
}

func (r *postgresRepo) CreateJob(ctx context.Context, job *domain.MailJob) (*domain.MailJob, error) {
	if job == nil {
		return nil, domain.ErrInvalidInput
	}
	if job.JobID == "" {
		job.JobID = uuid.NewString()
	}
	if len(job.Recipient) == 0 {
		return nil, domain.ErrInvalidInput
	}
	if job.Status == "" {
		job.Status = domain.StatusQueued
	}
	if job.MaxAttempts <= 0 {
		job.MaxAttempts = 3
	}
	if job.Category == "" {
		job.Category = domain.CategorySystem
	}
	var varsJSON []byte
	if job.Variables != nil {
		var err error
		varsJSON, err = json.Marshal(job.Variables)
		if err != nil {
			return nil, err
		}
	} else {
		varsJSON = []byte(`{}`)
	}

	row := r.db.QueryRow(ctx, `
		INSERT INTO mail_jobs (
			job_id, notification_id, user_id, recipient, template_id, subject, body, variables,
			category, status, attempts, max_attempts, last_error, queued_at, created_at, updated_at
		)
		VALUES (
			$1, NULLIF($2, '')::uuid, NULLIF($3, '')::uuid, $4, $5, $6, $7, $8::jsonb,
			$9, $10, $11, $12, $13, NOW(), NOW(), NOW()
		)
		RETURNING id, job_id, notification_id, user_id, recipient, template_id, subject, body, variables,
		          category, status, attempts, max_attempts, last_error, queued_at, processed_at, sent_at, failed_at, created_at, updated_at
	`, job.JobID, job.NotificationID, job.UserID, job.Recipient, job.TemplateID, job.Subject, job.Body, varsJSON, job.Category.String(), job.Status.String(), job.Attempts, job.MaxAttempts, job.LastError)

	return scanJob(row)
}

func (r *postgresRepo) GetJobByID(ctx context.Context, id string) (*domain.MailJob, error) {
	row := r.db.QueryRow(ctx, `
		SELECT id, job_id, notification_id, user_id, recipient, template_id, subject, body, variables,
		       category, status, attempts, max_attempts, last_error, queued_at, processed_at, sent_at, failed_at, created_at, updated_at
		FROM mail_jobs
		WHERE id = $1
	`, strings.TrimSpace(id))
	job, err := scanJob(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return job, nil
}

func (r *postgresRepo) GetJobByJobID(ctx context.Context, jobID string) (*domain.MailJob, error) {
	row := r.db.QueryRow(ctx, `
		SELECT id, job_id, notification_id, user_id, recipient, template_id, subject, body, variables,
		       category, status, attempts, max_attempts, last_error, queued_at, processed_at, sent_at, failed_at, created_at, updated_at
		FROM mail_jobs
		WHERE job_id = $1
	`, strings.TrimSpace(jobID))
	job, err := scanJob(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return job, nil
}

func (r *postgresRepo) ListJobs(ctx context.Context, page, pageSize int, status, category string) ([]*domain.MailJob, int, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize

	where := `WHERE 1=1`
	args := []any{}
	if s := strings.TrimSpace(status); s != "" {
		args = append(args, s)
		where += fmt.Sprintf(` AND status = $%d`, len(args))
	}
	if c := strings.TrimSpace(category); c != "" {
		args = append(args, c)
		where += fmt.Sprintf(` AND category = $%d`, len(args))
	}

	var total int
	countSQL := `SELECT COUNT(*) FROM mail_jobs ` + where
	if err := r.db.QueryRow(ctx, countSQL, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	args = append(args, pageSize, offset)
	listSQL := `
		SELECT id, job_id, notification_id, user_id, recipient, template_id, subject, body, variables,
		       category, status, attempts, max_attempts, last_error, queued_at, processed_at, sent_at, failed_at, created_at, updated_at
		FROM mail_jobs
		` + where + `
		ORDER BY queued_at DESC
		LIMIT $` + fmt.Sprintf("%d", len(args)-1) + ` OFFSET $` + fmt.Sprintf("%d", len(args))

	rows, err := r.db.Query(ctx, listSQL, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	items := make([]*domain.MailJob, 0)
	for rows.Next() {
		job, err := scanJob(rows)
		if err != nil {
			return nil, 0, err
		}
		items = append(items, job)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

func (r *postgresRepo) UpdateJobStatus(ctx context.Context, id string, status domain.Status, lastError string, attempts int32, processedAt, sentAt, failedAt *time.Time) (*domain.MailJob, error) {
	row := r.db.QueryRow(ctx, `
		UPDATE mail_jobs
		SET status = $2,
		    last_error = $3,
		    attempts = $4,
		    processed_at = COALESCE($5, processed_at),
		    sent_at = COALESCE($6, sent_at),
		    failed_at = COALESCE($7, failed_at),
		    updated_at = NOW()
		WHERE id = $1
		RETURNING id, job_id, notification_id, user_id, recipient, template_id, subject, body, variables,
		          category, status, attempts, max_attempts, last_error, queued_at, processed_at, sent_at, failed_at, created_at, updated_at
	`, strings.TrimSpace(id), status.String(), lastError, attempts, processedAt, sentAt, failedAt)

	job, err := scanJob(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return job, nil
}

func (r *postgresRepo) GetTemplate(ctx context.Context, templateID string) (*domain.MailTemplate, error) {
	row := r.db.QueryRow(ctx, `
		SELECT template_id, subject, body_template, channel, is_active, created_at, updated_at
		FROM mail_templates
		WHERE template_id = $1
	`, strings.TrimSpace(templateID))
	var t domain.MailTemplate
	if err := row.Scan(&t.TemplateID, &t.Subject, &t.BodyTemplate, &t.Channel, &t.IsActive, &t.CreatedAt, &t.UpdatedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return &t, nil
}

func (r *postgresRepo) ListTemplates(ctx context.Context) ([]*domain.MailTemplate, error) {
	rows, err := r.db.Query(ctx, `
		SELECT template_id, subject, body_template, channel, is_active, created_at, updated_at
		FROM mail_templates
		ORDER BY template_id
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]*domain.MailTemplate, 0)
	for rows.Next() {
		var t domain.MailTemplate
		if err := rows.Scan(&t.TemplateID, &t.Subject, &t.BodyTemplate, &t.Channel, &t.IsActive, &t.CreatedAt, &t.UpdatedAt); err != nil {
			return nil, err
		}
		items = append(items, &t)
	}
	return items, rows.Err()
}

func (r *postgresRepo) UpsertTemplate(ctx context.Context, template *domain.MailTemplate) (*domain.MailTemplate, error) {
	if template == nil {
		return nil, domain.ErrInvalidInput
	}
	if strings.TrimSpace(template.TemplateID) == "" {
		return nil, domain.ErrInvalidInput
	}
	row := r.db.QueryRow(ctx, `
		INSERT INTO mail_templates (template_id, subject, body_template, channel, is_active, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, NOW(), NOW())
		ON CONFLICT (template_id) DO UPDATE SET
			subject = EXCLUDED.subject,
			body_template = EXCLUDED.body_template,
			channel = EXCLUDED.channel,
			is_active = EXCLUDED.is_active,
			updated_at = NOW()
		RETURNING template_id, subject, body_template, channel, is_active, created_at, updated_at
	`, template.TemplateID, template.Subject, template.BodyTemplate, template.Channel, template.IsActive)
	var out domain.MailTemplate
	if err := row.Scan(&out.TemplateID, &out.Subject, &out.BodyTemplate, &out.Channel, &out.IsActive, &out.CreatedAt, &out.UpdatedAt); err != nil {
		return nil, err
	}
	return &out, nil
}

func (r *postgresRepo) SeedDefaultTemplates(ctx context.Context, templates []*domain.MailTemplate) error {
	for _, t := range templates {
		if _, err := r.UpsertTemplate(ctx, t); err != nil {
			return err
		}
	}
	return nil
}

func (r *postgresRepo) MarkDedup(ctx context.Context, jobID string, ttl time.Duration) (bool, error) {
	if r.cache == nil {
		return true, nil
	}
	if ttl <= 0 {
		ttl = r.dedupTTL
	}
	return r.cache.SetNX(ctx, cache.DedupKey(jobID), "1", ttl).Result()
}

func (r *postgresRepo) TrackBucket(ctx context.Context, category string, status domain.Status, occurredAt time.Time) error {
	if r.cache == nil {
		return nil
	}
	hour := occurredAt.UTC().Format("2006010215")
	keys := []string{
		cache.StatsHourKey(hour),
		cache.StatsCategoryHourKey(category, hour),
		cache.StatsStatusHourKey(status.String(), hour),
	}
	for _, key := range keys {
		if err := r.cache.Incr(ctx, key).Err(); err != nil {
			return err
		}
		_ = r.cache.Expire(ctx, key, r.cacheTTL).Err()
	}
	return nil
}

func scanJob(row interface{ Scan(dest ...any) error }) (*domain.MailJob, error) {
	var job domain.MailJob
	var recipient []string
	var varsRaw []byte
	var notificationID pgtype.UUID
	var userID pgtype.UUID
	var processedAt pgtype.Timestamptz
	var sentAt pgtype.Timestamptz
	var failedAt pgtype.Timestamptz
	if err := row.Scan(
		&job.ID,
		&job.JobID,
		&notificationID,
		&userID,
		&recipient,
		&job.TemplateID,
		&job.Subject,
		&job.Body,
		&varsRaw,
		&job.Category,
		&job.Status,
		&job.Attempts,
		&job.MaxAttempts,
		&job.LastError,
		&job.QueuedAt,
		&processedAt,
		&sentAt,
		&failedAt,
		&job.CreatedAt,
		&job.UpdatedAt,
	); err != nil {
		return nil, err
	}
	job.Recipient = recipient
	if notificationID.Valid {
		job.NotificationID = uuid.UUID(notificationID.Bytes).String()
	}
	if userID.Valid {
		job.UserID = uuid.UUID(userID.Bytes).String()
	}
	if len(varsRaw) > 0 {
		_ = json.Unmarshal(varsRaw, &job.Variables)
	}
	if processedAt.Valid {
		t := processedAt.Time
		job.ProcessedAt = &t
	}
	if sentAt.Valid {
		t := sentAt.Time
		job.SentAt = &t
	}
	if failedAt.Valid {
		t := failedAt.Time
		job.FailedAt = &t
	}
	return &job, nil
}
