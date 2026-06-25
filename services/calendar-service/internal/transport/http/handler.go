package http

import (
	"net/http"
	"strconv"
	"time"

	"github.com/Temych228/DocflowWeb/services/calendar-service/internal/domain"
	"github.com/Temych228/DocflowWeb/services/calendar-service/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

type Handler struct {
	svc   service.CalendarService
	db    *pgxpool.Pool
	cache *redis.Client
}

func New(svc service.CalendarService, db *pgxpool.Pool, cache *redis.Client) *Handler {
	return &Handler{svc: svc, db: db, cache: cache}
}

func (h *Handler) Register(r *gin.Engine) {
	r.GET("/health", h.health)
	r.GET("/ready", h.ready)

	events := r.Group("/events")
	{
		events.POST("", h.createEvent)
		events.PUT("/:id", h.updateEvent)
		events.DELETE("/:id", h.deleteEvent)
		events.GET("/by-day/:year/:month/:day", h.getEventsByDay)
		events.GET("/by-week/:year/:week", h.getEventsByWeek)
		events.GET("/by-month/:year/:month", h.getEventsByMonth)
		events.GET("/by-user/:user_id", h.getEventsByUser)
		events.GET("/upcoming/:user_id", h.getUpcomingDeadlines)
		events.POST("/filter", h.filterEvents)
		events.GET("/stats/:user_id", h.getEventStats)
	}
}

func (h *Handler) health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func (h *Handler) ready(c *gin.Context) {
	ctx := c.Request.Context()

	if err := h.db.Ping(ctx); err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"status": "not ready", "reason": "database"})
		return
	}

	if h.cache != nil {
		if err := h.cache.Ping(ctx).Err(); err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"status": "not ready", "reason": "cache"})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{"status": "ready"})
}

func (h *Handler) createEvent(c *gin.Context) {
	var req struct {
		Title       string    `json:"title" binding:"required"`
		Description string    `json:"description"`
		EventType   string    `json:"event_type"`
		UserID      string    `json:"user_id" binding:"required"`
		RefID       string    `json:"ref_id"`
		RefType     string    `json:"ref_type"`
		StartTime   time.Time `json:"start_time" binding:"required"`
		EndTime     time.Time `json:"end_time"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	input := domain.CreateEventInput{
		Title:       req.Title,
		Description: req.Description,
		EventType:   domain.EventType(req.EventType),
		UserID:      req.UserID,
		RefID:       req.RefID,
		RefType:     req.RefType,
		StartTime:   req.StartTime,
		EndTime:     req.EndTime,
	}

	event, err := h.svc.CreateEvent(c.Request.Context(), input)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, event)
}

func (h *Handler) updateEvent(c *gin.Context) {
	eventID := c.Param("id")

	var req struct {
		Title       string    `json:"title"`
		Description string    `json:"description"`
		StartTime   time.Time `json:"start_time"`
		EndTime     time.Time `json:"end_time"`
		UpdatedBy   string    `json:"updated_by" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	input := domain.UpdateEventInput{
		ID:        eventID,
		UpdatedBy: req.UpdatedBy,
	}

	if req.Title != "" {
		input.Title = &req.Title
	}
	if req.Description != "" {
		input.Description = &req.Description
	}
	if !req.StartTime.IsZero() {
		input.StartTime = &req.StartTime
	}
	if !req.EndTime.IsZero() {
		input.EndTime = &req.EndTime
	}

	event, err := h.svc.UpdateEvent(c.Request.Context(), input)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, event)
}

func (h *Handler) deleteEvent(c *gin.Context) {
	eventID := c.Param("id")

	var req struct {
		DeletedBy string `json:"deleted_by" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.svc.DeleteEvent(c.Request.Context(), eventID, req.DeletedBy); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Event deleted"})
}

func (h *Handler) getEventsByDay(c *gin.Context) {
	userID := c.Query("user_id")
	if userID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "user_id is required"})
		return
	}

	year, _ := strconv.Atoi(c.Param("year"))
	month, _ := strconv.Atoi(c.Param("month"))
	day, _ := strconv.Atoi(c.Param("day"))

	events, total, err := h.svc.GetEventsByDay(c.Request.Context(), userID, year, month, day)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"events": events, "total": total})
}

func (h *Handler) getEventsByWeek(c *gin.Context) {
	userID := c.Query("user_id")
	if userID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "user_id is required"})
		return
	}

	year, _ := strconv.Atoi(c.Param("year"))
	week, _ := strconv.Atoi(c.Param("week"))

	events, total, err := h.svc.GetEventsByWeek(c.Request.Context(), userID, year, week)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"events": events, "total": total})
}

func (h *Handler) getEventsByMonth(c *gin.Context) {
	userID := c.Query("user_id")
	if userID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "user_id is required"})
		return
	}

	year, _ := strconv.Atoi(c.Param("year"))
	month, _ := strconv.Atoi(c.Param("month"))

	events, total, err := h.svc.GetEventsByMonth(c.Request.Context(), userID, year, month)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"events": events, "total": total})
}

func (h *Handler) getEventsByUser(c *gin.Context) {
	userID := c.Param("user_id")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	filter := domain.ListFilter{
		UserID:   userID,
		Page:     page,
		PageSize: pageSize,
	}

	events, total, err := h.svc.GetEventsByUser(c.Request.Context(), filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"events": events, "total": total})
}

func (h *Handler) getUpcomingDeadlines(c *gin.Context) {
	userID := c.Param("user_id")
	days, _ := strconv.Atoi(c.DefaultQuery("days", "7"))

	events, err := h.svc.GetUpcomingDeadlines(c.Request.Context(), userID, days)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"events": events, "total": len(events)})
}

func (h *Handler) filterEvents(c *gin.Context) {
	var req struct {
		UserID   string `json:"user_id" binding:"required"`
		RefType  string `json:"ref_type"`
		Page     int    `json:"page"`
		PageSize int    `json:"page_size"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.Page == 0 {
		req.Page = 1
	}
	if req.PageSize == 0 {
		req.PageSize = 20
	}

	filter := domain.EventFilter{
		UserID:   req.UserID,
		RefType:  req.RefType,
		Page:     req.Page,
		PageSize: req.PageSize,
	}

	events, total, err := h.svc.FilterEvents(c.Request.Context(), filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"events": events, "total": total})
}

func (h *Handler) getEventStats(c *gin.Context) {
	userID := c.Param("user_id")

	stats, err := h.svc.GetEventStats(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, stats)
}
