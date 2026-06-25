package handlers

import (
	"net/http"
	"time"

	"github.com/Temych228/DocflowWeb/api-gateway/internal/clients"
	"github.com/Temych228/DocflowWeb/api-gateway/internal/middleware"
	"github.com/Temych228/DocflowWeb/api-gateway/pkg/dto"
	docpb "github.com/Temych228/docflow-protos-final/document/v1"
	"github.com/gin-gonic/gin"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type DocumentHandler struct {
	docClient *clients.DocumentClient
}

func NewDocumentHandler(docClient *clients.DocumentClient) *DocumentHandler {
	return &DocumentHandler{docClient: docClient}
}

func (h *DocumentHandler) ListDocuments(c *gin.Context) {
	page := parseQueryInt(c, "page", 1)
	pageSize := parseQueryInt(c, "page_size", 10)

	resp, err := h.docClient.ListDocuments(c.Request.Context(), int32(page), int32(pageSize))
	if err != nil {
		c.JSON(500, dto.NewErrorResponse(dto.ErrorInternal, grpcErrMessage(err), nil))
		return
	}

	c.JSON(200, dto.NewSuccessResponse(gin.H{"documents": resp.Documents, "total": resp.Total}))
}

func (h *DocumentHandler) CreateDocument(c *gin.Context) {
	creatorID := middleware.ExtractUserID(c)
	if creatorID == "" {
		c.JSON(401, dto.NewErrorResponse(dto.ErrorUnauthorized, "User ID not found", nil))
		return
	}

	var req CreateDocumentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, dto.NewErrorResponse(dto.ErrorValidation, err.Error(), nil))
		return
	}

	docType := req.DocType
	if docType == "" {
		docType = req.Type
	}
	if docType == "" {
		c.JSON(400, dto.NewErrorResponse(dto.ErrorValidation, "doc_type is required", nil))
		return
	}

	var deadline *timestamppb.Timestamp
	if req.Deadline != "" {
		if t, err := time.Parse(time.RFC3339, req.Deadline); err == nil {
			deadline = timestamppb.New(t)
		}
	}

	resp, err := h.docClient.Client.CreateDocument(c.Request.Context(), &docpb.CreateDocumentRequest{
		Title:       req.Title,
		Description: req.Description,
		DocType:     docType,
		CreatorId:   creatorID,
		Deadline:    deadline,
	})
	if err != nil {
		c.JSON(500, dto.NewErrorResponse(dto.ErrorInternal, grpcErrMessage(err), nil))
		return
	}

	c.JSON(201, dto.NewSuccessResponse(resp.Document))
}

func (h *DocumentHandler) GetDocument(c *gin.Context) {
	docID := c.Param("id")

	resp, err := h.docClient.Client.GetDocument(c.Request.Context(), &docpb.GetDocumentRequest{Id: docID})
	if err != nil {
		c.JSON(404, dto.NewErrorResponse(dto.ErrorNotFound, "Document not found", nil))
		return
	}

	c.JSON(200, dto.NewSuccessResponse(resp.Document))
}

func (h *DocumentHandler) UpdateDocument(c *gin.Context) {
	docID := c.Param("id")
	userID := middleware.ExtractUserID(c)

	var req UpdateDocumentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, dto.NewErrorResponse(dto.ErrorValidation, err.Error(), nil))
		return
	}

	var deadline *timestamppb.Timestamp
	if req.Deadline != "" {
		if t, err := time.Parse(time.RFC3339, req.Deadline); err == nil {
			deadline = timestamppb.New(t)
		}
	}

	resp, err := h.docClient.Client.UpdateDocument(c.Request.Context(), &docpb.UpdateDocumentRequest{
		Id:          docID,
		Title:       req.Title,
		Description: req.Description,
		DocType:     req.DocType,
		Deadline:    deadline,
		UpdatedBy:   userID,
	})
	if err != nil {
		c.JSON(500, dto.NewErrorResponse(dto.ErrorInternal, grpcErrMessage(err), nil))
		return
	}

	c.JSON(200, dto.NewSuccessResponse(resp.Document))
}

func (h *DocumentHandler) DeleteDocument(c *gin.Context) {
	docID := c.Param("id")
	userID := middleware.ExtractUserID(c)

	_, err := h.docClient.Client.DeleteDocument(c.Request.Context(), &docpb.DeleteDocumentRequest{
		Id:        docID,
		DeletedBy: userID,
	})
	if err != nil {
		c.JSON(500, dto.NewErrorResponse(dto.ErrorInternal, grpcErrMessage(err), nil))
		return
	}

	c.JSON(200, dto.NewSuccessResponse(gin.H{"deleted": true}))
}

