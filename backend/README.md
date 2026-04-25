# Backend (Go + chi)

This backend uses OpenAPI-generated server interfaces and models.

## Generation

Use the root generation script:

```bash
make generate
```

This runs:

- `oapi-codegen` for API structs/interfaces in `backend/internal/apigen`
- `sqlc` for PostgreSQL models/queries in `backend/internal/db`
- `orval` for frontend API hooks/types

## Migrations

Migration tool: `sql-migrate`

Migration files live in `backend/db/migrations` and are the schema source for `sqlc`.

Apply migrations locally (when PostgreSQL is running):

```bash
make db-up
```

Create and inspect migrations:

```bash
make db-new NAME=add_sessions_index
make db-status
```

## Run

```bash
go run ./cmd/api
```

## Next Implementation Steps

1. Generate API types and chi handlers.
2. Implement strict server interface methods.
3. Add PostgreSQL repositories and migrations.
4. Add auth/session middleware and policy evaluation service.
