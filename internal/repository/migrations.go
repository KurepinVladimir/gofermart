package repository

import "context"

const createUsersSQL = `
CREATE TABLE IF NOT EXISTS users (
  id BIGSERIAL PRIMARY KEY,
  login TEXT NOT NULL UNIQUE,
  password_hash TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
`

// используем TEXT + CHECK вместо ENUM, чтобы не ловить сложности с ALTER TYPE
const createOrdersSQL = `
CREATE TABLE IF NOT EXISTS orders (
  id BIGSERIAL PRIMARY KEY,
  user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  number  TEXT NOT NULL UNIQUE,
  status  TEXT NOT NULL DEFAULT 'NEW' CHECK (status IN ('NEW','PROCESSING','INVALID','PROCESSED')),
  accrual NUMERIC(18,2),
  uploaded_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
  processed_at TIMESTAMPTZ
);
CREATE INDEX IF NOT EXISTS idx_orders_user_uploaded_at ON orders(user_id, uploaded_at DESC);
`

func (p *Postgres) RunMigrations(ctx context.Context) error {
	if _, err := p.Pool.Exec(ctx, createUsersSQL); err != nil {
		return err
	}
	if _, err := p.Pool.Exec(ctx, createOrdersSQL); err != nil {
		return err
	}
	return nil
}
