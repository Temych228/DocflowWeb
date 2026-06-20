\connect user_db;

CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE IF NOT EXISTS users (
                                     id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email TEXT NOT NULL,
    first_name TEXT NOT NULL,
    last_name TEXT NOT NULL DEFAULT '',
    phone TEXT NOT NULL DEFAULT '',
    department TEXT NOT NULL DEFAULT '',
    avatar_url TEXT NOT NULL DEFAULT '',
    role TEXT NOT NULL DEFAULT 'employee',
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    is_verified BOOLEAN NOT NULL DEFAULT FALSE,
    is_banned BOOLEAN NOT NULL DEFAULT FALSE,
    ban_reason TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ NULL,
    CONSTRAINT users_role_check CHECK (role IN ('employee', 'manager', 'admin'))
    );

CREATE UNIQUE INDEX IF NOT EXISTS users_email_active_idx
    ON users (email)
    WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS users_role_idx ON users (role);
CREATE INDEX IF NOT EXISTS users_deleted_at_idx ON users (deleted_at);

CREATE TABLE IF NOT EXISTS user_stats (
                                          user_id UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    total_docs INT NOT NULL DEFAULT 0,
    completed_docs INT NOT NULL DEFAULT 0,
    overdue_docs INT NOT NULL DEFAULT 0,
    total_tasks INT NOT NULL DEFAULT 0,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
    );