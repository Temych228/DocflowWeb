package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"

	"github.com/Temych228/DocflowWeb/services/user-service/internal/domain"
)

const (
	cacheKeyID    = "user:id:"
	cacheKeyEmail = "user:email:"
	defaultTTL    = 5 * time.Minute
)

type userRepo struct {
	db    *pgxpool.Pool
	cache *redis.Client
	ttl   time.Duration
}

func New(db *pgxpool.Pool, cache *redis.Client, ttl time.Duration) UserRepository {
	if ttl <= 0 {
		ttl = defaultTTL
	}
	return &userRepo{db: db, cache: cache, ttl: ttl}
}

func (r *userRepo) GetByID(ctx context.Context, id string) (*domain.User, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return nil, domain.ErrInvalidInput
	}

	if u, err := r.fromCache(ctx, cacheKeyID+id); err == nil {
		return u, nil
	}

	u, err := r.queryOne(ctx, `
		SELECT id, email, first_name, last_name, phone, department, avatar_url, role,
		       is_active, is_verified, is_banned, ban_reason, created_at, updated_at
		FROM users
		WHERE id = $1 AND deleted_at IS NULL
	`, id)
	if err != nil {
		return nil, err
	}

	r.toCache(ctx, u)
	return u, nil
}

func (r *userRepo) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	email = normalizeEmail(email)
	if email == "" {
		return nil, domain.ErrInvalidInput
	}

	if u, err := r.fromCache(ctx, cacheKeyEmail+email); err == nil {
		return u, nil
	}

	u, err := r.queryOne(ctx, `
		SELECT id, email, first_name, last_name, phone, department, avatar_url, role,
		       is_active, is_verified, is_banned, ban_reason, created_at, updated_at
		FROM users
		WHERE email = $1 AND deleted_at IS NULL
	`, email)
	if err != nil {
		return nil, err
	}

	r.toCache(ctx, u)
	return u, nil
}

func (r *userRepo) List(ctx context.Context, page, pageSize int, role string) ([]*domain.User, int, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize

	role = strings.TrimSpace(strings.ToLower(role))

	countSQL := `SELECT COUNT(*) FROM users WHERE deleted_at IS NULL`
	var total int
	var err error

	if role != "" {
		countSQL += ` AND role = $1`
		err = r.db.QueryRow(ctx, countSQL, role).Scan(&total)
	} else {
		err = r.db.QueryRow(ctx, countSQL).Scan(&total)
	}
	if err != nil {
		return nil, 0, fmt.Errorf("count users: %w", err)
	}

	sql := `
		SELECT id, email, first_name, last_name, phone, department, avatar_url, role,
		       is_active, is_verified, is_banned, ban_reason, created_at, updated_at
		FROM users
		WHERE deleted_at IS NULL
	`
	args := []any{}
	if role != "" {
		sql += ` AND role = $1`
		args = append(args, role)
	}
	sql += ` ORDER BY created_at DESC LIMIT $` + fmt.Sprintf("%d", len(args)+1) + ` OFFSET $` + fmt.Sprintf("%d", len(args)+2)
	args = append(args, pageSize, offset)

	rows, err := r.db.Query(ctx, sql, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list users: %w", err)
	}
	defer rows.Close()

	users := make([]*domain.User, 0)
	for rows.Next() {
		u, err := scanUser(rows)
		if err != nil {
			return nil, 0, err
		}
		users = append(users, u)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}

	return users, total, nil
}

