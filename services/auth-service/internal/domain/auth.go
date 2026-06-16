package domain

import (
	"errors"
	"strings"
	"time"
	"unicode"
)

type Role string

const (
	RoleEmployee Role = "employee"
	RoleManager  Role = "manager"
	RoleAdmin    Role = "admin"
)

type AuthUser struct {
	ID           string    `db:"id"`
	Email        string    `db:"email"`
	PasswordHash string    `db:"password_hash"`
	Role         Role      `db:"role"`
	IsActive     bool      `db:"is_active"`
	IsVerified   bool      `db:"is_verified"`
	IsBanned     bool      `db:"is_banned"`
	CreatedAt    time.Time `db:"created_at"`
	UpdatedAt    time.Time `db:"updated_at"`
}

type RefreshToken struct {
	ID        string     `db:"id"`
	UserID    string     `db:"user_id"`
	TokenHash string     `db:"token_hash"`
	ExpiresAt time.Time  `db:"expires_at"`
	RevokedAt *time.Time `db:"revoked_at"`
	CreatedAt time.Time  `db:"created_at"`
}

type RegisterInput struct {
	Email    string
	Password string
	Name     string
	Role     Role
}

type LoginInput struct {
	Email    string
	Password string
}

var (
	ErrUserNotFound  = errors.New("user not found")
	ErrEmailTaken    = errors.New("email already taken")
	ErrInvalidInput  = errors.New("invalid input")
	ErrUnauthorized  = errors.New("unauthorized")
	ErrTokenExpired  = errors.New("token expired")
	ErrTokenInvalid  = errors.New("token invalid")
	ErrTokenNotFound = errors.New("token not found")
	ErrUserBanned    = errors.New("user is banned")
	ErrUserInactive  = errors.New("user is inactive")
)

func ValidatePassword(password string) error {
	password = strings.TrimSpace(password)
	if len(password) < 8 {
		return ErrInvalidInput
	}

	var hasUpper, hasLower, hasDigit, hasSpecial bool
	for _, r := range password {
		switch {
		case unicode.IsUpper(r):
			hasUpper = true
		case unicode.IsLower(r):
			hasLower = true
		case unicode.IsDigit(r):
			hasDigit = true
		case unicode.IsPunct(r) || unicode.IsSymbol(r):
			hasSpecial = true
		}
	}

	if !hasUpper || !hasLower || !hasDigit || !hasSpecial {
		return ErrInvalidInput
	}

	return nil
}

func (i *RegisterInput) Validate() error {
	i.Email = strings.TrimSpace(strings.ToLower(i.Email))
	i.Password = strings.TrimSpace(i.Password)
	i.Name = strings.TrimSpace(i.Name)

	if i.Email == "" || i.Password == "" || i.Name == "" {
		return ErrInvalidInput
	}

	if i.Role == "" {
		i.Role = RoleEmployee
	}

	if i.Role != RoleEmployee && i.Role != RoleManager && i.Role != RoleAdmin {
		return ErrInvalidInput
	}

	return ValidatePassword(i.Password)
}

func (i *LoginInput) Validate() error {
	i.Email = strings.TrimSpace(strings.ToLower(i.Email))
	i.Password = strings.TrimSpace(i.Password)

	if i.Email == "" || i.Password == "" {
		return ErrInvalidInput
	}

	return nil
}
