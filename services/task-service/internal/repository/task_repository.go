package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Temych228/DocflowWeb/services/task-service/internal/domain"
)

type taskRepository struct {
	db *pgxpool.Pool
}

func NewTaskRepository(db *pgxpool.Pool) TaskRepository {
	return &taskRepository{db: db}
}

func (r *taskRepository) Create(ctx context.Context, task *domain.Task) error {
	row := r.db.QueryRow(ctx, `
		INSERT INTO tasks (title, description, document_id, assignee_id, creator_id, status, priority, deadline)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, status, created_at, updated_at
	`, task.Title, task.Description, task.DocumentID, task.AssigneeID, task.CreatorID,
		task.Status, task.Priority, task.Deadline)

	err := row.Scan(&task.ID, &task.Status, &task.CreatedAt, &task.UpdatedAt)
	if err != nil {
		return fmt.Errorf("create task: %w", err)
	}
	return nil
}

func (r *taskRepository) GetByID(ctx context.Context, id string) (*domain.Task, error) {
	row := r.db.QueryRow(ctx, `
		SELECT id, title, description, document_id, assignee_id, creator_id, status, priority,
			deadline, created_at, updated_at
		FROM tasks
		WHERE id = $1
	`, id)

	task, err := scanTask(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrTaskNotFound
		}
		return nil, fmt.Errorf("get task by id: %w", err)
	}
	return task, nil
}

func (r *taskRepository) Update(ctx context.Context, task *domain.Task) error {
	row := r.db.QueryRow(ctx, `
		UPDATE tasks
		SET title = $1, description = $2, priority = $3, deadline = $4, updated_at = NOW()
		WHERE id = $5
		RETURNING updated_at
	`, task.Title, task.Description, task.Priority, task.Deadline, task.ID)

	err := row.Scan(&task.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.ErrTaskNotFound
		}
		return fmt.Errorf("update task: %w", err)
	}
	return nil
}

func (r *taskRepository) Delete(ctx context.Context, id string) error {
	tag, err := r.db.Exec(ctx, `DELETE FROM tasks WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete task: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrTaskNotFound
	}
	return nil
}

func (r *taskRepository) ListByDocument(ctx context.Context, documentID string, filter domain.ListFilter) ([]*domain.Task, int, error) {
	var total int
	if err := r.db.QueryRow(ctx, `SELECT COUNT(*) FROM tasks WHERE document_id = $1`, documentID).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count tasks by document: %w", err)
	}

	rows, err := r.db.Query(ctx, `
		SELECT id, title, description, document_id, assignee_id, creator_id, status, priority,
			deadline, created_at, updated_at
		FROM tasks
		WHERE document_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`, documentID, filter.PageSize, filter.Offset())
	if err != nil {
		return nil, 0, fmt.Errorf("list tasks by document: %w", err)
	}
	defer rows.Close()

	tasks, err := scanTasks(rows)
	if err != nil {
		return nil, 0, err
	}
	return tasks, total, nil
}

func (r *taskRepository) ListByAssignee(ctx context.Context, assigneeID string, filter domain.ListFilter) ([]*domain.Task, int, error) {
	var total int
	if err := r.db.QueryRow(ctx, `SELECT COUNT(*) FROM tasks WHERE assignee_id = $1`, assigneeID).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count tasks by assignee: %w", err)
	}

	rows, err := r.db.Query(ctx, `
		SELECT id, title, description, document_id, assignee_id, creator_id, status, priority,
			deadline, created_at, updated_at
		FROM tasks
		WHERE assignee_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`, assigneeID, filter.PageSize, filter.Offset())
	if err != nil {
		return nil, 0, fmt.Errorf("list tasks by assignee: %w", err)
	}
	defer rows.Close()

	tasks, err := scanTasks(rows)
	if err != nil {
		return nil, 0, err
	}
	return tasks, total, nil
}

func (r *taskRepository) Filter(ctx context.Context, filter domain.AdvancedFilter) ([]*domain.Task, int, error) {
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
	if filter.Priority != "" {
		conditions = append(conditions, fmt.Sprintf("priority = $%d", idx))
		args = append(args, string(filter.Priority))
		idx++
	}
	if filter.AssigneeID != "" {
		conditions = append(conditions, fmt.Sprintf("assignee_id = $%d", idx))
		args = append(args, filter.AssigneeID)
		idx++
	}
	if filter.DocumentID != "" {
		conditions = append(conditions, fmt.Sprintf("document_id = $%d", idx))
		args = append(args, filter.DocumentID)
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
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM tasks %s", where)
	if err := r.db.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count filtered tasks: %w", err)
	}

	args = append(args, filter.PageSize, filter.Offset())
	query := fmt.Sprintf(`
		SELECT id, title, description, document_id, assignee_id, creator_id, status, priority,
			deadline, created_at, updated_at
		FROM tasks
		%s
		ORDER BY created_at DESC
		LIMIT $%d OFFSET $%d
	`, where, idx, idx+1)

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("filter tasks: %w", err)
	}
	defer rows.Close()

	tasks, err := scanTasks(rows)
	if err != nil {
		return nil, 0, err
	}
	return tasks, total, nil
}

func (r *taskRepository) SetAssignee(ctx context.Context, id, assigneeID string) error {
	tag, err := r.db.Exec(ctx, `UPDATE tasks SET assignee_id = $1, updated_at = NOW() WHERE id = $2`, assigneeID, id)
	if err != nil {
		return fmt.Errorf("set assignee: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrTaskNotFound
	}
	return nil
}

func (r *taskRepository) UpdateStatus(ctx context.Context, id string, status domain.TaskStatus) error {
	tag, err := r.db.Exec(ctx, `UPDATE tasks SET status = $1, updated_at = NOW() WHERE id = $2`, status, id)
	if err != nil {
		return fmt.Errorf("update task status: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrTaskNotFound
	}
	return nil
}

func (r *taskRepository) FindOverdue(ctx context.Context) ([]*domain.Task, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, title, description, document_id, assignee_id, creator_id, status, priority,
			deadline, created_at, updated_at
		FROM tasks
		WHERE deadline IS NOT NULL
			AND deadline < NOW()
			AND status NOT IN ('done', 'overdue')
	`)
	if err != nil {
		return nil, fmt.Errorf("find overdue tasks: %w", err)
	}
	defer rows.Close()

	return scanTasks(rows)
}

