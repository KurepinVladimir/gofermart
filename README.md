gofermart# Gophermart

Накопительная система лояльности (HTTP API).

## Запуск локально

### Требования
- Go 1.22+
- PostgreSQL 14+ (локально)
- (Опционально) Docker для локального развёртывания

### Переменные окружения
- `RUN_ADDRESS` — адрес HTTP (по умолчанию `:8080`)
- `DATABASE_URI` — DSN PostgreSQL (обязательно)
- `ACCRUAL_SYSTEM_ADDRESS` — базовый URL внешней системы начислений (опционально; для локальной проверки можно запустить `cmd/accrualstub`)

> Флаги `-a`, `-d`, `-r` перекрывают значения из ENV.

### Пример запуска
```bash
# Windows cmd.exe
set RUN_ADDRESS=:8080
set DATABASE_URI=postgres://app:app@127.0.0.1:5432/gophermart?sslmode=disable
go run .\cmd\gophermart
