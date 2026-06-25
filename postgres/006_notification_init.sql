\connect notif_db;

CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE IF NOT EXISTS notifications (
                                             id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL,
    document_id UUID NULL,
    task_id UUID NULL,
    notif_category TEXT NOT NULL,
    title TEXT NOT NULL,
    body TEXT NOT NULL,
    ref_id TEXT NOT NULL DEFAULT '',
    ref_type TEXT NOT NULL DEFAULT '',
    proto_type INT NOT NULL DEFAULT 0,
    is_read BOOLEAN NOT NULL DEFAULT FALSE,
    sent_email BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    read_at TIMESTAMPTZ NULL,
    deleted_at TIMESTAMPTZ NULL
    );

CREATE INDEX IF NOT EXISTS idx_notifications_user_created
    ON notifications (user_id, created_at DESC)
    WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_notifications_user_unread
    ON notifications (user_id, is_read)
    WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_notifications_ref
    ON notifications (ref_type, ref_id)
    WHERE deleted_at IS NULL;

CREATE TABLE IF NOT EXISTS notification_preferences (
                                                        user_id UUID PRIMARY KEY,
                                                        email_enabled BOOLEAN NOT NULL DEFAULT TRUE,
                                                        push_enabled BOOLEAN NOT NULL DEFAULT TRUE,
                                                        deadline_notif BOOLEAN NOT NULL DEFAULT TRUE,
                                                        assigned_notif BOOLEAN NOT NULL DEFAULT TRUE,
                                                        status_notif BOOLEAN NOT NULL DEFAULT TRUE,
                                                        overdue_notif BOOLEAN NOT NULL DEFAULT TRUE,
                                                        updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
    );

CREATE TABLE IF NOT EXISTS notification_templates (
                                                      template_id TEXT PRIMARY KEY,
                                                      subject TEXT NOT NULL,
                                                      body_template TEXT NOT NULL,
                                                      is_active BOOLEAN NOT NULL DEFAULT TRUE,
                                                      created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
    );