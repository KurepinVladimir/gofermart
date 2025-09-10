package repository

import "context"

// SQL для создания таблицы пользователей.
// UNIQUE по login – чтобы логины не повторялись.
const createUsersSQL = `
CREATE TABLE IF NOT EXISTS users (
  id BIGSERIAL PRIMARY KEY,
  login TEXT NOT NULL UNIQUE,
  password_hash TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
`

// Запускаем все миграции при старте приложения.
func (p *Postgres) RunMigrations(ctx context.Context) error {
	_, err := p.Pool.Exec(ctx, createUsersSQL)
	return err
}
