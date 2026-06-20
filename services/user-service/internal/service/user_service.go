package service

import (
	"context"
	"strings"

	"github.com/Temych228/DocflowWeb/services/user-service/internal/domain"
)

type UserRepo interface {
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

type UserService struct {
	repo UserRepo
}

func New(repo UserRepo) *UserService {
	return &UserService{repo: repo}
}

func (s *UserService) GetUser(ctx context.Context, id string) (*domain.User, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *UserService) GetUserByEmail(ctx context.Context, email string) (*domain.User, error) {
	return s.repo.GetByEmail(ctx, email)
}

func (s *UserService) UpdateUser(ctx context.Context, id string, input domain.UpdateInput) (*domain.User, error) {
	if err := input.Validate(); err != nil {
		return nil, err
	}
	return s.repo.Update(ctx, id, input)
}

func (s *UserService) DeleteUser(ctx context.Context, id string) error {
	return s.repo.Delete(ctx, id)
}

func (s *UserService) ListUsers(ctx context.Context, page, pageSize int, role string) ([]*domain.User, int, error) {
	return s.repo.List(ctx, page, pageSize, strings.TrimSpace(role))
}

func (s *UserService) CreateUser(ctx context.Context, input domain.CreateInput) (*domain.User, error) {
	if err := input.Validate(); err != nil {
		return nil, err
	}
	return s.repo.Create(ctx, input)
}

func (s *UserService) CreateUserWithID(ctx context.Context, id string, input domain.CreateInput) (*domain.User, error) {
	if err := input.Validate(); err != nil {
		return nil, err
	}
	return s.repo.CreateWithID(ctx, id, input)
}

func (s *UserService) GetUsersBatch(ctx context.Context, ids []string) ([]*domain.User, error) {
	return s.repo.GetBatch(ctx, ids)
}

func (s *UserService) CheckUserExists(ctx context.Context, email string) (bool, string, error) {
	return s.repo.CheckExists(ctx, email)
}

func (s *UserService) VerifyUserEmail(ctx context.Context, userID string) error {
	return s.repo.VerifyEmail(ctx, userID)
}

func (s *UserService) BanUser(ctx context.Context, userID, adminID, reason string) error {
	return s.repo.Ban(ctx, userID, adminID, reason)
}

func (s *UserService) GetUserStats(ctx context.Context) (*domain.UserStats, error) {
	return s.repo.GetStats(ctx)
}
