package clients

import (
	"context"
	"fmt"

	authpb "github.com/Temych228/docflow-protos-final/auth/v1"
	calendarpb "github.com/Temych228/docflow-protos-final/calendar/v1"
	docpb "github.com/Temych228/docflow-protos-final/document/v1"
	mailpb "github.com/Temych228/docflow-protos-final/mail/v1"
	notifpb "github.com/Temych228/docflow-protos-final/notification/v1"
	taskpb "github.com/Temych228/docflow-protos-final/task/v1"
	userpb "github.com/Temych228/docflow-protos-final/user/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type AuthClient struct {
	Client authpb.AuthServiceClient
	conn   *grpc.ClientConn
}

func NewAuthClient(addr string) (*AuthClient, error) {
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to auth service: %w", err)
	}
	return &AuthClient{
		Client: authpb.NewAuthServiceClient(conn),
		conn:   conn,
	}, nil
}

func (c *AuthClient) ValidateToken(ctx context.Context, token string) (*authpb.ValidateTokenResponse, error) {
	return c.Client.ValidateToken(ctx, &authpb.ValidateTokenRequest{
		AccessToken: token,
	})
}

func (c *AuthClient) Register(ctx context.Context, req *authpb.RegisterRequest) (*authpb.RegisterResponse, error) {
	return c.Client.Register(ctx, req)
}

func (c *AuthClient) Login(ctx context.Context, req *authpb.LoginRequest) (*authpb.LoginResponse, error) {
	return c.Client.Login(ctx, req)
}

func (c *AuthClient) RefreshToken(ctx context.Context, req *authpb.RefreshTokenRequest) (*authpb.RefreshTokenResponse, error) {
	return c.Client.RefreshToken(ctx, req)
}

func (c *AuthClient) Logout(ctx context.Context, req *authpb.LogoutRequest) (*authpb.LogoutResponse, error) {
	return c.Client.Logout(ctx, req)
}

func (c *AuthClient) VerifyEmail(ctx context.Context, req *authpb.VerifyEmailRequest) (*authpb.VerifyEmailResponse, error) {
	return c.Client.VerifyEmail(ctx, req)
}

func (c *AuthClient) ForgotPassword(ctx context.Context, req *authpb.ForgotPasswordRequest) (*authpb.ForgotPasswordResponse, error) {
	return c.Client.ForgotPassword(ctx, req)
}

func (c *AuthClient) ResetPassword(ctx context.Context, req *authpb.ResetPasswordRequest) (*authpb.ResetPasswordResponse, error) {
	return c.Client.ResetPassword(ctx, req)
}

func (c *AuthClient) ChangePassword(ctx context.Context, req *authpb.ChangePasswordRequest) (*authpb.ChangePasswordResponse, error) {
	return c.Client.ChangePassword(ctx, req)
}

func (c *AuthClient) Close() error {
	return c.conn.Close()
}

type UserClient struct {
	Client userpb.UserServiceClient
	conn   *grpc.ClientConn
}

func NewUserClient(addr string) (*UserClient, error) {
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to user service: %w", err)
	}
	return &UserClient{
		Client: userpb.NewUserServiceClient(conn),
		conn:   conn,
	}, nil
}

func (c *UserClient) GetUser(ctx context.Context, userID string) (*userpb.GetUserResponse, error) {
	return c.Client.GetUser(ctx, &userpb.GetUserRequest{Id: userID})
}

func (c *UserClient) ListUsers(ctx context.Context, page, pageSize int32) (*userpb.ListUsersResponse, error) {
	return c.Client.ListUsers(ctx, &userpb.ListUsersRequest{Page: page, PageSize: pageSize})
}

func (c *UserClient) CreateUser(ctx context.Context, req *userpb.CreateUserRequest) (*userpb.CreateUserResponse, error) {
	return c.Client.CreateUser(ctx, req)
}

func (c *UserClient) GetUserByEmail(ctx context.Context, email string) (*userpb.GetUserResponse, error) {
	return c.Client.GetUserByEmail(ctx, &userpb.GetUserByEmailRequest{Email: email})
}

func (c *UserClient) UpdateUser(ctx context.Context, req *userpb.UpdateUserRequest) (*userpb.UpdateUserResponse, error) {
	return c.Client.UpdateUser(ctx, req)
}

func (c *UserClient) DeleteUser(ctx context.Context, id string) (*userpb.DeleteUserResponse, error) {
	return c.Client.DeleteUser(ctx, &userpb.DeleteUserRequest{Id: id})
}

func (c *UserClient) CheckUserExists(ctx context.Context, id string) (*userpb.CheckUserExistsResponse, error) {
	return c.Client.CheckUserExists(ctx, &userpb.CheckUserExistsRequest{Id: id})
}

func (c *UserClient) Close() error {
	return c.conn.Close()
}

type DocumentClient struct {
	Client docpb.DocumentServiceClient
	conn   *grpc.ClientConn
}

