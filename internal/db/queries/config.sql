-- name: GetConfig :one
SELECT * FROM config
WHERE key = $1;

-- name: GetAllConfig :many
SELECT * FROM config
ORDER BY key;

-- name: SetConfig :one
INSERT INTO config (key, value)
VALUES ($1, $2)
ON CONFLICT (key) DO UPDATE
SET value = EXCLUDED.value,
    updated_at = NOW()
RETURNING *;

-- name: DeleteConfig :exec
DELETE FROM config
WHERE key = $1;

-- name: GetConfigByPrefix :many
SELECT * FROM config
WHERE key LIKE $1 || '%'
ORDER BY key;
