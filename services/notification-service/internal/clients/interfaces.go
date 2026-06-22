package clients

import "context"

type UserLookup interface {
	GetUserByID(ctx context.Context, userID string) (*UserInfo, error)
}

type UserInfo struct {
	ID    string
	Email string
	Name  string
}