func (r *taskRepository) MarkOverdueByIDs(ctx context.Context, ids []string) error {
	if len(ids) == 0 {
		return nil
	}
	_, err := r.db.Exec(ctx, `
		UPDATE tasks
		SET status = $1, updated_at = NOW()
		WHERE id = ANY($2)
	`, domain.StatusOverdue, ids)
	if err != nil {
		return fmt.Errorf("mark tasks overdue: %w", err)
	}
	return nil
}

func (r *taskRepository) Stats(ctx context.Context) (*domain.TaskStats, error) {
	stats := &domain.TaskStats{}
	err := r.db.QueryRow(ctx, `
		SELECT
			COUNT(*) AS total,
			COUNT(*) FILTER (WHERE status = 'open')        AS open,
			COUNT(*) FILTER (WHERE status = 'in_progress') AS in_progress,
			COUNT(*) FILTER (WHERE status = 'done')        AS done,
			COUNT(*) FILTER (WHERE status = 'overdue')     AS overdue
		FROM tasks
	`).Scan(&stats.Total, &stats.Open, &stats.InProgress, &stats.Done, &stats.Overdue)
	if err != nil {
		return nil, fmt.Errorf("get task stats: %w", err)
	}
	return stats, nil
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanTask(row rowScanner) (*domain.Task, error) {
	var (
		task       domain.Task
		assigneeID *string
		deadline   *time.Time
	)

	err := row.Scan(
		&task.ID, &task.Title, &task.Description, &task.DocumentID, &assigneeID,
		&task.CreatorID, &task.Status, &task.Priority, &deadline,
		&task.CreatedAt, &task.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	task.AssigneeID = assigneeID
	task.Deadline = deadline

	return &task, nil
}

func scanTasks(rows pgx.Rows) ([]*domain.Task, error) {
	var tasks []*domain.Task
	for rows.Next() {
		task, err := scanTask(rows)
		if err != nil {
			return nil, fmt.Errorf("scan task: %w", err)
		}
		tasks = append(tasks, task)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate tasks: %w", err)
	}
	return tasks, nil
}
