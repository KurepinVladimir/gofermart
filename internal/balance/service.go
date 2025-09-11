package balance

import (
	"context"

	"github.com/KurepinVladimir/gofermart/internal/repository"
)

type Repo interface {
	GetBalance(ctx context.Context, userID int64) (repository.Balance, error)
}

type Service struct{ repo Repo }

func NewService(r Repo) *Service { return &Service{repo: r} }

func (s *Service) Get(ctx context.Context, userID int64) (repository.Balance, error) {
	return s.repo.GetBalance(ctx, userID)
}
