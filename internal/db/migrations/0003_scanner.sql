-- Migration: 0003_scanner.sql
-- Description: Add scanner tables for library file management

-- =============================================================================
-- Add unique constraint to media_items for upsert operations
-- =============================================================================
-- This allows the scanner to safely upsert media items based on natural keys
-- Note: We use COALESCE(parent_id, -1) to handle NULL parent_id in unique index
-- =============================================================================

CREATE UNIQUE INDEX media_items_natural_key_idx
ON media_items(kind, title, COALESCE(year, -1), COALESCE(parent_id, -1));

-- =============================================================================
-- media_relations - Track hierarchical relationships between media items
-- =============================================================================
-- This table creates explicit parent-child relationships between media items.
-- Examples:
--   - TV Series -> Seasons (relation: "series-season")
--   - Seasons -> Episodes (relation: "season-episode")
--   - Music Artist -> Albums (relation: "artist-album")
--   - Albums -> Tracks (relation: "album-track")
--   - Book Series -> Books (relation: "series-book")
--
-- The sort_index field stores ordering information like season number, episode
-- number, or track number for proper sorting.
-- =============================================================================

CREATE TABLE media_relations (
    id BIGSERIAL PRIMARY KEY,
    parent_id BIGINT NOT NULL REFERENCES media_items(id) ON DELETE CASCADE,
    child_id BIGINT NOT NULL REFERENCES media_items(id) ON DELETE CASCADE,
    relation TEXT NOT NULL,        -- Type of relationship (e.g., "series-season", "season-episode")
    sort_index NUMERIC,             -- Ordering within relationship (season_number, episode_number, track_number)
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,  -- Additional relation metadata
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Ensure each parent-child-relation combination is unique
CREATE UNIQUE INDEX media_relations_unique_idx ON media_relations(parent_id, child_id, relation);

-- Index for efficiently finding all children of a parent
CREATE INDEX media_relations_parent_idx ON media_relations(parent_id);

-- Index for efficiently finding all parents of a child
CREATE INDEX media_relations_child_idx ON media_relations(child_id);

-- Index for finding relations by type
CREATE INDEX media_relations_relation_idx ON media_relations(relation);

-- Auto-update timestamp trigger
CREATE TRIGGER update_media_relations_updated_at
    BEFORE UPDATE ON media_relations
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- =============================================================================
-- media_files - Track physical files associated with media items
-- =============================================================================
-- This table maps filesystem paths to media items. Each file can be associated
-- with a media item (movies, episodes, tracks, books, etc.). The scanner uses
-- this table to:
--   1. Track which files have already been imported
--   2. Detect changes to files (via hash)
--   3. Clean up orphaned media items when files are deleted
--   4. Avoid duplicate imports
--
-- The path field is unique to prevent duplicate file entries.
-- The hash field can be used to detect file modifications.
-- =============================================================================

CREATE TABLE media_files (
    id BIGSERIAL PRIMARY KEY,
    media_item_id BIGINT REFERENCES media_items(id) ON DELETE CASCADE,
    path TEXT NOT NULL UNIQUE,      -- Absolute file path on disk
    size BIGINT,                     -- File size in bytes
    hash TEXT,                       -- SHA-256 or similar hash for detecting changes
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Index for efficiently finding all files for a media item
CREATE INDEX media_files_item_idx ON media_files(media_item_id);

-- Index for finding files by path prefix (useful for directory scans)
CREATE INDEX media_files_path_idx ON media_files(path text_pattern_ops);

-- Auto-update timestamp trigger
CREATE TRIGGER update_media_files_updated_at
    BEFORE UPDATE ON media_files
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- =============================================================================
-- scanner_state - Track library scanner status and progress
-- =============================================================================
-- This is a SINGLE-ROW table (enforced by CHECK constraint) that tracks the
-- current state of the library scanner. It stores:
--   - Whether a scan is currently running
--   - Start/finish timestamps
--   - Progress counters (files scanned, items created/updated)
--   - Error log for debugging
--   - General log for tracking scanner activity
--
-- The scanner MUST check this table before starting a new scan to prevent
-- concurrent scans from interfering with each other.
-- =============================================================================

CREATE TABLE scanner_state (
    id INT PRIMARY KEY DEFAULT 1,                     -- Always 1 (single row table)
    running BOOLEAN NOT NULL DEFAULT FALSE,           -- Is a scan currently in progress?
    started_at TIMESTAMPTZ,                           -- When current/last scan started
    finished_at TIMESTAMPTZ,                          -- When last scan finished
    files_scanned INT NOT NULL DEFAULT 0,             -- Number of files processed
    items_created INT NOT NULL DEFAULT 0,             -- Number of new media items created
    items_updated INT NOT NULL DEFAULT 0,             -- Number of existing items updated
    errors JSONB NOT NULL DEFAULT '[]'::jsonb,        -- Array of error messages
    log JSONB NOT NULL DEFAULT '[]'::jsonb,           -- Array of log entries
    CONSTRAINT single_row_check CHECK (id = 1)        -- Enforce single row
);

-- Create GIN index for efficient JSON queries on logs
CREATE INDEX scanner_state_errors_idx ON scanner_state USING GIN(errors);
CREATE INDEX scanner_state_log_idx ON scanner_state USING GIN(log);

-- Insert the single row with default values
INSERT INTO scanner_state (id, running) VALUES (1, FALSE)
ON CONFLICT (id) DO NOTHING;

-- =============================================================================
-- Helper Function: Get or Create Scanner State
-- =============================================================================
-- This function ensures the scanner_state row always exists and returns it.
-- Useful for applications that want to ensure the row is present.
-- =============================================================================

CREATE OR REPLACE FUNCTION get_scanner_state()
RETURNS scanner_state AS $$
DECLARE
    state scanner_state;
BEGIN
    SELECT * INTO state FROM scanner_state WHERE id = 1;

    IF NOT FOUND THEN
        INSERT INTO scanner_state (id, running)
        VALUES (1, FALSE)
        RETURNING * INTO state;
    END IF;

    RETURN state;
END;
$$ LANGUAGE plpgsql;
