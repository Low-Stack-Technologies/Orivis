# Orivis Overview

Orivis is a self-hostable SSO platform by Low-Stack Technologies.

It is designed for teams that want full control over authentication while keeping setup and operations simple.

## Core Product Surfaces

1. **OAuth2/OIDC Provider**
   - External applications redirect users to Orivis to authenticate.
   - Orivis issues tokens and user identity claims.

2. **Forward Auth**
   - Reverse proxy asks Orivis to allow or deny each request.
   - Orivis returns identity headers for allowed requests.

## Authentication Methods

- Password
- TOTP (2FA)
- Passkeys (WebAuthn)
- External OAuth providers (Google first)

Each user can link multiple sign-in methods to the same identity.

## Policy and Governance

Administrators can control access for both OAuth2 and forward-auth:

- Platform-level mode: `allow_any`, `allowlist`, `denylist`
- Subject-level overrides for users and groups

See `docs/policy-model.md` for precedence details.
