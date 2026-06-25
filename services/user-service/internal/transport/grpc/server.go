package grpcserver

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/Temych228/DocflowWeb/services/user-service/internal/domain"
	"github.com/Temych228/DocflowWeb/services/user-service/internal/service"
	userv1 "github.com/Temych228/docflow-protos-final/user/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type Server struct {
	userv1.UnimplementedUserServiceServer
	svc *service.UserService
}

func New(svc *service.UserService) *Server {
	return &Server{svc: svc}
}

func (s *Server) CreateUser(ctx context.Context, req *userv1.CreateUserRequest) (*userv1.CreateUserResponse, error) {
	user, err := s.svc.CreateUser(ctx, domain.CreateInput{
		Email: req.GetEmail(),
		Name:  req.GetName(),
		Role:  protoRoleToDomain(req.GetRole()),
	})
	if err != nil {
		return nil, mapError(err)
	}
	return &userv1.CreateUserResponse{User: toProtoUser(user)}, nil
}

func (s *Server) GetUser(ctx context.Context, req *userv1.GetUserRequest) (*userv1.GetUserResponse, error) {
	if strings.TrimSpace(req.GetId()) == "" {
		return nil, status.Error(codes.InvalidArgument, "id is required")
	}
	user, err := s.svc.GetUser(ctx, req.GetId())
	if err != nil {
		return nil, mapError(err)
	}
	return &userv1.GetUserResponse{User: toProtoUser(user)}, nil
}

func (s *Server) GetUserByEmail(ctx context.Context, req *userv1.GetUserByEmailRequest) (*userv1.GetUserResponse, error) {
	if strings.TrimSpace(req.GetEmail()) == "" {
		return nil, status.Error(codes.InvalidArgument, "email is required")
	}
	user, err := s.svc.GetUserByEmail(ctx, req.GetEmail())
	if err != nil {
		return nil, mapError(err)
	}
	return &userv1.GetUserResponse{User: toProtoUser(user)}, nil
}

func (s *Server) UpdateUser(ctx context.Context, req *userv1.UpdateUserRequest) (*userv1.UpdateUserResponse, error) {
	if strings.TrimSpace(req.GetId()) == "" {
		return nil, status.Error(codes.InvalidArgument, "id is required")
	}

	input := domain.UpdateInput{
		Name: req.GetName(),
	}
	if role, ok := optionalProtoRoleToDomain(req.GetRole()); ok {
		input.Role = role
	}

	user, err := s.svc.UpdateUser(ctx, req.GetId(), input)
	if err != nil {
		return nil, mapError(err)
	}
	return &userv1.UpdateUserResponse{User: toProtoUser(user)}, nil
}

func (s *Server) DeleteUser(ctx context.Context, req *userv1.DeleteUserRequest) (*userv1.DeleteUserResponse, error) {
	if strings.TrimSpace(req.GetId()) == "" {
		return nil, status.Error(codes.InvalidArgument, "id is required")
	}
	if err := s.svc.DeleteUser(ctx, req.GetId()); err != nil {
		return nil, mapError(err)
	}
	return &userv1.DeleteUserResponse{Success: true}, nil
}

func (s *Server) ListUsers(ctx context.Context, req *userv1.ListUsersRequest) (*userv1.ListUsersResponse, error) {
	users, total, err := s.svc.ListUsers(ctx, int(req.GetPage()), int(req.GetPageSize()), "")
	if err != nil {
		return nil, mapError(err)
	}
	out := make([]*userv1.User, 0, len(users))
	for _, user := range users {
		out = append(out, toProtoUser(user))
	}
	return &userv1.ListUsersResponse{Users: out, Total: int32(total)}, nil
}

func (s *Server) GetUsersBatch(ctx context.Context, req *userv1.GetUsersBatchRequest) (*userv1.GetUsersBatchResponse, error) {
	users, err := s.svc.GetUsersBatch(ctx, req.GetIds())
	if err != nil {
		return nil, mapError(err)
	}
	out := make([]*userv1.User, 0, len(users))
	for _, user := range users {
		out = append(out, toProtoUser(user))
	}
	return &userv1.GetUsersBatchResponse{Users: out}, nil
}

func (s *Server) CheckUserExists(ctx context.Context, req *userv1.CheckUserExistsRequest) (*userv1.CheckUserExistsResponse, error) {
	email := strings.TrimSpace(req.GetId())
	if email == "" {
		return nil, status.Error(codes.InvalidArgument, "email is required")
	}

	exists, _, err := s.svc.CheckUserExists(ctx, email)
	if err != nil {
		return nil, mapError(err)
	}
	return &userv1.CheckUserExistsResponse{Exists: exists}, nil
}

func (s *Server) VerifyUserEmail(ctx context.Context, req *userv1.VerifyUserEmailRequest) (*userv1.VerifyUserEmailResponse, error) {
	if strings.TrimSpace(req.GetId()) == "" {
		return nil, status.Error(codes.InvalidArgument, "id is required")
	}
	if err := s.svc.VerifyUserEmail(ctx, req.GetId()); err != nil {
		return nil, mapError(err)
	}
	return &userv1.VerifyUserEmailResponse{Success: true}, nil
}

