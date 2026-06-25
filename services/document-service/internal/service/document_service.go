package service

import (
	"bytes"
	"context"
	"encoding/csv"
	"fmt"
	"strings"
	"time"

	"github.com/Temych228/DocflowWeb/services/document-service/internal/clients"
	"github.com/Temych228/DocflowWeb/services/document-service/internal/domain"
	"github.com/Temych228/DocflowWeb/services/document-service/internal/repository"
)

type documentService struct {
	documents    repository.DocumentRepository
	history      repository.HistoryRepository
	notification clients.NotificationClient
	calendar     clients.CalendarClient
}

func New(
	documents repository.DocumentRepository,
	history repository.HistoryRepository,
	notification clients.NotificationClient,
	calendar clients.CalendarClient,
) DocumentService {
	return &documentService{
		documents:    documents,
		history:      history,
		notification: notification,
		calendar:     calendar,
	}
}

func (s *documentService) CreateDocument(ctx context.Context, input domain.CreateDocumentInput) (*domain.Document, error) {
	if err := input.Validate(); err != nil {
		return nil, err
	}

	tags := input.Tags
	if tags == nil {
		tags = []string{}
	}

	doc := &domain.Document{
		Title:         input.Title,
		Description:   input.Description,
		Type:          input.Type,
		Status:        domain.StatusDraft,
		CreatorID:     input.CreatorID,
		ResponsibleID: input.ResponsibleID,
		Deadline:      input.Deadline,
		FileURL:       input.FileURL,
		Tags:          tags,
	}

	if err := s.documents.Create(ctx, doc); err != nil {
		return nil, fmt.Errorf("create document: %w", err)
	}

	entry := &domain.HistoryEntry{
		DocumentID: doc.ID,
		ChangedBy:  input.CreatorID,
		Field:      "status",
		OldValue:   "",
		NewValue:   string(doc.Status),
	}
	if err := s.history.Create(ctx, entry); err != nil {
		return nil, fmt.Errorf("record history: %w", err)
	}

	s.notifyDocumentEvent(ctx, doc, "new_document", "New document created",
		fmt.Sprintf("Document %q was created", doc.Title))

	if doc.Deadline != nil {
		s.createCalendarEvent(ctx, doc, "document_due")
	}

	return doc, nil
}

func (s *documentService) GetDocument(ctx context.Context, id string) (*domain.Document, error) {
	if id == "" {
		return nil, domain.ErrInvalidInput
	}
	return s.documents.GetByID(ctx, id)
}

func (s *documentService) UpdateDocument(ctx context.Context, input domain.UpdateDocumentInput) (*domain.Document, error) {
	if err := input.Validate(); err != nil {
		return nil, err
	}

	doc, err := s.documents.GetByID(ctx, input.ID)
	if err != nil {
		return nil, err
	}

	if doc.Status == domain.StatusArchived {
		return nil, domain.ErrForbidden
	}

	if input.Title != nil {
		doc.Title = *input.Title
	}
	if input.Description != nil {
		doc.Description = *input.Description
	}
	if input.Type != nil {
		doc.Type = *input.Type
	}
	if input.Deadline != nil {
		doc.Deadline = input.Deadline
	}

	if err := s.documents.Update(ctx, doc); err != nil {
		return nil, fmt.Errorf("update document: %w", err)
	}

	entry := &domain.HistoryEntry{
		DocumentID: doc.ID,
		ChangedBy:  input.ActorID,
		Field:      "document",
		OldValue:   "",
		NewValue:   "fields updated",
	}
	if err := s.history.Create(ctx, entry); err != nil {
		return nil, fmt.Errorf("record history: %w", err)
	}

	if input.Deadline != nil {
		s.createCalendarEvent(ctx, doc, "document_due")
	}

	return doc, nil
}

func (s *documentService) DeleteDocument(ctx context.Context, id, actorID string) error {
	if id == "" || actorID == "" {
		return domain.ErrInvalidInput
	}

	doc, err := s.documents.GetByID(ctx, id)
	if err != nil {
		return err
	}

	if doc.Status != domain.StatusDraft {
		return domain.ErrForbidden
	}

	return s.documents.Delete(ctx, id)
}

func (s *documentService) ListDocuments(ctx context.Context, filter domain.ListFilter) ([]*domain.Document, int, error) {
	filter.Normalize()
	return s.documents.List(ctx, filter)
}

