-- name: CreateAuthChallenge :one
INSERT INTO auth_challenges (challenge_type, user_id, data, expires_at)
VALUES (sqlc.arg(challenge_type), sqlc.narg(user_id)::uuid, sqlc.arg(data), sqlc.arg(expires_at))
RETURNING id::text AS id,
          challenge_type,
          user_id::text AS user_id,
          data,
          expires_at,
          consumed_at,
          created_at;

-- name: GetAuthChallengeByID :one
SELECT id::text AS id,
       challenge_type,
       user_id::text AS user_id,
       data,
       expires_at,
       consumed_at,
       created_at
FROM auth_challenges
WHERE id = sqlc.arg(challenge_id)::uuid;

-- name: ConsumeAuthChallenge :exec
UPDATE auth_challenges
SET consumed_at = NOW()
WHERE id = sqlc.arg(challenge_id)::uuid
  AND consumed_at IS NULL;

-- name: CreateExternalIdentity :one
INSERT INTO external_identities (user_id, provider, provider_subject, email, metadata)
VALUES (
  sqlc.arg(user_id)::uuid,
  sqlc.arg(provider),
  sqlc.arg(provider_subject),
  sqlc.narg(email),
  sqlc.arg(metadata)
)
RETURNING id::text AS id,
          user_id::text AS user_id,
          provider,
          provider_subject,
          email,
          metadata,
          created_at;

-- name: GetExternalIdentityByProviderSubject :one
SELECT id::text AS id,
       user_id::text AS user_id,
       provider,
       provider_subject,
       email,
       metadata,
       created_at
FROM external_identities
WHERE provider = sqlc.arg(provider)
  AND provider_subject = sqlc.arg(provider_subject);

-- name: CreateWebAuthnCredential :one
INSERT INTO webauthn_credentials (user_id, credential_id, public_key, sign_count, transports, metadata)
VALUES (
  sqlc.arg(user_id)::uuid,
  sqlc.arg(credential_id),
  sqlc.narg(public_key),
  sqlc.arg(sign_count),
  sqlc.arg(transports),
  sqlc.arg(metadata)
)
RETURNING id::text AS id,
          user_id::text AS user_id,
          credential_id,
          public_key,
          sign_count,
          transports,
          metadata,
          created_at;

-- name: GetWebAuthnCredentialByCredentialID :one
SELECT id::text AS id,
       user_id::text AS user_id,
       credential_id,
       public_key,
       sign_count,
       transports,
       metadata,
       created_at
FROM webauthn_credentials
WHERE credential_id = sqlc.arg(credential_id);

-- name: ListWebAuthnCredentialsByUserID :many
SELECT id::text AS id,
       user_id::text AS user_id,
       credential_id,
       public_key,
       sign_count,
       transports,
       metadata,
       created_at
FROM webauthn_credentials
WHERE user_id = sqlc.arg(user_id)::uuid
ORDER BY created_at DESC;

-- name: CreateSession :one
INSERT INTO sessions (user_id, refresh_token_hash, expires_at)
VALUES (sqlc.arg(user_id)::uuid, sqlc.arg(refresh_token_hash), sqlc.arg(expires_at))
RETURNING id::text AS id,
          user_id::text AS user_id,
          refresh_token_hash,
          expires_at,
          created_at;

-- name: GetSessionByRefreshTokenHash :one
SELECT id::text AS id,
       user_id::text AS user_id,
       refresh_token_hash,
       expires_at,
       created_at
FROM sessions
WHERE refresh_token_hash = sqlc.arg(refresh_token_hash);

-- name: DeleteSessionByID :execrows
DELETE FROM sessions
WHERE id = sqlc.arg(session_id)::uuid;

-- name: CreateOAuthAuthorizationCode :one
INSERT INTO oauth_authorization_codes (
  code_hash,
  client_id,
  user_id,
  redirect_uri,
  scope,
  code_challenge,
  code_challenge_method,
  expires_at
)
VALUES (
  sqlc.arg(code_hash),
  sqlc.arg(client_id),
  sqlc.arg(user_id)::uuid,
  sqlc.arg(redirect_uri),
  sqlc.arg(scope),
  sqlc.narg(code_challenge),
  sqlc.narg(code_challenge_method),
  sqlc.arg(expires_at)
)
RETURNING id::text AS id,
          code_hash,
          client_id,
          user_id::text AS user_id,
          redirect_uri,
          scope,
          code_challenge,
          code_challenge_method,
          expires_at,
          consumed_at,
          created_at;

