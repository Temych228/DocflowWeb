package handlers

import (
	"time"

	"github.com/Temych228/DocflowWeb/api-gateway/internal/clients"
	"github.com/Temych228/DocflowWeb/api-gateway/internal/middleware"
	"github.com/Temych228/DocflowWeb/api-gateway/pkg/dto"
	calendarpb "github.com/Temych228/docflow-protos-final/calendar/v1"
	"github.com/gin-gonic/gin"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type CalendarHandler struct {
	calendarClient *clients.CalendarClient
}

func NewCalendarHandler(calendarClient *clients.CalendarClient) *CalendarHandler {
	return &CalendarHandler{calendarClient: calendarClient}
}

var eventTypeMap = map[string]calendarpb.EventType{
	"deadline": calendarpb.EventType_EVENT_TYPE_DEADLINE,
	"task":     calendarpb.EventType_EVENT_TYPE_TASK,
	"meeting":  calendarpb.EventType_EVENT_TYPE_MEETING,
	"reminder": calendarpb.EventType_EVENT_TYPE_REMINDER,
}

func (h *CalendarHandler) CreateEvent(c *gin.Context) {
	var req CreateEventRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, dto.NewErrorResponse(dto.ErrorValidation, err.Error(), nil))
		return
	}

	userID := req.UserID
	if userID == "" {
		userID = middleware.ExtractUserID(c)
	}

	var startTime, endTime *timestamppb.Timestamp
	if req.StartTime != "" {
		if t, err := time.Parse(time.RFC3339, req.StartTime); err == nil {
			startTime = timestamppb.New(t)
		}
	}
	if req.EndTime != "" {
		if t, err := time.Parse(time.RFC3339, req.EndTime); err == nil {
			endTime = timestamppb.New(t)
		}
	}

	pbType := calendarpb.EventType_EVENT_TYPE_UNSPECIFIED
	if et, ok := eventTypeMap[req.EventType]; ok {
		pbType = et
	}

	resp, err := h.calendarClient.Client.CreateEvent(c.Request.Context(), &calendarpb.CreateEventRequest{
		Title:       req.Title,
		Description: req.Description,
		EventType:   pbType,
		UserId:      userID,
		StartTime:   startTime,
		EndTime:     endTime,
	})
	if err != nil {
		c.JSON(500, dto.NewErrorResponse(dto.ErrorInternal, grpcErrMessage(err), nil))
		return
	}

	c.JSON(201, dto.NewSuccessResponse(resp.Event))
}

func (h *CalendarHandler) GetEventsByDay(c *gin.Context) {
	userID := c.Param("user_id")
	year := parseQueryInt(c, "year", time.Now().Year())
	month := parseQueryInt(c, "month", int(time.Now().Month()))
	day := parseQueryInt(c, "day", time.Now().Day())

	resp, err := h.calendarClient.Client.GetEventsByDay(c.Request.Context(), &calendarpb.GetEventsByDayRequest{
		UserId: userID,
		Year:   int32(year),
		Month:  int32(month),
		Day:    int32(day),
	})
	if err != nil {
		c.JSON(500, dto.NewErrorResponse(dto.ErrorInternal, grpcErrMessage(err), nil))
		return
	}

	c.JSON(200, dto.NewSuccessResponse(gin.H{"events": resp.Events, "total": resp.Total}))
}

func (h *CalendarHandler) GetEventsByWeek(c *gin.Context) {
	userID := c.Param("user_id")
	year := parseQueryInt(c, "year", time.Now().Year())
	_, week := time.Now().ISOWeek()
	weekNum := parseQueryInt(c, "week", week)

	resp, err := h.calendarClient.Client.GetEventsByWeek(c.Request.Context(), &calendarpb.GetEventsByWeekRequest{
		UserId: userID,
		Year:   int32(year),
		Week:   int32(weekNum),
	})
	if err != nil {
		c.JSON(500, dto.NewErrorResponse(dto.ErrorInternal, grpcErrMessage(err), nil))
		return
	}

	c.JSON(200, dto.NewSuccessResponse(gin.H{"events": resp.Events, "total": resp.Total}))
}

func (h *CalendarHandler) GetEventsByMonth(c *gin.Context) {
	userID := c.Param("user_id")
	year := parseQueryInt(c, "year", time.Now().Year())
	month := parseQueryInt(c, "month", int(time.Now().Month()))

	resp, err := h.calendarClient.Client.GetEventsByMonth(c.Request.Context(), &calendarpb.GetEventsByMonthRequest{
		UserId: userID,
		Year:   int32(year),
		Month:  int32(month),
	})
	if err != nil {
		c.JSON(500, dto.NewErrorResponse(dto.ErrorInternal, grpcErrMessage(err), nil))
		return
	}

	c.JSON(200, dto.NewSuccessResponse(gin.H{"events": resp.Events, "total": resp.Total}))
}

func (h *CalendarHandler) GetEventsByUser(c *gin.Context) {
	userID := c.Param("user_id")
	page := parseQueryInt(c, "page", 1)
	pageSize := parseQueryInt(c, "page_size", 10)

	resp, err := h.calendarClient.Client.GetEventsByUser(c.Request.Context(), &calendarpb.GetEventsByUserRequest{
		UserId:   userID,
		Page:     int32(page),
		PageSize: int32(pageSize),
	})
	if err != nil {
		c.JSON(500, dto.NewErrorResponse(dto.ErrorInternal, grpcErrMessage(err), nil))
		return
	}

	c.JSON(200, dto.NewSuccessResponse(gin.H{"events": resp.Events, "total": resp.Total}))
}

