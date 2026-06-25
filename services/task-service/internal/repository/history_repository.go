package repository

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Temych228/DocflowWeb/services/task-service/internal/domain"
)

type historyRepository struct {
	db *pgxpool.Pool
}

func NewHistoryRepository(db *pgxpool.Pool) HistoryRepository {
	return &historyRepository{db: db}
}

func (r *historyRepository) Create(ctx context.Context, entry *domain.HistoryEntry) error {
	row := r.db.QueryRow(ctx, `
		INSERT INTO task_history (task_id, actor_id, field, old_value, new_value)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, changed_at
	`, entry.TaskID, entry.ChangedBy, entry.Field, entry.OldValue, entry.NewValue)

	if err := row.Scan(&entry.ID, &entry.ChangedAt); err != nil {
		return fmt.Errorf("create task history entry: %w", err)
	}
	return nil
}

func (r *historyRepository) List(ctx context.Context, taskID string) ([]*domain.HistoryEntry, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, task_id, actor_id, field, old_value, new_value, changed_at
		FROM task_history
		WHERE task_id = $1
		ORDER BY changed_at DESC
	`, taskID)
	if err != nil {
		return nil, fmt.Errorf("list task history: %w", err)
	}
	defer rows.Close()

	return scanHistoryEntries(rows)
}

func scanHistoryEntries(rows pgx.Rows) ([]*domain.HistoryEntry, error) {
	var entries []*domain.HistoryEntry
	for rows.Next() {
		var entry domain.HistoryEntry
		err := rows.Scan(
			&entry.ID, &entry.TaskID, &entry.ChangedBy,
			&entry.Field, &entry.OldValue, &entry.NewValue, &entry.ChangedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan task history entry: %w", err)
		}
		entries = append(entries, &entry)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate task history entries: %w", err)
	}
	return entries, nil
}
