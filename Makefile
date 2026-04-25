.PHONY: help doctor install install-root install-frontend generate lint test test-contract test-bdd build build-frontend build-backend dev dev-frontend dev-backend db-start db-stop db-logs db-up db-down db-status db-new

help:
	@printf "Orivis developer commands\n\n"
	@printf "  make doctor          Check required local tools\n"
	@printf "  make install         Install root + frontend dependencies\n"
	@printf "  make generate        Generate Go, SQLc, and Orval artifacts\n"
	@printf "  make lint            Lint the OpenAPI specification\n"
	@printf "  make test            Run contract, BDD, and Go tests\n"
	@printf "  make build           Build frontend and compile backend\n"
	@printf "  make dev             Run backend API + frontend dev server\n"
	@printf "  make db-start        Start local PostgreSQL via Docker Compose\n"
	@printf "  make db-up           Apply sql-migrate migrations\n"
	@printf "  make db-down         Roll back one sql-migrate migration\n"
	@printf "  make db-status       Show sql-migrate status\n"
	@printf "  make db-new NAME=x   Create a new migration file\n"

doctor:
	@bash -c 'set -euo pipefail; \
	missing=0; \
	check_cmd() { \
	  name="$$1"; hint="$$2"; \
	  if command -v "$$name" >/dev/null 2>&1; then \
	    printf "[ok] %s\n" "$$name"; \
	  else \
	    printf "[missing] %s - %s\n" "$$name" "$$hint"; \
	    missing=1; \
	  fi; \
	}; \
	check_cmd go "install Go 1.22+ from https://go.dev/dl/"; \
	check_cmd node "install Node.js 20+ from https://nodejs.org/"; \
	check_cmd npm "npm ships with Node.js"; \
	check_cmd docker "install Docker Engine/Desktop"; \
	check_cmd oapi-codegen "go install github.com/deepmap/oapi-codegen/v2/cmd/oapi-codegen@latest"; \
	check_cmd sqlc "go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest"; \
	check_cmd sql-migrate "go install github.com/rubenv/sql-migrate/...@latest"; \
	if command -v docker >/dev/null 2>&1 && docker compose version >/dev/null 2>&1; then \
	  printf "[ok] docker compose\n"; \
	else \
	  printf "[missing] docker compose - install Docker Compose v2 plugin\n"; \
	  missing=1; \
	fi; \
	if [ -x frontend/node_modules/.bin/orval ]; then \
	  printf "[ok] orval (frontend local)\n"; \
	else \
	  printf "[missing] orval (frontend local) - run: npm --prefix frontend install\n"; \
	  missing=1; \
	fi; \
	if [ "$$missing" -ne 0 ]; then \
	  printf "\nOne or more required tools are missing.\n"; \
	  exit 1; \
	fi; \
	printf "\nAll required tools are available.\n"'

install: install-root install-frontend

install-root:
	npm install

install-frontend:
	npm --prefix frontend install

generate:
	./generate.sh

lint:
	npm run lint:openapi

test: test-contract test-bdd
	cd backend && go test ./...

test-contract:
	npm run test:contract

test-bdd:
	npm run test:bdd

build: build-frontend build-backend

build-frontend:
	npm --prefix frontend run build

build-backend:
	cd backend && go test ./...

dev:
	@bash -c 'set -euo pipefail; trap "kill 0" EXIT; (cd backend && go run ./cmd/api) & npm --prefix frontend run dev'

dev-frontend:
	npm --prefix frontend run dev

dev-backend:
	cd backend && go run ./cmd/api

db-start:
	docker compose up -d postgres

db-stop:
	docker compose down

db-logs:
	docker compose logs -f postgres

db-up:
	./scripts/db-up.sh

db-down:
	./scripts/db-down.sh

db-status:
	./scripts/db-status.sh

db-new:
	@if [ -z "$(NAME)" ]; then echo "usage: make db-new NAME=add_users_table"; exit 1; fi
	./scripts/db-new.sh "$(NAME)"
