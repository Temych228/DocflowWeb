package clients

import (
	"context"
	"fmt"

	notifpb "github.com/Temych228/docflow-protos-final/notification/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type notificationClient struct {
	client notifpb.NotificationServiceClient
}

func NewNotificationClient(addr string) (NotificationClient, error) {
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("notification grpc dial %s: %w", addr, err)
	}
	return &notificationClient{client: notifpb.NewNotificationServiceClient(conn)}, nil
}

func (c *notificationClient) SendVerificationEmail(ctx context.Context, userID, email, name, token string) error {
	_, err := c.client.SendEmail(ctx, &notifpb.SendEmailRequest{
		To:         email,
		Subject:    "Email verification",
		TemplateId: "verification",
		TemplateVars: map[string]string{
			"FirstName":        name,
			"VerificationLink": fmt.Sprintf("https://dms.local/verify?token=%s", token),
		},
	})
	return err
}

func (c *notificationClient) SendPasswordResetEmail(ctx context.Context, userID, email, token string) error {
	_, err := c.client.SendEmail(ctx, &notifpb.SendEmailRequest{
		To:         email,
		Subject:    "Reset Password",
		TemplateId: "password_reset",
		TemplateVars: map[string]string{
			"ResetLink": fmt.Sprintf("https://dms.local/reset-password?token=%s", token),
		},
	})
	return err
}