func NewDocumentClient(addr string) (*DocumentClient, error) {
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to document service: %w", err)
	}
	return &DocumentClient{
		Client: docpb.NewDocumentServiceClient(conn),
		conn:   conn,
	}, nil
}

func (c *DocumentClient) ListDocuments(ctx context.Context, page, pageSize int32) (*docpb.ListDocumentsResponse, error) {
	return c.Client.ListDocuments(ctx, &docpb.ListDocumentsRequest{Page: page, PageSize: pageSize})
}

func (c *DocumentClient) Close() error {
	return c.conn.Close()
}

type TaskClient struct {
	Client taskpb.TaskServiceClient
	conn   *grpc.ClientConn
}

func NewTaskClient(addr string) (*TaskClient, error) {
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to task service: %w", err)
	}
	return &TaskClient{
		Client: taskpb.NewTaskServiceClient(conn),
		conn:   conn,
	}, nil
}

func (c *TaskClient) GetTaskStats(ctx context.Context) (*taskpb.GetTaskStatsResponse, error) {
	return c.Client.GetTaskStats(ctx, &taskpb.GetTaskStatsRequest{})
}

func (c *TaskClient) Close() error {
	return c.conn.Close()
}

type CalendarClient struct {
	Client calendarpb.CalendarServiceClient
	conn   *grpc.ClientConn
}

func NewCalendarClient(addr string) (*CalendarClient, error) {
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to calendar service: %w", err)
	}
	return &CalendarClient{
		Client: calendarpb.NewCalendarServiceClient(conn),
		conn:   conn,
	}, nil
}

func (c *CalendarClient) GetEventStats(ctx context.Context, userID string) (*calendarpb.GetEventStatsResponse, error) {
	return c.Client.GetEventStats(ctx, &calendarpb.GetEventStatsRequest{UserId: userID})
}

func (c *CalendarClient) Close() error {
	return c.conn.Close()
}

type NotificationClient struct {
	Client notifpb.NotificationServiceClient
	conn   *grpc.ClientConn
}

func NewNotificationClient(addr string) (*NotificationClient, error) {
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to notification service: %w", err)
	}
	return &NotificationClient{
		Client: notifpb.NewNotificationServiceClient(conn),
		conn:   conn,
	}, nil
}

func (c *NotificationClient) Close() error {
	return c.conn.Close()
}

type MailClient struct {
	Client mailpb.MailServiceClient
	conn   *grpc.ClientConn
}

func NewMailClient(addr string) (*MailClient, error) {
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to mail service: %w", err)
	}
	return &MailClient{
		Client: mailpb.NewMailServiceClient(conn),
		conn:   conn,
	}, nil
}

func (c *MailClient) Close() error {
	return c.conn.Close()
}

func (c *MailClient) SendEmail(ctx context.Context, req *mailpb.SendEmailRequest) (*mailpb.SendEmailResponse, error) {
	return c.Client.SendEmail(ctx, req)
}

func (c *MailClient) SendBulkEmail(ctx context.Context, req *mailpb.SendBulkEmailRequest) (*mailpb.SendBulkEmailResponse, error) {
	return c.Client.SendBulkEmail(ctx, req)
}

func (c *MailClient) SubmitMailJob(ctx context.Context, job *mailpb.MailJob) (*mailpb.SubmitMailJobResponse, error) {
	return c.Client.SubmitMailJob(ctx, &mailpb.SubmitMailJobRequest{Job: job})
}

func (c *MailClient) GetMailJob(ctx context.Context, id string) (*mailpb.GetMailJobResponse, error) {
	return c.Client.GetMailJob(ctx, &mailpb.GetMailJobRequest{Id: id})
}

func (c *MailClient) ListMailJobs(ctx context.Context, page, pageSize int32, status, category string) (*mailpb.ListMailJobsResponse, error) {
	return c.Client.ListMailJobs(ctx, &mailpb.ListMailJobsRequest{Page: page, PageSize: pageSize, Status: status, Category: category})
}

func (c *MailClient) GetTemplate(ctx context.Context, id string) (*mailpb.GetTemplateResponse, error) {
	return c.Client.GetTemplate(ctx, &mailpb.GetTemplateRequest{TemplateId: id})
}

func (c *MailClient) ListTemplates(ctx context.Context) (*mailpb.ListTemplatesResponse, error) {
	return c.Client.ListTemplates(ctx, &mailpb.ListTemplatesRequest{})
}

func (c *MailClient) UpdateTemplate(ctx context.Context, template *mailpb.MailTemplate) (*mailpb.UpdateTemplateResponse, error) {
	return c.Client.UpdateTemplate(ctx, &mailpb.UpdateTemplateRequest{Template: template})
}

func (c *MailClient) GetStats(ctx context.Context, from, to *timestamppb.Timestamp) (*mailpb.GetStatsResponse, error) {
	return c.Client.GetStats(ctx, &mailpb.GetStatsRequest{From: from, To: to})
}
