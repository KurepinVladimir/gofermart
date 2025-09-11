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

const createBalancesSQL = `
CREATE TABLE IF NOT EXISTS balances (
  user_id   BIGINT PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
  current   NUMERIC(18,2) NOT NULL DEFAULT 0,
  withdrawn NUMERIC(18,2) NOT NULL DEFAULT 0
);
`

const createWithdrawalsSQL = `
CREATE TABLE IF NOT EXISTS withdrawals (
  id           BIGSERIAL PRIMARY KEY,
  user_id      BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  order_number TEXT   NOT NULL UNIQUE,           -- защитимся от повторного списания по тому же номеру
  sum          NUMERIC(18,2) NOT NULL CHECK (sum > 0),
  processed_at TIMESTAMPTZ   NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_withdrawals_user_processed_at
  ON withdrawals(user_id, processed_at DESC);
`

func (p *Postgres) RunMigrations(ctx context.Context) error {
	if _, err := p.Pool.Exec(ctx, createUsersSQL); err != nil {
		return err
	}
	if _, err := p.Pool.Exec(ctx, createOrdersSQL); err != nil {
		return err
	}
	if _, err := p.Pool.Exec(ctx, createBalancesSQL); err != nil {
		return err
	}
	// НОВОЕ:
	if _, err := p.Pool.Exec(ctx, createWithdrawalsSQL); err != nil {
		return err
	}
	return nil
}
