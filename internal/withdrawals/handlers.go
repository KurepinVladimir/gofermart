package withdrawals

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/KurepinVladimir/gofermart/internal/auth"
	"github.com/KurepinVladimir/gofermart/internal/orders" // для Luhn
)

type Handlers struct{ svc *Service }

func NewHandlers(s *Service) *Handlers { return &Handlers{svc: s} }

type withdrawReq struct {
	Order string  `json:"order"`
	Sum   float64 `json:"sum"`
}

func (h *Handlers) PostWithdraw(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.ContextUserID(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	var req withdrawReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Order == "" || req.Sum <= 0 {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	// Номер заказа должен пройти Луна
	if !orders.LuhnValid(req.Order) {
		http.Error(w, "unprocessable", http.StatusUnprocessableEntity) // 422
		return
	}

	err := h.svc.Withdraw(r.Context(), userID, req.Order, req.Sum)
	if err != nil {
		switch err {
		case ErrInsufficientFunds:
			http.Error(w, "payment required", http.StatusPaymentRequired) // 402
			return
		case ErrDuplicateOrderUsage:
			http.Error(w, "unprocessable", http.StatusUnprocessableEntity) // 422
			return
		default:
			http.Error(w, "server error", http.StatusInternalServerError)
			return
		}
	}

	w.WriteHeader(http.StatusOK)
}

type withdrawalResp struct {
	Order       string  `json:"order"`
	Sum         float64 `json:"sum"`
	ProcessedAt string  `json:"processed_at"`
}

func (h *Handlers) GetWithdrawals(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.ContextUserID(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	rows, err := h.svc.List(r.Context(), userID)
	if err != nil {
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}
	if len(rows) == 0 {
		w.WriteHeader(http.StatusNoContent) // 204
		return
	}

	resp := make([]withdrawalResp, 0, len(rows))
	for _, rw := range rows {
		resp = append(resp, withdrawalResp{
			Order:       rw.Order,
			Sum:         rw.Sum,
			ProcessedAt: rw.ProcessedAt.Format(time.RFC3339),
		})
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}