func (s *documentService) AssignResponsible(ctx context.Context, id, responsibleID, actorID string) (*domain.Document, error) {
	if id == "" || responsibleID == "" || actorID == "" {
		return nil, domain.ErrInvalidInput
	}

	doc, err := s.documents.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if doc.Status == domain.StatusArchived {
		return nil, domain.ErrForbidden
	}

	oldResponsible := ""
	if doc.ResponsibleID != nil {
		oldResponsible = *doc.ResponsibleID
	}

	if err := s.documents.SetResponsible(ctx, id, responsibleID); err != nil {
		return nil, fmt.Errorf("assign responsible: %w", err)
	}
	doc.ResponsibleID = &responsibleID

	entry := &domain.HistoryEntry{
		DocumentID: id,
		ChangedBy:  actorID,
		Field:      "responsible_id",
		OldValue:   oldResponsible,
		NewValue:   responsibleID,
	}
	if err := s.history.Create(ctx, entry); err != nil {
		return nil, fmt.Errorf("record history: %w", err)
	}

	if doc.Status == domain.StatusDraft {
		oldStatus := doc.Status
		if err := s.documents.UpdateStatus(ctx, id, domain.StatusAssigned); err != nil {
			return nil, fmt.Errorf("transition to assigned: %w", err)
		}
		doc.Status = domain.StatusAssigned

		statusEntry := &domain.HistoryEntry{
			DocumentID: id,
			ChangedBy:  actorID,
			Field:      "status",
			OldValue:   string(oldStatus),
			NewValue:   string(doc.Status),
		}
		if err := s.history.Create(ctx, statusEntry); err != nil {
			return nil, fmt.Errorf("record history: %w", err)
		}
	}

	s.notifyDocumentEvent(ctx, doc, "assigned", "Document assigned to you",
		fmt.Sprintf("You were assigned as responsible for document %q", doc.Title))

	return doc, nil
}

func (s *documentService) ChangeStatus(ctx context.Context, id string, newStatus domain.DocumentStatus, actorID, comment string) (*domain.Document, error) {
	if id == "" || actorID == "" {
		return nil, domain.ErrInvalidInput
	}
	if !newStatus.Valid() {
		return nil, domain.ErrInvalidInput
	}

	doc, err := s.documents.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if !domain.CanTransition(doc.Status, newStatus) {
		return nil, domain.ErrInvalidTransition
	}

	oldStatus := doc.Status

	if newStatus == domain.StatusArchived {
		if err := s.documents.Archive(ctx, id); err != nil {
			return nil, fmt.Errorf("archive document: %w", err)
		}
		doc.Status = domain.StatusArchived
		now := time.Now()
		doc.ArchivedAt = &now
	} else {
		if err := s.documents.UpdateStatus(ctx, id, newStatus); err != nil {
			return nil, fmt.Errorf("change status: %w", err)
		}
		doc.Status = newStatus
	}

	if newStatus != domain.StatusOverdue {
		doc.IsOverdue = false
	}

	entry := &domain.HistoryEntry{
		DocumentID: id,
		ChangedBy:  actorID,
		Field:      "status",
		OldValue:   string(oldStatus),
		NewValue:   string(newStatus),
	}
	if err := s.history.Create(ctx, entry); err != nil {
		return nil, fmt.Errorf("record history: %w", err)
	}

	s.notifyDocumentEvent(ctx, doc, "status_changed",
		"Document status changed",
		fmt.Sprintf("Document %q status changed from %s to %s", doc.Title, oldStatus, newStatus))

	return doc, nil
}

func (s *documentService) ArchiveDocument(ctx context.Context, id, actorID string) (*domain.Document, error) {
	if id == "" || actorID == "" {
		return nil, domain.ErrInvalidInput
	}

	doc, err := s.documents.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if doc.Status == domain.StatusArchived {
		return doc, nil
	}

	if !domain.CanTransition(doc.Status, domain.StatusArchived) {
		return nil, domain.ErrInvalidTransition
	}

	oldStatus := doc.Status
	if err := s.documents.Archive(ctx, id); err != nil {
		return nil, fmt.Errorf("archive document: %w", err)
	}

	doc.Status = domain.StatusArchived
	now := time.Now()
	doc.ArchivedAt = &now

	entry := &domain.HistoryEntry{
		DocumentID: id,
		ChangedBy:  actorID,
		Field:      "status",
		OldValue:   string(oldStatus),
		NewValue:   string(domain.StatusArchived),
	}
	if err := s.history.Create(ctx, entry); err != nil {
		return nil, fmt.Errorf("record history: %w", err)
	}

	return doc, nil
}

func (s *documentService) GetDocumentHistory(ctx context.Context, filter domain.HistoryFilter) ([]*domain.HistoryEntry, int, error) {
	if filter.DocumentID == "" {
		return nil, 0, domain.ErrInvalidInput
	}
	filter.Normalize()
	return s.history.List(ctx, filter)
}

func (s *documentService) FilterDocuments(ctx context.Context, filter domain.AdvancedFilter) ([]*domain.Document, int, error) {
	filter.Normalize()
	return s.documents.Filter(ctx, filter)
}

