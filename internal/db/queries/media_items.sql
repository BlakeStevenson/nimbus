-- name: GetMediaItem :one
SELECT * FROM media_items
WHERE id = $1;

-- name: ListMediaItems :many
SELECT * FROM media_items
WHERE
    (sqlc.narg('kind')::text IS NULL OR kind = sqlc.narg('kind'))
    AND (
        -- If parent_id is explicitly provided, use it (even if it's NULL to find top-level items)
        (sqlc.narg('parent_id')::bigint IS NOT NULL AND parent_id = sqlc.narg('parent_id'))
        OR
        -- Otherwise, apply top_level_only filter if set
        (sqlc.narg('parent_id')::bigint IS NULL AND (NOT sqlc.narg('top_level_only')::boolean OR parent_id IS NULL))
    )
    AND (
        sqlc.narg('search')::text IS NULL
        OR title ILIKE '%' || sqlc.narg('search') || '%'
        OR sort_title ILIKE '%' || sqlc.narg('search') || '%'
    )
ORDER BY sort_title, created_at DESC
LIMIT sqlc.narg('limit')
OFFSET sqlc.narg('offset');

-- name: CountMediaItems :one
SELECT COUNT(*) FROM media_items
WHERE
    (sqlc.narg('kind')::text IS NULL OR kind = sqlc.narg('kind'))
    AND (
        -- If parent_id is explicitly provided, use it (even if it's NULL to find top-level items)
        (sqlc.narg('parent_id')::bigint IS NOT NULL AND parent_id = sqlc.narg('parent_id'))
        OR
        -- Otherwise, apply top_level_only filter if set
        (sqlc.narg('parent_id')::bigint IS NULL AND (NOT sqlc.narg('top_level_only')::boolean OR parent_id IS NULL))
    )
    AND (
        sqlc.narg('search')::text IS NULL
        OR title ILIKE '%' || sqlc.narg('search') || '%'
        OR sort_title ILIKE '%' || sqlc.narg('search') || '%'
    );

-- name: CreateMediaItem :one
INSERT INTO media_items (
    kind,
    title,
    sort_title,
    year,
    external_ids,
    metadata,
    parent_id
) VALUES (
    $1, $2, $3, $4, $5, $6, $7
)
RETURNING *;

-- name: UpdateMediaItem :one
UPDATE media_items
SET
    title = COALESCE(sqlc.narg('title'), title),
    sort_title = COALESCE(sqlc.narg('sort_title'), sort_title),
    year = COALESCE(sqlc.narg('year'), year),
    external_ids = COALESCE(sqlc.narg('external_ids'), external_ids),
    metadata = COALESCE(sqlc.narg('metadata'), metadata),
    parent_id = COALESCE(sqlc.narg('parent_id'), parent_id)
WHERE id = sqlc.arg('id')
RETURNING *;

-- name: DeleteMediaItem :exec
DELETE FROM media_items
WHERE id = $1;

-- name: ListMediaItemsByKind :many
SELECT * FROM media_items
WHERE kind = $1
ORDER BY sort_title, created_at DESC
LIMIT $2
OFFSET $3;

-- name: ListChildMediaItems :many
SELECT * FROM media_items
WHERE parent_id = $1
ORDER BY sort_title, created_at DESC;

-- name: GetMediaItemsByExternalID :many
SELECT * FROM media_items
WHERE external_ids @> sqlc.arg('external_id')::jsonb;

-- =============================================================================
-- Scanner-specific queries
-- =============================================================================

-- name: GetMediaItemByTitleAndYear :one
SELECT * FROM media_items
WHERE title = $1 AND year = $2 AND kind = $3
LIMIT 1;

-- name: GetMediaItemByTitleYearAndParent :one
SELECT * FROM media_items
WHERE title = $1
  AND (year = $2 OR (year IS NULL AND $2::int IS NULL))
  AND kind = $3
  AND (parent_id = $4 OR (parent_id IS NULL AND $4::bigint IS NULL))
LIMIT 1;

-- name: UpsertMediaItem :one
INSERT INTO media_items (
    kind,
    title,
    sort_title,
    year,
    external_ids,
    metadata,
    parent_id
) VALUES (
    $1, $2, $3, $4, $5, $6, $7
)
ON CONFLICT (kind, title, COALESCE(year, -1), COALESCE(parent_id, -1))
DO UPDATE SET
    sort_title = EXCLUDED.sort_title,
    external_ids = media_items.external_ids || EXCLUDED.external_ids,
    metadata = media_items.metadata || EXCLUDED.metadata,
    updated_at = NOW()
RETURNING *;

-- name: UpdateMediaMetadata :one
UPDATE media_items
SET
    metadata = media_items.metadata || sqlc.arg('metadata')::jsonb,
    updated_at = NOW()
WHERE id = sqlc.arg('id')
RETURNING *;

-- name: UpdateMediaExternalIDs :one
UPDATE media_items
SET
    external_ids = media_items.external_ids || sqlc.arg('external_ids')::jsonb,
    updated_at = NOW()
WHERE id = sqlc.arg('id')
RETURNING *;
