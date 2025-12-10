-- name: CreateDownload :one
INSERT INTO downloads (
    plugin_id,
    name,
    status,
    progress,
    url,
    file_content,
    file_name,
    priority,
    metadata,
    created_by_user_id
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
RETURNING *;

-- name: GetDownload :one
SELECT * FROM downloads WHERE id = $1;

-- name: ListDownloads :many
SELECT * FROM downloads
WHERE (
    CASE
        WHEN $1 != '' THEN plugin_id = $1
        ELSE TRUE
    END
)
AND (
    CASE
        WHEN $2 != '' THEN status = $2
        ELSE TRUE
    END
)
ORDER BY
    CASE WHEN status IN ('queued', 'downloading')
         THEN COALESCE(queue_position, 999999)
         ELSE 999999
    END,
    created_at DESC
LIMIT $3
OFFSET $4;

-- name: ListAllDownloads :many
SELECT * FROM downloads
ORDER BY
    CASE WHEN status IN ('queued', 'downloading')
         THEN COALESCE(queue_position, 999999)
         ELSE 999999
    END,
    created_at DESC;

-- name: ListDownloadsByPlugin :many
SELECT * FROM downloads
WHERE plugin_id = $1
ORDER BY
    CASE WHEN status IN ('queued', 'downloading')
         THEN COALESCE(queue_position, 999999)
         ELSE 999999
    END,
    created_at DESC;

-- name: ListDownloadsByStatus :many
SELECT * FROM downloads
WHERE status = $1
ORDER BY created_at DESC;

-- name: UpdateDownloadStatus :one
UPDATE downloads
SET
    status = $1,
    updated_at = CURRENT_TIMESTAMP,
    started_at = CASE
        WHEN $1 = 'downloading' AND started_at IS NULL THEN CURRENT_TIMESTAMP
        ELSE started_at
    END,
    completed_at = CASE
        WHEN $1 IN ('completed', 'failed', 'cancelled') AND completed_at IS NULL THEN CURRENT_TIMESTAMP
        ELSE completed_at
    END
WHERE id = $2
RETURNING *;

-- name: UpdateDownloadProgress :exec
UPDATE downloads
SET
    progress = $1,
    downloaded_bytes = $2,
    total_bytes = $3,
    updated_at = CURRENT_TIMESTAMP
WHERE id = $4;

-- name: UpdateDownloadError :exec
UPDATE downloads
SET
    error_message = $1,
    retry_count = retry_count + 1,
    updated_at = CURRENT_TIMESTAMP
WHERE id = $2;

-- name: UpdateDownloadDestination :exec
UPDATE downloads
SET
    destination_path = $1,
    updated_at = CURRENT_TIMESTAMP
WHERE id = $2;

-- name: UpdateDownloadQueuePosition :exec
UPDATE downloads
SET
    queue_position = $1,
    updated_at = CURRENT_TIMESTAMP
WHERE id = $2;

-- name: DeleteDownload :exec
DELETE FROM downloads WHERE id = $1;

-- name: DeleteCompletedDownloads :exec
DELETE FROM downloads
WHERE status = 'completed'
AND completed_at < CURRENT_TIMESTAMP - ($1 || ' days')::INTERVAL;

-- name: CountDownloadsByStatus :one
SELECT COUNT(*) as count FROM downloads WHERE status = $1;

-- name: CountDownloadsByPlugin :one
SELECT COUNT(*) as count FROM downloads WHERE plugin_id = $1;

-- name: GetActiveDownloads :many
SELECT * FROM downloads
WHERE status IN ('queued', 'downloading')
ORDER BY COALESCE(queue_position, 999999), created_at;

-- Download logs queries

-- name: AddDownloadLog :exec
INSERT INTO download_logs (download_id, level, message)
VALUES ($1, $2, $3);

-- name: GetDownloadLogs :many
SELECT * FROM download_logs
WHERE download_id = $1
ORDER BY created_at DESC
LIMIT $2;

-- name: GetRecentDownloadLogs :many
SELECT * FROM download_logs
WHERE download_id = $1
AND created_at > $2
ORDER BY created_at DESC;

-- name: DeleteDownloadLogs :exec
DELETE FROM download_logs WHERE download_id = $1;

-- name: DeleteOldDownloadLogs :exec
DELETE FROM download_logs
WHERE created_at < CURRENT_TIMESTAMP - ($1 || ' days')::INTERVAL;
