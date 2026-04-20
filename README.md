# Plata

Асинхронный сервис котировок валютных курсов.

Клиент инициирует обновление котировки по валютной паре (напр. `EUR/MXN`) и получает идентификатор запроса. 
Само обновление идёт в фоне: воркер ходит во внешний API и сохраняет результат в БД. 
По идентификатору или по паре клиент позже забирает значение.

## Тесты

Юнит-тесты запускаются без инфраструктуры:

```bash
make test                   # go test ./...
```

Интеграционные тесты (под тегом `integration`) работают против локальной
Postgres из `docker-compose.yml`. Они автоматически применяют миграции из
`migrations/` и очищают таблицу `quote_updates` перед каждым тестом:

```bash
make up                     # docker compose up -d
make test-integration       # go test -tags=integration ./...
```

Переопределить DSN (например в CI):

```bash
TEST_DB_URL="postgres://user:pass@host:5432/dbname?sslmode=disable" make test-integration
```

## Стек

- Go 1.24+
- `net/http` + [chi](https://github.com/go-chi/chi) — HTTP-роутинг
- PostgreSQL + [pgx](https://github.com/jackc/pgx) — БД
- [exchangeratesapi.io](https://exchangeratesapi.io/) — источник котировок

