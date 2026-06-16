package repository

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"

	"github.com/Temych228/DocflowWeb/services/auth-service/internal/domain"
)

type authRepo struct {
	db    *pgxpool.Pool
	cache *redis.Client
}

func New(db *pgxpool.Pool, cache *redis.Client) AuthRepository {
	return &authRepo{db: db, cache: cache}
}

func (r *authRepo) CreateUser(ctx context.Context, email, passwordHash string, role domain.Role) (string, error) {
	var id string
	err := r.db.QueryRow(ctx, `
		INSERT INTO auth_users (email, password_hash, role)
		VALUES ($1, $2, $3)
		RETURNING id
	`, normalizeEmail(email), passwordHash, string(role)).Scan(&id)
	if err != nil {
		if isUniqueViolation(err) {
			return "", domain.ErrEmailTaken
		}
		return "", fmt.Errorf("create user: %w", err)
	}
	return id, nil
}

func (r *authRepo) GetUserByEmail(ctx context.Context, email string) (*domain.AuthUser, error) {
	var u domain.AuthUser
	err := r.db.QueryRow(ctx, `
		SELECT id, email, password_hash, role, is_active, is_verified, created_at, updated_at
		FROM auth_users
		WHERE email = $1
	`, normalizeEmail(email)).Scan(
		&u.ID, &u.Email, &u.PasswordHash, &u.Role,
		&u.IsActive, &u.IsVerified, &u.CreatedAt, &u.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrUserNotFound
		}
		return nil, fmt.Errorf("get user by email: %w", err)
	}
	return &u, nil
}

func (r *authRepo) GetUserByID(ctx context.Context, id string) (*domain.AuthUser, error) {
	var u domain.AuthUser
	err := r.db.QueryRow(ctx, `
		SELECT id, email, password_hash, role, is_active, is_verified, created_at, updated_at
		FROM auth_users
		WHERE id = $1
	`, strings.TrimSpace(id)).Scan(
		&u.ID, &u.Email, &u.PasswordHash, &u.Role,
		&u.IsActive, &u.IsVerified, &u.CreatedAt, &u.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrUserNotFound
		}
		return nil, fmt.Errorf("get user by id: %w", err)
	}
	return &u, nil
}

