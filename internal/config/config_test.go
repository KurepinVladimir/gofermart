package config

import (
	"flag"
	"os"
	"testing"
)

// сбросить пакетные флаги перед каждым вызовом Load()
func resetFlags() {
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ContinueOnError)
}

func TestLoad_FromEnv(t *testing.T) {
	t.Setenv("RUN_ADDRESS", ":9090")
	t.Setenv("DATABASE_URI", "postgres://user:pass@localhost:5432/db?sslmode=disable")
	t.Setenv("ACCRUAL_SYSTEM_ADDRESS", "http://localhost:8081")

	// Load использует пакетные флаги, их надо сбросить
	resetFlags()

	// Без флагов — берём из ENV
	origArgs := os.Args
	defer func() { os.Args = origArgs }()
	os.Args = []string{"testbin"} // флагов нет

	cfg := Load()

	if cfg.RunAddr != ":9090" {
		t.Fatalf("RunAddr: expected %q, got %q", ":9090", cfg.RunAddr)
	}
	if cfg.Database != "postgres://user:pass@localhost:5432/db?sslmode=disable" {
		t.Fatalf("Database: unexpected %q", cfg.Database)
	}
	if cfg.Accrual != "http://localhost:8081" {
		t.Fatalf("Accrual: expected %q, got %q", "http://localhost:8081", cfg.Accrual)
	}
}

func TestLoad_FlagsOverrideEnv(t *testing.T) {
	// ENV даём одни значения...
	t.Setenv("RUN_ADDRESS", ":9090")
	t.Setenv("DATABASE_URI", "postgres://env/env@localhost:5432/env?sslmode=disable")
	t.Setenv("ACCRUAL_SYSTEM_ADDRESS", "http://env:9000")

	// ...но флаги должны их переопределить
	resetFlags()

	origArgs := os.Args
	defer func() { os.Args = origArgs }()
	os.Args = []string{
		"testbin",
		"-a", "127.0.0.1:7777",
		"-d", "postgres://flag/flag@db:5432/flag?sslmode=disable",
		"-r", "http://flag:8088",
	}

	cfg := Load()

	if cfg.RunAddr != "127.0.0.1:7777" {
		t.Fatalf("RunAddr flag override failed: %q", cfg.RunAddr)
	}
	if cfg.Database != "postgres://flag/flag@db:5432/flag?sslmode=disable" {
		t.Fatalf("Database flag override failed: %q", cfg.Database)
	}
	if cfg.Accrual != "http://flag:8088" {
		t.Fatalf("Accrual flag override failed: %q", cfg.Accrual)
	}
}
