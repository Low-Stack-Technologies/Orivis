-- name: CreateUser :one
INSERT INTO users (email, username, password_hash)
VALUES ($1, $2, $3)
RETURNING *;

-- name: GetUserByID :one
SELECT *
FROM users
WHERE id = $1;

-- name: GetUserByEmail :one
SELECT *
FROM users
WHERE email = $1;

-- name: ListUserGroups :many
SELECT g.*
FROM groups g
INNER JOIN user_groups ug ON ug.group_id = g.id
WHERE ug.user_id = $1
ORDER BY g.name ASC;

-- name: ListUserAuthMethods :many
SELECT *
FROM user_auth_methods
WHERE user_id = $1
ORDER BY created_at DESC;
