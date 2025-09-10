package repository

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Postgres struct {
	Pool *pgxpool.Pool
}

func NewPostgres(ctx context.Context, dsn string) (*Postgres, error) {
	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, err
	}
	cfg.MaxConns = 10
	cfg.MinConns = 2
	cfg.HealthCheckPeriod = time.Minute

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, err
	}
	return &Postgres{Pool: pool}, nil
}

func (p *Postgres) Close() {
	if p.Pool != nil {
		p.Pool.Close()
	}
}

func (p *Postgres) Ping(ctx context.Context) error {
	return p.Pool.Ping(ctx)
}

// Создать пользователя и вернуть его id
func (p *Postgres) CreateUser(ctx context.Context, login, passwordHash string) (int64, error) {
	var id int64
	err := p.Pool.QueryRow(ctx,
		`INSERT INTO users (login, password_hash) VALUES ($1, $2) RETURNING id`,
		login, passwordHash,
	).Scan(&id)
	return id, err
}

// Получить id и hash пароля по логину
func (p *Postgres) GetUserByLogin(ctx context.Context, login string) (int64, string, error) {
	var (
		id   int64
		hash string
	)
	err := p.Pool.QueryRow(ctx,
		`SELECT id, password_hash FROM users WHERE login=$1`,
		login,
	).Scan(&id, &hash)
	if err != nil {
		return 0, "", err // в т.ч. pgx.ErrNoRows
	}
	return id, hash, nil
}

// Узнать владельца заказа, если он уже есть
func (p *Postgres) GetOrderOwner(ctx context.Context, number string) (int64, bool, error) {
	var uid int64
	err := p.Pool.QueryRow(ctx, `SELECT user_id FROM orders WHERE number=$1`, number).Scan(&uid)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, false, nil
	}
	if err != nil {
		return 0, false, err
	}
	return uid, true, nil
}

// Вставить новый заказ
func (p *Postgres) InsertOrder(ctx context.Context, userID int64, number string) error {
	_, err := p.Pool.Exec(ctx, `INSERT INTO orders (user_id, number) VALUES ($1,$2)`, userID, number)
	return err
}
