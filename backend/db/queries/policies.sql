-- name: GetPlatformPolicy :one
SELECT *
FROM platform_policies
WHERE auth_surface = $1
  AND platform_id = $2;

-- name: UpsertPlatformPolicy :one
INSERT INTO platform_policies (
  auth_surface,
  platform_id,
  mode,
  entries,
  updated_at
)
VALUES ($1, $2, $3, $4, NOW())
ON CONFLICT (auth_surface, platform_id)
DO UPDATE SET
  mode = EXCLUDED.mode,
  entries = EXCLUDED.entries,
  updated_at = NOW()
RETURNING *;

-- name: GetSubjectPolicyOverride :one
SELECT *
FROM subject_policy_overrides
WHERE auth_surface = $1
  AND subject_type = $2
  AND subject_id = $3
  AND platform_id = $4;

-- name: UpsertSubjectPolicyOverride :one
INSERT INTO subject_policy_overrides (
  auth_surface,
  subject_type,
  subject_id,
  platform_id,
  decision,
  reason,
  updated_at
)
VALUES ($1, $2, $3, $4, $5, $6, NOW())
ON CONFLICT (auth_surface, subject_type, subject_id, platform_id)
DO UPDATE SET
  decision = EXCLUDED.decision,
  reason = EXCLUDED.reason,
  updated_at = NOW()
RETURNING *;