func (r *authRepo) MarkVerified(ctx context.Context, userID string) error {
	tag, err := r.db.Exec(ctx, `
		UPDATE auth_users
		SET is_verified = true, updated_at = NOW()
		WHERE id = $1
	`, strings.TrimSpace(userID))
	if err != nil {
		return fmt.Errorf("mark verified: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrUserNotFound
	}
	return nil
}

func (r *authRepo) UpdatePassword(ctx context.Context, userID, newHash string) error {
	tag, err := r.db.Exec(ctx, `
		UPDATE auth_users
		SET password_hash = $2, updated_at = NOW()
		WHERE id = $1
	`, strings.TrimSpace(userID), newHash)
	if err != nil {
		return fmt.Errorf("update password: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrUserNotFound
	}
	return nil
}

func (r *authRepo) SetActive(ctx context.Context, userID string, active bool) error {
	tag, err := r.db.Exec(ctx, `
		UPDATE auth_users
		SET is_active = $2, updated_at = NOW()
		WHERE id = $1
	`, strings.TrimSpace(userID), active)
	if err != nil {
		return fmt.Errorf("set active: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrUserNotFound
	}
	return nil
}

func (r *authRepo) CreateRefreshToken(ctx context.Context, userID, token string, expiresAt time.Time) error {
	hash := hashToken(token)
	_, err := r.db.Exec(ctx, `
		INSERT INTO refresh_tokens (user_id, token_hash, expires_at)
		VALUES ($1, $2, $3)
	`, strings.TrimSpace(userID), hash, expiresAt)
	if err != nil {
		return fmt.Errorf("create refresh token: %w", err)
	}
	return nil
}

func (r *authRepo) FindRefreshToken(ctx context.Context, token string) (string, time.Time, bool, error) {
	hash := hashToken(token)

	var (
		userID    string
		expiresAt time.Time
		revokedAt *time.Time
	)

	err := r.db.QueryRow(ctx, `
		SELECT user_id, expires_at, revoked_at
		FROM refresh_tokens
		WHERE token_hash = $1
	`, hash).Scan(&userID, &expiresAt, &revokedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", time.Time{}, false, domain.ErrTokenNotFound
		}
		return "", time.Time{}, false, fmt.Errorf("find refresh token: %w", err)
	}

	if revokedAt != nil {
		return "", time.Time{}, false, domain.ErrUnauthorized
	}

	return userID, expiresAt, true, nil
}

func (r *authRepo) RevokeRefreshToken(ctx context.Context, token string) error {
	hash := hashToken(token)
	_, err := r.db.Exec(ctx, `
		UPDATE refresh_tokens
		SET revoked_at = NOW()
		WHERE token_hash = $1 AND revoked_at IS NULL
	`, hash)
	if err != nil {
		return fmt.Errorf("revoke refresh token: %w", err)
	}
	return nil
}

func (r *authRepo) RevokeAllRefreshTokens(ctx context.Context, userID string) (int32, error) {
	tag, err := r.db.Exec(ctx, `
		UPDATE refresh_tokens
		SET revoked_at = NOW()
		WHERE user_id = $1 AND revoked_at IS NULL
	`, strings.TrimSpace(userID))
	if err != nil {
		return 0, fmt.Errorf("revoke all refresh tokens: %w", err)
	}
	return int32(tag.RowsAffected()), nil
}

func (r *authRepo) StoreVerificationToken(ctx context.Context, userID, token string, expiresAt time.Time) error {
	return r.storeTempToken(ctx, "verify:"+token, userID, expiresAt)
}

func (r *authRepo) GetVerificationToken(ctx context.Context, token string) (string, error) {
	return r.getTempToken(ctx, "verify:"+token)
}

func (r *authRepo) DeleteVerificationToken(ctx context.Context, token string) error {
	return r.deleteTempToken(ctx, "verify:"+token)
}

func (r *authRepo) StorePasswordResetToken(ctx context.Context, userID, token string, expiresAt time.Time) error {
	return r.storeTempToken(ctx, "reset:"+token, userID, expiresAt)
}

func (r *authRepo) GetPasswordResetToken(ctx context.Context, token string) (string, error) {
	return r.getTempToken(ctx, "reset:"+token)
}

func (r *authRepo) DeletePasswordResetToken(ctx context.Context, token string) error {
	return r.deleteTempToken(ctx, "reset:"+token)
}

func (r *authRepo) StoreSession(ctx context.Context, userID string, sessionJSON []byte, ttl time.Duration) error {
	if r.cache == nil {
		return nil
	}
	return r.cache.Set(ctx, "auth:session:"+userID, sessionJSON, ttl).Err()
}

func (r *authRepo) GetSession(ctx context.Context, userID string) ([]byte, error) {
	if r.cache == nil {
		return nil, domain.ErrTokenNotFound
	}
	val, err := r.cache.Get(ctx, "auth:session:"+userID).Bytes()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, domain.ErrTokenNotFound
		}
		return nil, fmt.Errorf("get session: %w", err)
	}
	return val, nil
}

func (r *authRepo) DeleteSession(ctx context.Context, userID string) error {
	if r.cache == nil {
		return nil
	}
	_, err := r.cache.Del(ctx, "auth:session:"+userID).Result()
	return err
}

func (r *authRepo) storeTempToken(ctx context.Context, key, value string, expiresAt time.Time) error {
	if r.cache == nil {
		return nil
	}
	return r.cache.Set(ctx, key, value, time.Until(expiresAt)).Err()
}

func (r *authRepo) getTempToken(ctx context.Context, key string) (string, error) {
	if r.cache == nil {
		return "", domain.ErrTokenNotFound
	}
	val, err := r.cache.Get(ctx, key).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return "", domain.ErrTokenNotFound
		}
		return "", fmt.Errorf("get temp token: %w", err)
	}
	return val, nil
}

func (r *authRepo) deleteTempToken(ctx context.Context, key string) error {
	if r.cache == nil {
		return nil
	}
	_, err := r.cache.Del(ctx, key).Result()
	return err
}

func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

func hashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func isUniqueViolation(err error) bool {
	return strings.Contains(strings.ToLower(err.Error()), "unique")
}
