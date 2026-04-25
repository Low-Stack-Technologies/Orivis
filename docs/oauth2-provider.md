# OAuth2/OIDC Provider Integration

Orivis acts as an Authorization Server for third-party applications.

## Endpoints

- Discovery: `GET /.well-known/openid-configuration`
- Authorization: `GET /oauth2/authorize`
- Token: `POST /oauth2/token`
- Introspection: `POST /oauth2/introspect`
- Revocation: `POST /oauth2/revoke`
- JWKS: `GET /oauth2/jwks`
- UserInfo: `GET /oauth2/userinfo`

## Supported Grant Types (v0.1)

- `authorization_code` (with PKCE)
- `refresh_token`
- `client_credentials`

## Policy Enforcement

OAuth2 authorization checks are policy-aware:

- Platform policy mode
- User override for platform
- Group override for platform

If denied, Orivis returns a policy error and records an audit event.
