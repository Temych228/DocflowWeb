\connect mail_db;

CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE IF NOT EXISTS mail_jobs (
                                         id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    job_id TEXT NOT NULL UNIQUE,
    notification_id UUID NULL,
    user_id UUID NULL,
    recipient TEXT[] NOT NULL,
    template_id TEXT NOT NULL DEFAULT '',
    subject TEXT NOT NULL,
    body TEXT NOT NULL,
    variables JSONB NOT NULL DEFAULT '{}'::jsonb,
    category TEXT NOT NULL DEFAULT 'system',
    status TEXT NOT NULL DEFAULT 'queued',
    attempts INT NOT NULL DEFAULT 0,
    max_attempts INT NOT NULL DEFAULT 3,
    last_error TEXT NOT NULL DEFAULT '',
    queued_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    processed_at TIMESTAMPTZ NULL,
    sent_at TIMESTAMPTZ NULL,
    failed_at TIMESTAMPTZ NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT mail_jobs_status_check CHECK (status IN ('queued', 'processing', 'sent', 'failed', 'bounced', 'cancelled'))
    );

CREATE INDEX IF NOT EXISTS idx_mail_jobs_status_created
    ON mail_jobs (status, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_mail_jobs_category_created
    ON mail_jobs (category, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_mail_jobs_template_created
    ON mail_jobs (template_id, created_at DESC);

CREATE TABLE IF NOT EXISTS mail_templates (
                                              template_id TEXT PRIMARY KEY,
                                              subject TEXT NOT NULL,
                                              body_template TEXT NOT NULL,
                                              channel TEXT NOT NULL DEFAULT 'email',
                                              is_active BOOLEAN NOT NULL DEFAULT TRUE,
                                              created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
    );