func (h *DocumentHandler) AssignResponsible(c *gin.Context) {
	docID := c.Param("id")
	userID := middleware.ExtractUserID(c)

	var req AssignRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, dto.NewErrorResponse(dto.ErrorValidation, err.Error(), nil))
		return
	}

	resp, err := h.docClient.Client.AssignResponsible(c.Request.Context(), &docpb.AssignResponsibleRequest{
		DocumentId:    docID,
		ResponsibleId: req.ResponsibleID,
		AssignedBy:    userID,
	})
	if err != nil {
		c.JSON(500, dto.NewErrorResponse(dto.ErrorInternal, grpcErrMessage(err), nil))
		return
	}

	c.JSON(200, dto.NewSuccessResponse(resp.Document))
}

func (h *DocumentHandler) ChangeStatus(c *gin.Context) {
	docID := c.Param("id")
	userID := middleware.ExtractUserID(c)

	var req ChangeStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, dto.NewErrorResponse(dto.ErrorValidation, err.Error(), nil))
		return
	}

	statusMap := map[string]docpb.DocumentStatus{
		"draft":       docpb.DocumentStatus_DOCUMENT_STATUS_DRAFT,
		"assigned":    docpb.DocumentStatus_DOCUMENT_STATUS_ASSIGNED,
		"in_progress": docpb.DocumentStatus_DOCUMENT_STATUS_IN_PROGRESS,
		"completed":   docpb.DocumentStatus_DOCUMENT_STATUS_COMPLETED,
		"overdue":     docpb.DocumentStatus_DOCUMENT_STATUS_OVERDUE,
		"archived":    docpb.DocumentStatus_DOCUMENT_STATUS_ARCHIVED,
	}
	status, ok := statusMap[req.Status]
	if !ok {
		c.JSON(400, dto.NewErrorResponse(dto.ErrorValidation, "Invalid status value", nil))
		return
	}

	resp, err := h.docClient.Client.ChangeStatus(c.Request.Context(), &docpb.ChangeStatusRequest{
		DocumentId: docID,
		NewStatus:  status,
		ChangedBy:  userID,
	})
	if err != nil {
		c.JSON(500, dto.NewErrorResponse(dto.ErrorInternal, grpcErrMessage(err), nil))
		return
	}

	c.JSON(200, dto.NewSuccessResponse(resp.Document))
}

func (h *DocumentHandler) ArchiveDocument(c *gin.Context) {
	docID := c.Param("id")
	userID := middleware.ExtractUserID(c)

	_, err := h.docClient.Client.ArchiveDocument(c.Request.Context(), &docpb.ArchiveDocumentRequest{
		Id:         docID,
		ArchivedBy: userID,
	})
	if err != nil {
		c.JSON(500, dto.NewErrorResponse(dto.ErrorInternal, grpcErrMessage(err), nil))
		return
	}

	c.JSON(200, dto.NewSuccessResponse(gin.H{"archived": true}))
}

func (h *DocumentHandler) GetDocumentHistory(c *gin.Context) {
	docID := c.Param("id")

	resp, err := h.docClient.Client.GetDocumentHistory(c.Request.Context(), &docpb.GetDocumentHistoryRequest{
		DocumentId: docID,
	})
	if err != nil {
		c.JSON(500, dto.NewErrorResponse(dto.ErrorInternal, grpcErrMessage(err), nil))
		return
	}

	c.JSON(200, dto.NewSuccessResponse(gin.H{"entries": resp.Entries}))
}

