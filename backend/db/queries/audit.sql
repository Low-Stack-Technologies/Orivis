-- name: CreateAuditEvent :one
INSERT INTO audit_events (
  actor_type,
  actor_id,
  action,
  target_type,
  target_id,
  metadata
)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING *;

-- name: ListAuditEvents :many
SELECT *
FROM audit_events
ORDER BY created_at DESC
LIMIT $1 OFFSET $2;

-- name: CountAuditEvents :one
SELECT COUNT(*)
FROM audit_events;
