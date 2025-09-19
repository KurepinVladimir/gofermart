package orders

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/KurepinVladimir/gofermart/internal/auth"
	"github.com/KurepinVladimir/gofermart/internal/repository"
)

//  фейковая реализация orders.Repo

type fakeOrdersRepo struct {
	// номер -> владелец
	owner map[string]int64
	// номер -> запись
	rows map[string]repository.OrderRow
}

func newFakeOrdersRepo() *fakeOrdersRepo {
	return &fakeOrdersRepo{
		owner: map[string]int64{},
		rows:  map[string]repository.OrderRow{},
	}
}

func (f *fakeOrdersRepo) GetOrderOwner(_ context.Context, number string) (int64, bool, error) {
	uid, ok := f.owner[number]
	return uid, ok, nil
}

func (f *fakeOrdersRepo) InsertOrder(_ context.Context, userID int64, number string) error {
	// эмулируем уникальность: если уже есть — не перезаписываем
	if _, exists := f.owner[number]; exists {
		return nil
	}
	f.owner[number] = userID
	f.rows[number] = repository.OrderRow{
		Number:     number,
		Status:     "NEW",
		Accrual:    sql.NullFloat64{}, // пустое значение (accrual отсутствует)
		UploadedAt: time.Now(),
	}
	return nil
}

func (f *fakeOrdersRepo) ListOrdersByUser(_ context.Context, userID int64) ([]repository.OrderRow, error) {
	out := make([]repository.OrderRow, 0, len(f.rows))
	for num, uid := range f.owner {
		if uid == userID {
			out = append(out, f.rows[num])
		}
	}
	// порядок не критичен
	return out, nil
}

// ---- сами тесты ----

func TestPostOrder_HandlerCodes(t *testing.T) {
	repo := newFakeOrdersRepo()
	svc := NewService(repo)
	h := NewHandlers(svc)

	// вызывает POST /api/user/orders от имени userID
	call := func(userID int64, body string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodPost, "/api/user/orders", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "text/plain")
		req = req.WithContext(auth.WithUserID(req.Context(), userID))
		rr := httptest.NewRecorder()
		http.HandlerFunc(h.PostOrder).ServeHTTP(rr, req)
		return rr
	}

	// 422 — неверный номер по Луну
	if rr := call(1, "12345678902"); rr.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422, got %d", rr.Code)
	}

	// 202 — новый валидный
	if rr := call(1, "79927398713"); rr.Code != http.StatusAccepted {
		t.Fatalf("expected 202, got %d", rr.Code)
	}

	// 200 — повтор этим же пользователем
	if rr := call(1, "79927398713"); rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}

	// 409 — этот номер уже «принадлежит» другому пользователю
	repo.owner["4000000000000002"] = 777
	repo.rows["4000000000000002"] = repository.OrderRow{
		Number:     "4000000000000002",
		Status:     "PROCESSED",
		Accrual:    sql.NullFloat64{Float64: 123.45, Valid: true},
		UploadedAt: time.Now(),
	}
	if rr := call(1, "4000000000000002"); rr.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d", rr.Code)
	}
}

func TestGetOrders_HandlerJSON(t *testing.T) {
	repo := newFakeOrdersRepo()
	svc := NewService(repo)
	h := NewHandlers(svc)

	// подготовим запись другого тестового заказа для пользователя 7
	repo.owner["4000000000000002"] = 7
	repo.rows["4000000000000002"] = repository.OrderRow{
		Number:     "4000000000000002",
		Status:     "PROCESSED",
		Accrual:    sql.NullFloat64{Float64: 123.45, Valid: true},
		UploadedAt: time.Date(2025, 9, 11, 19, 51, 50, 0, time.Local),
	}

	req := httptest.NewRequest(http.MethodGet, "/api/user/orders", nil)
	req = req.WithContext(auth.WithUserID(req.Context(), 7))
	rr := httptest.NewRecorder()

	http.HandlerFunc(h.GetOrders).ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status %d, body=%s", rr.Code, rr.Body.String())
	}
	if ct := rr.Header().Get("Content-Type"); ct != "application/json" {
		t.Fatalf("Content-Type = %q", ct)
	}

	var got []struct {
		Number     string   `json:"number"`
		Status     string   `json:"status"`
		Accrual    *float64 `json:"accrual,omitempty"`
		UploadedAt string   `json:"uploaded_at"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &got); err != nil {
		t.Fatalf("json err: %v; body=%s", err, rr.Body.String())
	}
	if len(got) != 1 || got[0].Number != "4000000000000002" || got[0].Status != "PROCESSED" || got[0].Accrual == nil {
		t.Fatalf("unexpected response: %+v", got)
	}
}
