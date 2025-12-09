-- name: CreateUser :one
INSERT INTO users (
    username,
    email,
    is_active,
    is_admin,
    metadata
) VALUES (
    $1, $2, $3, $4, $5
) RETURNING *;

-- name: GetUserByID :one
SELECT * FROM users
WHERE id = $1 LIMIT 1;

-- name: GetUserByUsername :one
SELECT * FROM users
WHERE username = $1 LIMIT 1;

-- name: GetUserByEmail :one
SELECT * FROM users
WHERE email = $1 LIMIT 1;

-- name: UpdateUser :one
UPDATE users
SET
    username = COALESCE(sqlc.narg(username), username),
    email = COALESCE(sqlc.narg(email), email),
    is_active = COALESCE(sqlc.narg(is_active), is_active),
    is_admin = COALESCE(sqlc.narg(is_admin), is_admin),
    metadata = COALESCE(sqlc.narg(metadata), metadata)
WHERE id = $1
RETURNING *;

-- name: DeleteUser :exec
DELETE FROM users
WHERE id = $1;

-- name: ListUsers :many
SELECT * FROM users
ORDER BY created_at DESC
LIMIT $1 OFFSET $2;

-- name: CountUsers :one
SELECT COUNT(*) FROM users;
