package grpc

import (
	"context"
	"strings"

	calendarv1 "github.com/Temych228/docflow-protos-final/calendar/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/Temych228/DocflowWeb/services/calendar-service/internal/domain"
	"github.com/Temych228/DocflowWeb/services/calendar-service/internal/service"
)

type Handler struct {
	calendarv1.UnimplementedCalendarServiceServer
	svc service.CalendarService
}

func New(svc service.CalendarService) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) CreateEvent(ctx context.Context, req *calendarv1.CreateEventRequest) (*calendarv1.CreateEventResponse, error) {
	if req.Title == "" || req.UserId == "" {
		return nil, status.Error(codes.InvalidArgument, "title and user_id are required")
	}
	if req.StartTime == nil {
		return nil, status.Error(codes.InvalidArgument, "start_time is required")
	}

	input := domain.CreateEventInput{
		Title:       req.Title,
		Description: req.Description,
		EventType:   protoEventTypeToDomain(req.EventType),
		UserID:      req.UserId,
		RefID:       req.RefId,
		RefType:     req.RefType,
		StartTime:   req.StartTime.AsTime(),
	}
	if req.EndTime != nil {
		input.EndTime = req.EndTime.AsTime()
	}

	event, err := h.svc.CreateEvent(ctx, input)
	if err != nil {
		return nil, domainErrToStatus(err)
	}

	return &calendarv1.CreateEventResponse{
		Event: domainToProto(event),
	}, nil
}

func (h *Handler) GetEventsByDay(ctx context.Context, req *calendarv1.GetEventsByDayRequest) (*calendarv1.GetEventsResponse, error) {
	if req.UserId == "" {
		return nil, status.Error(codes.InvalidArgument, "user_id is required")
	}

	events, total, err := h.svc.GetEventsByDay(ctx, req.UserId, int(req.Year), int(req.Month), int(req.Day))
	if err != nil {
		return nil, domainErrToStatus(err)
	}

	return &calendarv1.GetEventsResponse{
		Events: domainsToProto(events),
		Total:  int32(total),
	}, nil
}

func (h *Handler) GetEventsByWeek(ctx context.Context, req *calendarv1.GetEventsByWeekRequest) (*calendarv1.GetEventsResponse, error) {
	if req.UserId == "" {
		return nil, status.Error(codes.InvalidArgument, "user_id is required")
	}

	events, total, err := h.svc.GetEventsByWeek(ctx, req.UserId, int(req.Year), int(req.Week))
	if err != nil {
		return nil, domainErrToStatus(err)
	}

	return &calendarv1.GetEventsResponse{
		Events: domainsToProto(events),
		Total:  int32(total),
	}, nil
}

func (h *Handler) GetEventsByMonth(ctx context.Context, req *calendarv1.GetEventsByMonthRequest) (*calendarv1.GetEventsResponse, error) {
	if req.UserId == "" {
		return nil, status.Error(codes.InvalidArgument, "user_id is required")
	}

	events, total, err := h.svc.GetEventsByMonth(ctx, req.UserId, int(req.Year), int(req.Month))
	if err != nil {
		return nil, domainErrToStatus(err)
	}

	return &calendarv1.GetEventsResponse{
		Events: domainsToProto(events),
		Total:  int32(total),
	}, nil
}

func (h *Handler) GetEventsByUser(ctx context.Context, req *calendarv1.GetEventsByUserRequest) (*calendarv1.GetEventsResponse, error) {
	if req.UserId == "" {
		return nil, status.Error(codes.InvalidArgument, "user_id is required")
	}

	filter := domain.ListFilter{
		UserID:   req.UserId,
		Page:     int(req.Page),
		PageSize: int(req.PageSize),
	}

	events, total, err := h.svc.GetEventsByUser(ctx, filter)
	if err != nil {
		return nil, domainErrToStatus(err)
	}

	return &calendarv1.GetEventsResponse{
		Events: domainsToProto(events),
		Total:  int32(total),
	}, nil
}

func (h *Handler) GetUpcomingDeadlines(ctx context.Context, req *calendarv1.GetUpcomingDeadlinesRequest) (*calendarv1.GetEventsResponse, error) {
	if req.UserId == "" {
		return nil, status.Error(codes.InvalidArgument, "user_id is required")
	}

	events, err := h.svc.GetUpcomingDeadlines(ctx, req.UserId, int(req.Days))
	if err != nil {
		return nil, domainErrToStatus(err)
	}

	return &calendarv1.GetEventsResponse{
		Events: domainsToProto(events),
		Total:  int32(len(events)),
	}, nil
}

func (h *Handler) DeleteEvent(ctx context.Context, req *calendarv1.DeleteEventRequest) (*calendarv1.DeleteEventResponse, error) {
	if req.Id == "" || req.DeletedBy == "" {
		return nil, status.Error(codes.InvalidArgument, "id and deleted_by are required")
	}

	if err := h.svc.DeleteEvent(ctx, req.Id, req.DeletedBy); err != nil {
		return nil, domainErrToStatus(err)
	}

	return &calendarv1.DeleteEventResponse{Success: true}, nil
}

