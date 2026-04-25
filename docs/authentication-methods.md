# Authentication Methods

## Local Credentials

- Password login starts at `POST /v1/auth/login/password`.
- If second factor is required, API returns `challenge_required`.

## TOTP (2FA)

- User submits a 6-digit code at `POST /v1/auth/challenge/totp`.
- Successful challenge upgrades session to authenticated.

## Passkeys (WebAuthn)

1. Fetch challenge options at `POST /v1/auth/challenge/webauthn/options`.
2. Verify assertion at `POST /v1/auth/challenge/webauthn/verify`.

## External OAuth Providers

- Start provider login via `POST /v1/auth/providers/{provider}/start`.
- Complete login via `POST /v1/auth/providers/{provider}/callback`.
- Google is modeled as the first provider.

## Linked Sign-In Methods

- List methods: `GET /v1/me/methods`
- Link method: `POST /v1/me/methods`
- Unlink method: `DELETE /v1/me/methods/{methodId}`

One Orivis user can hold multiple methods simultaneously.
