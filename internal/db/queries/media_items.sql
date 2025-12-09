-- name: GetMediaItem :one
SELECT * FROM media_items
WHERE id = $1;

-- name: ListMediaItems :many
SELECT * FROM media_items
WHERE
    (sqlc.narg('kind')::text IS NULL OR kind = sqlc.narg('kind'))
    AND (sqlc.narg('parent_id')::bigint IS NULL OR parent_id = sqlc.narg('parent_id'))
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
    AND (sqlc.narg('parent_id')::bigint IS NULL OR parent_id = sqlc.narg('parent_id'))
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
