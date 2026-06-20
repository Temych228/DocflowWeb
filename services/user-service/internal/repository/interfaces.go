package repository

import (
	"context"

	"github.com/Temych228/DocflowWeb/services/user-service/internal/domain"
)

type UserRepository interface {
	GetByID(ctx context.Context, id string) (*domain.User, error)
	GetByEmail(ctx context.Context, email string) (*domain.User, error)
	List(ctx context.Context, page, pageSize int, role string) ([]*domain.User, int, error)
	GetBatch(ctx context.Context, ids []string) ([]*domain.User, error)
	CheckExists(ctx context.Context, email string) (bool, string, error)
	Create(ctx context.Context, input domain.CreateInput) (*domain.User, error)
	CreateWithID(ctx context.Context, id string, input domain.CreateInput) (*domain.User, error)
	Update(ctx context.Context, id string, input domain.UpdateInput) (*domain.User, error)
	VerifyEmail(ctx context.Context, id string) error
	Ban(ctx context.Context, id, adminID, reason string) error
	Delete(ctx context.Context, id string) error
	GetStats(ctx context.Context) (*domain.UserStats, error)
}
