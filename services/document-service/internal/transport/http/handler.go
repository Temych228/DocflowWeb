package http

import (
	"net/http"
	"strconv"
	"time"

	"github.com/Temych228/DocflowWeb/services/document-service/internal/domain"
	"github.com/Temych228/DocflowWeb/services/document-service/internal/service"
	"github.com/gin-gonic/gin"
)

type Handler struct {
	docService service.DocumentService
}

func New(docService service.DocumentService) *Handler {
	return &Handler{docService: docService}
}

func (h *Handler) Register(r *gin.Engine) {
	docs := r.Group("/documents")
	{
		docs.POST("", h.createDocument)
		docs.GET("", h.listDocuments)
		docs.GET("/:id", h.getDocument)
		docs.PUT("/:id", h.updateDocument)
		docs.DELETE("/:id", h.deleteDocument)
		docs.POST("/:id/assign", h.assignResponsible)
		docs.PATCH("/:id/status", h.changeStatus)
		docs.POST("/:id/archive", h.archiveDocument)
		docs.GET("/:id/history", h.getDocumentHistory)
		docs.POST("/filter", h.filterDocuments)
		docs.POST("/mark-overdue", h.markOverdue)
		docs.GET("/export/csv", h.exportDocumentsCSV)
		docs.GET("/stats", h.getDocumentStats)
	}
}

