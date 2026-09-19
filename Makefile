# Loads .env when present so targets see DATABASE_URL and friends.
ifneq (,$(wildcard .env))
include .env
export
endif

.PHONY: dev-db test-db migrate seed serve worker sqlc test vet

dev-db:
	docker compose up -d --wait postgres

# Creates the empty database that database-backed tests use (TEST_DATABASE_URL).
test-db:
	docker compose exec -T postgres psql -U ground -d ground -tc "SELECT 1 FROM pg_database WHERE datname = 'ground_test'" | grep -q 1 || \
		docker compose exec -T postgres createdb -U ground ground_test

migrate:
	go run ./cmd/app migrate

seed:
	go run ./cmd/app seed --samples

serve:
	go run ./cmd/app serve

worker:
	go run ./cmd/app worker

sqlc:
	go tool sqlc generate

vet:
	go vet ./...

# Tests that need Postgres skip themselves unless TEST_DATABASE_URL is set.
# -p 1: those tests share one database, so packages must not run in parallel.
test:
	go test -p 1 ./...
