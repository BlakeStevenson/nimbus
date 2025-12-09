-- name: GetConfig :one
SELECT * FROM config
WHERE key = $1;

-- name: GetAllConfig :many
SELECT * FROM config
ORDER BY key;

-- name: SetConfig :one
INSERT INTO config (key, value, metadata)
VALUES ($1, $2, COALESCE($3, '{}'::jsonb))
ON CONFLICT (key) DO UPDATE
SET value = EXCLUDED.value,
    metadata = CASE
        WHEN $3 IS NULL THEN config.metadata
        ELSE EXCLUDED.metadata
    END,
    updated_at = NOW()
RETURNING *;

-- name: DeleteConfig :exec
DELETE FROM config
WHERE key = $1;

-- name: GetConfigByPrefix :many
SELECT * FROM config
WHERE key LIKE $1 || '%'
ORDER BY key;
