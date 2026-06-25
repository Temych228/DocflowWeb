package grpcserver

import (
	"context"
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"

	documentv1 "github.com/Temych228/docflow-protos-final/document/v1"

	"github.com/Temych228/DocflowWeb/services/document-service/internal/domain"
	"github.com/Temych228/DocflowWeb/services/document-service/internal/service"
	apperrors "github.com/Temych228/DocflowWeb/services/document-service/pkg/errors"
)

type Server struct {
	documentv1.UnimplementedDocumentServiceServer
	svc service.DocumentService
}

func New(svc service.DocumentService) *Server {
	return &Server{svc: svc}
}

var statusToProto = map[domain.DocumentStatus]documentv1.DocumentStatus{
	domain.StatusDraft:      documentv1.DocumentStatus_DOCUMENT_STATUS_DRAFT,
	domain.StatusAssigned:   documentv1.DocumentStatus_DOCUMENT_STATUS_ASSIGNED,
	domain.StatusInProgress: documentv1.DocumentStatus_DOCUMENT_STATUS_IN_PROGRESS,
	domain.StatusCompleted:  documentv1.DocumentStatus_DOCUMENT_STATUS_COMPLETED,
	domain.StatusOverdue:    documentv1.DocumentStatus_DOCUMENT_STATUS_OVERDUE,
	domain.StatusArchived:   documentv1.DocumentStatus_DOCUMENT_STATUS_ARCHIVED,
}

var statusFromProto = map[documentv1.DocumentStatus]domain.DocumentStatus{
	documentv1.DocumentStatus_DOCUMENT_STATUS_DRAFT:       domain.StatusDraft,
	documentv1.DocumentStatus_DOCUMENT_STATUS_ASSIGNED:    domain.StatusAssigned,
	documentv1.DocumentStatus_DOCUMENT_STATUS_IN_PROGRESS: domain.StatusInProgress,
	documentv1.DocumentStatus_DOCUMENT_STATUS_COMPLETED:   domain.StatusCompleted,
	documentv1.DocumentStatus_DOCUMENT_STATUS_OVERDUE:     domain.StatusOverdue,
	documentv1.DocumentStatus_DOCUMENT_STATUS_ARCHIVED:    domain.StatusArchived,
}

func docStatusToProto(s domain.DocumentStatus) documentv1.DocumentStatus {
	if v, ok := statusToProto[s]; ok {
		return v
	}
	return documentv1.DocumentStatus_DOCUMENT_STATUS_UNSPECIFIED
}

func docStatusFromProto(s documentv1.DocumentStatus) domain.DocumentStatus {
	if v, ok := statusFromProto[s]; ok {
		return v
	}
	return ""
}

func timeToProto(t *time.Time) *timestamppb.Timestamp {
	if t == nil {
		return nil
	}
	return timestamppb.New(*t)
}

func timeFromProto(ts *timestamppb.Timestamp) *time.Time {
	if ts == nil {
		return nil
	}
	if ts == nil {
		return nil
	}
	t := ts.AsTime()
	return &t
}

func stringPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func stringFromPtr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func documentToProto(doc *domain.Document) *documentv1.Document {
	if doc == nil {
		return nil
	}
	return &documentv1.Document{
		Id:            doc.ID,
		Title:         doc.Title,
		Description:   doc.Description,
		DocType:       string(doc.Type),
		Status:        docStatusToProto(doc.Status),
		CreatorId:     doc.CreatorID,
		ResponsibleId: stringFromPtr(doc.ResponsibleID),
		Deadline:      timeToProto(doc.Deadline),
		CreatedAt:     timeToProto(&doc.CreatedAt),
		UpdatedAt:     timeToProto(&doc.UpdatedAt),
	}
}

func documentsToProto(docs []*domain.Document) []*documentv1.Document {
	out := make([]*documentv1.Document, len(docs))
	for i, doc := range docs {
		out[i] = documentToProto(doc)
	}
	return out
}

func historyEntryToProto(entry *domain.HistoryEntry) *documentv1.DocumentHistoryEntry {
	if entry == nil {
		return nil
	}
	return &documentv1.DocumentHistoryEntry{
		Id:         entry.ID,
		DocumentId: entry.DocumentID,
		ChangedBy:  entry.ChangedBy,
		Field:      entry.Field,
		OldValue:   entry.OldValue,
		NewValue:   entry.NewValue,
		ChangedAt:  timeToProto(&entry.ChangedAt),
	}
}

