package repository

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/Temych228/DocflowWeb/services/notification-service/internal/cache"
	"github.com/Temych228/DocflowWeb/services/notification-service/internal/domain"
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

func New(db *pgxpool.Pool, cacheClient *redis.Client, cacheTTL, dedupTTL time.Duration) NotificationRepository {
	return &postgresRepo{
		db:       db,
		cache:    cacheClient,
		cacheTTL: cacheTTL,
		dedupTTL: dedupTTL,
	}
}

func (r *postgresRepo) CreateNotification(ctx context.Context, n *domain.Notification) (*domain.Notification, error) {
	if n == nil {
		return nil, domain.ErrInvalidInput
	}

	row := r.db.QueryRow(ctx, `
		INSERT INTO notifications (
			user_id, document_id, task_id, notif_category, title, body,
			ref_id, ref_type, proto_type, is_read, sent_email, created_at, updated_at
		)
		VALUES (
			$1, NULLIF($2, '')::uuid, NULLIF($3, '')::uuid, $4, $5, $6,
			$7, $8, $9, $10, $11, NOW(), NOW()
		)
		RETURNING id, user_id, document_id, task_id, notif_category, title, body,
		          ref_id, ref_type, proto_type, is_read, sent_email, created_at, updated_at, read_at, deleted_at
	`, n.UserID, n.DocumentID, n.TaskID, n.Category.String(), n.Title, n.Body, n.RefID, n.RefType, n.ProtoType, n.IsRead, n.SentEmail)

	return scanNotification(row)
}

func (r *postgresRepo) SetEmailQueued(ctx context.Context, notificationID string) error {
	tag, err := r.db.Exec(ctx, `
		UPDATE notifications
		SET sent_email = TRUE, updated_at = NOW()
		WHERE id = $1 AND deleted_at IS NULL
	`, strings.TrimSpace(notificationID))
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *postgresRepo) GetHistory(ctx context.Context, userID string, page, pageSize int) ([]*domain.Notification, int, error) {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return nil, 0, domain.ErrInvalidInput
	}
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize

	var total int
	if err := r.db.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM notifications
		WHERE user_id = $1 AND deleted_at IS NULL
	`, userID).Scan(&total); err != nil {
		return nil, 0, err
	}

	rows, err := r.db.Query(ctx, `
		SELECT id, user_id, document_id, task_id, notif_category, title, body,
		       ref_id, ref_type, proto_type, is_read, sent_email, created_at, updated_at, read_at, deleted_at
		FROM notifications
		WHERE user_id = $1 AND deleted_at IS NULL
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`, userID, pageSize, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	items := make([]*domain.Notification, 0)
	for rows.Next() {
		n, err := scanNotification(rows)
		if err != nil {
			return nil, 0, err
		}
		items = append(items, n)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}

	return items, total, nil
}

func (r *postgresRepo) MarkRead(ctx context.Context, notificationID, userID string) (*domain.Notification, error) {
	row := r.db.QueryRow(ctx, `
		UPDATE notifications
		SET is_read = TRUE, read_at = COALESCE(read_at, NOW()), updated_at = NOW()
		WHERE id = $1 AND user_id = $2 AND deleted_at IS NULL
		RETURNING id, user_id, document_id, task_id, notif_category, title, body,
		          ref_id, ref_type, proto_type, is_read, sent_email, created_at, updated_at, read_at, deleted_at
	`, strings.TrimSpace(notificationID), strings.TrimSpace(userID))

	n, err := scanNotification(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}

	_ = r.InvalidateUnreadCountCache(ctx, userID)
	return n, nil
}

func (r *postgresRepo) MarkAllRead(ctx context.Context, userID string) (int32, error) {
	tag, err := r.db.Exec(ctx, `
		UPDATE notifications
		SET is_read = TRUE, read_at = COALESCE(read_at, NOW()), updated_at = NOW()
		WHERE user_id = $1 AND is_read = FALSE AND deleted_at IS NULL
	`, strings.TrimSpace(userID))
	if err != nil {
		return 0, err
	}
	_ = r.InvalidateUnreadCountCache(ctx, userID)
	return int32(tag.RowsAffected()), nil
}

func (r *postgresRepo) Delete(ctx context.Context, notificationID, userID string) error {
	tag, err := r.db.Exec(ctx, `
		UPDATE notifications
		SET deleted_at = NOW(), updated_at = NOW()
		WHERE id = $1 AND user_id = $2 AND deleted_at IS NULL
	`, strings.TrimSpace(notificationID), strings.TrimSpace(userID))
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	_ = r.InvalidateUnreadCountCache(ctx, userID)
	return nil
}

func (r *postgresRepo) GetUnreadCount(ctx context.Context, userID string) (int32, error) {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return 0, domain.ErrInvalidInput
	}

	if r.cache != nil {
		if val, err := r.cache.Get(ctx, cache.UnreadKey(userID)).Int64(); err == nil {
			return int32(val), nil
		}
	}

	var count int32
	if err := r.db.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM notifications
		WHERE user_id = $1 AND is_read = FALSE AND deleted_at IS NULL
	`, userID).Scan(&count); err != nil {
		return 0, err
	}

	_ = r.SetUnreadCountCache(ctx, userID, count, r.cacheTTL)
	return count, nil
}

