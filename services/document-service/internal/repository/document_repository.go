package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Temych228/DocflowWeb/services/document-service/internal/domain"
)

type postgresDocumentRepository struct {
	db *pgxpool.Pool
}

func NewDocumentRepository(db *pgxpool.Pool) *postgresDocumentRepository {
	return &postgresDocumentRepository{db: db}
}

func (r *postgresDocumentRepository) Create(ctx context.Context, doc *domain.Document) error {
	row := r.db.QueryRow(ctx, `
		INSERT INTO documents (title, description, type, status, creator_id, responsible_id,
			deadline, file_url, tags)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id, status, is_overdue, created_at, updated_at
	`, doc.Title, doc.Description, doc.Type, doc.Status, doc.CreatorID, doc.ResponsibleID,
		doc.Deadline, doc.FileURL, doc.Tags)

	err := row.Scan(&doc.ID, &doc.Status, &doc.IsOverdue, &doc.CreatedAt, &doc.UpdatedAt)
	if err != nil {
		return fmt.Errorf("create document: %w", err)
	}
	return nil
}

func (r *postgresDocumentRepository) GetByID(ctx context.Context, id string) (*domain.Document, error) {
	row := r.db.QueryRow(ctx, `
		SELECT id, title, description, type, status, creator_id, responsible_id,
			deadline, file_url, tags, is_overdue, created_at, updated_at, archived_at
		FROM documents
		WHERE id = $1
	`, id)

	doc, err := scanDocument(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrDocumentNotFound
		}
		return nil, fmt.Errorf("get document by id: %w", err)
	}
	return doc, nil
}

func (r *postgresDocumentRepository) Update(ctx context.Context, doc *domain.Document) error {
	row := r.db.QueryRow(ctx, `
		UPDATE documents
		SET title = $1, description = $2, type = $3, deadline = $4, updated_at = NOW()
		WHERE id = $5
		RETURNING updated_at
	`, doc.Title, doc.Description, doc.Type, doc.Deadline, doc.ID)

	err := row.Scan(&doc.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.ErrDocumentNotFound
		}
		return fmt.Errorf("update document: %w", err)
	}
	return nil
}

func (r *postgresDocumentRepository) Delete(ctx context.Context, id string) error {
	tag, err := r.db.Exec(ctx, `DELETE FROM documents WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete document: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrDocumentNotFound
	}
	return nil
}

func (r *postgresDocumentRepository) List(ctx context.Context, filter domain.ListFilter) ([]*domain.Document, int, error) {
	var total int
	if err := r.db.QueryRow(ctx, `SELECT COUNT(*) FROM documents`).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count documents: %w", err)
	}

	rows, err := r.db.Query(ctx, `
		SELECT id, title, description, type, status, creator_id, responsible_id,
			deadline, file_url, tags, is_overdue, created_at, updated_at, archived_at
		FROM documents
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2
	`, filter.PageSize, filter.Offset())
	if err != nil {
		return nil, 0, fmt.Errorf("list documents: %w", err)
	}
	defer rows.Close()

	docs, err := scanDocuments(rows)
	if err != nil {
		return nil, 0, err
	}
	return docs, total, nil
}

func (r *postgresDocumentRepository) Filter(ctx context.Context, filter domain.AdvancedFilter) ([]*domain.Document, int, error) {
	var (
		conditions []string
		args       []any
		idx        = 1
	)

	if filter.Status != "" {
		conditions = append(conditions, fmt.Sprintf("status = $%d", idx))
		args = append(args, string(filter.Status))
		idx++
	}
	if filter.Type != "" {
		conditions = append(conditions, fmt.Sprintf("type = $%d", idx))
		args = append(args, string(filter.Type))
		idx++
	}
	if filter.ResponsibleID != "" {
		conditions = append(conditions, fmt.Sprintf("responsible_id = $%d", idx))
		args = append(args, filter.ResponsibleID)
		idx++
	}
	if filter.CreatorID != "" {
		conditions = append(conditions, fmt.Sprintf("creator_id = $%d", idx))
		args = append(args, filter.CreatorID)
		idx++
	}
	if filter.DeadlineFrom != nil {
		conditions = append(conditions, fmt.Sprintf("deadline >= $%d", idx))
		args = append(args, *filter.DeadlineFrom)
		idx++
	}
	if filter.DeadlineTo != nil {
		conditions = append(conditions, fmt.Sprintf("deadline <= $%d", idx))
		args = append(args, *filter.DeadlineTo)
		idx++
	}

	where := ""
	if len(conditions) > 0 {
		where = "WHERE " + strings.Join(conditions, " AND ")
	}

	var total int
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM documents %s", where)
	if err := r.db.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count filtered documents: %w", err)
	}

	args = append(args, filter.PageSize, filter.Offset())
	query := fmt.Sprintf(`
		SELECT id, title, description, type, status, creator_id, responsible_id,
			deadline, file_url, tags, is_overdue, created_at, updated_at, archived_at
		FROM documents
		%s
		ORDER BY created_at DESC
		LIMIT $%d OFFSET $%d
	`, where, idx, idx+1)

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("filter documents: %w", err)
	}
	defer rows.Close()

	docs, err := scanDocuments(rows)
	if err != nil {
		return nil, 0, err
	}
	return docs, total, nil
}

func (r *postgresDocumentRepository) UpdateStatus(ctx context.Context, id string, status domain.DocumentStatus) error {
	tag, err := r.db.Exec(ctx, `UPDATE documents SET status = $1, updated_at = NOW() WHERE id = $2`, status, id)
	if err != nil {
		return fmt.Errorf("update document status: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrDocumentNotFound
	}
	return nil
}

func (r *postgresDocumentRepository) SetResponsible(ctx context.Context, id string, responsibleID string) error {
	tag, err := r.db.Exec(ctx, `UPDATE documents SET responsible_id = $1, updated_at = NOW() WHERE id = $2`, responsibleID, id)
	if err != nil {
		return fmt.Errorf("set responsible: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrDocumentNotFound
	}
	return nil
}

func (r *postgresDocumentRepository) Archive(ctx context.Context, id string) error {
	tag, err := r.db.Exec(ctx, `
		UPDATE documents
		SET status = $1, archived_at = NOW(), updated_at = NOW()
		WHERE id = $2
	`, domain.StatusArchived, id)
	if err != nil {
		return fmt.Errorf("archive document: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrDocumentNotFound
	}
	return nil
}

func (r *postgresDocumentRepository) FindOverdue(ctx context.Context) ([]*domain.Document, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, title, description, type, status, creator_id, responsible_id,
			deadline, file_url, tags, is_overdue, created_at, updated_at, archived_at
		FROM documents
		WHERE deadline IS NOT NULL
			AND deadline < NOW()
			AND status NOT IN ('completed', 'overdue', 'archived')
	`)
	if err != nil {
		return nil, fmt.Errorf("find overdue documents: %w", err)
	}
	defer rows.Close()

	return scanDocuments(rows)
}

