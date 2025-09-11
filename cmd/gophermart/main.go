package main

import (
	"context"
	"os/signal"
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

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	pg, err := repository.NewPostgres(ctx, cfg.Database)
	if err != nil {
		logger.Log.Fatal("db connect failed", zap.Error(err))
	}
	defer pg.Close()

	//запускаем миграции
	if err := pg.RunMigrations(ctx); err != nil {
		logger.Log.Fatal("migrations failed", zap.Error(err))
	}

	// http server
	srv := app.NewServer(cfg.RunAddr, pg)

	// ЗАПУСК HTTP
	go func() {
		logger.Log.Info("server started", zap.String("addr", cfg.RunAddr))
		if err := srv.Start(); err != nil {
			logger.Log.Error("server stopped", zap.Error(err))
			stop()
		}
	}()

	// === НОВОЕ: воркер начислений ===
	if cfg.Accrual != "" {
		cli := accrual.New(cfg.Accrual)
		w := orders.NewWorker(pg, cli, 50, 2*time.Second) // batch=50, тик=2с
		go w.Run(ctx)
	} else {
		logger.Log.Warn("ACCRUAL_SYSTEM_ADDRESS not set — accrual worker disabled")
	}

	<-ctx.Done()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = srv.Shutdown(shutdownCtx)
	logger.Log.Info("server exited")
}
