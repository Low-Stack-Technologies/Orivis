# Orivis

Orivis is an SSO platform by [Low-Stack Technologies](https://www.low-stack.tech/), built to let teams roll their own authentication with a clean operator experience.

The platform supports:

- OAuth2 / OpenID Connect provider mode for third-party applications
- Forward-auth for reverse proxies (Traefik-style behavior first)
- Multiple sign-in methods per user (password, TOTP, passkeys/WebAuthn, external OAuth2 providers)
- Fine-grained access policy controls at platform, group, and user level

## Repository Layout

- `openapi/` API contract (source of truth)
- `tests/contract/` contract tests against the OpenAPI spec
- `tests/bdd/` Gherkin feature specs for TDD behavior coverage
- `docs/` architecture and operator/developer documentation
- `backend/` Go (chi) backend bootstrap and codegen config
- `frontend/` Vite React + TypeScript + Tailwind + Orval + React Query bootstrap docs/config

## Current Status

This repository currently contains the contract-first foundation:

1. OpenAPI 3.1 specification for core and admin flows
2. Contract tests that validate the API design
3. BDD scenarios that capture expected behavior
4. Architecture and integration documentation

## Quick Start

```bash
make doctor
make install
make generate
make lint
make test
make build
```

Tool installation guidance is documented in `docs/tooling.md`.

## Local PostgreSQL

```bash
make db-start
make db-up
make db-status
```

Use `make db-down` to roll back one migration and `make db-stop` to stop the container.

## Technology Direction

- **Frontend:** Vite, React, TypeScript, Tailwind CSS, Orval, React Query
- **Backend:** Go, chi router, OpenAPI-generated structs/interfaces
- **Database:** PostgreSQL

## Up Next

- Implement backend handlers behind generated interfaces
- Add PostgreSQL-backed repositories and migrations
- Build dashboard UI using generated Orval hooks
- Add end-to-end tests across auth, OAuth2, and forward-auth flows
