package repository

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Temych228/DocflowWeb/services/document-service/internal/domain"
)

type postgresHistoryRepository struct {
	db *pgxpool.Pool
}

func NewHistoryRepository(db *pgxpool.Pool) *postgresHistoryRepository {
	return &postgresHistoryRepository{db: db}
}

func (r *postgresHistoryRepository) Create(ctx context.Context, entry *domain.HistoryEntry) error {
	row := r.db.QueryRow(ctx, `
		INSERT INTO document_history (document_id, changed_by, field, old_value, new_value)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, changed_at
	`, entry.DocumentID, entry.ChangedBy, entry.Field, entry.OldValue, entry.NewValue)

	if err := row.Scan(&entry.ID, &entry.ChangedAt); err != nil {
		return fmt.Errorf("create history entry: %w", err)
	}
	return nil
}

func (r *postgresHistoryRepository) List(ctx context.Context, filter domain.HistoryFilter) ([]*domain.HistoryEntry, int, error) {
	var total int
	err := r.db.QueryRow(ctx, `
		SELECT COUNT(*) FROM document_history WHERE document_id = $1
	`, filter.DocumentID).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("count history entries: %w", err)
	}

	rows, err := r.db.Query(ctx, `
		SELECT id, document_id, changed_by, field, old_value, new_value, changed_at
		FROM document_history
		WHERE document_id = $1
		ORDER BY changed_at DESC
		LIMIT $2 OFFSET $3
	`, filter.DocumentID, filter.PageSize, filter.Offset())
	if err != nil {
		return nil, 0, fmt.Errorf("list history entries: %w", err)
	}
	defer rows.Close()

	entries, err := scanHistoryEntries(rows)
	if err != nil {
		return nil, 0, err
	}
	return entries, total, nil
}

func scanHistoryEntries(rows pgx.Rows) ([]*domain.HistoryEntry, error) {
	var entries []*domain.HistoryEntry
	for rows.Next() {
		var entry domain.HistoryEntry
		err := rows.Scan(
			&entry.ID, &entry.DocumentID, &entry.ChangedBy,
			&entry.Field, &entry.OldValue, &entry.NewValue, &entry.ChangedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan history entry: %w", err)
		}
		entries = append(entries, &entry)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate history entries: %w", err)
	}
	return entries, nil
}
