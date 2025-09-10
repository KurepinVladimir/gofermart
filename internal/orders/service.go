package orders

import "context"

type Repo interface {
	GetOrderOwner(ctx context.Context, number string) (int64, bool, error)
	InsertOrder(ctx context.Context, userID int64, number string) error
}

type Service struct{ repo Repo }

func NewService(r Repo) *Service { return &Service{repo: r} }

func (s *Service) GetOwner(ctx context.Context, number string) (int64, bool, error) {
	return s.repo.GetOrderOwner(ctx, number)
}

func (s *Service) Create(ctx context.Context, userID int64, number string) error {
	return s.repo.InsertOrder(ctx, userID, number)
}
