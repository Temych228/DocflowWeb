package clients

import (
	"context"
	"fmt"
	"strings"

	userv1 "github.com/Temych228/docflow-protos-final/user/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type UserClient struct {
	conn   *grpc.ClientConn
	client userv1.UserServiceClient
}

func NewUserClient(addr string) (*UserClient, error) {
	addr = strings.TrimSpace(addr)
	if addr == "" {
		return nil, nil
	}

	conn, err := grpc.Dial(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}

	return &UserClient{
		conn:   conn,
		client: userv1.NewUserServiceClient(conn),
	}, nil
}

func (c *UserClient) GetUserByID(ctx context.Context, userID string) (*UserInfo, error) {
	if c == nil || c.client == nil {
		return nil, fmt.Errorf("user client is not initialized")
	}

	resp, err := c.client.GetUser(ctx, &userv1.GetUserRequest{Id: strings.TrimSpace(userID)})
	if err != nil {
		return nil, err
	}

	user := resp.GetUser()
	if user == nil {
		return nil, fmt.Errorf("user is empty")
	}

	return &UserInfo{
		ID:    user.GetId(),
		Email: user.GetEmail(),
		Name:  user.GetName(),
	}, nil
}

func (c *UserClient) Close() error {
	if c == nil || c.conn == nil {
		return nil
	}
	return c.conn.Close()
}
