package service

import (
	"context"
	"time"
)

type AuthService interface {
	Register(ctx context.Context, input RegisterInput) (string, error)
	Login(ctx context.Context, input LoginInput) (*TokenPair, error)
	RefreshToken(ctx context.Context, refreshToken string) (*TokenPair, error)
	Logout(ctx context.Context, userID, refreshToken string) error
	ValidateToken(tokenString string) (*TokenClaims, error)
	VerifyEmail(ctx context.Context, token string) error
	ForgotPassword(ctx context.Context, email string) error
	ResetPassword(ctx context.Context, token, newPassword string) error
	ChangePassword(ctx context.Context, userID, oldPassword, newPassword string) error
	GetSession(ctx context.Context, userID string) (*SessionInfo, error)
	RevokeAllSessions(ctx context.Context, userID string) (int32, error)
}

type RegisterInput struct {
	Email    string
	Password string
	Name     string
	Role     string
}

type LoginInput struct {
	Email    string
	Password string
}

type TokenPair struct {
	AccessToken  string
	RefreshToken string
	ExpiresIn    int64
	UserID       string
	Email        string
	Role         string
}

type TokenClaims struct {
	UserID string
	Email  string
	Role   string
}

type SessionInfo struct {
	UserID    string
	Email     string
	Role      string
	CreatedAt time.Time
	ExpiresAt time.Time
}
