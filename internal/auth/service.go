package auth

import (
	"context"
	"errors"

	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"golang.org/x/crypto/bcrypt"
)

type Repo interface {
	CreateUser(ctx context.Context, login, passwordHash string) (int64, error)
	GetUserByLogin(ctx context.Context, login string) (int64, string, error)
}

type Service struct {
	repo Repo
}

func NewService(repo Repo) *Service { return &Service{repo: repo} }

var (
	ErrLoginTaken   = errors.New("login_taken")
	ErrInvalidCreds = errors.New("invalid_credentials")
)

func hashPassword(pw string) (string, error) {
	b, err := bcrypt.GenerateFromPassword([]byte(pw), bcrypt.DefaultCost)
	return string(b), err
}

func checkPassword(hash, pw string) error {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(pw))
}

func (s *Service) Register(ctx context.Context, login, password string) (int64, error) {
	h, err := hashPassword(password)
	if err != nil {
		return 0, err
	}
	id, err := s.repo.CreateUser(ctx, login, h)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == pgerrcode.UniqueViolation {
			return 0, ErrLoginTaken
		}
		return 0, err
	}
	return id, nil
}

func (s *Service) Login(ctx context.Context, login, password string) (int64, error) {
	id, hash, err := s.repo.GetUserByLogin(ctx, login)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, ErrInvalidCreds
		}
		return 0, err
	}
	if err := checkPassword(hash, password); err != nil {
		return 0, ErrInvalidCreds
	}
	return id, nil
}
