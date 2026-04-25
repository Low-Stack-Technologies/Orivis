# Development Workflow

## Contract-First Loop

1. Edit `openapi/orivis.openapi.yaml`.
2. Run lint and contract tests.
3. Regenerate backend/frontend/db artifacts.
4. Implement code against generated types.
5. Validate behavior with BDD scenarios.

## Local Commands

```bash
make lint
make test
make build
```

Use `make dev` to run backend and frontend together.

## Generation Commands

Canonical generation entrypoint:

```bash
make generate
```

The script runs:

- `oapi-codegen` for Go API types/interfaces
- `sqlc` for PostgreSQL models/query code
- `orval` for frontend hooks/types

Migration tool:

- `sql-migrate` with config at `backend/dbconfig.yml`
- Migrations in `backend/db/migrations`

Migration commands:

```bash
make db-start
make db-up
make db-status
make db-new NAME=describe_change
make db-down
make db-stop
```
