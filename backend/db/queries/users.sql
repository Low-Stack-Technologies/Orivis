-- name: CreateUser :one
INSERT INTO users (email, username, password_hash)
VALUES (sqlc.arg(email), sqlc.arg(username), sqlc.arg(password_hash))
RETURNING id::text AS id, email, username, password_hash, created_at, updated_at;

-- name: GetUserByID :one
SELECT id::text AS id, email, username, password_hash, created_at, updated_at
FROM users
WHERE id = sqlc.arg(user_id)::uuid;

-- name: GetUserByEmail :one
SELECT id::text AS id, email, username, password_hash, created_at, updated_at
FROM users
WHERE email = sqlc.arg(email);

-- name: GetUserByIdentifier :one
SELECT id::text AS id, email, username, password_hash, created_at, updated_at
FROM users
WHERE email = sqlc.arg(identifier) OR username = sqlc.arg(identifier);

-- name: ListUserGroups :many
SELECT g.id::text AS id, g.name, g.created_at
FROM groups g
INNER JOIN user_groups ug ON ug.group_id = g.id
WHERE ug.user_id = sqlc.arg(user_id)::uuid
ORDER BY g.name ASC;

-- name: ListUserAuthMethods :many
SELECT id::text AS id,
       user_id::text AS user_id,
       method_type,
       provider_subject,
       secret_ref,
       metadata,
       created_at
FROM user_auth_methods
WHERE user_id = sqlc.arg(user_id)::uuid
ORDER BY created_at DESC;

-- name: CreateUserAuthMethod :one
INSERT INTO user_auth_methods (user_id, method_type, provider_subject, secret_ref, metadata)
VALUES (
  sqlc.arg(user_id)::uuid,
  sqlc.arg(method_type),
  sqlc.narg(provider_subject),
  sqlc.narg(secret_ref),
  sqlc.arg(metadata)
)
RETURNING id::text AS id,
          user_id::text AS user_id,
          method_type,
          provider_subject,
          secret_ref,
          metadata,
          created_at;

-- name: DeleteUserAuthMethod :execrows
DELETE FROM user_auth_methods
WHERE id = sqlc.arg(method_id)::uuid
  AND user_id = sqlc.arg(user_id)::uuid;

-- name: UpsertTotpFactor :exec
INSERT INTO totp_factors (user_id, secret, enabled, updated_at)
VALUES (sqlc.arg(user_id)::uuid, sqlc.arg(secret), sqlc.arg(enabled), NOW())
ON CONFLICT (user_id) DO UPDATE SET
  secret = EXCLUDED.secret,
  enabled = EXCLUDED.enabled,
  updated_at = NOW();

-- name: GetTotpFactorByUserID :one
SELECT user_id::text AS user_id,
       secret,
       enabled,
       created_at,
       updated_at
FROM totp_factors
WHERE user_id = sqlc.arg(user_id)::uuid;
