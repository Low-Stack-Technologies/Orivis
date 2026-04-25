# Frontend (Vite + React + TS)

Frontend stack target:

- Vite
- React + TypeScript
- Tailwind CSS
- React Query
- Orval-generated API hooks and types

## Install

```bash
npm --prefix frontend install
```

## Run

```bash
npm --prefix frontend run dev
```

For full-stack local development, use the root command:

```bash
make dev
```

See `docs/development-workflow.md` for the complete setup/runbook.

During local development, Vite proxies all `/v1/*` API requests to `http://localhost:8080/v1/*`.

## API Code Generation

Use `frontend/orval.config.ts` with the root OpenAPI contract.

Example command:

```bash
npm --prefix frontend run generate:api
```

Generated hooks and models are intended for `frontend/src/api/generated`.
