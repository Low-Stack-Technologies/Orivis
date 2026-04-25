-- name: GetPlatformPolicy :one
SELECT id,
       auth_surface,
       platform_id,
       mode,
       entries,
       updated_at
FROM platform_policies
WHERE auth_surface = sqlc.arg(auth_surface)
  AND platform_id = sqlc.arg(platform_id);

-- name: UpsertPlatformPolicy :one
INSERT INTO platform_policies (
  auth_surface,
  platform_id,
  mode,
  entries,
  updated_at
)
VALUES (
  sqlc.arg(auth_surface),
  sqlc.arg(platform_id),
  sqlc.arg(mode),
  sqlc.arg(entries),
  NOW()
)
ON CONFLICT (auth_surface, platform_id)
DO UPDATE SET
  mode = EXCLUDED.mode,
  entries = EXCLUDED.entries,
  updated_at = NOW()
RETURNING *;

-- name: GetSubjectPolicyOverride :one
SELECT id,
       auth_surface,
       subject_type,
       subject_id::text AS subject_id,
       platform_id,
       decision,
       reason,
       updated_at
FROM subject_policy_overrides
WHERE auth_surface = sqlc.arg(auth_surface)
  AND subject_type = sqlc.arg(subject_type)
  AND subject_id = sqlc.arg(subject_id)::uuid
  AND platform_id = sqlc.arg(platform_id);

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
VALUES (
  sqlc.arg(auth_surface),
  sqlc.arg(subject_type),
  sqlc.arg(subject_id)::uuid,
  sqlc.arg(platform_id),
  sqlc.arg(decision),
  sqlc.narg(reason),
  NOW()
)
ON CONFLICT (auth_surface, subject_type, subject_id, platform_id)
DO UPDATE SET
  decision = EXCLUDED.decision,
  reason = EXCLUDED.reason,
  updated_at = NOW()
RETURNING id,
          auth_surface,
          subject_type,
          subject_id::text AS subject_id,
          platform_id,
          decision,
          reason,
          updated_at;
