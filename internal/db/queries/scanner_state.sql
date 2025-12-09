-- scanner_state.sql
-- SQLC queries for managing library scanner state (single-row table)

-- =============================================================================
-- GetScannerState - Retrieve the current scanner state
-- =============================================================================
-- Always returns the single row (id = 1)
-- name: GetScannerState :one
SELECT * FROM scanner_state
WHERE id = 1;

-- =============================================================================
-- StartScan - Mark a scan as started
-- =============================================================================
-- Sets running = true, started_at = now, resets counters
-- name: StartScan :one
UPDATE scanner_state
SET
    running = TRUE,
    started_at = NOW(),
    finished_at = NULL,
    files_scanned = 0,
    items_created = 0,
    items_updated = 0,
    errors = '[]'::jsonb,
    log = '[]'::jsonb
WHERE id = 1
RETURNING *;

-- =============================================================================
-- FinishScan - Mark a scan as completed
-- =============================================================================
-- Sets running = false, finished_at = now
-- name: FinishScan :one
UPDATE scanner_state
SET
    running = FALSE,
    finished_at = NOW()
WHERE id = 1
RETURNING *;

-- =============================================================================
-- IncrementFilesScanned - Increment the files_scanned counter
-- =============================================================================
-- name: IncrementFilesScanned :one
UPDATE scanner_state
SET files_scanned = files_scanned + 1
WHERE id = 1
RETURNING *;

-- =============================================================================
-- IncrementItemsCreated - Increment the items_created counter
-- =============================================================================
-- name: IncrementItemsCreated :one
UPDATE scanner_state
SET items_created = items_created + 1
WHERE id = 1
RETURNING *;

-- =============================================================================
-- IncrementItemsUpdated - Increment the items_updated counter
-- =============================================================================
-- name: IncrementItemsUpdated :one
UPDATE scanner_state
SET items_updated = items_updated + 1
WHERE id = 1
RETURNING *;

-- =============================================================================
-- UpdateScanProgress - Update multiple counters at once
-- =============================================================================
-- More efficient than multiple individual updates
-- name: UpdateScanProgress :one
UPDATE scanner_state
SET
    files_scanned = files_scanned + $1,
    items_created = items_created + $2,
    items_updated = items_updated + $3
WHERE id = 1
RETURNING *;

-- =============================================================================
-- AppendScanError - Add an error message to the errors array
-- =============================================================================
-- Errors are stored as JSON objects: {"timestamp": "...", "message": "..."}
-- name: AppendScanError :one
UPDATE scanner_state
SET errors = errors || sqlc.arg('error')::jsonb
WHERE id = 1
RETURNING *;

-- =============================================================================
-- AppendScanLog - Add a log entry to the log array
-- =============================================================================
-- Log entries are JSON objects: {"timestamp": "...", "level": "info", "message": "..."}
-- name: AppendScanLog :one
UPDATE scanner_state
SET log = log || sqlc.arg('log_entry')::jsonb
WHERE id = 1
RETURNING *;

-- =============================================================================
-- SetScannerRunning - Manually set the running flag
-- =============================================================================
-- Useful for error recovery or manual state management
-- name: SetScannerRunning :one
UPDATE scanner_state
SET running = $1
WHERE id = 1
RETURNING *;

-- =============================================================================
-- ResetScannerState - Reset all scanner state to defaults
-- =============================================================================
-- Use with caution - clears all progress and logs
-- name: ResetScannerState :one
UPDATE scanner_state
SET
    running = FALSE,
    started_at = NULL,
    finished_at = NULL,
    files_scanned = 0,
    items_created = 0,
    items_updated = 0,
    errors = '[]'::jsonb,
    log = '[]'::jsonb
WHERE id = 1
RETURNING *;

-- =============================================================================
-- ClearScanLogs - Clear the log array while keeping counters
-- =============================================================================
-- Useful to prevent log from growing too large
-- name: ClearScanLogs :one
UPDATE scanner_state
SET log = '[]'::jsonb
WHERE id = 1
RETURNING *;

-- =============================================================================
-- ClearScanErrors - Clear the errors array
-- =============================================================================
-- name: ClearScanErrors :one
UPDATE scanner_state
SET errors = '[]'::jsonb
WHERE id = 1
RETURNING *;