func (h *Handler) createDocument(c *gin.Context) {
	var req struct {
		Title         string     `json:"title" binding:"required"`
		Description   string     `json:"description"`
		Type          string     `json:"type" binding:"required"`
		CreatorID     string     `json:"creator_id" binding:"required"`
		ResponsibleID *string    `json:"responsible_id"`
		Deadline      *time.Time `json:"deadline"`
		FileURL       string     `json:"file_url" binding:"required"`
		Tags          []string   `json:"tags"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	doc, err := h.docService.CreateDocument(c.Request.Context(), domain.CreateDocumentInput{
		Title:         req.Title,
		Description:   req.Description,
		Type:          domain.DocumentType(req.Type),
		CreatorID:     req.CreatorID,
		ResponsibleID: req.ResponsibleID,
		Deadline:      req.Deadline,
		FileURL:       req.FileURL,
		Tags:          req.Tags,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, doc)
}

func (h *Handler) listDocuments(c *gin.Context) {
	page := parseQuery(c, "page", 1)
	pageSize := parseQuery(c, "page_size", 10)
	creatorID := c.Query("creator_id")
	responsibleID := c.Query("responsible_id")

	docs, total, err := h.docService.ListDocuments(c.Request.Context(), domain.ListFilter{
		Page:          page,
		PageSize:      pageSize,
		CreatorID:     creatorID,
		ResponsibleID: responsibleID,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"documents": docs, "total": total})
}

func (h *Handler) getDocument(c *gin.Context) {
	docID := c.Param("id")

	doc, err := h.docService.GetDocument(c.Request.Context(), docID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Document not found"})
		return
	}

	c.JSON(http.StatusOK, doc)
}

func (h *Handler) updateDocument(c *gin.Context) {
	docID := c.Param("id")

	var req struct {
		Title       *string    `json:"title"`
		Description *string    `json:"description"`
		Type        *string    `json:"type"`
		Deadline    *time.Time `json:"deadline"`
		FileURL     *string    `json:"file_url"`
		Tags        []string   `json:"tags"`
		ActorID     string     `json:"actor_id" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var docType *domain.DocumentType
	if req.Type != nil {
		dt := domain.DocumentType(*req.Type)
		docType = &dt
	}

	doc, err := h.docService.UpdateDocument(c.Request.Context(), domain.UpdateDocumentInput{
		ID:          docID,
		Title:       req.Title,
		Description: req.Description,
		Type:        docType,
		Deadline:    req.Deadline,
		FileURL:     req.FileURL,
		Tags:        req.Tags,
		ActorID:     req.ActorID,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, doc)
}

func (h *Handler) deleteDocument(c *gin.Context) {
	docID := c.Param("id")

	var req struct {
		ActorID string `json:"actor_id" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.docService.DeleteDocument(c.Request.Context(), docID, req.ActorID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Document deleted"})
}

func (h *Handler) assignResponsible(c *gin.Context) {
	docID := c.Param("id")

	var req struct {
		ResponsibleID string `json:"responsible_id" binding:"required"`
		ActorID       string `json:"actor_id" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	doc, err := h.docService.AssignResponsible(c.Request.Context(), docID, req.ResponsibleID, req.ActorID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, doc)
}

func (h *Handler) changeStatus(c *gin.Context) {
	docID := c.Param("id")

	var req struct {
		NewStatus string `json:"new_status" binding:"required"`
		ActorID   string `json:"actor_id" binding:"required"`
		Comment   string `json:"comment"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	doc, err := h.docService.ChangeStatus(c.Request.Context(), docID, domain.DocumentStatus(req.NewStatus), req.ActorID, req.Comment)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, doc)
}

func (h *Handler) archiveDocument(c *gin.Context) {
	docID := c.Param("id")

	var req struct {
		ActorID string `json:"actor_id" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	doc, err := h.docService.ArchiveDocument(c.Request.Context(), docID, req.ActorID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, doc)
}

func (h *Handler) getDocumentHistory(c *gin.Context) {
	docID := c.Param("id")
	page := parseQuery(c, "page", 1)
	pageSize := parseQuery(c, "page_size", 10)

	history, total, err := h.docService.GetDocumentHistory(c.Request.Context(), domain.HistoryFilter{
		DocumentID: docID,
		Page:       page,
		PageSize:   pageSize,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"entries": history, "total": total})
}

func (h *Handler) filterDocuments(c *gin.Context) {
	var req struct {
		Status        string     `json:"status"`
		Type          string     `json:"type"`
		ResponsibleID string     `json:"responsible_id"`
		CreatorID     string     `json:"creator_id"`
		DeadlineFrom  *time.Time `json:"deadline_from"`
		DeadlineTo    *time.Time `json:"deadline_to"`
		Page          int        `json:"page"`
		PageSize      int        `json:"page_size"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.Page == 0 {
		req.Page = 1
	}
	if req.PageSize == 0 {
		req.PageSize = 10
	}

	docs, total, err := h.docService.FilterDocuments(c.Request.Context(), domain.AdvancedFilter{
		Status:        domain.DocumentStatus(req.Status),
		Type:          domain.DocumentType(req.Type),
		ResponsibleID: req.ResponsibleID,
		CreatorID:     req.CreatorID,
		DeadlineFrom:  req.DeadlineFrom,
		DeadlineTo:    req.DeadlineTo,
		Page:          req.Page,
		PageSize:      req.PageSize,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"documents": docs, "total": total})
}

func (h *Handler) markOverdue(c *gin.Context) {
	docs, err := h.docService.MarkOverdue(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"marked_documents": docs, "count": len(docs)})
}

func (h *Handler) exportDocumentsCSV(c *gin.Context) {
	var filter domain.AdvancedFilter
	status := c.Query("status")
	docType := c.Query("type")
	responsibleID := c.Query("responsible_id")
	creatorID := c.Query("creator_id")

	if status != "" {
		filter.Status = domain.DocumentStatus(status)
	}
	if docType != "" {
		filter.Type = domain.DocumentType(docType)
	}
	filter.ResponsibleID = responsibleID
	filter.CreatorID = creatorID

	data, _, err := h.docService.ExportDocumentsCSV(c.Request.Context(), filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Header("Content-Type", "text/csv")
	c.Header("Content-Disposition", "attachment; filename=documents.csv")
	c.Data(http.StatusOK, "text/csv; charset=utf-8", data)
}

func (h *Handler) getDocumentStats(c *gin.Context) {
	stats, err := h.docService.GetDocumentStats(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, stats)
}

func parseQuery(c *gin.Context, key string, defaultVal int) int {
	val := c.Query(key)
	if val == "" {
		return defaultVal
	}
	parsed, err := strconv.Atoi(val)
	if err != nil {
		return defaultVal
	}
	return parsed
}
