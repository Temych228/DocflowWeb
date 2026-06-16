package clients

import (
	"context"
	"fmt"

	userpb "github.com/Temych228/docflow-protos-final/user/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type userClient struct {
	client userpb.UserServiceClient
}

func NewUserClient(addr string) (UserClient, error) {
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("user grpc dial: %w", err)
	}
	return &userClient{client: userpb.NewUserServiceClient(conn)}, nil
}

func (c *userClient) CreateUser(ctx context.Context, req CreateUserRequest) error {
	role := roleStringToProto(req.Role)
	_, err := c.client.CreateUser(ctx, &userpb.CreateUserRequest{
		Email:    req.Email,
		Name:     req.Name,
		Role:     role,
		Password: req.Password,
	})
	return err
}

func (c *userClient) VerifyUserEmail(ctx context.Context, userID string) error {
	_, err := c.client.VerifyUserEmail(ctx, &userpb.VerifyUserEmailRequest{
		Id: userID,
	})
	return err
}

func roleStringToProto(role string) userpb.UserRole {
	switch role {
	case "manager":
		return userpb.UserRole_USER_ROLE_MANAGER
	case "admin":
		return userpb.UserRole_USER_ROLE_ADMIN
	default:
		return userpb.UserRole_USER_ROLE_EMPLOYEE
	}
}
