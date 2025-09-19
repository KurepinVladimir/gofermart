package main

import (
	"context"
	"errors"
	"net/http"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"go.uber.org/zap"

	"github.com/KurepinVladimir/gofermart/internal/accrual"
	"github.com/KurepinVladimir/gofermart/internal/app"
	"github.com/KurepinVladimir/gofermart/internal/config"
	"github.com/KurepinVladimir/gofermart/internal/logger"
	"github.com/KurepinVladimir/gofermart/internal/orders"
	"github.com/KurepinVladimir/gofermart/internal/repository"
)

func main() {
	if err := logger.Init(); err != nil {
		panic(err)
	}
	defer logger.Close()

	cfg := config.Load()
	if cfg.Database == "" {
		logger.Log.Fatal("DATABASE_URI is required")
	}

	// общий контекст завершения по SIGINT/SIGTERM
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	pg, err := repository.NewPostgres(ctx, cfg.Database)
	if err != nil {
		logger.Log.Fatal("db connect failed", zap.Error(err))
	}
	defer pg.Close()

	// миграции
	if err := pg.RunMigrations(ctx); err != nil {
		logger.Log.Fatal("migrations failed", zap.Error(err))
	}

	// HTTP
	srv := app.NewServer(cfg.RunAddr, pg)

	// будем корректно ждать фоновые горутины (воркеры)
	var wg sync.WaitGroup

	// запуск HTTP
	go func() {
		logger.Log.Info("server started", zap.String("addr", cfg.RunAddr))
		if err := srv.Start(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Log.Error("server stopped unexpectedly", zap.Error(err))
			// инициируем общий shutdown
			stop()
		}
	}()

	// воркер начислений
	if cfg.Accrual != "" {
		cli := accrual.New(cfg.Accrual)
		w := orders.NewWorker(pg, cli, 50, 2*time.Second) // batch=50, тик=2с
		wg.Add(1)
		go func() {
			defer wg.Done()
			w.Run(ctx) // корректно выйдет по ctx.Done()
		}()
	} else {
		logger.Log.Warn("ACCRUAL_SYSTEM_ADDRESS not set — accrual worker disabled")
	}

	// ждём сигнал
	<-ctx.Done()

	// мягко глушим HTTP; даём время активным запросам завершиться
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Log.Warn("http shutdown", zap.Error(err))
	}

	// ждём фоновые горутины (воркеры)
	wg.Wait()

	logger.Log.Info("server exited")
}
