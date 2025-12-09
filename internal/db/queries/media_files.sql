-- media_files.sql
-- SQLC queries for managing media files associated with media items

-- =============================================================================
-- GetMediaFile - Retrieve a media file by ID
-- =============================================================================
-- name: GetMediaFile :one
SELECT * FROM media_files
WHERE id = $1;

-- =============================================================================
-- GetMediaFileByPath - Find a media file by its filesystem path
-- =============================================================================
-- Used by scanner to check if a file has already been imported
-- name: GetMediaFileByPath :one
SELECT * FROM media_files
WHERE path = $1;

-- =============================================================================
-- ListMediaFilesByItem - Get all files associated with a media item
-- =============================================================================
-- Useful for displaying file locations or cleaning up files
-- name: ListMediaFilesByItem :many
SELECT * FROM media_files
WHERE media_item_id = $1
ORDER BY path;

-- =============================================================================
-- ListMediaFiles - Get all media files with pagination
-- =============================================================================
-- name: ListMediaFiles :many
SELECT * FROM media_files
ORDER BY created_at DESC
LIMIT $1 OFFSET $2;

-- =============================================================================
-- CountMediaFiles - Count total media files
-- =============================================================================
-- name: CountMediaFiles :one
SELECT COUNT(*) FROM media_files;

-- =============================================================================
-- CreateMediaFile - Insert a new media file record
-- =============================================================================
-- name: CreateMediaFile :one
INSERT INTO media_files (
    media_item_id,
    path,
    size,
    hash
) VALUES (
    $1, $2, $3, $4
)
RETURNING *;

-- =============================================================================
-- UpdateMediaFile - Update an existing media file record
-- =============================================================================
-- name: UpdateMediaFile :one
UPDATE media_files
SET
    media_item_id = COALESCE(sqlc.narg('media_item_id'), media_item_id),
    size = COALESCE(sqlc.narg('size'), size),
    hash = COALESCE(sqlc.narg('hash'), hash),
    updated_at = NOW()
WHERE id = $1
RETURNING *;

-- =============================================================================
-- UpdateMediaFileByPath - Update a media file by path
-- =============================================================================
-- Used by scanner when re-scanning existing files
-- name: UpdateMediaFileByPath :one
UPDATE media_files
SET
    media_item_id = COALESCE(sqlc.narg('media_item_id'), media_item_id),
    size = COALESCE(sqlc.narg('size'), size),
    hash = COALESCE(sqlc.narg('hash'), hash),
    updated_at = NOW()
WHERE path = $1
RETURNING *;

-- =============================================================================
-- UpsertMediaFile - Insert or update a media file by path
-- =============================================================================
-- Atomically insert or update - scanner's primary operation
-- name: UpsertMediaFile :one
INSERT INTO media_files (
    media_item_id,
    path,
    size,
    hash
) VALUES (
    $1, $2, $3, $4
)
ON CONFLICT (path) DO UPDATE SET
    media_item_id = EXCLUDED.media_item_id,
    size = EXCLUDED.size,
    hash = EXCLUDED.hash,
    updated_at = NOW()
RETURNING *;

-- =============================================================================
-- DeleteMediaFile - Delete a media file by ID
-- =============================================================================
-- name: DeleteMediaFile :exec
DELETE FROM media_files
WHERE id = $1;

-- =============================================================================
-- DeleteMediaFileByPath - Delete a media file by path
-- =============================================================================
-- name: DeleteMediaFileByPath :exec
DELETE FROM media_files
WHERE path = $1;

-- =============================================================================
-- DeleteMediaFilesByItem - Delete all files for a media item
-- =============================================================================
-- name: DeleteMediaFilesByItem :exec
DELETE FROM media_files
WHERE media_item_id = $1;

-- =============================================================================
-- ListOrphanedMediaFiles - Find files with no associated media item
-- =============================================================================
-- Useful for cleanup operations
-- name: ListOrphanedMediaFiles :many
SELECT * FROM media_files
WHERE media_item_id IS NULL
ORDER BY path;
