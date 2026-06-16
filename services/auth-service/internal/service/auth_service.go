package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"github.com/Temych228/DocflowWeb/services/auth-service/internal/clients"
	"github.com/Temych228/DocflowWeb/services/auth-service/internal/config"
	"github.com/Temych228/DocflowWeb/services/auth-service/internal/domain"
	"github.com/Temych228/DocflowWeb/services/auth-service/internal/repository"
)

type authService struct {
	cfg    *config.Config
	repo   repository.AuthRepository
	notif  clients.NotificationClient
	userCl clients.UserClient
}

func New(
	cfg *config.Config,
	repo repository.AuthRepository,
	notif clients.NotificationClient,
	userCl clients.UserClient,
) AuthService {
	return &authService{cfg: cfg, repo: repo, notif: notif, userCl: userCl}
}

func (s *authService) Register(ctx context.Context, in RegisterInput) (string, error) {
	m := &domain.RegisterInput{
		Email:    in.Email,
		Password: in.Password,
		Name:     in.Name,
		Role:     domain.Role(in.Role),
	}
	if err := m.Validate(); err != nil {
		return "", err
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(m.Password), bcrypt.DefaultCost)
	if err != nil {
		return "", fmt.Errorf("hash password: %w", err)
	}

	userID, err := s.repo.CreateUser(ctx, m.Email, string(hash), m.Role)
	if err != nil {
		return "", err
	}

	verifyToken := uuid.New().String()
	_ = s.repo.StoreVerificationToken(ctx, userID, verifyToken, time.Now().Add(s.cfg.VerificationTTL))

	if s.notif != nil {
		_ = s.notif.SendVerificationEmail(ctx, userID, m.Email, m.Name, verifyToken)
	}

	if s.userCl != nil {
		_ = s.userCl.CreateUser(ctx, clients.CreateUserRequest{
			Email: m.Email,
			Name:  m.Name,
			Role:  string(m.Role),
		})
	}

	return userID, nil
}

func (s *authService) Login(ctx context.Context, in LoginInput) (*TokenPair, error) {
	m := &domain.LoginInput{Email: in.Email, Password: in.Password}
	if err := m.Validate(); err != nil {
		return nil, err
	}

	user, err := s.repo.GetUserByEmail(ctx, m.Email)
	if err != nil {
		if errors.Is(err, domain.ErrUserNotFound) {
			return nil, domain.ErrUnauthorized
		}
		return nil, err
	}

	if !user.IsActive {
		return nil, domain.ErrUserInactive
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(m.Password)); err != nil {
		return nil, domain.ErrUnauthorized
	}

	pair, err := s.issueTokenPair(ctx, user)
	if err != nil {
		return nil, err
	}

	session := SessionInfo{
		UserID:    user.ID,
		Email:     user.Email,
		Role:      string(user.Role),
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(s.cfg.AccessTokenTTL),
	}
	raw, _ := json.Marshal(session)
	_ = s.repo.StoreSession(ctx, user.ID, raw, s.cfg.AccessTokenTTL)

	return pair, nil
}

func (s *authService) RefreshToken(ctx context.Context, refreshToken string) (*TokenPair, error) {
	if refreshToken == "" {
		return nil, domain.ErrTokenInvalid
	}

	userID, expiresAt, _, err := s.repo.FindRefreshToken(ctx, refreshToken)
	if err != nil {
		return nil, err
	}
	if time.Now().After(expiresAt) {
		return nil, domain.ErrTokenExpired
	}

	if err := s.repo.RevokeRefreshToken(ctx, refreshToken); err != nil {
		return nil, err
	}

	user, err := s.repo.GetUserByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if !user.IsActive {
		return nil, domain.ErrUserInactive
	}

	return s.issueTokenPair(ctx, user)
}

func (s *authService) Logout(ctx context.Context, userID, refreshToken string) error {
	if refreshToken == "" {
		return domain.ErrTokenInvalid
	}

	if err := s.repo.RevokeRefreshToken(ctx, refreshToken); err != nil {
		return err
	}

	if userID != "" {
		_ = s.repo.DeleteSession(ctx, userID)
	}

	return nil
}

func (s *authService) ValidateToken(tokenString string) (*TokenClaims, error) {
	claims := &jwtClaims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, domain.ErrTokenInvalid
		}
		return []byte(s.cfg.JWTSecret), nil
	})
	if err != nil || !token.Valid {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, domain.ErrTokenExpired
		}
		return nil, domain.ErrTokenInvalid
	}

	return &TokenClaims{
		UserID: claims.UserID,
		Email:  claims.Email,
		Role:   claims.Role,
	}, nil
}

