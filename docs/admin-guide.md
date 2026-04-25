# Admin Guide

Admin endpoints require JWT bearer authentication.

## OAuth2 Policy Endpoints

- Platform policy: `GET|PUT /v1/admin/policies/oauth2/platforms/{platformId}`
- User override: `GET|PUT /v1/admin/policies/oauth2/users/{userId}/platforms/{platformId}`
- Group override: `GET|PUT /v1/admin/policies/oauth2/groups/{groupId}/platforms/{platformId}`

## Forward-Auth Policy Endpoints

- Platform policy: `GET|PUT /v1/admin/policies/forward-auth/platforms/{platformId}`
- User override: `GET|PUT /v1/admin/policies/forward-auth/users/{userId}/platforms/{platformId}`
- Group override: `GET|PUT /v1/admin/policies/forward-auth/groups/{groupId}/platforms/{platformId}`

## Auditing

- Audit events: `GET /v1/admin/audit-events`

All policy changes should generate audit events with actor and action metadata.
