package config

import (
	"flag"
	"os"
)

type Config struct {
	RunAddr  string
	Database string
}

func Load() Config {
	var cfg Config
	flag.StringVar(&cfg.RunAddr, "a", ":8080", "server listen address")
	flag.StringVar(&cfg.Database, "d", "", "postgres DSN")
	flag.Parse()

	if v := os.Getenv("RUN_ADDRESS"); v != "" {
		cfg.RunAddr = v
	}
	if v := os.Getenv("DATABASE_URI"); v != "" {
		cfg.Database = v
	}
	return cfg
}
