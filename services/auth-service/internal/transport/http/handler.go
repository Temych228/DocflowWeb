package grpc

import (
	"context"
	"errors"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/Temych228/DocflowWeb/services/auth-service/internal/domain"
	"github.com/Temych228/DocflowWeb/services/auth-service/internal/service"
	authpb "github.com/Temych228/docflow-protos-final/auth/v1"
)

type AuthHandler struct {
	authpb.UnimplementedAuthServiceServer
	svc service.AuthService
}

func New(svc service.AuthService) *AuthHandler {
	return &AuthHandler{svc: svc}
}

func (h *AuthHandler) Register(ctx context.Context, req *authpb.RegisterRequest) (*authpb.RegisterResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request is nil")
	}

	userID, err := h.svc.Register(ctx, service.RegisterInput{
		Email:     req.Email,
		Password:  req.Password,
		FirstName: req.FirstName,
		LastName:  req.LastName,
		Phone:     req.Phone,
		Role:      req.Role,
	})
	if err != nil {
		return nil, toGRPCError(err)
	}

	return &authpb.RegisterResponse{UserId: userID}, nil
}

func (h *AuthHandler) Login(ctx context.Context, req *authpb.LoginRequest) (*authpb.LoginResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request is nil")
	}

	pair, err := h.svc.Login(ctx, service.LoginInput{
		Email:    req.Email,
		Password: req.Password,
	})
	if err != nil {
		return nil, toGRPCError(err)
	}

	return &authpb.LoginResponse{
		AccessToken:  pair.AccessToken,
		RefreshToken: pair.RefreshToken,
		ExpiresAt:    timestamppb.New(pair.ExpiresAt),
		UserId:       pair.UserID,
		Email:        pair.Email,
		Role:         pair.Role,
	}, nil
}

func (h *AuthHandler) RefreshToken(ctx context.Context, req *authpb.RefreshTokenRequest) (*authpb.RefreshTokenResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request is nil")
	}

	pair, err := h.svc.RefreshToken(ctx, req.RefreshToken)
	if err != nil {
		return nil, toGRPCError(err)
	}

	return &authpb.RefreshTokenResponse{
		AccessToken:  pair.AccessToken,
		RefreshToken: pair.RefreshToken,
		ExpiresAt:    timestamppb.New(pair.ExpiresAt),
	}, nil
}

func (h *AuthHandler) Logout(ctx context.Context, req *authpb.LogoutRequest) (*authpb.LogoutResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request is nil")
	}

	if err := h.svc.Logout(ctx, req.RefreshToken); err != nil {
		return nil, toGRPCError(err)
	}

	return &authpb.LogoutResponse{Success: true}, nil
}

func (h *AuthHandler) ValidateToken(ctx context.Context, req *authpb.ValidateTokenRequest) (*authpb.ValidateTokenResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request is nil")
	}

	claims, err := h.svc.ValidateToken(req.AccessToken)
	if err != nil {
		return nil, toGRPCError(err)
	}

	return &authpb.ValidateTokenResponse{
		Valid:  true,
		UserId: claims.UserID,
		Email:  claims.Email,
		Role:   claims.Role,
	}, nil
}

func (h *AuthHandler) VerifyEmail(ctx context.Context, req *authpb.VerifyEmailRequest) (*authpb.VerifyEmailResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request is nil")
	}

	if err := h.svc.VerifyEmail(ctx, req.Token); err != nil {
		return nil, toGRPCError(err)
	}

	return &authpb.VerifyEmailResponse{Success: true}, nil
}

func (h *AuthHandler) ForgotPassword(ctx context.Context, req *authpb.ForgotPasswordRequest) (*authpb.ForgotPasswordResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request is nil")
	}

	resetToken, err := h.svc.ForgotPassword(ctx, req.Email)
	if err != nil {
		return nil, toGRPCError(err)
	}

	return &authpb.ForgotPasswordResponse{
		ResetToken: resetToken,
	}, nil
}

func (h *AuthHandler) ResetPassword(ctx context.Context, req *authpb.ResetPasswordRequest) (*authpb.ResetPasswordResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request is nil")
	}

	if err := h.svc.ResetPassword(ctx, req.Token, req.NewPassword); err != nil {
		return nil, toGRPCError(err)
	}

	return &authpb.ResetPasswordResponse{Success: true}, nil
}

func (h *AuthHandler) ChangePassword(ctx context.Context, req *authpb.ChangePasswordRequest) (*authpb.ChangePasswordResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request is nil")
	}

	if err := h.svc.ChangePassword(ctx, req.UserId, req.OldPassword, req.NewPassword); err != nil {
		return nil, toGRPCError(err)
	}

	return &authpb.ChangePasswordResponse{Success: true}, nil
}

func (h *AuthHandler) GetSession(ctx context.Context, req *authpb.GetSessionRequest) (*authpb.GetSessionResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request is nil")
	}

	session, err := h.svc.GetSession(ctx, req.AccessToken)
	if err != nil {
		return nil, toGRPCError(err)
	}

	return &authpb.GetSessionResponse{
		UserId:    session.UserID,
		Email:     session.Email,
		Role:      session.Role,
		ExpiresAt: timestamppb.New(session.ExpiresAt),
	}, nil
}

func (h *AuthHandler) RevokeAllSessions(ctx context.Context, req *authpb.RevokeAllSessionsRequest) (*authpb.RevokeAllSessionsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request is nil")
	}

	count, err := h.svc.RevokeAllSessions(ctx, req.UserId)
	if err != nil {
		return nil, toGRPCError(err)
	}

	return &authpb.RevokeAllSessionsResponse{RevokedCount: count}, nil
}

func toGRPCError(err error) error {
	switch {
	case errors.Is(err, models.ErrInvalidInput):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, models.ErrEmailTaken):
		return status.Error(codes.AlreadyExists, err.Error())
	case errors.Is(err, models.ErrUserNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, models.ErrUnauthorized):
		return status.Error(codes.Unauthenticated, err.Error())
	case errors.Is(err, models.ErrTokenExpired):
		return status.Error(codes.Unauthenticated, err.Error())
	case errors.Is(err, models.ErrTokenInvalid):
		return status.Error(codes.Unauthenticated, err.Error())
	case errors.Is(err, models.ErrTokenNotFound):
		return status.Error(codes.Unauthenticated, err.Error())
	case errors.Is(err, models.ErrUserBanned):
		return status.Error(codes.PermissionDenied, err.Error())
	case errors.Is(err, models.ErrUserInactive):
		return status.Error(codes.PermissionDenied, err.Error())
	default:
		return status.Error(codes.Internal, "internal error")
	}
}