-- name: GetOAuthAuthorizationCodeByHash :one
SELECT id::text AS id,
       code_hash,
       client_id,
       user_id::text AS user_id,
       redirect_uri,
       scope,
       code_challenge,
       code_challenge_method,
       expires_at,
       consumed_at,
       created_at
FROM oauth_authorization_codes
WHERE code_hash = sqlc.arg(code_hash);

-- name: ConsumeOAuthAuthorizationCode :exec
UPDATE oauth_authorization_codes
SET consumed_at = NOW()
WHERE id = sqlc.arg(code_id)::uuid
  AND consumed_at IS NULL;

-- name: GetOAuthClientByID :one
SELECT id,
       name,
       redirect_uris,
       scopes,
       confidential,
       created_at,
       client_secret_hash,
       require_pkce
FROM oauth_clients
WHERE id = sqlc.arg(client_id);

-- name: CreateOAuthClient :one
INSERT INTO oauth_clients (id, name, redirect_uris, scopes, confidential, client_secret_hash, require_pkce)
VALUES (
  sqlc.arg(id),
  sqlc.arg(name),
  sqlc.arg(redirect_uris),
  sqlc.arg(scopes),
  sqlc.arg(confidential),
  sqlc.narg(client_secret_hash),
  sqlc.arg(require_pkce)
)
RETURNING id,
          name,
          redirect_uris,
          scopes,
          confidential,
          created_at,
          client_secret_hash,
          require_pkce;

-- name: CreateOAuthAccessToken :one
INSERT INTO oauth_access_tokens (token_hash, token_jti, client_id, user_id, scope, expires_at)
VALUES (
  sqlc.arg(token_hash),
  sqlc.arg(token_jti),
  sqlc.arg(client_id),
  sqlc.narg(user_id)::uuid,
  sqlc.arg(scope),
  sqlc.arg(expires_at)
)
RETURNING id::text AS id,
          token_hash,
          token_jti,
          client_id,
          user_id::text AS user_id,
          scope,
          expires_at,
          revoked_at,
          created_at;

-- name: GetOAuthAccessTokenByHash :one
SELECT id::text AS id,
       token_hash,
       token_jti,
       client_id,
       user_id::text AS user_id,
       scope,
       expires_at,
       revoked_at,
       created_at
FROM oauth_access_tokens
WHERE token_hash = sqlc.arg(token_hash);

-- name: RevokeOAuthAccessTokenByHash :exec
UPDATE oauth_access_tokens
SET revoked_at = NOW()
WHERE token_hash = sqlc.arg(token_hash)
  AND revoked_at IS NULL;

-- name: CreateOAuthRefreshToken :one
INSERT INTO oauth_refresh_tokens (token_hash, access_token_id, client_id, user_id, scope, expires_at)
VALUES (
  sqlc.arg(token_hash),
  sqlc.arg(access_token_id)::uuid,
  sqlc.arg(client_id),
  sqlc.narg(user_id)::uuid,
  sqlc.arg(scope),
  sqlc.arg(expires_at)
)
RETURNING id::text AS id,
          token_hash,
          access_token_id::text AS access_token_id,
          client_id,
          user_id::text AS user_id,
          scope,
          expires_at,
          revoked_at,
          created_at;

-- name: GetOAuthRefreshTokenByHash :one
SELECT id::text AS id,
       token_hash,
       access_token_id::text AS access_token_id,
       client_id,
       user_id::text AS user_id,
       scope,
       expires_at,
       revoked_at,
       created_at
FROM oauth_refresh_tokens
WHERE token_hash = sqlc.arg(token_hash);

-- name: RevokeOAuthRefreshTokenByHash :exec
UPDATE oauth_refresh_tokens
SET revoked_at = NOW()
WHERE token_hash = sqlc.arg(token_hash)
  AND revoked_at IS NULL;
