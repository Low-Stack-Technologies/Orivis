# Development Workflow

## Local Runbook

### First-time setup

```bash
make doctor
make install
make generate
```

### Start local services

```bash
make db-start
make db-up
make dev
```

Local endpoints:

- Frontend: `http://localhost:5173`
- Backend API: `http://localhost:8080`
- Health: `http://localhost:8080/healthz`

Vite development proxy:

- Requests from frontend to `/v1/*` are proxied to `http://localhost:8080/v1/*`.

### Run backend/frontend separately

Use two terminals:

```bash
# Terminal 1
make dev-backend

# Terminal 2
make dev-frontend
```

### Stop local services

- Use `Ctrl+C` in each running dev terminal.
- Stop PostgreSQL container when done: `make db-stop`

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

When contract/schema files change, run generation first:

```bash
make generate
```

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
