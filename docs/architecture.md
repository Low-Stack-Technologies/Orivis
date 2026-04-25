# Architecture

## Technology Decisions

- **Backend**: Go + chi router
- **Database**: PostgreSQL
- **API contract**: OpenAPI 3.1
- **Frontend**: Vite + React + TypeScript + Tailwind CSS + React Query + Orval

## Contract-First Workflow

OpenAPI is the source of truth.

1. Design and review endpoint contracts in `openapi/orivis.openapi.yaml`.
2. Validate contract with lint and tests.
3. Generate Go structs/interfaces and TypeScript clients.
4. Implement handlers and UI against generated types.

## Runtime Components

1. **Identity API**
   - Registration, login, MFA challenges, auth method linking.
2. **OAuth2/OIDC Provider**
   - Authorization, token, introspection, revocation, JWKS, userinfo.
3. **Forward Auth Engine**
   - Policy-aware request decisions for reverse proxies.
4. **Policy Service**
   - Global platform rules and user/group overrides.
5. **Audit Service**
   - Immutable admin/security event history.

## Data Ownership

- PostgreSQL stores users, sessions, auth methods, clients, policy rules, and audit events.
- Cryptographic material (signing keys, passkey metadata) is versioned and rotated.
