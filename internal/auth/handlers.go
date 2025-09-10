package auth

import (
	"encoding/json"
	"net/http"
	"time"

	"go.uber.org/zap"

	"github.com/KurepinVladimir/gofermart/internal/logger"
)

type Handlers struct {
	svc *Service
}

func NewHandlers(svc *Service) *Handlers { return &Handlers{svc: svc} }

type creds struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

func (h *Handlers) Register(w http.ResponseWriter, r *http.Request) {
	var c creds
	if err := json.NewDecoder(r.Body).Decode(&c); err != nil || c.Login == "" || c.Password == "" {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	id, err := h.svc.Register(r.Context(), c.Login, c.Password)
	if err != nil {
		if err == ErrLoginTaken {
			http.Error(w, "conflict", http.StatusConflict) // 409
			return
		}
		logger.Log.Warn("register failed", zap.Error(err))
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}
	token, err := MakeToken(id, c.Login, 24*time.Hour)
	if err != nil {
		logger.Log.Error("token create failed", zap.Error(err))
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK) // по ТЗ: 200
	_ = json.NewEncoder(w).Encode(map[string]string{"token": token})
}

func (h *Handlers) Login(w http.ResponseWriter, r *http.Request) {
	var c creds
	if err := json.NewDecoder(r.Body).Decode(&c); err != nil || c.Login == "" || c.Password == "" {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	id, err := h.svc.Login(r.Context(), c.Login, c.Password)
	if err != nil {
		if err == ErrInvalidCreds {
			http.Error(w, "unauthorized", http.StatusUnauthorized) // 401
			return
		}
		logger.Log.Warn("login failed", zap.Error(err))
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}
	token, err := MakeToken(id, c.Login, 24*time.Hour)
	if err != nil {
		logger.Log.Error("token create failed", zap.Error(err))
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]string{"token": token})
}
