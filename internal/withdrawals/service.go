package withdrawals

import (
	"context"
	"errors"

	"github.com/KurepinVladimir/gofermart/internal/repository"
)

var (
	ErrInsufficientFunds   = repository.ErrInsufficientFunds
	ErrDuplicateOrderUsage = errors.New("withdraw_order_duplicate")
)

type Repo interface {
	Withdraw(ctx context.Context, userID int64, orderNumber string, amount float64) error
	ListWithdrawalsByUser(ctx context.Context, userID int64) ([]repository.WithdrawalRow, error)
}

type Service struct{ repo Repo }

func NewService(r Repo) *Service { return &Service{repo: r} }

func (s *Service) Withdraw(ctx context.Context, userID int64, orderNumber string, amount float64) error {
	return s.repo.Withdraw(ctx, userID, orderNumber, amount)
}

func (s *Service) List(ctx context.Context, userID int64) ([]repository.WithdrawalRow, error) {
	return s.repo.ListWithdrawalsByUser(ctx, userID)
}
