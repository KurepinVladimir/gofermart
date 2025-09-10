package app

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/KurepinVladimir/gofermart/internal/auth"
	"github.com/KurepinVladimir/gofermart/internal/repository"
)

type Server struct {
	httpServer *http.Server
	pg         *repository.Postgres
}

func NewServer(addr string, pg *repository.Postgres) *Server {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Compress(5)) // gzip-ответы при поддержке клиентом

	// Простой health-check: проверяем подключение к БД
	r.Get("/ping", func(w http.ResponseWriter, r *http.Request) {
		if err := pg.Ping(r.Context()); err != nil {
			http.Error(w, "db not ready", http.StatusServiceUnavailable)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	})

	// ===== Auth =====
	authSvc := auth.NewService(pg)
	authHandlers := auth.NewHandlers(authSvc)
	r.Post("/api/user/register", authHandlers.Register)
	r.Post("/api/user/login", authHandlers.Login)

	// ===== Health =====
	r.Get("/ping", func(w http.ResponseWriter, r *http.Request) {
		if err := pg.Ping(r.Context()); err != nil {
			http.Error(w, "db not ready", http.StatusServiceUnavailable)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	})

	s := &http.Server{
		Addr:    addr,
		Handler: r,
	}
	return &Server{httpServer: s, pg: pg}
}

func (s *Server) Start() error {
	return s.httpServer.ListenAndServe()
}

func (s *Server) Shutdown(ctx context.Context) error {
	return s.httpServer.Shutdown(ctx)
}
