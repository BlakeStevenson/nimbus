-- name: GetPlugin :one
SELECT * FROM plugins
WHERE id = $1;

-- name: ListPlugins :many
SELECT * FROM plugins
ORDER BY name;

-- name: ListEnabledPlugins :many
SELECT * FROM plugins
WHERE enabled = TRUE
ORDER BY name;

-- name: UpsertPlugin :one
INSERT INTO plugins (id, name, description, version, enabled, capabilities)
VALUES ($1, $2, $3, $4, $5, $6)
ON CONFLICT (id)
DO UPDATE SET
    name = EXCLUDED.name,
    description = EXCLUDED.description,
    version = EXCLUDED.version,
    capabilities = EXCLUDED.capabilities,
    updated_at = NOW()
RETURNING *;

-- name: EnablePlugin :exec
UPDATE plugins
SET enabled = TRUE, updated_at = NOW()
WHERE id = $1;

-- name: DisablePlugin :exec
UPDATE plugins
SET enabled = FALSE, updated_at = NOW()
WHERE id = $1;

-- name: DeletePlugin :exec
DELETE FROM plugins
WHERE id = $1;

-- name: PluginExists :one
SELECT EXISTS(SELECT 1 FROM plugins WHERE id = $1);
