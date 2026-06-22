package service

import (
	"bytes"
	"fmt"
	"html/template"
	"strings"

	"github.com/Temych228/DocflowWeb/services/mail-service/internal/domain"
)

var defaultTemplates = []*domain.MailTemplate{
	{
		TemplateID:   string(domain.CategoryVerification),
		Subject:      "Please verify your email",
		BodyTemplate: "<html><body><h3>Hello</h3><p>Please verify your email by clicking this link: {{.VerificationLink}}</p></body></html>",
		Channel:      "email",
		IsActive:     true,
	},
	{
		TemplateID:   string(domain.CategoryPasswordReset),
		Subject:      "Password reset request",
		BodyTemplate: "<html><body><h3>Hello</h3><p>Reset your password using this link: {{.ResetLink}}</p></body></html>",
		Channel:      "email",
		IsActive:     true,
	},
	{
		TemplateID:   string(domain.CategorySystem),
		Subject:      "System notification",
		BodyTemplate: "<html><body><h3>Notification</h3><p>{{.Title}}</p><p>{{.Body}}</p></body></html>",
		Channel:      "email",
		IsActive:     true,
	},
}

func defaultTemplateByID(id string) (*domain.MailTemplate, bool) {
	id = strings.TrimSpace(id)
	for _, t := range defaultTemplates {
		if t.TemplateID == id {
			copy := *t
			return &copy, true
		}
	}
	return nil, false
}

func mailTemplateForCategory(category domain.Category) *domain.MailTemplate {
	for _, t := range defaultTemplates {
		if t.TemplateID == string(category) {
			return t
		}
	}
	return &domain.MailTemplate{
		TemplateID:   string(category),
		Subject:      fmt.Sprintf("Notification: %s", category.String()),
		BodyTemplate: "<html><body><p>{{.Title}}</p><p>{{.Body}}</p></body></html>",
		Channel:      "email",
		IsActive:     true,
	}
}

func renderTemplate(body string, vars map[string]string) (string, error) {
	tpl, err := template.New("email").Parse(body)
	if err != nil {
		return "", err
	}
	var buf bytes.Buffer
	if err := tpl.Execute(&buf, vars); err != nil {
		return "", err
	}
	return buf.String(), nil
}
