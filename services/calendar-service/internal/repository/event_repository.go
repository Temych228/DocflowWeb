package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Temych228/DocflowWeb/services/calendar-service/internal/domain"
)

type postgresEventRepository struct {
	db *pgxpool.Pool
}

func NewEventRepository(db *pgxpool.Pool) EventRepository {
	return &postgresEventRepository{db: db}
}

func (r *postgresEventRepository) Create(ctx context.Context, event *domain.Event) error {
	row := r.db.QueryRow(ctx, `
		INSERT INTO events (user_id, title, description, event_type, ref_id, ref_type, start_time, end_time)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, created_at, updated_at
	`,
		event.UserID, event.Title, event.Description, string(event.EventType),
		event.RefID, event.RefType, event.StartTime, event.EndTime,
	)
	return row.Scan(&event.ID, &event.CreatedAt, &event.UpdatedAt)
}

func (r *postgresEventRepository) GetByID(ctx context.Context, id string) (*domain.Event, error) {
	row := r.db.QueryRow(ctx, `
		SELECT id, user_id, title, description, event_type, ref_id, ref_type,
		       start_time, end_time, created_at, updated_at
		FROM events
		WHERE id = $1
	`, id)

	event, err := scanEvent(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrEventNotFound
		}
		return nil, fmt.Errorf("get event by id: %w", err)
	}
	return event, nil
}

func (r *postgresEventRepository) Update(ctx context.Context, event *domain.Event) error {
	row := r.db.QueryRow(ctx, `
		UPDATE events
		SET title = $1, description = $2, start_time = $3, end_time = $4, updated_at = NOW()
		WHERE id = $5
		RETURNING updated_at
	`, event.Title, event.Description, event.StartTime, event.EndTime, event.ID)

	err := row.Scan(&event.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.ErrEventNotFound
		}
		return fmt.Errorf("update event: %w", err)
	}
	return nil
}

