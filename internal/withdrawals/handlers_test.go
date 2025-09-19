package withdrawals

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/KurepinVladimir/gofermart/internal/auth"
	"github.com/KurepinVladimir/gofermart/internal/repository"
)

// ---- фейковый репозиторий под интерфейс withdrawals.Service ----

type fakeRepo struct {
	mode string // "", "insufficient", "duplicate"
	log  []repository.WithdrawalRow
}

func (f *fakeRepo) Withdraw(_ context.Context, userID int64, order string, amount float64) error {
	switch f.mode {
	case "insufficient":
		return repository.ErrInsufficientFunds
	case "duplicate":
		return errors.New("withdraw_order_duplicate")
	default:
		f.log = append(f.log, repository.WithdrawalRow{
			Order:       order,
			Sum:         amount,
			ProcessedAt: time.Date(2025, 9, 12, 10, 0, 0, 0, time.UTC),
		})
		return nil
	}
}

func (f *fakeRepo) ListWithdrawalsByUser(_ context.Context, _ int64) ([]repository.WithdrawalRow, error) {
	return f.log, nil
}

// ---- тесты ----

func TestPostWithdraw_Codes(t *testing.T) {
	// 422 — номер не проходит Луна
	{
		repo := &fakeRepo{}
		svc := NewService(repo)
		h := NewHandlers(svc)

		body := []byte(`{"order":"12345678902","sum":10}`)
		req := httptest.NewRequest(http.MethodPost, "/api/user/balance/withdraw", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req = req.WithContext(auth.WithUserID(req.Context(), 1))
		rr := httptest.NewRecorder()

		http.HandlerFunc(h.PostWithdraw).ServeHTTP(rr, req)
		if rr.Code != http.StatusUnprocessableEntity {
			t.Fatalf("expected 422, got %d; body=%s", rr.Code, rr.Body.String())
		}
	}

	// 402 — недостаточно средств
	{
		repo := &fakeRepo{mode: "insufficient"}
		svc := NewService(repo)
		h := NewHandlers(svc)

		body := []byte(`{"order":"79927398713","sum":9999}`)
		req := httptest.NewRequest(http.MethodPost, "/api/user/balance/withdraw", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req = req.WithContext(auth.WithUserID(req.Context(), 1))
		rr := httptest.NewRecorder()

		http.HandlerFunc(h.PostWithdraw).ServeHTTP(rr, req)
		if rr.Code != http.StatusPaymentRequired {
			t.Fatalf("expected 402, got %d; body=%s", rr.Code, rr.Body.String())
		}
	}

	// 200 — успех
	{
		repo := &fakeRepo{}
		svc := NewService(repo)
		h := NewHandlers(svc)

		body := []byte(`{"order":"79927398713","sum":70.5}`)
		req := httptest.NewRequest(http.MethodPost, "/api/user/balance/withdraw", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req = req.WithContext(auth.WithUserID(req.Context(), 1))
		rr := httptest.NewRecorder()

		http.HandlerFunc(h.PostWithdraw).ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d; body=%s", rr.Code, rr.Body.String())
		}
	}
}

func TestGetWithdrawals_JSON(t *testing.T) {
	repo := &fakeRepo{
		log: []repository.WithdrawalRow{
			{
				Order:       "4000000000000002",
				Sum:         70.5,
				ProcessedAt: time.Date(2025, 9, 12, 10, 0, 0, 0, time.UTC),
			},
		},
	}
	svc := NewService(repo)
	h := NewHandlers(svc)

	req := httptest.NewRequest(http.MethodGet, "/api/user/withdrawals", nil)
	req = req.WithContext(auth.WithUserID(req.Context(), 1))
	rr := httptest.NewRecorder()

	http.HandlerFunc(h.GetWithdrawals).ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d; body=%s", rr.Code, rr.Body.String())
	}
	if ct := rr.Header().Get("Content-Type"); ct != "application/json" {
		t.Fatalf("unexpected Content-Type: %q", ct)
	}

	var got []struct {
		Order       string  `json:"order"`
		Sum         float64 `json:"sum"`
		ProcessedAt string  `json:"processed_at"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &got); err != nil {
		t.Fatalf("json: %v; body=%s", err, rr.Body.String())
	}
	if len(got) != 1 || got[0].Order != "4000000000000002" || got[0].Sum != 70.5 {
		t.Fatalf("unexpected response: %+v", got)
	}
}
