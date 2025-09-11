package balance

import (
	"encoding/json"
	"net/http"

	"github.com/KurepinVladimir/gofermart/internal/auth"
)

type Handlers struct {
	svc *Service
}

func NewHandlers(s *Service) *Handlers { return &Handlers{svc: s} }

type resp struct {
	Current   float64 `json:"current"`
	Withdrawn float64 `json:"withdrawn"`
}

func (h *Handlers) GetBalance(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.ContextUserID(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	b, err := h.svc.Get(r.Context(), userID)
	if err != nil {
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp{
		Current:   b.Current,
		Withdrawn: b.Withdrawn,
	})
}