func (h *CalendarHandler) GetUpcomingDeadlines(c *gin.Context) {
	userID := c.Param("user_id")
	days := parseQueryInt(c, "days", 7)

	resp, err := h.calendarClient.Client.GetUpcomingDeadlines(c.Request.Context(), &calendarpb.GetUpcomingDeadlinesRequest{
		UserId: userID,
		Days:   int32(days),
	})
	if err != nil {
		c.JSON(500, dto.NewErrorResponse(dto.ErrorInternal, grpcErrMessage(err), nil))
		return
	}

	c.JSON(200, dto.NewSuccessResponse(gin.H{"events": resp.Events, "total": resp.Total}))
}

func (h *CalendarHandler) GetEventStats(c *gin.Context) {
	userID := c.Param("user_id")

	resp, err := h.calendarClient.Client.GetEventStats(c.Request.Context(), &calendarpb.GetEventStatsRequest{
		UserId: userID,
	})
	if err != nil {
		c.JSON(500, dto.NewErrorResponse(dto.ErrorInternal, grpcErrMessage(err), nil))
		return
	}

	c.JSON(200, dto.NewSuccessResponse(gin.H{"stats": resp.Stats}))
}

func (h *CalendarHandler) DeleteEvent(c *gin.Context) {
	eventID := c.Param("id")
	userID := middleware.ExtractUserID(c)

	_, err := h.calendarClient.Client.DeleteEvent(c.Request.Context(), &calendarpb.DeleteEventRequest{
		Id:        eventID,
		DeletedBy: userID,
	})
	if err != nil {
		c.JSON(500, dto.NewErrorResponse(dto.ErrorInternal, grpcErrMessage(err), nil))
		return
	}

	c.JSON(200, dto.NewSuccessResponse(gin.H{"deleted": true}))
}

func (h *CalendarHandler) UpdateEvent(c *gin.Context) {
	eventID := c.Param("id")
	userID := middleware.ExtractUserID(c)

	var req UpdateEventRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, dto.NewErrorResponse(dto.ErrorValidation, err.Error(), nil))
		return
	}

	var startTime, endTime *timestamppb.Timestamp
	if req.StartTime != "" {
		if t, err := time.Parse(time.RFC3339, req.StartTime); err == nil {
			startTime = timestamppb.New(t)
		}
	}
	if req.EndTime != "" {
		if t, err := time.Parse(time.RFC3339, req.EndTime); err == nil {
			endTime = timestamppb.New(t)
		}
	}

	resp, err := h.calendarClient.Client.UpdateEvent(c.Request.Context(), &calendarpb.UpdateEventRequest{
		Id:          eventID,
		Title:       req.Title,
		Description: req.Description,
		StartTime:   startTime,
		EndTime:     endTime,
		UpdatedBy:   userID,
	})
	if err != nil {
		c.JSON(500, dto.NewErrorResponse(dto.ErrorInternal, grpcErrMessage(err), nil))
		return
	}

	c.JSON(200, dto.NewSuccessResponse(resp.Event))
}

func (h *CalendarHandler) FilterEvents(c *gin.Context) {
	var req FilterEventsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, dto.NewErrorResponse(dto.ErrorValidation, err.Error(), nil))
		return
	}

	pbType := calendarpb.EventType_EVENT_TYPE_UNSPECIFIED
	if et, ok := eventTypeMap[req.EventType]; ok {
		pbType = et
	}

	var from, to *timestamppb.Timestamp
	if req.From != "" {
		if t, err := time.Parse(time.RFC3339, req.From); err == nil {
			from = timestamppb.New(t)
		}
	}
	if req.To != "" {
		if t, err := time.Parse(time.RFC3339, req.To); err == nil {
			to = timestamppb.New(t)
		}
	}

	resp, err := h.calendarClient.Client.FilterEvents(c.Request.Context(), &calendarpb.FilterEventsRequest{
		UserId:    req.UserID,
		EventType: pbType,
		RefType:   req.RefType,
		From:      from,
		To:        to,
		Page:      req.Page,
		PageSize:  req.PageSize,
	})
	if err != nil {
		c.JSON(500, dto.NewErrorResponse(dto.ErrorInternal, grpcErrMessage(err), nil))
		return
	}

	c.JSON(200, dto.NewSuccessResponse(gin.H{"events": resp.Events, "total": resp.Total}))
}

type CreateEventRequest struct {
	Title       string `json:"title" binding:"required"`
	Description string `json:"description"`
	EventType   string `json:"event_type"`
	UserID      string `json:"user_id"`
	StartTime   string `json:"start_time"`
	EndTime     string `json:"end_time"`
}

type UpdateEventRequest struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	StartTime   string `json:"start_time"`
	EndTime     string `json:"end_time"`
}

type FilterEventsRequest struct {
	UserID    string `json:"user_id"`
	EventType string `json:"event_type"`
	RefType   string `json:"ref_type"`
	From      string `json:"from"`
	To        string `json:"to"`
	Page      int32  `json:"page,omitempty"`
	PageSize  int32  `json:"page_size,omitempty"`
}
