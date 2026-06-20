package domain

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

type Role string

const (
	RoleEmployee Role = "employee"
	RoleManager  Role = "manager"
	RoleAdmin    Role = "admin"
)

type User struct {
	ID         string    `db:"id" json:"id"`
	Email      string    `db:"email" json:"email"`
	FirstName  string    `db:"first_name" json:"first_name"`
	LastName   string    `db:"last_name" json:"last_name"`
	Phone      string    `db:"phone" json:"phone"`
	Department string    `db:"department" json:"department"`
	AvatarURL  string    `db:"avatar_url" json:"avatar_url"`
	Role       Role      `db:"role" json:"role"`
	IsActive   bool      `db:"is_active" json:"is_active"`
	IsVerified bool      `db:"is_verified" json:"is_verified"`
	IsBanned   bool      `db:"is_banned" json:"is_banned"`
	BanReason  string    `db:"ban_reason" json:"ban_reason"`
	CreatedAt  time.Time `db:"created_at" json:"created_at"`
	UpdatedAt  time.Time `db:"updated_at" json:"updated_at"`
}

type CreateInput struct {
	Email      string
	Name       string
	FirstName  string
	LastName   string
	Phone      string
	Department string
	AvatarURL  string
	Role       Role
}

type UpdateInput struct {
	Name       string
	FirstName  string
	LastName   string
	Phone      string
	Department string
	AvatarURL  string
	Role       Role
}

type UserStats struct {
	Total         int32 `json:"total"`
	Active        int32 `json:"active"`
	Banned        int32 `json:"banned"`
	Employees     int32 `json:"employees"`
	Managers      int32 `json:"managers"`
	Admins        int32 `json:"admins"`
	TotalDocs     int32 `json:"total_docs"`
	CompletedDocs int32 `json:"completed_docs"`
	OverdueDocs   int32 `json:"overdue_docs"`
	TotalTasks    int32 `json:"total_tasks"`
}

var (
	ErrUserNotFound = errors.New("user not found")
	ErrEmailTaken   = errors.New("email already taken")
	ErrUserBanned   = errors.New("user is banned")
	ErrInvalidInput = errors.New("invalid input")
)

func (r Role) Valid() bool {
	switch r {
	case RoleEmployee, RoleManager, RoleAdmin:
		return true
	default:
		return false
	}
}

func (u *User) FullName() string {
	first := strings.TrimSpace(u.FirstName)
	last := strings.TrimSpace(u.LastName)
	switch {
	case first == "" && last == "":
		return ""
	case first == "":
		return last
	case last == "":
		return first
	default:
		return first + " " + last
	}
}

func (i *CreateInput) Validate() error {
	i.Email = normalizeEmail(i.Email)
	i.Name = strings.TrimSpace(i.Name)
	i.FirstName = strings.TrimSpace(i.FirstName)
	i.LastName = strings.TrimSpace(i.LastName)
	i.Phone = strings.TrimSpace(i.Phone)
	i.Department = strings.TrimSpace(i.Department)
	i.AvatarURL = strings.TrimSpace(i.AvatarURL)

	if i.Email == "" {
		return ErrInvalidInput
	}

	if i.FirstName == "" && i.Name != "" {
		i.FirstName, i.LastName = splitFullName(i.Name)
	}

	if i.FirstName == "" {
		return ErrInvalidInput
	}

	if i.LastName == "" && i.Name != "" {
		_, i.LastName = splitFullName(i.Name)
	}

	if i.Role == "" {
		i.Role = RoleEmployee
	}

	if !i.Role.Valid() {
		return ErrInvalidInput
	}

	return nil
}

func (i *UpdateInput) Validate() error {
	i.Name = strings.TrimSpace(i.Name)
	i.FirstName = strings.TrimSpace(i.FirstName)
	i.LastName = strings.TrimSpace(i.LastName)
	i.Phone = strings.TrimSpace(i.Phone)
	i.Department = strings.TrimSpace(i.Department)
	i.AvatarURL = strings.TrimSpace(i.AvatarURL)

	if i.Name != "" && i.FirstName == "" {
		i.FirstName, i.LastName = splitFullName(i.Name)
	}

	if i.Role != "" && !i.Role.Valid() {
		return ErrInvalidInput
	}

	if i.FirstName == "" && i.LastName == "" && i.Phone == "" && i.Department == "" && i.AvatarURL == "" && i.Role == "" {
		return ErrInvalidInput
	}

	return nil
}

func splitFullName(name string) (string, string) {
	parts := strings.Fields(strings.TrimSpace(name))
	if len(parts) == 0 {
		return "", ""
	}
	if len(parts) == 1 {
		return parts[0], ""
	}
	return parts[0], strings.Join(parts[1:], " ")
}

func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

func ParseRole(s string) Role {
	r := Role(strings.ToLower(strings.TrimSpace(s)))
	if r.Valid() {
		return r
	}
	return RoleEmployee
}

func (r Role) String() string {
	return string(r)
}

func NewName(firstName, lastName string) string {
	firstName = strings.TrimSpace(firstName)
	lastName = strings.TrimSpace(lastName)
	switch {
	case firstName == "" && lastName == "":
		return ""
	case firstName == "":
		return lastName
	case lastName == "":
		return firstName
	default:
		return fmt.Sprintf("%s %s", firstName, lastName)
	}
}
