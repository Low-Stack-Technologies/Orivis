# Tooling

This project uses contract and schema generation heavily. Install these tools before contributing.

## Required Tools

- `go` (1.22+)
- `node` and `npm` (20+ recommended)
- `make`
- `docker` + `docker compose`
- `oapi-codegen`
- `sqlc`
- `sql-migrate`

## Install Commands

Install Go tools:

```bash
go install github.com/deepmap/oapi-codegen/v2/cmd/oapi-codegen@latest
go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest
go install github.com/rubenv/sql-migrate/...@latest
```

Install frontend dependencies (includes `orval`):

```bash
npm --prefix frontend install
```

Install root dependencies (lint + tests):

```bash
npm install
```

## Core Developer Commands

```bash
make doctor
make install
make generate
make lint
make test
make build
```

For local app development:

```bash
make dev
```

## Verify Tool Availability

```bash
oapi-codegen --help
sqlc version
sql-migrate --help
npm --prefix frontend run generate:api -- --help
```

## Migration Tooling

- Migrations are authored in `backend/db/migrations` with `sql-migrate` format.
- `sqlc` reads those migration files as schema input.
- Local `sql-migrate` config lives at `backend/dbconfig.yml`.

Migration helpers:

```bash
make db-start
make db-up
make db-status
make db-new NAME=add_new_table
make db-down
make db-stop
```
