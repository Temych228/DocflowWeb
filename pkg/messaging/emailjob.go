package messaging

import "time"

type EmailJob struct {
	JobID          string            `json:"job_id"`
	NotificationID string            `json:"notification_id"`
	UserID         string            `json:"user_id"`
	Recipient      []string          `json:"recipient"`
	TemplateID     string            `json:"template_id"`
	Subject        string            `json:"subject"`
	Body           string            `json:"body"`
	Variables      map[string]string `json:"variables,omitempty"`
	Category       string            `json:"category"`
	CreatedAt      time.Time         `json:"created_at"`
}
