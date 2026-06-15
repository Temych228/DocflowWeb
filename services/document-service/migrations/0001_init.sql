\connect doc_db;

CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TYPE doc_status AS ENUM (
    'draft', 'assigned', 'in_progress', 'completed', 'overdue', 'archived'
);

CREATE TYPE doc_type AS ENUM (
    'contract', 'invoice', 'report', 'memo', 'order', 'other'
);

CREATE TABLE IF NOT EXISTS documents (
                                         id             UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    title          TEXT        NOT NULL,
    description    TEXT        NOT NULL DEFAULT '',
    type           doc_type    NOT NULL DEFAULT 'other',
    status         doc_status  NOT NULL DEFAULT 'draft',
    creator_id     UUID        NOT NULL,
    responsible_id UUID        NULL,
    deadline       TIMESTAMPTZ NULL,
    file_url       TEXT        NOT NULL DEFAULT '',
    tags           TEXT[]      NOT NULL DEFAULT '{}',
    is_overdue     BOOLEAN     NOT NULL DEFAULT FALSE,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    archived_at    TIMESTAMPTZ NULL
    );

CREATE TABLE IF NOT EXISTS document_history (
                                                id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    document_id UUID        NOT NULL REFERENCES documents(id) ON DELETE CASCADE,
    changed_by  UUID        NOT NULL,
    field       TEXT        NOT NULL,
    old_value   TEXT        NOT NULL DEFAULT '',
    new_value   TEXT        NOT NULL DEFAULT '',
    changed_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
    );

CREATE INDEX IF NOT EXISTS idx_documents_status       ON documents(status);
CREATE INDEX IF NOT EXISTS idx_documents_creator      ON documents(creator_id);
CREATE INDEX IF NOT EXISTS idx_documents_responsible  ON documents(responsible_id);
CREATE INDEX IF NOT EXISTS idx_documents_deadline     ON documents(deadline);
CREATE INDEX IF NOT EXISTS idx_doc_history_document   ON document_history(document_id);
CREATE INDEX IF NOT EXISTS idx_doc_history_changed_by ON document_history(changed_by);