\connect task_db;

CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TYPE task_status AS ENUM (
    'pending', 'in_progress', 'completed', 'overdue', 'cancelled'
);

CREATE TYPE task_priority AS ENUM (
    'low', 'medium', 'high', 'critical'
);

CREATE TABLE IF NOT EXISTS tasks (
                                     id           UUID          PRIMARY KEY DEFAULT gen_random_uuid(),
    document_id  UUID          NOT NULL,
    title        TEXT          NOT NULL,
    description  TEXT          NOT NULL DEFAULT '',
    status       task_status   NOT NULL DEFAULT 'pending',
    priority     task_priority NOT NULL DEFAULT 'medium',
    creator_id   UUID          NOT NULL,
    assignee_id  UUID          NULL,
    deadline     TIMESTAMPTZ   NULL,
    is_overdue   BOOLEAN       NOT NULL DEFAULT FALSE,
    created_at   TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    completed_at TIMESTAMPTZ   NULL
    );

CREATE TABLE IF NOT EXISTS task_history (
                                            id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    task_id     UUID        NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
    actor_id    UUID        NOT NULL,
    action      TEXT        NOT NULL,
    old_status  task_status NULL,
    new_status  task_status NULL,
    comment     TEXT        NOT NULL DEFAULT '',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
    );

CREATE INDEX IF NOT EXISTS idx_tasks_document    ON tasks(document_id);
CREATE INDEX IF NOT EXISTS idx_tasks_assignee    ON tasks(assignee_id);
CREATE INDEX IF NOT EXISTS idx_tasks_status      ON tasks(status);
CREATE INDEX IF NOT EXISTS idx_tasks_deadline    ON tasks(deadline);
CREATE INDEX IF NOT EXISTS idx_tasks_is_overdue  ON tasks(is_overdue);
CREATE INDEX IF NOT EXISTS idx_task_history_task ON task_history(task_id);

CREATE OR REPLACE FUNCTION set_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_tasks_updated_at ON tasks;
CREATE TRIGGER trg_tasks_updated_at
    BEFORE UPDATE ON tasks
    FOR EACH ROW
    EXECUTE FUNCTION set_updated_at();