func (r *postgresRepo) SetUnreadCountCache(ctx context.Context, userID string, count int32, ttl time.Duration) error {
	if r.cache == nil {
		return nil
	}
	if ttl <= 0 {
		ttl = r.cacheTTL
	}
	return r.cache.Set(ctx, cache.UnreadKey(userID), count, ttl).Err()
}

func (r *postgresRepo) InvalidateUnreadCountCache(ctx context.Context, userID string) error {
	if r.cache == nil {
		return nil
	}
	return r.cache.Del(ctx, cache.UnreadKey(userID)).Err()
}

func (r *postgresRepo) GetPreferences(ctx context.Context, userID string) (*domain.Preferences, error) {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return nil, domain.ErrInvalidInput
	}

	if r.cache != nil {
		if data, err := r.cache.Get(ctx, cache.PrefsKey(userID)).Bytes(); err == nil {
			var p domain.Preferences
			if json.Unmarshal(data, &p) == nil {
				return &p, nil
			}
		}
	}

	row := r.db.QueryRow(ctx, `
		SELECT user_id, email_enabled, push_enabled, deadline_notif, assigned_notif, status_notif, overdue_notif, updated_at
		FROM notification_preferences
		WHERE user_id = $1
	`, userID)

	var p domain.Preferences
	if err := row.Scan(&p.UserID, &p.EmailEnabled, &p.PushEnabled, &p.DeadlineNotif, &p.AssignedNotif, &p.StatusNotif, &p.OverdueNotif, &p.UpdatedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			p = defaultPreferences(userID)
			_, _ = r.UpsertPreferences(ctx, &p)
			return &p, nil
		}
		return nil, err
	}

	_ = r.setPrefsCache(ctx, &p)
	return &p, nil
}

func (r *postgresRepo) UpsertPreferences(ctx context.Context, p *domain.Preferences) (*domain.Preferences, error) {
	if p == nil {
		return nil, domain.ErrInvalidInput
	}
	p.UserID = strings.TrimSpace(p.UserID)
	if p.UserID == "" {
		return nil, domain.ErrInvalidInput
	}

	row := r.db.QueryRow(ctx, `
		INSERT INTO notification_preferences (
			user_id, email_enabled, push_enabled, deadline_notif, assigned_notif, status_notif, overdue_notif, updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, NOW())
		ON CONFLICT (user_id) DO UPDATE SET
			email_enabled = EXCLUDED.email_enabled,
			push_enabled = EXCLUDED.push_enabled,
			deadline_notif = EXCLUDED.deadline_notif,
			assigned_notif = EXCLUDED.assigned_notif,
			status_notif = EXCLUDED.status_notif,
			overdue_notif = EXCLUDED.overdue_notif,
			updated_at = NOW()
		RETURNING user_id, email_enabled, push_enabled, deadline_notif, assigned_notif, status_notif, overdue_notif, updated_at
	`, p.UserID, p.EmailEnabled, p.PushEnabled, p.DeadlineNotif, p.AssignedNotif, p.StatusNotif, p.OverdueNotif)

	var out domain.Preferences
	if err := row.Scan(&out.UserID, &out.EmailEnabled, &out.PushEnabled, &out.DeadlineNotif, &out.AssignedNotif, &out.StatusNotif, &out.OverdueNotif, &out.UpdatedAt); err != nil {
		return nil, err
	}

	_ = r.setPrefsCache(ctx, &out)
	return &out, nil
}