func (s *authService) VerifyEmail(ctx context.Context, token string) error {
	if token == "" {
		return domain.ErrInvalidInput
	}

	userID, err := s.repo.GetVerificationToken(ctx, token)
	if err != nil {
		if errors.Is(err, domain.ErrTokenNotFound) {
			return domain.ErrTokenInvalid
		}
		return err
	}

	if err := s.repo.MarkVerified(ctx, userID); err != nil {
		return err
	}
	_ = s.repo.DeleteVerificationToken(ctx, token)

	if s.userCl != nil {
		_ = s.userCl.VerifyUserEmail(ctx, userID)
	}

	return nil
}

func (s *authService) ForgotPassword(ctx context.Context, email string) error {
	if email == "" {
		return domain.ErrInvalidInput
	}

	user, err := s.repo.GetUserByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, domain.ErrUserNotFound) {
			return nil
		}
		return err
	}

	resetToken := uuid.New().String()
	if err := s.repo.StorePasswordResetToken(ctx, user.ID, resetToken, time.Now().Add(s.cfg.PasswordResetTTL)); err != nil {
		return err
	}

	if s.notif != nil {
		_ = s.notif.SendPasswordResetEmail(ctx, user.ID, user.Email, resetToken)
	}

	return nil
}

func (s *authService) ResetPassword(ctx context.Context, token, newPassword string) error {
	if token == "" || newPassword == "" {
		return domain.ErrInvalidInput
	}
	if err := domain.ValidatePassword(newPassword); err != nil {
		return err
	}

	userID, err := s.repo.GetPasswordResetToken(ctx, token)
	if err != nil {
		if errors.Is(err, domain.ErrTokenNotFound) {
			return domain.ErrTokenInvalid
		}
		return err
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("hash password: %w", err)
	}

	if err := s.repo.UpdatePassword(ctx, userID, string(hash)); err != nil {
		return err
	}

	_ = s.repo.DeletePasswordResetToken(ctx, token)
	_, _ = s.repo.RevokeAllRefreshTokens(ctx, userID)
	_ = s.repo.DeleteSession(ctx, userID)

	return nil
}

func (s *authService) ChangePassword(ctx context.Context, userID, oldPassword, newPassword string) error {
	if userID == "" || oldPassword == "" || newPassword == "" {
		return domain.ErrInvalidInput
	}
	if err := domain.ValidatePassword(newPassword); err != nil {
		return err
	}

	user, err := s.repo.GetUserByID(ctx, userID)
	if err != nil {
		return err
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(oldPassword)); err != nil {
		return domain.ErrUnauthorized
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("hash password: %w", err)
	}

	if err := s.repo.UpdatePassword(ctx, userID, string(hash)); err != nil {
		return err
	}

	_, _ = s.repo.RevokeAllRefreshTokens(ctx, userID)
	_ = s.repo.DeleteSession(ctx, userID)

	return nil
}

func (s *authService) GetSession(ctx context.Context, userID string) (*SessionInfo, error) {
	if userID == "" {
		return nil, domain.ErrInvalidInput
	}

	raw, err := s.repo.GetSession(ctx, userID)
	if err == nil {
		var session SessionInfo
		if jsonErr := json.Unmarshal(raw, &session); jsonErr == nil {
			return &session, nil
		}
	}

	user, err := s.repo.GetUserByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	return &SessionInfo{
		UserID:    user.ID,
		Email:     user.Email,
		Role:      string(user.Role),
		CreatedAt: user.CreatedAt,
		ExpiresAt: time.Now().Add(s.cfg.AccessTokenTTL),
	}, nil
}

func (s *authService) RevokeAllSessions(ctx context.Context, userID string) (int32, error) {
	if userID == "" {
		return 0, domain.ErrInvalidInput
	}

	count, err := s.repo.RevokeAllRefreshTokens(ctx, userID)
	if err != nil {
		return 0, err
	}

	_ = s.repo.DeleteSession(ctx, userID)

	return count, nil
}

type jwtClaims struct {
	UserID string `json:"user_id"`
	Email  string `json:"email"`
	Role   string `json:"role"`
	jwt.RegisteredClaims
}

func (s *authService) issueTokenPair(ctx context.Context, user *domain.AuthUser) (*TokenPair, error) {
	expiresAt := time.Now().Add(s.cfg.AccessTokenTTL)

	claims := &jwtClaims{
		UserID: user.ID,
		Email:  user.Email,
		Role:   string(user.Role),
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ID:        uuid.New().String(),
		},
	}

	accessToken, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(s.cfg.JWTSecret))
	if err != nil {
		return nil, fmt.Errorf("sign access token: %w", err)
	}

	refreshToken := uuid.New().String()
	if err := s.repo.CreateRefreshToken(ctx, user.ID, refreshToken, time.Now().Add(s.cfg.RefreshTokenTTL)); err != nil {
		return nil, err
	}

	return &TokenPair{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    int64(s.cfg.AccessTokenTTL.Seconds()), // proto: expires_in (секунды)
		UserID:       user.ID,
		Email:        user.Email,
		Role:         string(user.Role),
	}, nil
}