func (h *DocumentHandler) FilterDocuments(c *gin.Context) {
	var req FilterDocumentsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, dto.NewErrorResponse(dto.ErrorValidation, err.Error(), nil))
		return
	}

	statusMap := map[string]docpb.DocumentStatus{
		"draft":       docpb.DocumentStatus_DOCUMENT_STATUS_DRAFT,
		"assigned":    docpb.DocumentStatus_DOCUMENT_STATUS_ASSIGNED,
		"in_progress": docpb.DocumentStatus_DOCUMENT_STATUS_IN_PROGRESS,
		"completed":   docpb.DocumentStatus_DOCUMENT_STATUS_COMPLETED,
		"overdue":     docpb.DocumentStatus_DOCUMENT_STATUS_OVERDUE,
		"archived":    docpb.DocumentStatus_DOCUMENT_STATUS_ARCHIVED,
	}

	pbStatus := docpb.DocumentStatus_DOCUMENT_STATUS_UNSPECIFIED
	if req.Status != "" {
		if s, ok := statusMap[req.Status]; ok {
			pbStatus = s
		}
	}

	resp, err := h.docClient.Client.FilterDocuments(c.Request.Context(), &docpb.FilterDocumentsRequest{
		Status:        pbStatus,
		DocType:       req.DocType,
		ResponsibleId: req.ResponsibleID,
		CreatorId:     req.CreatorID,
		Page:          req.Page,
		PageSize:      req.PageSize,
	})
	if err != nil {
		c.JSON(500, dto.NewErrorResponse(dto.ErrorInternal, grpcErrMessage(err), nil))
		return
	}

	c.JSON(200, dto.NewSuccessResponse(gin.H{"documents": resp.Documents, "total": resp.Total}))
}

func (h *DocumentHandler) MarkOverdue(c *gin.Context) {
	resp, err := h.docClient.Client.MarkOverdue(c.Request.Context(), &docpb.MarkOverdueRequest{})
	if err != nil {
		c.JSON(500, dto.NewErrorResponse(dto.ErrorInternal, grpcErrMessage(err), nil))
		return
	}

	c.JSON(200, dto.NewSuccessResponse(gin.H{"marked_count": resp.MarkedCount}))
}

func (h *DocumentHandler) ExportCSV(c *gin.Context) {
	statusStr := c.DefaultQuery("status", "")
	docType := c.DefaultQuery("doc_type", "")

	statusMap := map[string]docpb.DocumentStatus{
		"draft":       docpb.DocumentStatus_DOCUMENT_STATUS_DRAFT,
		"assigned":    docpb.DocumentStatus_DOCUMENT_STATUS_ASSIGNED,
		"in_progress": docpb.DocumentStatus_DOCUMENT_STATUS_IN_PROGRESS,
		"completed":   docpb.DocumentStatus_DOCUMENT_STATUS_COMPLETED,
		"overdue":     docpb.DocumentStatus_DOCUMENT_STATUS_OVERDUE,
		"archived":    docpb.DocumentStatus_DOCUMENT_STATUS_ARCHIVED,
	}

	pbStatus := docpb.DocumentStatus_DOCUMENT_STATUS_UNSPECIFIED
	if s, ok := statusMap[statusStr]; ok {
		pbStatus = s
	}

	resp, err := h.docClient.Client.ExportDocumentsCSV(c.Request.Context(), &docpb.ExportCSVRequest{
		Status:  pbStatus,
		DocType: docType,
	})
	if err != nil {
		c.JSON(500, dto.NewErrorResponse(dto.ErrorInternal, grpcErrMessage(err), nil))
		return
	}

	filename := resp.Filename
	if filename == "" {
		filename = "documents_export.csv"
	}

	c.Header("Content-Disposition", "attachment; filename="+filename)
	c.Data(http.StatusOK, "text/csv; charset=utf-8", resp.Data)
}

type CreateDocumentRequest struct {
	Title         string   `json:"title" binding:"required"`
	Description   string   `json:"description"`
	DocType       string   `json:"doc_type,omitempty"`
	Type          string   `json:"type,omitempty"`
	ResponsibleID string   `json:"responsible_id,omitempty"`
	FileURL       string   `json:"file_url" binding:"required"`
	Tags          []string `json:"tags,omitempty"`
	Deadline      string   `json:"deadline,omitempty"`
}

type UpdateDocumentRequest struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	DocType     string `json:"doc_type"`
	Deadline    string `json:"deadline,omitempty"`
}

type FilterDocumentsRequest struct {
	Status        string `json:"status,omitempty"`
	DocType       string `json:"doc_type,omitempty"`
	ResponsibleID string `json:"responsible_id,omitempty"`
	CreatorID     string `json:"creator_id,omitempty"`
	Page          int32  `json:"page,omitempty"`
	PageSize      int32  `json:"page_size,omitempty"`
}

type AssignRequest struct {
	ResponsibleID string `json:"responsible_id" binding:"required"`
}

type ChangeStatusRequest struct {
	Status string `json:"status" binding:"required"`
}
