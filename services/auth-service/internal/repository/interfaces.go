package repository

import (
	"context"
	"time"

	"github.com/Temych228/DocflowWeb/services/auth-service/internal/domain"
)

type AuthRepository interface {
	CreateUser(ctx context.Context, email, passwordHash string, role domain.Role) (string, error)
	GetUserByEmail(ctx context.Context, email string) (*domain.AuthUser, error)
	GetUserByID(ctx context.Context, id string) (*domain.AuthUser, error)
	MarkVerified(ctx context.Context, userID string) error
	UpdatePassword(ctx context.Context, userID, newHash string) error
	SetActive(ctx context.Context, userID string, active bool) error

	CreateRefreshToken(ctx context.Context, userID, token string, expiresAt time.Time) error
	FindRefreshToken(ctx context.Context, token string) (userID string, expiresAt time.Time, ok bool, err error)
	RevokeRefreshToken(ctx context.Context, token string) error
	RevokeAllRefreshTokens(ctx context.Context, userID string) (int32, error)

	StoreVerificationToken(ctx context.Context, userID, token string, expiresAt time.Time) error
	GetVerificationToken(ctx context.Context, token string) (string, error)
	DeleteVerificationToken(ctx context.Context, token string) error

	StorePasswordResetToken(ctx context.Context, userID, token string, expiresAt time.Time) error
	GetPasswordResetToken(ctx context.Context, token string) (string, error)
	DeletePasswordResetToken(ctx context.Context, token string) error

	StoreSession(ctx context.Context, userID string, sessionJSON []byte, ttl time.Duration) error
	GetSession(ctx context.Context, userID string) ([]byte, error)
	DeleteSession(ctx context.Context, userID string) error
}
