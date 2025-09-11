package orders

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/KurepinVladimir/gofermart/internal/auth"
)

type Handlers struct {
	svc *Service
}

func NewHandlers(s *Service) *Handlers { return &Handlers{svc: s} }

func (h *Handlers) PostOrder(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.ContextUserID(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil || len(body) == 0 {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	number := strings.TrimSpace(string(body))

	// только цифры
	for _, ch := range number {
		if ch < '0' || ch > '9' {
			http.Error(w, "unprocessable", http.StatusUnprocessableEntity) // 422
			return
		}
	}
	// Луна
	if !LuhnValid(number) {
		http.Error(w, "unprocessable", http.StatusUnprocessableEntity) // 422
		return
	}

	owner, exists, err := h.svc.GetOwner(r.Context(), number)
	if err != nil {
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}
	if exists {
		if owner == userID {
			w.WriteHeader(http.StatusOK) // уже загружен этим пользователем
		} else {
			http.Error(w, "conflict", http.StatusConflict) // 409: чужой номер
		}
		return
	}

	if err := h.svc.Create(r.Context(), userID, number); err != nil {
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusAccepted) // 202: новый принят
}

// Ответная структура
type orderResponse struct {
	Number     string   `json:"number"`
	Status     string   `json:"status"`
	Accrual    *float64 `json:"accrual,omitempty"`
	UploadedAt string   `json:"uploaded_at"`
}

func (h *Handlers) GetOrders(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.ContextUserID(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	rows, err := h.svc.ListByUser(r.Context(), userID)
	if err != nil {
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}
	if len(rows) == 0 {
		w.WriteHeader(http.StatusNoContent) // 204
		return
	}

	resp := make([]orderResponse, 0, len(rows))
	for _, rr := range rows {
		var accrualPtr *float64
		if rr.Accrual.Valid {
			v := rr.Accrual.Float64
			accrualPtr = &v
		}
		resp = append(resp, orderResponse{
			Number:     rr.Number,
			Status:     rr.Status,
			Accrual:    accrualPtr,
			UploadedAt: rr.UploadedAt.Format(time.RFC3339),
		})
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}
