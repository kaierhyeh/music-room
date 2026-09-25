# Load .env if present (used by migrate-up, run, etc.)
ifneq (,$(wildcard .env))
include .env
export
endif

.PHONY: deps tools sqlc migrate-up migrate-down run build test e2e swagger benchmark docker-up  docker-down seed seed-tracks seed-bulk seed-reset

deps: ## Download Go dependencies
	go mod download

tools: ## Install dev tools (sqlc, golang-migrate, swag, k6)
	go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest
	go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest
	go install github.com/swaggo/swag/cmd/swag@latest
	go install go.k6.io/k6@latest

sqlc: ## Regenerate Go code from SQL files
	sqlc generate

migrate-up: ## Apply all migrations
	migrate -path migrations -database "$(DATABASE_URL)" up

migrate-down: ## Roll back the last migration
	migrate -path migrations -database "$(DATABASE_URL)" down 1

run: ## Start the server locally
	go run ./cmd/server

build: ## Compile the binary
	go build -o bin/server ./cmd/server

test-integration: ## Run the integration tests against a throwaway database
	@docker compose exec -T postgres psql -U musicroom -d postgres \
		-c "DROP DATABASE IF EXISTS musicroom_test;" \
		-c "CREATE DATABASE musicroom_test OWNER musicroom;" >/dev/null
	TEST_DATABASE_URL="postgres://musicroom:musicroom@localhost:5432/musicroom_test?sslmode=disable" \
		go test ./internal/api/... -v

test: ## Run tests
	go test ./...

e2e: ## Run the end-to-end suite against the running stack (make docker-up first)
	go test -tags e2e -count=1 -v ./e2e/

swagger: ## Regenerate OpenAPI docs
	swag init -g cmd/server/main.go -o docs

benchmark: ## Run k6 load benchmarks (stack must be up: make docker-up)
	k6 run benchmarks/auth.js
	@echo "Set TOKEN, EVENT_ID, TRACK_ID env vars, then run:"
	@echo "  k6 run benchmarks/vote-ws.js"
	@echo "  k6 run benchmarks/playlist-ws.js"

# --- seeds ----------------------------------------------------------------
# Everything runs through the postgres container, so no local psql is needed.
# PSQL is overridable if you would rather point at another database.
PSQL ?= docker compose exec -T postgres psql -v ON_ERROR_STOP=1 -U musicroom -d musicroom
SEED_BULK_USERS ?= 500

seed: ## Load the demo dataset (6 users, 6 playlists, 4 events). Password123!
	$(PSQL) -f - < seeds/tracks.sql
	$(PSQL) -f - < seeds/seed.sql
	@echo "seeded -- log in as alice@musicroom.local / Password123!"

seed-tracks: ## Refresh seeds/tracks.sql from Deezer (preview URLs expire)
	./seeds/fetch-tracks.sh > seeds/tracks.sql
	@echo "seeds/tracks.sql regenerated -- run 'make seed' to apply it"

seed-bulk: ## Add SEED_BULK_USERS load-test accounts on top of the demo data
	$(PSQL) -v users=$(SEED_BULK_USERS) -f - < seeds/seed-bulk.sql

seed-reset: ## Drop every seeded account, then reload the demo dataset
	$(PSQL) -f - < seeds/teardown.sql
	$(MAKE) seed

docker-up: ## Start the stack (postgres, app, adminer)
	docker compose up --build

docker-down: ## Stop the stack
	docker compose down