func (s *Server) WatchUser(req *userv1.WatchUserRequest, stream userv1.UserService_WatchUserServer) error {
	if strings.TrimSpace(req.GetId()) == "" {
		return status.Error(codes.InvalidArgument, "id is required")
	}

	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	var last *domain.User

	for {
		select {
		case <-stream.Context().Done():
			return nil
		case <-ticker.C:
			user, err := s.svc.GetUser(stream.Context(), req.GetId())
			if err != nil {
				return mapError(err)
			}
			if last != nil && sameUser(last, user) {
				continue
			}
			eventType := "snapshot"
			if last != nil {
				eventType = "updated"
			}
			if err := stream.Send(&userv1.UserEvent{
				UserId:     user.ID,
				EventType:  eventType,
				User:       toProtoUser(user),
				OccurredAt: timestamppb.Now(),
			}); err != nil {
				return status.Error(codes.Internal, err.Error())
			}
			last = user
		}
	}
}

func (s *Server) GetUserStats(ctx context.Context, req *userv1.GetUserStatsRequest) (*userv1.GetUserStatsResponse, error) {
	stats, err := s.svc.GetUserStats(ctx)
	if err != nil {
		return nil, mapError(err)
	}
	return &userv1.GetUserStatsResponse{
		Stats: &userv1.UserStats{
			Total:     stats.Total,
			Active:    stats.Active,
			Banned:    stats.Banned,
			Employees: stats.Employees,
			Managers:  stats.Managers,
			Admins:    stats.Admins,
		},
	}, nil
}

func (s *Server) BanUser(ctx context.Context, req *userv1.BanUserRequest) (*userv1.BanUserResponse, error) {
	if strings.TrimSpace(req.GetUserId()) == "" {
		return nil, status.Error(codes.InvalidArgument, "user_id is required")
	}
	if strings.TrimSpace(req.GetAdminId()) == "" {
		return nil, status.Error(codes.InvalidArgument, "admin_id is required")
	}
	if err := s.svc.BanUser(ctx, req.GetUserId(), req.GetAdminId(), req.GetReason()); err != nil {
		return nil, mapError(err)
	}
	return &userv1.BanUserResponse{Success: true}, nil
}

func toProtoUser(user *domain.User) *userv1.User {
	if user == nil {
		return nil
	}
	return &userv1.User{
		Id:            user.ID,
		Email:         user.Email,
		Name:          user.FullName(),
		Role:          domainToProtoRole(user.Role),
		IsActive:      user.IsActive,
		EmailVerified: user.IsVerified,
		IsBanned:      user.IsBanned,
		BanReason:     user.BanReason,
		CreatedAt:     timestamppb.New(user.CreatedAt),
		UpdatedAt:     timestamppb.New(user.UpdatedAt),
	}
}

func domainToProtoRole(role domain.Role) userv1.UserRole {
	switch role {
	case domain.RoleEmployee:
		return userv1.UserRole_USER_ROLE_EMPLOYEE
	case domain.RoleManager:
		return userv1.UserRole_USER_ROLE_MANAGER
	case domain.RoleAdmin:
		return userv1.UserRole_USER_ROLE_ADMIN
	default:
		return userv1.UserRole_USER_ROLE_UNSPECIFIED
	}
}

func protoRoleToDomain(role userv1.UserRole) domain.Role {
	switch role {
	case userv1.UserRole_USER_ROLE_EMPLOYEE:
		return domain.RoleEmployee
	case userv1.UserRole_USER_ROLE_MANAGER:
		return domain.RoleManager
	case userv1.UserRole_USER_ROLE_ADMIN:
		return domain.RoleAdmin
	default:
		return domain.RoleEmployee
	}
}

func optionalProtoRoleToDomain(role userv1.UserRole) (domain.Role, bool) {
	switch role {
	case userv1.UserRole_USER_ROLE_EMPLOYEE:
		return domain.RoleEmployee, true
	case userv1.UserRole_USER_ROLE_MANAGER:
		return domain.RoleManager, true
	case userv1.UserRole_USER_ROLE_ADMIN:
		return domain.RoleAdmin, true
	default:
		return "", false
	}
}

func sameUser(a, b *domain.User) bool {
	if a == nil || b == nil {
		return false
	}
	return a.FullName() == b.FullName() &&
		a.Role == b.Role &&
		a.IsActive == b.IsActive &&
		a.IsVerified == b.IsVerified &&
		a.IsBanned == b.IsBanned &&
		a.BanReason == b.BanReason
}

func mapError(err error) error {
	switch {
	case errors.Is(err, domain.ErrUserNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, domain.ErrEmailTaken):
		return status.Error(codes.AlreadyExists, err.Error())
	case errors.Is(err, domain.ErrInvalidInput):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, domain.ErrUserBanned):
		return status.Error(codes.PermissionDenied, err.Error())
	default:
		return status.Error(codes.Internal, fmt.Sprintf("%v", err))
	}
}
