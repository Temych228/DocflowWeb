package clients

import "context"

type NotificationClient interface {
	SendVerificationEmail(ctx context.Context, userID, email, name, token string) error
	SendPasswordResetEmail(ctx context.Context, userID, email, token string) error
}

type UserClient interface {
	CreateUser(ctx context.Context, req CreateUserRequest) error
	VerifyUserEmail(ctx context.Context, userID string) error
}

type CreateUserRequest struct {
	Email    string
	Name     string
	Password string
	Role     string
}
