.PHONY: build run test test-integration test-all up down migrate

# -- App --------------------------------------------------------------

build:
	go build -o bin/server ./cmd/server

run:
	go run ./cmd/server

# -- Infra ------------------------------------------------------------

up:
	docker compose up -d

down:
	docker compose down

migrate:
	psql "$${DB_URL:-postgres://plata:plata@localhost:5432/plata?sslmode=disable}" -f migrations/001_init.sql

# -- Tests ------------------------------------------------------------

# Fast unit tests only (no Postgres required).
test:
	go test ./...

# Integration tests — require Postgres available at localhost:5432
# (or TEST_DB_URL). Run `make up` first, or the target pipeline brings it up.
test-integration:
	go test -tags=integration ./... -count=1

# Full suite: bring Postgres up, run all tests.
test-all: up
	go test -tags=integration ./... -count=1