func (r *postgresRepo) GetTemplate(ctx context.Context, templateID string) (*domain.Template, error) {
	templateID = strings.TrimSpace(templateID)
	if templateID == "" {
		return nil, domain.ErrInvalidInput
	}

	row := r.db.QueryRow(ctx, `
		SELECT template_id, subject, body_template, is_active, created_at, updated_at
		FROM notification_templates
		WHERE template_id = $1
	`, templateID)

	var t domain.Template
	if err := row.Scan(&t.TemplateID, &t.Subject, &t.BodyTemplate, &t.IsActive, &t.CreatedAt, &t.UpdatedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return &t, nil
}

func (r *postgresRepo) UpsertTemplate(ctx context.Context, t *domain.Template) (*domain.Template, error) {
	if t == nil {
		return nil, domain.ErrInvalidInput
	}
	t.TemplateID = strings.TrimSpace(t.TemplateID)
	if t.TemplateID == "" {
		return nil, domain.ErrInvalidInput
	}

	row := r.db.QueryRow(ctx, `
		INSERT INTO notification_templates (
			template_id, subject, body_template, is_active, created_at, updated_at
		)
		VALUES ($1, $2, $3, $4, NOW(), NOW())
		ON CONFLICT (template_id) DO UPDATE SET
			subject = EXCLUDED.subject,
			body_template = EXCLUDED.body_template,
			is_active = EXCLUDED.is_active,
			updated_at = NOW()
		RETURNING template_id, subject, body_template, is_active, created_at, updated_at
	`, t.TemplateID, t.Subject, t.BodyTemplate, t.IsActive)

	var out domain.Template
	if err := row.Scan(&out.TemplateID, &out.Subject, &out.BodyTemplate, &out.IsActive, &out.CreatedAt, &out.UpdatedAt); err != nil {
		return nil, err
	}
	return &out, nil
}

func (r *postgresRepo) MarkEventProcessed(ctx context.Context, eventKey string, ttl time.Duration) (bool, error) {
	eventKey = strings.TrimSpace(eventKey)
	if eventKey == "" {
		return false, domain.ErrInvalidInput
	}
	if r.cache == nil {
		return true, nil
	}
	if ttl <= 0 {
		ttl = r.dedupTTL
	}
	return r.cache.SetNX(ctx, cache.DedupKey(eventKey), "1", ttl).Result()
}

func (r *postgresRepo) SeedDefaultTemplates(ctx context.Context, templates []*domain.Template) error {
	for _, t := range templates {
		if _, err := r.UpsertTemplate(ctx, t); err != nil {
			return err
		}
	}
	return nil
}

func (r *postgresRepo) setPrefsCache(ctx context.Context, p *domain.Preferences) error {
	if r.cache == nil || p == nil {
		return nil
	}
	data, err := json.Marshal(p)
	if err != nil {
		return err
	}
	return r.cache.Set(ctx, cache.PrefsKey(p.UserID), data, r.cacheTTL).Err()
}

func scanNotification(row interface{ Scan(dest ...any) error }) (*domain.Notification, error) {
	var n domain.Notification
	var documentID pgtype.UUID
	var taskID pgtype.UUID
	var readAt pgtype.Timestamptz
	var deletedAt pgtype.Timestamptz

	if err := row.Scan(
		&n.ID,
		&n.UserID,
		&documentID,
		&taskID,
		&n.Category,
		&n.Title,
		&n.Body,
		&n.RefID,
		&n.RefType,
		&n.ProtoType,
		&n.IsRead,
		&n.SentEmail,
		&n.CreatedAt,
		&n.UpdatedAt,
		&readAt,
		&deletedAt,
	); err != nil {
		return nil, err
	}

	if documentID.Valid {
		n.DocumentID = uuid.UUID(documentID.Bytes).String()
	}
	if taskID.Valid {
		n.TaskID = uuid.UUID(taskID.Bytes).String()
	}
	if readAt.Valid {
		t := readAt.Time
		n.ReadAt = &t
	}
	if deletedAt.Valid {
		t := deletedAt.Time
		n.DeletedAt = &t
	}

	return &n, nil
}

func defaultPreferences(userID string) domain.Preferences {
	return domain.Preferences{
		UserID:        userID,
		EmailEnabled:  true,
		PushEnabled:   true,
		DeadlineNotif: true,
		AssignedNotif: true,
		StatusNotif:   true,
		OverdueNotif:  true,
		UpdatedAt:     time.Now().UTC(),
	}
}