func historyEntriesToProto(entries []*domain.HistoryEntry) []*documentv1.DocumentHistoryEntry {
	out := make([]*documentv1.DocumentHistoryEntry, len(entries))
	for i, entry := range entries {
		out[i] = historyEntryToProto(entry)
	}
	return out
}

func (s *Server) CreateDocument(ctx context.Context, req *documentv1.CreateDocumentRequest) (*documentv1.CreateDocumentResponse, error) {
	input := domain.CreateDocumentInput{
		Title:         req.GetTitle(),
		Description:   req.GetDescription(),
		Type:          domain.DocumentType(req.GetDocType()),
		CreatorID:     req.GetCreatorId(),
		ResponsibleID: nil,
		Deadline:      timeFromProto(req.GetDeadline()),
		FileURL:       "",
		Tags:          []string{},
	}

	doc, err := s.svc.CreateDocument(ctx, input)
	if err != nil {
		return nil, apperrors.ToGRPCError(err)
	}

	return &documentv1.CreateDocumentResponse{Document: documentToProto(doc)}, nil
}

func (s *Server) GetDocument(ctx context.Context, req *documentv1.GetDocumentRequest) (*documentv1.GetDocumentResponse, error) {
	doc, err := s.svc.GetDocument(ctx, req.GetId())
	if err != nil {
		return nil, apperrors.ToGRPCError(err)
	}

	return &documentv1.GetDocumentResponse{Document: documentToProto(doc)}, nil
}

func (s *Server) UpdateDocument(ctx context.Context, req *documentv1.UpdateDocumentRequest) (*documentv1.UpdateDocumentResponse, error) {
	input := domain.UpdateDocumentInput{
		ID:      req.GetId(),
		ActorID: req.GetUpdatedBy(),
	}

	if req.GetTitle() != "" {
		title := req.GetTitle()
		input.Title = &title
	}
	if req.GetDescription() != "" {
		description := req.GetDescription()
		input.Description = &description
	}
	if req.GetDocType() != "" {
		docType := domain.DocumentType(req.GetDocType())
		input.Type = &docType
	}
	if req.GetDeadline() != nil {
		input.Deadline = timeFromProto(req.GetDeadline())
	}

	doc, err := s.svc.UpdateDocument(ctx, input)
	if err != nil {
		return nil, apperrors.ToGRPCError(err)
	}

	return &documentv1.UpdateDocumentResponse{Document: documentToProto(doc)}, nil
}

func (s *Server) DeleteDocument(ctx context.Context, req *documentv1.DeleteDocumentRequest) (*documentv1.DeleteDocumentResponse, error) {
	if err := s.svc.DeleteDocument(ctx, req.GetId(), req.GetDeletedBy()); err != nil {
		return nil, apperrors.ToGRPCError(err)
	}

	return &documentv1.DeleteDocumentResponse{Success: true}, nil
}

func (s *Server) ListDocuments(ctx context.Context, req *documentv1.ListDocumentsRequest) (*documentv1.ListDocumentsResponse, error) {
	filter := domain.ListFilter{
		Page:     int(req.GetPage()),
		PageSize: int(req.GetPageSize()),
	}

	docs, total, err := s.svc.ListDocuments(ctx, filter)
	if err != nil {
		return nil, apperrors.ToGRPCError(err)
	}

	return &documentv1.ListDocumentsResponse{
		Documents: documentsToProto(docs),
		Total:     int32(total),
	}, nil
}

func (s *Server) AssignResponsible(ctx context.Context, req *documentv1.AssignResponsibleRequest) (*documentv1.AssignResponsibleResponse, error) {
	doc, err := s.svc.AssignResponsible(ctx, req.GetDocumentId(), req.GetResponsibleId(), req.GetAssignedBy())
	if err != nil {
		return nil, apperrors.ToGRPCError(err)
	}

	return &documentv1.AssignResponsibleResponse{Document: documentToProto(doc)}, nil
}

