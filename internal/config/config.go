package config

import (
	"flag"
	"os"
)

type Config struct {
	RunAddr  string
	Database string
	Accrual  string // базовый URL внешней системы, напр. http://localhost:8081
}

func Load() Config {
	var cfg Config

	// 1) дефолты из ENV (если есть)
	runDefault := getenv("RUN_ADDRESS", ":8080")
	dbDefault := getenv("DATABASE_URI", "")
	accDefault := getenv("ACCRUAL_SYSTEM_ADDRESS", "")

	// 2) объявляем флаги с этими дефолтами
	flag.StringVar(&cfg.RunAddr, "a", runDefault, "server listen address")
	flag.StringVar(&cfg.Database, "d", dbDefault, "postgres DSN")
	flag.StringVar(&cfg.Accrual, "r", accDefault, "accrual system base URL")
	flag.Parse()

	// 3) готово: если флаги не заданы — останутся значения из ENV/дефолта,
	// если заданы — флаги переопределяют ENV.
	return cfg
}

func getenv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
