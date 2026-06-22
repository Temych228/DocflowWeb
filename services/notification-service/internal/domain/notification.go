package domain

import (
	"errors"
	"strings"
	"time"
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
)

type Channel string

const (
	ChannelEmail   Channel = "email"
	ChannelPush    Channel = "push"
	ChannelSystem  Channel = "system"
	ChannelLogging Channel = "logging"
)

type Notification struct {
	ID         string     `db:"id" json:"id"`
	UserID     string     `db:"user_id" json:"user_id"`
	DocumentID string     `db:"document_id" json:"document_id"`
	TaskID     string     `db:"task_id" json:"task_id"`
	Category   Category   `db:"notif_category" json:"category"`
	Title      string     `db:"title" json:"title"`
	Body       string     `db:"body" json:"body"`
	RefID      string     `db:"ref_id" json:"ref_id"`
	RefType    string     `db:"ref_type" json:"ref_type"`
	ProtoType  int32      `db:"proto_type" json:"proto_type"`
	IsRead     bool       `db:"is_read" json:"is_read"`
	SentEmail  bool       `db:"sent_email" json:"sent_email"`
	CreatedAt  time.Time  `db:"created_at" json:"created_at"`
	UpdatedAt  time.Time  `db:"updated_at" json:"updated_at"`
	ReadAt     *time.Time `db:"read_at" json:"read_at"`
	DeletedAt  *time.Time `db:"deleted_at" json:"deleted_at"`
}

type Preferences struct {
	UserID        string    `db:"user_id" json:"user_id"`
	EmailEnabled  bool      `db:"email_enabled" json:"email_enabled"`
	PushEnabled   bool      `db:"push_enabled" json:"push_enabled"`
	DeadlineNotif bool      `db:"deadline_notif" json:"deadline_notif"`
	AssignedNotif bool      `db:"assigned_notif" json:"assigned_notif"`
	StatusNotif   bool      `db:"status_notif" json:"status_notif"`
	OverdueNotif  bool      `db:"overdue_notif" json:"overdue_notif"`
	UpdatedAt     time.Time `db:"updated_at" json:"updated_at"`
}

type Template struct {
	TemplateID   string    `db:"template_id" json:"template_id"`
	Subject      string    `db:"subject" json:"subject"`
	BodyTemplate string    `db:"body_template" json:"body_template"`
	IsActive     bool      `db:"is_active" json:"is_active"`
	CreatedAt    time.Time `db:"created_at" json:"created_at"`
	UpdatedAt    time.Time `db:"updated_at" json:"updated_at"`
}

type LogEvent struct {
	Service   string         `json:"service"`
	Action    string         `json:"action"`
	Level     string         `json:"level"`
	Message   string         `json:"message"`
	RequestID string         `json:"request_id,omitempty"`
	UserID    string         `json:"user_id,omitempty"`
	Meta      map[string]any `json:"meta,omitempty"`
	At        time.Time      `json:"at"`
}

type EmailJob struct {
	JobID          string            `json:"job_id"`
	NotificationID string            `json:"notification_id"`
	UserID         string            `json:"user_id"`
	Recipient      []string          `json:"recipient"`
	TemplateID     string            `json:"template_id"`
	Subject        string            `json:"subject"`
	Body           string            `json:"body"`
	Variables      map[string]string `json:"variables,omitempty"`
	Category       Category          `json:"category"`
	CreatedAt      time.Time         `json:"created_at"`
}

type StatsSnapshot struct {
	Total      int64            `json:"total"`
	Read       int64            `json:"read"`
	Unread     int64            `json:"unread"`
	ByCategory map[string]int64 `json:"by_category"`
	ByHour     map[string]int64 `json:"by_hour"`
	ByChannel  map[string]int64 `json:"by_channel"`
	From       time.Time        `json:"from"`
	To         time.Time        `json:"to"`
}

var (
	ErrNotFound     = errors.New("notification not found")
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
	case CategorySystem:
		return CategorySystem
	default:
		return CategorySystem
	}
}

func (c Category) String() string {
	return string(c)
}

func (c Channel) String() string {
	return string(c)
}