func (r *userRepo) GetBatch(ctx context.Context, ids []string) ([]*domain.User, error) {
	if len(ids) == 0 {
		return []*domain.User{}, nil
	}

	out := make([]*domain.User, 0, len(ids))
	missing := make([]string, 0, len(ids))

	for _, id := range ids {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		if u, err := r.fromCache(ctx, cacheKeyID+id); err == nil {
			out = append(out, u)
			continue
		}
		missing = append(missing, id)
	}

	if len(missing) == 0 {
		return out, nil
	}

	rows, err := r.db.Query(ctx, `
		SELECT id, email, first_name, last_name, phone, department, avatar_url, role,
		       is_active, is_verified, is_banned, ban_reason, created_at, updated_at
		FROM users
		WHERE id = ANY($1::uuid[]) AND deleted_at IS NULL
	`, missing)
	if err != nil {
		return nil, fmt.Errorf("get users batch: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		u, err := scanUser(rows)
		if err != nil {
			return nil, err
		}
		r.toCache(ctx, u)
		out = append(out, u)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return out, nil
}

func (r *userRepo) CheckExists(ctx context.Context, email string) (bool, string, error) {
	email = normalizeEmail(email)
	if email == "" {
		return false, "", domain.ErrInvalidInput
	}

	var exists bool
	var id string
	err := r.db.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM users WHERE email = $1 AND deleted_at IS NULL
		),
		COALESCE((
			SELECT id FROM users WHERE email = $1 AND deleted_at IS NULL LIMIT 1
		), '')
	`, email).Scan(&exists, &id)
	if err != nil {
		return false, "", fmt.Errorf("check user exists: %w", err)
	}

	return exists, id, nil
}

func (r *userRepo) Create(ctx context.Context, input domain.CreateInput) (*domain.User, error) {
	if err := input.Validate(); err != nil {
		return nil, err
	}
	return r.insert(ctx, "", input, false)
}

func (r *userRepo) CreateWithID(ctx context.Context, id string, input domain.CreateInput) (*domain.User, error) {
	if err := input.Validate(); err != nil {
		return nil, err
	}
	id = strings.TrimSpace(id)
	if id == "" {
		return nil, domain.ErrInvalidInput
	}
	return r.insert(ctx, id, input, true)
}

func (r *userRepo) insert(ctx context.Context, id string, input domain.CreateInput, fixedID bool) (*domain.User, error) {
	_ = domain.NewName(input.FirstName, input.LastName)

	if fixedID {
		user, err := r.queryOne(ctx, `
			INSERT INTO users (
				id, email, first_name, last_name, phone, department, avatar_url, role,
				is_active, is_verified, is_banned, ban_reason, created_at, updated_at
			)
			VALUES (
				$1, $2, $3, $4, $5, $6, $7, $8,
				true, false, false, '', NOW(), NOW()
			)
			ON CONFLICT (id) DO NOTHING
			RETURNING id, email, first_name, last_name, phone, department, avatar_url, role,
			          is_active, is_verified, is_banned, ban_reason, created_at, updated_at
		`, id, input.Email, input.FirstName, input.LastName, input.Phone, input.Department, input.AvatarURL, string(input.Role))
		if err == nil {
			_, _ = r.db.Exec(ctx, `INSERT INTO user_stats (user_id) VALUES ($1) ON CONFLICT DO NOTHING`, user.ID)
			r.toCache(ctx, user)
			return user, nil
		}
		if !errors.Is(err, domain.ErrUserNotFound) {
			if existing, getErr := r.GetByID(ctx, id); getErr == nil {
				return existing, nil
			}
			if existing, getErr := r.GetByEmail(ctx, input.Email); getErr == nil {
				return existing, nil
			}
			return nil, err
		}
		return nil, err
	}

	user, err := r.queryOne(ctx, `
		INSERT INTO users (
			email, first_name, last_name, phone, department, avatar_url, role,
			is_active, is_verified, is_banned, ban_reason, created_at, updated_at
		)
		VALUES (
			$1, $2, $3, $4, $5, $6, $7,
			true, false, false, '', NOW(), NOW()
		)
		ON CONFLICT (email) DO NOTHING
		RETURNING id, email, first_name, last_name, phone, department, avatar_url, role,
		          is_active, is_verified, is_banned, ban_reason, created_at, updated_at
	`, input.Email, input.FirstName, input.LastName, input.Phone, input.Department, input.AvatarURL, string(input.Role))
	if err == nil {
		_, _ = r.db.Exec(ctx, `INSERT INTO user_stats (user_id) VALUES ($1) ON CONFLICT DO NOTHING`, user.ID)
		r.toCache(ctx, user)
		return user, nil
	}

	if existing, getErr := r.GetByEmail(ctx, input.Email); getErr == nil {
		return existing, nil
	}
	return nil, err
}

func (r *userRepo) Update(ctx context.Context, id string, input domain.UpdateInput) (*domain.User, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return nil, domain.ErrInvalidInput
	}
	if err := input.Validate(); err != nil {
		return nil, err
	}

	current, err := r.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	firstName := current.FirstName
	lastName := current.LastName
	phone := current.Phone
	department := current.Department
	avatarURL := current.AvatarURL
	role := current.Role

	if input.FirstName != "" {
		firstName = input.FirstName
	}
	if input.LastName != "" {
		lastName = input.LastName
	}
	if input.Phone != "" {
		phone = input.Phone
	}
	if input.Department != "" {
		department = input.Department
	}
	if input.AvatarURL != "" {
		avatarURL = input.AvatarURL
	}
	if input.Role != "" {
		role = input.Role
	}

	u, err := r.queryOne(ctx, `
		UPDATE users
		SET first_name = $2,
		    last_name = $3,
		    phone = $4,
		    department = $5,
		    avatar_url = $6,
		    role = $7,
		    updated_at = NOW()
		WHERE id = $1 AND deleted_at IS NULL
		RETURNING id, email, first_name, last_name, phone, department, avatar_url, role,
		          is_active, is_verified, is_banned, ban_reason, created_at, updated_at
	`, id, firstName, lastName, phone, department, avatarURL, string(role))
	if err != nil {
		return nil, err
	}

	r.invalidateCache(ctx, u.ID, u.Email)
	r.toCache(ctx, u)
	return u, nil
}

func (r *userRepo) VerifyEmail(ctx context.Context, id string) error {
	tag, err := r.db.Exec(ctx, `
		UPDATE users
		SET is_verified = true, updated_at = NOW()
		WHERE id = $1 AND deleted_at IS NULL
	`, strings.TrimSpace(id))
	if err != nil {
		return fmt.Errorf("verify email: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrUserNotFound
	}
	r.invalidateCacheByID(ctx, id)
	return nil
}

func (r *userRepo) Ban(ctx context.Context, id, adminID, reason string) error {
	tag, err := r.db.Exec(ctx, `
		UPDATE users
		SET is_banned = true, is_active = false, ban_reason = $2, updated_at = NOW()
		WHERE id = $1 AND deleted_at IS NULL
	`, strings.TrimSpace(id), strings.TrimSpace(reason))
	if err != nil {
		return fmt.Errorf("ban user: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrUserNotFound
	}
	r.invalidateCacheByID(ctx, id)
	return nil
}

func (r *userRepo) Delete(ctx context.Context, id string) error {
	var email string
	err := r.db.QueryRow(ctx, `
		UPDATE users
		SET deleted_at = NOW(), updated_at = NOW()
		WHERE id = $1 AND deleted_at IS NULL
		RETURNING email
	`, strings.TrimSpace(id)).Scan(&email)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.ErrUserNotFound
		}
		return fmt.Errorf("delete user: %w", err)
	}

	r.invalidateCache(ctx, id, email)
	return nil
}

func (r *userRepo) GetStats(ctx context.Context) (*domain.UserStats, error) {
	var s domain.UserStats
	err := r.db.QueryRow(ctx, `
		SELECT
			COUNT(*) AS total,
			COUNT(*) FILTER (WHERE is_active = true) AS active,
			COUNT(*) FILTER (WHERE is_banned = true) AS banned,
			COUNT(*) FILTER (WHERE role = 'employee') AS employees,
			COUNT(*) FILTER (WHERE role = 'manager') AS managers,
			COUNT(*) FILTER (WHERE role = 'admin') AS admins,
			COALESCE(SUM(us.total_docs), 0) AS total_docs,
			COALESCE(SUM(us.completed_docs), 0) AS completed_docs,
			COALESCE(SUM(us.overdue_docs), 0) AS overdue_docs,
			COALESCE(SUM(us.total_tasks), 0) AS total_tasks
		FROM users u
		LEFT JOIN user_stats us ON us.user_id = u.id
		WHERE u.deleted_at IS NULL
	`).Scan(
		&s.Total,
		&s.Active,
		&s.Banned,
		&s.Employees,
		&s.Managers,
		&s.Admins,
		&s.TotalDocs,
		&s.CompletedDocs,
		&s.OverdueDocs,
		&s.TotalTasks,
	)
	if err != nil {
		return nil, fmt.Errorf("get user stats: %w", err)
	}
	return &s, nil
}

func (r *userRepo) queryOne(ctx context.Context, sql string, args ...any) (*domain.User, error) {
	u, err := scanUser(r.db.QueryRow(ctx, sql, args...))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrUserNotFound
		}
		return nil, fmt.Errorf("query user: %w", err)
	}
	return u, nil
}

func scanUser(row interface{ Scan(dest ...any) error }) (*domain.User, error) {
	var u domain.User
	err := row.Scan(
		&u.ID,
		&u.Email,
		&u.FirstName,
		&u.LastName,
		&u.Phone,
		&u.Department,
		&u.AvatarURL,
		&u.Role,
		&u.IsActive,
		&u.IsVerified,
		&u.IsBanned,
		&u.BanReason,
		&u.CreatedAt,
		&u.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *userRepo) fromCache(ctx context.Context, key string) (*domain.User, error) {
	if r.cache == nil {
		return nil, redis.Nil
	}
	data, err := r.cache.Get(ctx, key).Bytes()
	if err != nil {
		return nil, err
	}
	var u domain.User
	if err := json.Unmarshal(data, &u); err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *userRepo) toCache(ctx context.Context, u *domain.User) {
	if r.cache == nil || u == nil {
		return
	}
	data, err := json.Marshal(u)
	if err != nil {
		return
	}
	_ = r.cache.Set(ctx, cacheKeyID+u.ID, data, r.ttl).Err()
	_ = r.cache.Set(ctx, cacheKeyEmail+normalizeEmail(u.Email), data, r.ttl).Err()
}

func (r *userRepo) invalidateCache(ctx context.Context, id, email string) {
	if r.cache == nil {
		return
	}
	_, _ = r.cache.Del(ctx, cacheKeyID+strings.TrimSpace(id), cacheKeyEmail+normalizeEmail(email)).Result()
}

func (r *userRepo) invalidateCacheByID(ctx context.Context, id string) {
	if r.cache == nil {
		return
	}
	_, _ = r.cache.Del(ctx, cacheKeyID+strings.TrimSpace(id)).Result()
}

func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}