func (r *postgresEventRepository) Delete(ctx context.Context, id string) error {
	tag, err := r.db.Exec(ctx, `DELETE FROM events WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete event: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrEventNotFound
	}
	return nil
}

func (r *postgresEventRepository) GetByRange(ctx context.Context, filter domain.RangeFilter) ([]*domain.Event, int, error) {
	var total int
	err := r.db.QueryRow(ctx, `
		SELECT COUNT(*) FROM events
		WHERE user_id = $1 AND start_time >= $2 AND start_time < $3
	`, filter.UserID, filter.From, filter.To).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("count events by range: %w", err)
	}

	rows, err := r.db.Query(ctx, `
		SELECT id, user_id, title, description, event_type, ref_id, ref_type,
		       start_time, end_time, created_at, updated_at
		FROM events
		WHERE user_id = $1 AND start_time >= $2 AND start_time < $3
		ORDER BY start_time ASC
	`, filter.UserID, filter.From, filter.To)
	if err != nil {
		return nil, 0, fmt.Errorf("query events by range: %w", err)
	}
	defer rows.Close()

	events, err := scanEvents(rows)
	if err != nil {
		return nil, 0, err
	}
	return events, total, nil
}

func (r *postgresEventRepository) GetByUser(ctx context.Context, filter domain.ListFilter) ([]*domain.Event, int, error) {
	var total int
	err := r.db.QueryRow(ctx, `
		SELECT COUNT(*) FROM events WHERE user_id = $1
	`, filter.UserID).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("count events by user: %w", err)
	}

	rows, err := r.db.Query(ctx, `
		SELECT id, user_id, title, description, event_type, ref_id, ref_type,
		       start_time, end_time, created_at, updated_at
		FROM events
		WHERE user_id = $1
		ORDER BY start_time ASC
		LIMIT $2 OFFSET $3
	`, filter.UserID, filter.PageSize, filter.Offset())
	if err != nil {
		return nil, 0, fmt.Errorf("query events by user: %w", err)
	}
	defer rows.Close()

	events, err := scanEvents(rows)
	if err != nil {
		return nil, 0, err
	}
	return events, total, nil
}

func (r *postgresEventRepository) GetUpcomingDeadlines(ctx context.Context, userID string, days int) ([]*domain.Event, error) {
	if days <= 0 {
		days = 7
	}
	until := time.Now().AddDate(0, 0, days)

	rows, err := r.db.Query(ctx, `
		SELECT id, user_id, title, description, event_type, ref_id, ref_type,
		       start_time, end_time, created_at, updated_at
		FROM events
		WHERE user_id = $1
		  AND event_type = 'deadline'
		  AND start_time >= NOW()
		  AND start_time <= $2
		ORDER BY start_time ASC
	`, userID, until)
	if err != nil {
		return nil, fmt.Errorf("query upcoming deadlines: %w", err)
	}
	defer rows.Close()

	return scanEvents(rows)
}

func (r *postgresEventRepository) Filter(ctx context.Context, filter domain.EventFilter) ([]*domain.Event, int, error) {
	var (
		conditions []string
		args       []any
		idx        = 1
	)

	if filter.UserID != "" {
		conditions = append(conditions, fmt.Sprintf("user_id = $%d", idx))
		args = append(args, filter.UserID)
		idx++
	}
	if filter.EventType != nil {
		conditions = append(conditions, fmt.Sprintf("event_type = $%d", idx))
		args = append(args, string(*filter.EventType))
		idx++
	}
	if filter.RefType != "" {
		conditions = append(conditions, fmt.Sprintf("ref_type = $%d", idx))
		args = append(args, filter.RefType)
		idx++
	}
	if filter.From != nil {
		conditions = append(conditions, fmt.Sprintf("start_time >= $%d", idx))
		args = append(args, *filter.From)
		idx++
	}
	if filter.To != nil {
		conditions = append(conditions, fmt.Sprintf("start_time <= $%d", idx))
		args = append(args, *filter.To)
		idx++
	}

	where := ""
	if len(conditions) > 0 {
		where = "WHERE " + strings.Join(conditions, " AND ")
	}

	var total int
	countQ := fmt.Sprintf("SELECT COUNT(*) FROM events %s", where)
	if err := r.db.QueryRow(ctx, countQ, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count filtered events: %w", err)
	}

	args = append(args, filter.PageSize, filter.Offset())
	query := fmt.Sprintf(`
		SELECT id, user_id, title, description, event_type, ref_id, ref_type,
		       start_time, end_time, created_at, updated_at
		FROM events
		%s
		ORDER BY start_time ASC
		LIMIT $%d OFFSET $%d
	`, where, idx, idx+1)

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("filter events: %w", err)
	}
	defer rows.Close()

	events, err := scanEvents(rows)
	if err != nil {
		return nil, 0, err
	}
	return events, total, nil
}

func (r *postgresEventRepository) Stats(ctx context.Context, userID string) (*domain.EventStats, error) {
	stats := &domain.EventStats{}

	var q string
	var args []any

	if userID != "" {
		q = `
			SELECT
				COUNT(*) AS total,
				COUNT(*) FILTER (WHERE event_type = 'deadline') AS deadlines,
				COUNT(*) FILTER (WHERE event_type = 'task')     AS tasks,
				COUNT(*) FILTER (WHERE event_type = 'meeting')  AS meetings,
				COUNT(*) FILTER (WHERE event_type = 'reminder') AS reminders
			FROM events
			WHERE user_id = $1
		`
		args = []any{userID}
	} else {
		q = `
			SELECT
				COUNT(*) AS total,
				COUNT(*) FILTER (WHERE event_type = 'deadline') AS deadlines,
				COUNT(*) FILTER (WHERE event_type = 'task')     AS tasks,
				COUNT(*) FILTER (WHERE event_type = 'meeting')  AS meetings,
				COUNT(*) FILTER (WHERE event_type = 'reminder') AS reminders
			FROM events
		`
	}

	err := r.db.QueryRow(ctx, q, args...).Scan(
		&stats.Total, &stats.Deadlines, &stats.Tasks, &stats.Meetings, &stats.Reminders,
	)
	if err != nil {
		return nil, fmt.Errorf("get event stats: %w", err)
	}
	return stats, nil
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanEvent(row rowScanner) (*domain.Event, error) {
	var e domain.Event
	err := row.Scan(
		&e.ID, &e.UserID, &e.Title, &e.Description, &e.EventType,
		&e.RefID, &e.RefType, &e.StartTime, &e.EndTime,
		&e.CreatedAt, &e.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &e, nil
}

func scanEvents(rows pgx.Rows) ([]*domain.Event, error) {
	var events []*domain.Event
	for rows.Next() {
		e, err := scanEvent(rows)
		if err != nil {
			return nil, fmt.Errorf("scan event: %w", err)
		}
		events = append(events, e)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate events: %w", err)
	}
	return events, nil
}
