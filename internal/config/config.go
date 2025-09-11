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
	flag.StringVar(&cfg.RunAddr, "a", ":8080", "server listen address")
	flag.StringVar(&cfg.Database, "d", "", "postgres DSN")
	flag.StringVar(&cfg.Accrual, "r", "", "accrual system base URL")
	flag.Parse()

	if v := os.Getenv("RUN_ADDRESS"); v != "" {
		cfg.RunAddr = v
	}
	if v := os.Getenv("DATABASE_URI"); v != "" {
		cfg.Database = v
	}
	if v := os.Getenv("ACCRUAL_SYSTEM_ADDRESS"); v != "" {
		cfg.Accrual = v
	}
	return cfg
}
