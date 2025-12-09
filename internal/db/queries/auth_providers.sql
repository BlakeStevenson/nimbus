-- name: CreateAuthProvider :one
INSERT INTO auth_providers (
    user_id,
    provider_type,
    provider_id,
    credentials,
    metadata,
    is_primary
) VALUES (
    $1, $2, $3, $4, $5, $6
) RETURNING *;

-- name: GetAuthProvider :one
SELECT * FROM auth_providers
WHERE id = $1 LIMIT 1;

-- name: GetAuthProviderByUserAndType :one
SELECT * FROM auth_providers
WHERE user_id = $1 AND provider_type = $2
LIMIT 1;

-- name: GetAuthProviderByTypeAndProviderID :one
SELECT * FROM auth_providers
WHERE provider_type = $1 AND provider_id = $2
LIMIT 1;

-- name: ListAuthProvidersByUser :many
SELECT * FROM auth_providers
WHERE user_id = $1
ORDER BY is_primary DESC, created_at ASC;

-- name: GetPrimaryAuthProvider :one
SELECT * FROM auth_providers
WHERE user_id = $1 AND is_primary = true
LIMIT 1;

-- name: UpdateAuthProvider :one
UPDATE auth_providers
SET
    credentials = COALESCE(sqlc.narg(credentials), credentials),
    metadata = COALESCE(sqlc.narg(metadata), metadata),
    is_primary = COALESCE(sqlc.narg(is_primary), is_primary),
    last_used_at = COALESCE(sqlc.narg(last_used_at), last_used_at)
WHERE id = $1
RETURNING *;

-- name: UpdateAuthProviderLastUsed :exec
UPDATE auth_providers
SET last_used_at = NOW()
WHERE id = $1;

-- name: DeleteAuthProvider :exec
DELETE FROM auth_providers
WHERE id = $1;

-- name: SetPrimaryAuthProvider :exec
UPDATE auth_providers
SET is_primary = (id = $2)
WHERE user_id = $1;
