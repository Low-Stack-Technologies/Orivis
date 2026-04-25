# Forward Auth (Traefik-First)

Orivis supports request-by-request authorization for reverse proxies using a forward-auth style contract.

## Endpoint

- `GET /v1/forward-auth/check`

## Expected Request Headers

- `X-Forwarded-Method`
- `X-Forwarded-Host`
- `X-Forwarded-Uri`
- Optional: `X-Forwarded-Proto`, `X-Forwarded-For`

## Decision Outcomes

- `200` allow
- `401` unauthenticated
- `403` authenticated but forbidden by policy

When allowed, Orivis returns identity headers like:

- `X-Orivis-Subject`
- `X-Orivis-Email`
- `X-Orivis-Groups`
- `X-Orivis-Decision-Reason`

## Policy Controls

Forward-auth uses the same policy hierarchy as OAuth2:

- Platform mode (allow any / allowlist / denylist)
- User override
- Group override
