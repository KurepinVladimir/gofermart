
# Gofermart — накопительная система лояльности

Сервис реализует HTTP API для регистрации пользователей, приёма номеров заказов, расчёта и учёта баллов лояльности, а также списаний.  
Хранилище — PostgreSQL. Для расчёта начислений используется внешний сервис «accrual» (в проекте есть заглушка/стаб).

## Содержание

- [Возможности](#возможности)
- [Требования](#требования)
- [Быстрый старт](#быстрый-старт)
- [Переменные окружения и флаги](#переменные-окружения-и-флаги)
- [Настройка PostgreSQL](#настройка-postgresql)
- [Запуск сервиса](#запуск-сервиса)
- [Проверка работоспособности](#проверка-работоспособности)
- [Запуск accrual-стаба](#запуск-accrual-стаба)
- [Тестирование](#тестирование)
- [Линтер и CI](#линтер-и-ci)
- [Архитектура и важные моменты](#архитектура-и-важные-моменты)
- [Справка по API](#справка-по-api)
- [Трюблшутинг](#трюблшутинг)

---

## Возможности

- Регистрация / логин (JWT, передача через `Authorization: Bearer`).
- Загрузка номеров заказов (проверка Луна, уникальность).
- Фоновая сверка заказов с внешним сервисом начислений и зачисление баллов.
- Просмотр статусов заказов.
- Просмотр текущего баланса и суммы всех списаний.
- Списание баллов на оплату заказа.
- Сжатие ответов/запросов (включается только когда имеет смысл).
- Graceful shutdown (корректная остановка HTTP и фоновых воркеров).
- Схема БД через миграции.

---

## Требования

- Go **1.24+** (toolchain подтягивается автоматически в CI).
- PostgreSQL **12+** (локально можно и новее).
- (Опционально) Docker + Docker Compose — для развёртывания всего окружения.  
  > Если Docker недоступен, запускайте PostgreSQL и сервисы локально.

---

## Быстрый старт

### 1) Клонирование и зависимости

git clone <repo-url>
cd gofermart
go mod download
`

### 2) Переменные (минимум)

* `DATABASE_URI` — строка подключения к вашей БД для **gophermart**
* `JWT_SECRET` — секрет для подписи JWT
* (опционально) `ACCRUAL_SYSTEM_ADDRESS` — адрес сервиса начислений, чтобы включить воркер

### 3) Запуск

:: Windows (PowerShell или cmd)
set RUN_ADDRESS=:8080
set DATABASE_URI=postgres://app:app@127.0.0.1:5432/gophermart?sslmode=disable
set JWT_SECRET=supersecret
:: при наличии accrual-стаба:
:: set ACCRUAL_SYSTEM_ADDRESS=http://localhost:8081

go run .\cmd\gophermart

---

## Переменные окружения и флаги

**Важно:** флаги **перекрывают** значения из окружения.

| Назначение                         | Переменная               | Флаг | Пример                                                         |
| ---------------------------------- | ------------------------ | ---- | -------------------------------------------------------------- |
| Адрес HTTP-сервера                 | `RUN_ADDRESS`            | `-a` | `:8080` или `127.0.0.1:8080`                                   |
| Подключение к БД (gophermart)      | `DATABASE_URI`           | `-d` | `postgres://app:app@127.0.0.1:5432/gophermart?sslmode=disable` |
| Адрес сервиса начислений (accrual) | `ACCRUAL_SYSTEM_ADDRESS` | `-r` | `http://localhost:8081`                                        |
| Секрет для JWT                     | `JWT_SECRET`             | —    | `supersecret`                                                  |

Пример с флагами:

go run ./cmd/gophermart -a :8080 -d "postgres://app:app@127.0.0.1:5432/gophermart?sslmode=disable" -r "http://localhost:8081"

---

## Настройка PostgreSQL

Создайте пользователя и БД (пример):

psql -U postgres -h 127.0.0.1 -c "CREATE USER app WITH PASSWORD 'app';"
psql -U postgres -h 127.0.0.1 -c "CREATE DATABASE gophermart OWNER app;"


Строка подключения для сервиса:

postgres://app:app@127.0.0.1:5432/gophermart?sslmode=disable

> Для **accrual** используйте **другую** БД (на том же инстансе можно):

postgres://app:app@127.0.0.1:5432/accrual?sslmode=disable

Миграции gophermart выполняются автоматически при старте.

---

## Запуск сервиса


go run ./cmd/gophermart
# или собрать бинарь:
go build -o gophermart ./cmd/gophermart
./gophermart -a :8080 -d "<DATABASE_URI>" -r "<ACCRUAL_SYSTEM_ADDRESS>"


В логах должен быть `server started`. Если `ACCRUAL_SYSTEM_ADDRESS` не задан — воркер начислений будет выключен (предупреждение в логах).

Graceful shutdown: `Ctrl+C` или SIGTERM — сервер корректно завершится, фоновые горутины остановятся.

## Проверка работоспособности

Пинг:
curl -i http://localhost:8080/ping


Регистрация:

curl -i -X POST http://localhost:8080/api/user/register \
  -H "Content-Type: application/json" \
  -d '{"login":"vova1","password":"secret"}'


Логин:

curl -i -X POST http://localhost:8080/api/user/login \
  -H "Content-Type: application/json" \
  -d '{"login":"vova1","password":"secret"}'


Сохраните токен из ответа и используйте в заголовке:

TOKEN=<скопируйте_из_ответа>

# загрузка заказа (text/plain), валидный Лун
curl -i -X POST http://localhost:8080/api/user/orders \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: text/plain" \
  --data "79927398713"

# список заказов
curl -i -X GET http://localhost:8080/api/user/orders \
  -H "Authorization: Bearer $TOKEN"

# баланс
curl -i -X GET http://localhost:8080/api/user/balance \
  -H "Authorization: Bearer $TOKEN"

# списание
curl -i -X POST http://localhost:8080/api/user/balance/withdraw \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"order":"4000000000000002","sum":70.5}'

# список списаний
curl -i -X GET http://localhost:8080/api/user/withdrawals \
  -H "Authorization: Bearer $TOKEN"


---

## Запуск accrual-стаба

В репозитории есть бинарь/стаб (или команда сборки) для «accrual».
Запустите его на `:8081` (или другом порту) и укажите адрес сервису gophermart через `ACCRUAL_SYSTEM_ADDRESS`.

Пример:

# отдельный терминал
./cmd/accrual/accrual_linux_amd64 -a :8081 -d "postgres://app:app@127.0.0.1:5432/accrual?sslmode=disable"

# далее в другом окне
export ACCRUAL_SYSTEM_ADDRESS=http://localhost:8081
go run ./cmd/gophermart


> Для автотестов практикума gophermart и accrual **должны использовать разные БД**.

---

## Тестирование

### Юнит-тесты

# Windows (обычно без -race, т.к. требуется cgo)
go test ./internal/... -count=1

Некоторые тесты используют `t.Setenv`, поэтому **не параллелятся** (`t.Parallel()` не используется там, где меняется окружение).



## Архитектура и важные моменты

* Слоистая архитектура:

  * `internal/app` — HTTP-сервер, маршрутизация (chi), мидлвари.
  * `internal/auth` — аутентификация/авторизация, JWT.
  * `internal/orders` — приём/листание заказов, валидатор Луна, воркер сверки.
  * `internal/balance` — текущий баланс и учёт списаний.
  * `internal/withdrawals` — HTTP для списаний.
  * `internal/repository` — доступ к БД (pgx/pgxpool), миграции.
  * `internal/accrual` — клиент для внешнего сервиса начислений.
  * `internal/config` — конфигурация (ENV + флаги, флаги приоритетнее).
  * `internal/logger` — zap.
* Пароли хранятся не в открытом виде (хэширование).
* Воркер начислений устойчив к рестартам — состояние хранится в БД.
* Сжатие включается только если клиент умеет и есть смысл сжать.
* Graceful shutdown завершает HTTP и фоновые горутины с тайм-аутом.

---

## Справка по API

Ключевые эндпоинты (см. ТЗ для полного описания):

* `POST /api/user/register` — регистрация (200/409/400/500). Возвращает JWT.
* `POST /api/user/login` — логин (200/401/400/500). Возвращает JWT.
* `POST /api/user/orders` — загрузка номера заказа (202/200/409/422/401/400).
* `GET  /api/user/orders` — список заказов пользователя (200/204/401).
* `GET  /api/user/balance` — баланс (200/401).
* `POST /api/user/balance/withdraw` — списание (200/402/422/401).
* `GET  /api/user/withdrawals` — список списаний (200/204/401).

Аутентификация: `Authorization: Bearer <JWT>`.

---

## Трюблшутинг

* **`JWT_SECRET is not set`** — задайте переменную окружения `JWT_SECRET` перед регистрацией/логином.
* **`db not ready` / 503 на `/ping`** — проверьте подключение к PostgreSQL, хост/порт/логин/пароль, что БД существует.
* **`409 Conflict` при регистрации** — логин уже занят.
* **Windows и `-race`** — флаг `-race` требует CGO; если не включено, запускайте тесты без `-race`.
* **Ошибки линтера в CI вида `undefined: chi/jwt/pgx`** — убедитесь, что в workflow есть шаг `go mod download`, и `go.mod/go.sum` закоммичены.

