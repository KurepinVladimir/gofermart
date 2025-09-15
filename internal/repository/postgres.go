package repository

import (
	"context"
	"database/sql"
	"errors"
	"time"

	//"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
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

// Ошибка недостатка средств — будем маппить на HTTP 402
var ErrInsufficientFunds = errors.New("insufficient_funds")

// OrderRow — то, что достаём из БД для ответа
type OrderRow struct {
	Number     string
	Status     string
	Accrual    sql.NullFloat64 // может быть NULL (когда еще не посчитали)
	UploadedAt time.Time
}

// ListOrdersByUser — все заказы пользователя, новые первыми
func (p *Postgres) ListOrdersByUser(ctx context.Context, userID int64) ([]OrderRow, error) {
	rows, err := p.Pool.Query(ctx, `
		SELECT number, status, accrual, uploaded_at
		FROM orders
		WHERE user_id = $1
		ORDER BY uploaded_at DESC
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []OrderRow
	for rows.Next() {
		var r OrderRow
		if err := rows.Scan(&r.Number, &r.Status, &r.Accrual, &r.UploadedAt); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// Создать строку баланса для пользователя (если её ещё нет)
func (p *Postgres) EnsureBalanceRow(ctx context.Context, userID int64) error {
	_, err := p.Pool.Exec(ctx, `
		INSERT INTO balances (user_id) VALUES ($1)
		ON CONFLICT (user_id) DO NOTHING
	`, userID)
	return err
}

// Прочитать баланс
type Balance struct {
	Current   float64
	Withdrawn float64
}

func (p *Postgres) GetBalance(ctx context.Context, userID int64) (Balance, error) {
	var b Balance
	// сначала гарантируем, что строка существует
	if err := p.EnsureBalanceRow(ctx, userID); err != nil {
		return b, err
	}
	err := p.Pool.QueryRow(ctx, `
		SELECT current, withdrawn
		FROM balances
		WHERE user_id = $1
	`, userID).Scan(&b.Current, &b.Withdrawn)
	return b, err
}

// Withdraw — атомарно списывает средства и фиксирует запись в withdrawals.
// Гарантии:
//   - строка баланса создаётся при отсутствии
//   - баланс блокируется SELECT ... FOR UPDATE
//   - при недостатке — ErrInsufficientFunds (ничего не меняем)
//   - уникальность order_number предотвращает повторное списание
func (p *Postgres) Withdraw(ctx context.Context, userID int64, orderNumber string, amount float64) error {
	tx, err := p.Pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// Убедимся, что есть строка баланса
	if _, err := tx.Exec(ctx,
		`INSERT INTO balances (user_id) VALUES ($1) ON CONFLICT (user_id) DO NOTHING`,
		userID,
	); err != nil {
		return err
	}

	// Заблокируем баланс пользователя
	var current, withdrawn float64
	if err := tx.QueryRow(ctx,
		`SELECT current, withdrawn FROM balances WHERE user_id=$1 FOR UPDATE`,
		userID,
	).Scan(&current, &withdrawn); err != nil {
		return err
	}

	if current < amount {
		return ErrInsufficientFunds
	}

	// Обновим баланс
	newCurrent := current - amount
	newWithdrawn := withdrawn + amount
	if _, err := tx.Exec(ctx,
		`UPDATE balances SET current=$1, withdrawn=$2 WHERE user_id=$3`,
		newCurrent, newWithdrawn, userID,
	); err != nil {
		return err
	}

	// Запишем факт списания
	if _, err := tx.Exec(ctx,
		`INSERT INTO withdrawals (user_id, order_number, sum) VALUES ($1, $2, $3)`,
		userID, orderNumber, amount,
	); err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			// этот номер уже использовался для списания (идемпотентность/защита)
			// По ТЗ нет отдельного кода, отдадим 422 — "неверный номер заказа" по смыслу.
			return errors.New("withdraw_order_duplicate")
		}
		return err
	}

	return tx.Commit(ctx)
}

type WithdrawalRow struct {
	Order       string
	Sum         float64
	ProcessedAt time.Time
}

func (p *Postgres) ListWithdrawalsByUser(ctx context.Context, userID int64) ([]WithdrawalRow, error) {
	rows, err := p.Pool.Query(ctx, `
		SELECT order_number, sum, processed_at
		FROM withdrawals
		WHERE user_id = $1
		ORDER BY processed_at DESC
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []WithdrawalRow
	for rows.Next() {
		var r WithdrawalRow
		if err := rows.Scan(&r.Order, &r.Sum, &r.ProcessedAt); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// Вытащить пачку заказов в статусах NEW/PROCESSING (для опроса внешней системы)
func (p *Postgres) PickPendingOrders(ctx context.Context, limit int) ([]string, error) {
	rows, err := p.Pool.Query(ctx, `
		SELECT number
		FROM orders
		WHERE status IN ('NEW','PROCESSING')
		ORDER BY uploaded_at ASC
		LIMIT $1
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var nums []string
	for rows.Next() {
		var n string
		if err := rows.Scan(&n); err != nil {
			return nil, err
		}
		nums = append(nums, n)
	}
	return nums, rows.Err()
}

// Поставить статус PROCESSING (когда внешняя система ещё считает)
func (p *Postgres) MarkProcessing(ctx context.Context, number string) error {
	_, err := p.Pool.Exec(ctx, `UPDATE orders SET status='PROCESSING' WHERE number=$1 AND status IN ('NEW','PROCESSING')`, number)
	return err
}

// Пометить заказ INVALID (окончательный статус)
func (p *Postgres) MarkInvalid(ctx context.Context, number string) error {
	_, err := p.Pool.Exec(ctx, `UPDATE orders SET status='INVALID', processed_at=now() WHERE number=$1`, number)
	return err
}

// Применить начисление и закрыть заказ (атомарно)
func (p *Postgres) ApplyAccrualByNumber(ctx context.Context, number string, accrual float64) error {
	tx, err := p.Pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// Блокируем заказ
	var userID int64
	var status string
	err = tx.QueryRow(ctx, `SELECT user_id, status FROM orders WHERE number=$1 FOR UPDATE`, number).Scan(&userID, &status)
	if err != nil {
		return err
	}
	// Идемпотентность: если уже закрыт — просто выходим
	if status == "PROCESSED" || status == "INVALID" {
		return tx.Commit(ctx)
	}

	// Баланс: убеждаемся, что строка есть
	if _, err := tx.Exec(ctx, `
		INSERT INTO balances (user_id) VALUES ($1)
		ON CONFLICT (user_id) DO NOTHING
	`, userID); err != nil {
		return err
	}

	// Начисляем, если есть что начислять (>0)
	if accrual > 0 {
		if _, err := tx.Exec(ctx, `
			UPDATE balances
			SET current = current + $1
			WHERE user_id = $2
		`, accrual, userID); err != nil {
			return err
		}
	}

	// Обновляем заказ
	if _, err := tx.Exec(ctx, `
		UPDATE orders
		SET status='PROCESSED', accrual=$1, processed_at=now()
		WHERE number=$2
	`, accrual, number); err != nil {
		return err
	}

	return tx.Commit(ctx)
}
