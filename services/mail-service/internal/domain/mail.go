package domain

import (
	"errors"
	"strings"
	"time"
)

type Status string

const (
	StatusQueued     Status = "queued"
	StatusProcessing Status = "processing"
	StatusSent       Status = "sent"
	StatusFailed     Status = "failed"
	StatusBounced    Status = "bounced"
	StatusCancelled  Status = "cancelled"
)

type Category string

const (
	CategoryDocumentAssigned Category = "document_assigned"
	CategoryStatusChanged    Category = "status_changed"
	CategoryDeadlineReminder Category = "deadline_reminder"
	CategoryOverdue          Category = "overdue"
	CategoryTaskAssigned     Category = "task_assigned"
	CategoryMention          Category = "mention"
	CategorySystem           Category = "system"
	CategoryPasswordReset    Category = "password_reset"
	CategoryVerification     Category = "verification"
)

type MailJob struct {
	ID             string            `db:"id" json:"id"`
	JobID          string            `db:"job_id" json:"job_id"`
	NotificationID string            `db:"notification_id" json:"notification_id"`
	UserID         string            `db:"user_id" json:"user_id"`
	Recipient      []string          `db:"recipient" json:"recipient"`
	TemplateID     string            `db:"template_id" json:"template_id"`
	Subject        string            `db:"subject" json:"subject"`
	Body           string            `db:"body" json:"body"`
	Variables      map[string]string `db:"variables" json:"variables,omitempty"`
	Category       Category          `db:"category" json:"category"`
	Status         Status            `db:"status" json:"status"`
	Attempts       int32             `db:"attempts" json:"attempts"`
	MaxAttempts    int32             `db:"max_attempts" json:"max_attempts"`
	LastError      string            `db:"last_error" json:"last_error"`
	QueuedAt       time.Time         `db:"queued_at" json:"queued_at"`
	ProcessedAt    *time.Time        `db:"processed_at" json:"processed_at"`
	SentAt         *time.Time        `db:"sent_at" json:"sent_at"`
	FailedAt       *time.Time        `db:"failed_at" json:"failed_at"`
	CreatedAt      time.Time         `db:"created_at" json:"created_at"`
	UpdatedAt      time.Time         `db:"updated_at" json:"updated_at"`
}

type MailTemplate struct {
	TemplateID   string    `db:"template_id" json:"template_id"`
	Subject      string    `db:"subject" json:"subject"`
	BodyTemplate string    `db:"body_template" json:"body_template"`
	Channel      string    `db:"channel" json:"channel"`
	IsActive     bool      `db:"is_active" json:"is_active"`
	CreatedAt    time.Time `db:"created_at" json:"created_at"`
	UpdatedAt    time.Time `db:"updated_at" json:"updated_at"`
}

type MailStats struct {
	Total      int64            `json:"total"`
	Queued     int64            `json:"queued"`
	Processing int64            `json:"processing"`
	Sent       int64            `json:"sent"`
	Failed     int64            `json:"failed"`
	ByCategory map[string]int64 `json:"by_category"`
	ByHour     map[string]int64 `json:"by_hour"`
	From       time.Time        `json:"from"`
	To         time.Time        `json:"to"`
}

type LogEvent struct {
	Service   string         `json:"service"`
	Action    string         `json:"action"`
	Level     string         `json:"level"`
	Message   string         `json:"message"`
	RequestID string         `json:"request_id,omitempty"`
	JobID     string         `json:"job_id,omitempty"`
	Meta      map[string]any `json:"meta,omitempty"`
	At        time.Time      `json:"at"`
}

type SMTPMessage struct {
	From    string
	To      []string
	Subject string
	HTML    string
	Text    string
}

var (
	ErrNotFound     = errors.New("mail job not found")
	ErrInvalidInput = errors.New("invalid input")
)

func NormalizeCategory(raw string) Category {
	switch Category(strings.ToLower(strings.TrimSpace(raw))) {
	case CategoryDocumentAssigned:
		return CategoryDocumentAssigned
	case CategoryStatusChanged:
		return CategoryStatusChanged
	case CategoryDeadlineReminder:
		return CategoryDeadlineReminder
	case CategoryOverdue:
		return CategoryOverdue
	case CategoryTaskAssigned:
		return CategoryTaskAssigned
	case CategoryMention:
		return CategoryMention
	case CategoryPasswordReset:
		return CategoryPasswordReset
	case CategoryVerification:
		return CategoryVerification
	case CategorySystem:
		return CategorySystem
	default:
		return CategorySystem
	}
}

func NormalizeStatus(raw string) Status {
	switch Status(strings.ToLower(strings.TrimSpace(raw))) {
	case StatusQueued:
		return StatusQueued
	case StatusProcessing:
		return StatusProcessing
	case StatusSent:
		return StatusSent
	case StatusFailed:
		return StatusFailed
	case StatusBounced:
		return StatusBounced
	case StatusCancelled:
		return StatusCancelled
	default:
		return StatusQueued
	}
}

func (s Status) String() string {
	return string(s)
}

func (c Category) String() string {
	return string(c)
}