func (h *Handler) UpdateEvent(ctx context.Context, req *calendarv1.UpdateEventRequest) (*calendarv1.UpdateEventResponse, error) {
	if req.Id == "" || req.UpdatedBy == "" {
		return nil, status.Error(codes.InvalidArgument, "id and updated_by are required")
	}

	input := domain.UpdateEventInput{
		ID:        req.Id,
		UpdatedBy: req.UpdatedBy,
	}

	if req.Title != "" {
		v := req.Title
		input.Title = &v
	}
	if req.Description != "" {
		v := req.Description
		input.Description = &v
	}
	if req.StartTime != nil {
		t := req.StartTime.AsTime()
		input.StartTime = &t
	}
	if req.EndTime != nil {
		t := req.EndTime.AsTime()
		input.EndTime = &t
	}

	event, err := h.svc.UpdateEvent(ctx, input)
	if err != nil {
		return nil, domainErrToStatus(err)
	}

	return &calendarv1.UpdateEventResponse{
		Event: domainToProto(event),
	}, nil
}

func (h *Handler) FilterEvents(ctx context.Context, req *calendarv1.FilterEventsRequest) (*calendarv1.GetEventsResponse, error) {
	filter := domain.EventFilter{
		UserID:   req.UserId,
		RefType:  req.RefType,
		Page:     int(req.Page),
		PageSize: int(req.PageSize),
	}

	if req.EventType != calendarv1.EventType_EVENT_TYPE_UNSPECIFIED {
		et := protoEventTypeToDomain(req.EventType)
		filter.EventType = &et
	}
	if req.From != nil {
		t := req.From.AsTime()
		filter.From = &t
	}
	if req.To != nil {
		t := req.To.AsTime()
		filter.To = &t
	}

	events, total, err := h.svc.FilterEvents(ctx, filter)
	if err != nil {
		return nil, domainErrToStatus(err)
	}

	return &calendarv1.GetEventsResponse{
		Events: domainsToProto(events),
		Total:  int32(total),
	}, nil
}

func (h *Handler) GetEventStats(ctx context.Context, req *calendarv1.GetEventStatsRequest) (*calendarv1.GetEventStatsResponse, error) {
	stats, err := h.svc.GetEventStats(ctx, req.UserId)
	if err != nil {
		return nil, domainErrToStatus(err)
	}

	return &calendarv1.GetEventStatsResponse{
		Stats: &calendarv1.EventStats{
			Total:     int32(stats.Total),
			Deadlines: int32(stats.Deadlines),
			Tasks:     int32(stats.Tasks),
			Meetings:  int32(stats.Meetings),
			Reminders: int32(stats.Reminders),
		},
	}, nil
}

func domainToProto(e *domain.Event) *calendarv1.Event {
	return &calendarv1.Event{
		Id:          e.ID,
		Title:       e.Title,
		Description: e.Description,
		EventType:   domainEventTypeToProto(e.EventType),
		UserId:      e.UserID,
		RefId:       e.RefID,
		RefType:     e.RefType,
		StartTime:   timestamppb.New(e.StartTime),
		EndTime:     timestamppb.New(e.EndTime),
		CreatedAt:   timestamppb.New(e.CreatedAt),
		UpdatedAt:   timestamppb.New(e.UpdatedAt),
	}
}

func domainsToProto(events []*domain.Event) []*calendarv1.Event {
	out := make([]*calendarv1.Event, 0, len(events))
	for _, e := range events {
		out = append(out, domainToProto(e))
	}
	return out
}

func domainEventTypeToProto(t domain.EventType) calendarv1.EventType {
	switch t {
	case domain.EventTypeDeadline:
		return calendarv1.EventType_EVENT_TYPE_DEADLINE
	case domain.EventTypeTask:
		return calendarv1.EventType_EVENT_TYPE_TASK
	case domain.EventTypeMeeting:
		return calendarv1.EventType_EVENT_TYPE_MEETING
	case domain.EventTypeReminder:
		return calendarv1.EventType_EVENT_TYPE_REMINDER
	default:
		return calendarv1.EventType_EVENT_TYPE_UNSPECIFIED
	}
}

func protoEventTypeToDomain(t calendarv1.EventType) domain.EventType {
	switch t {
	case calendarv1.EventType_EVENT_TYPE_DEADLINE:
		return domain.EventTypeDeadline
	case calendarv1.EventType_EVENT_TYPE_TASK:
		return domain.EventTypeTask
	case calendarv1.EventType_EVENT_TYPE_MEETING:
		return domain.EventTypeMeeting
	case calendarv1.EventType_EVENT_TYPE_REMINDER:
		return domain.EventTypeReminder
	default:
		return domain.EventTypeReminder
	}
}

func domainErrToStatus(err error) error {
	if err == nil {
		return nil
	}
	msg := err.Error()
	switch {
	case strings.Contains(msg, "not found"):
		return status.Error(codes.NotFound, msg)
	case strings.Contains(msg, "invalid input"):
		return status.Error(codes.InvalidArgument, msg)
	case strings.Contains(msg, "forbidden"):
		return status.Error(codes.PermissionDenied, msg)
	default:
		return status.Error(codes.Internal, msg)
	}
}
