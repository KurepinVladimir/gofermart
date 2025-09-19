package balance

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/KurepinVladimir/gofermart/internal/auth"
	"github.com/KurepinVladimir/gofermart/internal/repository"
)

// ---- фейковый репозиторий под интерфейс balance.Service ----

type fakeRepo struct {
	val repository.Balance
}

func (f *fakeRepo) GetBalance(_ context.Context, _ int64) (repository.Balance, error) {
	return f.val, nil
}

// ---- тест ----

func TestHandlers_GetBalance_OK(t *testing.T) {
	repo := &fakeRepo{
		val: repository.Balance{
			Current:   123.45,
			Withdrawn: 7,
		},
	}
	svc := NewService(repo)
	h := NewHandlers(svc)

	req := httptest.NewRequest(http.MethodGet, "/api/user/balance", nil)
	req = req.WithContext(auth.WithUserID(req.Context(), 42)) // подложим userID
	rr := httptest.NewRecorder()

	http.HandlerFunc(h.GetBalance).ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d; body=%s", rr.Code, rr.Body.String())
	}
	if ct := rr.Header().Get("Content-Type"); ct != "application/json" {
		t.Fatalf("unexpected Content-Type: %q", ct)
	}

	// Читаем как map, чтобы не зависеть от float32/float64 в реализации
	var got map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &got); err != nil {
		t.Fatalf("json unmarshal: %v; body=%s", err, rr.Body.String())
	}

	// Извлекаем числа как float64 (json.Unmarshal даёт float64)
	cur, ok1 := got["current"].(float64)
	wdr, ok2 := got["withdrawn"].(float64)
	if !ok1 || !ok2 {
		t.Fatalf("unexpected json types: %+v", got)
	}
	if cur != 123.45 || wdr != 7 {
		t.Fatalf("unexpected values: current=%v withdrawn=%v (raw=%+v)", cur, wdr, got)
	}
}
