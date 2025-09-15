package auth

import (
	"context"
	"testing"

	//"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"golang.org/x/crypto/bcrypt"
)

// in-mem repo
type memRepo struct {
	users map[string]struct {
		id   int64
		hash string
	}
	next int64
	// если login уже есть — имитируем unique_violation
}

func newMemRepo() *memRepo {
	return &memRepo{users: map[string]struct {
		id   int64
		hash string
	}{}, next: 1}
}

func (m *memRepo) CreateUser(ctx context.Context, login, passwordHash string) (int64, error) {
	if _, ok := m.users[login]; ok {
		return 0, &pgconn.PgError{Code: "23505"} // unique_violation
	}
	id := m.next
	m.next++
	m.users[login] = struct {
		id   int64
		hash string
	}{id: id, hash: passwordHash}
	return id, nil
}

func (m *memRepo) GetUserByLogin(ctx context.Context, login string) (int64, string, error) {
	u, ok := m.users[login]
	if !ok {
		return 0, "", pgx.ErrNoRows
	}
	return u.id, u.hash, nil
}

func TestService_Register_Login(t *testing.T) {
	t.Parallel()
	svc := NewService(newMemRepo())
	ctx := context.Background()

	// register
	id, err := svc.Register(ctx, "bob", "secret")
	if err != nil {
		t.Fatalf("register error: %v", err)
	}
	if id == 0 {
		t.Fatalf("got zero id")
	}
	// duplicate
	if _, err := svc.Register(ctx, "bob", "secret"); err != ErrLoginTaken {
		t.Fatalf("expected ErrLoginTaken, got %v", err)
	}

	// login ok
	id2, err := svc.Login(ctx, "bob", "secret")
	if err != nil {
		t.Fatalf("login error: %v", err)
	}
	if id2 != id {
		t.Fatalf("id mismatch: %d vs %d", id, id2)
	}

	// wrong password
	if _, err := svc.Login(ctx, "bob", "wrong"); err != ErrInvalidCreds {
		t.Fatalf("expected ErrInvalidCreds, got %v", err)
	}

	// unknown user
	if _, err := svc.Login(ctx, "nobody", "x"); err != ErrInvalidCreds {
		t.Fatalf("expected ErrInvalidCreds for unknown login, got %v", err)
	}
}

func TestService_hash_check(t *testing.T) {
	t.Parallel()
	h, err := hashPassword("p@ss")
	if err != nil {
		t.Fatalf("hashPassword: %v", err)
	}
	if err := checkPassword(h, "p@ss"); err != nil {
		t.Fatalf("checkPassword ok failed: %v", err)
	}
	if err := checkPassword(h, "nope"); err == nil {
		t.Fatalf("expected error for bad password")
	}
	// sanity: это действительно bcrypt-хэш
	if err := bcrypt.CompareHashAndPassword([]byte(h), []byte("p@ss")); err != nil {
		t.Fatalf("bcrypt compare failed: %v", err)
	}
}