func (s *Server) ChangeStatus(ctx context.Context, req *documentv1.ChangeStatusRequest) (*documentv1.ChangeStatusResponse, error) {
	newStatus := docStatusFromProto(req.GetNewStatus())

	doc, err := s.svc.ChangeStatus(ctx, req.GetDocumentId(), newStatus, req.GetChangedBy(), "")
	if err != nil {
		return nil, apperrors.ToGRPCError(err)
	}

	return &documentv1.ChangeStatusResponse{Document: documentToProto(doc)}, nil
}

func (s *Server) ArchiveDocument(ctx context.Context, req *documentv1.ArchiveDocumentRequest) (*documentv1.ArchiveDocumentResponse, error) {
	_, err := s.svc.ArchiveDocument(ctx, req.GetId(), req.GetArchivedBy())
	if err != nil {
		return nil, apperrors.ToGRPCError(err)
	}

	return &documentv1.ArchiveDocumentResponse{Success: true}, nil
}

func (s *Server) GetDocumentHistory(ctx context.Context, req *documentv1.GetDocumentHistoryRequest) (*documentv1.GetDocumentHistoryResponse, error) {
	filter := domain.HistoryFilter{
		DocumentID: req.GetDocumentId(),
		Page:       1,
		PageSize:   100,
	}

	entries, _, err := s.svc.GetDocumentHistory(ctx, filter)
	if err != nil {
		return nil, apperrors.ToGRPCError(err)
	}

	return &documentv1.GetDocumentHistoryResponse{
		Entries: historyEntriesToProto(entries),
	}, nil
}

func (s *Server) FilterDocuments(ctx context.Context, req *documentv1.FilterDocumentsRequest) (*documentv1.FilterDocumentsResponse, error) {
	filter := domain.AdvancedFilter{
		Status:        docStatusFromProto(req.GetStatus()),
		Type:          domain.DocumentType(req.GetDocType()),
		ResponsibleID: req.GetResponsibleId(),
		CreatorID:     req.GetCreatorId(),
		DeadlineFrom:  timeFromProto(req.GetDeadlineFrom()),
		DeadlineTo:    timeFromProto(req.GetDeadlineTo()),
		Page:          int(req.GetPage()),
		PageSize:      int(req.GetPageSize()),
	}

	docs, total, err := s.svc.FilterDocuments(ctx, filter)
	if err != nil {
		return nil, apperrors.ToGRPCError(err)
	}

	return &documentv1.FilterDocumentsResponse{
		Documents: documentsToProto(docs),
		Total:     int32(total),
	}, nil
}

func (s *Server) MarkOverdue(ctx context.Context, _ *documentv1.MarkOverdueRequest) (*documentv1.MarkOverdueResponse, error) {
	ids, err := s.svc.MarkOverdue(ctx)
	if err != nil {
		return nil, apperrors.ToGRPCError(err)
	}

	return &documentv1.MarkOverdueResponse{
		MarkedCount: int32(len(ids)),
	}, nil
}

func (s *Server) ExportDocumentsCSV(ctx context.Context, req *documentv1.ExportCSVRequest) (*documentv1.ExportCSVResponse, error) {
	filter := domain.AdvancedFilter{
		Status: docStatusFromProto(req.GetStatus()),
		Type:   domain.DocumentType(req.GetDocType()),
	}

	data, _, err := s.svc.ExportDocumentsCSV(ctx, filter)
	if err != nil {
		return nil, apperrors.ToGRPCError(err)
	}

	return &documentv1.ExportCSVResponse{
		Data:     data,
		Filename: "documents_export_" + time.Now().Format("20060102_150405") + ".csv",
	}, nil
}

func (s *Server) GetDocumentStats(ctx context.Context, _ *documentv1.GetDocumentStatsRequest) (*documentv1.GetDocumentStatsResponse, error) {
	stats, err := s.svc.GetDocumentStats(ctx)
	if err != nil {
		return nil, apperrors.ToGRPCError(err)
	}

	return &documentv1.GetDocumentStatsResponse{
		Stats: &documentv1.DocumentStats{
			Total:      int32(stats.Total),
			Draft:      int32(stats.Draft),
			Assigned:   int32(stats.Assigned),
			InProgress: int32(stats.InProgress),
			Completed:  int32(stats.Completed),
			Overdue:    int32(stats.Overdue),
			Archived:   int32(stats.Archived),
		},
	}, nil
}