func (r *postgresDocumentRepository) MarkOverdueByIDs(ctx context.Context, ids []string) error {
	if len(ids) == 0 {
		return nil
	}
	_, err := r.db.Exec(ctx, `
		UPDATE documents
		SET status = $1, is_overdue = TRUE, updated_at = NOW()
		WHERE id = ANY($2)
	`, domain.StatusOverdue, ids)
	if err != nil {
		return fmt.Errorf("mark overdue: %w", err)
	}
	return nil
}

func (r *postgresDocumentRepository) Stats(ctx context.Context) (*domain.DocumentStats, error) {
	stats := &domain.DocumentStats{}
	err := r.db.QueryRow(ctx, `
		SELECT
			COUNT(*) AS total,
			COUNT(*) FILTER (WHERE status = 'draft')       AS draft,
			COUNT(*) FILTER (WHERE status = 'assigned')    AS assigned,
			COUNT(*) FILTER (WHERE status = 'in_progress') AS in_progress,
			COUNT(*) FILTER (WHERE status = 'completed')   AS completed,
			COUNT(*) FILTER (WHERE status = 'overdue')     AS overdue,
			COUNT(*) FILTER (WHERE status = 'archived')    AS archived
		FROM documents
	`).Scan(
		&stats.Total, &stats.Draft, &stats.Assigned,
		&stats.InProgress, &stats.Completed, &stats.Overdue, &stats.Archived,
	)
	if err != nil {
		return nil, fmt.Errorf("get document stats: %w", err)
	}
	return stats, nil
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanDocument(row rowScanner) (*domain.Document, error) {
	var (
		doc           domain.Document
		responsibleID *string
		deadline      *time.Time
		archivedAt    *time.Time
	)

	err := row.Scan(
		&doc.ID, &doc.Title, &doc.Description, &doc.Type, &doc.Status,
		&doc.CreatorID, &responsibleID, &deadline, &doc.FileURL, &doc.Tags,
		&doc.IsOverdue, &doc.CreatedAt, &doc.UpdatedAt, &archivedAt,
	)
	if err != nil {
		return nil, err
	}

	doc.ResponsibleID = responsibleID
	doc.Deadline = deadline
	doc.ArchivedAt = archivedAt

	return &doc, nil
}

func scanDocuments(rows pgx.Rows) ([]*domain.Document, error) {
	var docs []*domain.Document
	for rows.Next() {
		doc, err := scanDocument(rows)
		if err != nil {
			return nil, fmt.Errorf("scan document: %w", err)
		}
		docs = append(docs, doc)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate documents: %w", err)
	}
	return docs, nil
}
