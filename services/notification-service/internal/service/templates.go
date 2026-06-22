package service

import (
	"bytes"
	"fmt"
	"html/template"
	"strings"

	"github.com/Temych228/DocflowWeb/services/notification-service/internal/domain"
)

var defaultTemplates = []*domain.Template{
	{
		TemplateID:   "document_assigned",
		Subject:      "A new document has been assigned to you",
		BodyTemplate: "<html><body><h3>Hello {{.UserName}}</h3><p>A new document has been assigned to you.</p><p><b>Title:</b> {{.Title}}</p><p><b>Document ID:</b> {{.RefID}}</p></body></html>",
		IsActive:     true,
	},
	{
		TemplateID:   "status_changed",
		Subject:      "Document status changed",
		BodyTemplate: "<html><body><h3>Hello {{.UserName}}</h3><p>The document status has changed.</p><p><b>Title:</b> {{.Title}}</p><p><b>New status:</b> {{.Status}}</p></body></html>",
		IsActive:     true,
	},
	{
		TemplateID:   "deadline_reminder",
		Subject:      "Deadline reminder",
		BodyTemplate: "<html><body><h3>Hello {{.UserName}}</h3><p>This is a deadline reminder.</p><p><b>Title:</b> {{.Title}}</p><p><b>Deadline:</b> {{.Deadline}}</p></body></html>",
		IsActive:     true,
	},
	{
		TemplateID:   "overdue",
		Subject:      "Item is overdue",
		BodyTemplate: "<html><body><h3>Hello {{.UserName}}</h3><p>An item is overdue.</p><p><b>Title:</b> {{.Title}}</p><p><b>Reference:</b> {{.RefID}}</p></body></html>",
		IsActive:     true,
	},
	{
		TemplateID:   "task_assigned",
		Subject:      "A new task has been assigned to you",
		BodyTemplate: "<html><body><h3>Hello {{.UserName}}</h3><p>A new task has been assigned to you.</p><p><b>Title:</b> {{.Title}}</p><p><b>Task ID:</b> {{.RefID}}</p></body></html>",
		IsActive:     true,
	},
}

func defaultTemplateByID(id string) (*domain.Template, bool) {
	id = strings.TrimSpace(id)
	for _, t := range defaultTemplates {
		if t.TemplateID == id {
			copy := *t
			return &copy, true
		}
	}
	return nil, false
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

func buildVars(userName, title, refID, refType, status, deadline string) map[string]string {
	return map[string]string{
		"UserName": userName,
		"Title":    title,
		"RefID":    refID,
		"RefType":  refType,
		"Status":   status,
		"Deadline": deadline,
	}
}

func templateVarsFromNotification(n *domain.Notification) map[string]string {
	return buildVars("", n.Title, n.RefID, n.RefType, string(n.Category), "")
}

func templateIDForCategory(category domain.Category) string {
	switch category {
	case domain.CategoryDocumentAssigned:
		return "document_assigned"
	case domain.CategoryStatusChanged:
		return "status_changed"
	case domain.CategoryDeadlineReminder:
		return "deadline_reminder"
	case domain.CategoryOverdue:
		return "overdue"
	case domain.CategoryTaskAssigned:
		return "task_assigned"
	default:
		return "overdue"
	}
}

func notificationTemplateForCategory(category domain.Category) *domain.Template {
	id := templateIDForCategory(category)
	if t, ok := defaultTemplateByID(id); ok {
		return t
	}
	return &domain.Template{
		TemplateID:   id,
		Subject:      fmt.Sprintf("Notification: %s", id),
		BodyTemplate: "<html><body><p>{{.Title}}</p></body></html>",
		IsActive:     true,
	}
}