func (s *documentService) MarkOverdue(ctx context.Context) ([]string, error) {
	docs, err := s.documents.FindOverdue(ctx)
	if err != nil {
		return nil, fmt.Errorf("find overdue documents: %w", err)
	}
	if len(docs) == 0 {
		return nil, nil
	}

	ids := make([]string, len(docs))
	for i, doc := range docs {
		ids[i] = doc.ID
	}

	if err := s.documents.MarkOverdueByIDs(ctx, ids); err != nil {
		return nil, fmt.Errorf("mark documents overdue: %w", err)
	}

	for _, doc := range docs {
		oldStatus := doc.Status
		newStatus := domain.StatusOverdue

		entry := &domain.HistoryEntry{
			DocumentID: doc.ID,
			ChangedBy:  "00000000-0000-0000-0000-000000000000",
			Field:      "status",
			OldValue:   string(oldStatus),
			NewValue:   string(newStatus),
		}
		_ = s.history.Create(ctx, entry)

		doc.Status = newStatus
		doc.IsOverdue = true
		s.notifyDocumentEvent(ctx, doc, "overdue", "Document is overdue",
			fmt.Sprintf("Document %q has passed its deadline", doc.Title))
	}

	return ids, nil
}

func (s *documentService) ExportDocumentsCSV(ctx context.Context, filter domain.AdvancedFilter) ([]byte, int, error) {
	filter.PageSize = 10000
	filter.Page = 1
	docs, _, err := s.documents.Filter(ctx, filter)
	if err != nil {
		return nil, 0, fmt.Errorf("filter documents for export: %w", err)
	}

	var buf bytes.Buffer
	writer := csv.NewWriter(&buf)

	header := []string{
		"id", "title", "description", "type", "status",
		"creator_id", "responsible_id", "deadline", "is_overdue",
		"created_at", "updated_at",
	}
	if err := writer.Write(header); err != nil {
		return nil, 0, fmt.Errorf("write csv header: %w", err)
	}

	for _, doc := range docs {
		responsible := ""
		if doc.ResponsibleID != nil {
			responsible = *doc.ResponsibleID
		}
		deadline := ""
		if doc.Deadline != nil {
			deadline = doc.Deadline.Format(time.RFC3339)
		}

		row := []string{
			doc.ID,
			doc.Title,
			doc.Description,
			string(doc.Type),
			string(doc.Status),
			doc.CreatorID,
			responsible,
			deadline,
			fmt.Sprintf("%t", doc.IsOverdue),
			doc.CreatedAt.Format(time.RFC3339),
			doc.UpdatedAt.Format(time.RFC3339),
		}
		if err := writer.Write(row); err != nil {
			return nil, 0, fmt.Errorf("write csv row: %w", err)
		}
	}

	writer.Flush()
	if err := writer.Error(); err != nil {
		return nil, 0, fmt.Errorf("flush csv writer: %w", err)
	}

	return buf.Bytes(), len(docs), nil
}

func (s *documentService) GetDocumentStats(ctx context.Context) (*domain.DocumentStats, error) {
	return s.documents.Stats(ctx)
}

func (s *documentService) notifyDocumentEvent(ctx context.Context, doc *domain.Document, category, title, body string) {
	if s.notification == nil {
		return
	}

	recipients := map[string]struct{}{
		doc.CreatorID: {},
	}
	if doc.ResponsibleID != nil && *doc.ResponsibleID != "" {
		recipients[*doc.ResponsibleID] = struct{}{}
	}

	for userID := range recipients {
		_ = s.notification.CreateNotification(ctx, clients.CreateNotificationInput{
			UserID:        userID,
			DocumentID:    doc.ID,
			NotifCategory: category,
			Title:         title,
			Body:          body,
			RefID:         doc.ID,
			RefType:       "document",
		})
	}
}

func (s *documentService) createCalendarEvent(ctx context.Context, doc *domain.Document, eventType string) {
	if s.calendar == nil || doc.Deadline == nil {
		return
	}

	recipients := map[string]struct{}{
		doc.CreatorID: {},
	}
	if doc.ResponsibleID != nil && *doc.ResponsibleID != "" {
		recipients[*doc.ResponsibleID] = struct{}{}
	}

	for userID := range recipients {
		_ = s.calendar.CreateEvent(ctx, clients.CreateEventInput{
			UserID:      userID,
			Title:       fmt.Sprintf("Deadline: %s", doc.Title),
			Description: strings.TrimSpace(doc.Description),
			EventType:   eventType,
			StartAt:     *doc.Deadline,
			EndAt:       *doc.Deadline,
			AllDay:      false,
			RefID:       doc.ID,
			RefType:     "document",
		})
	}
}
