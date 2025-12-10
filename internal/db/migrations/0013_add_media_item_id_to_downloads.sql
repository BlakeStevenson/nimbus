-- Migration: Add media_item_id column to downloads table
-- This links downloads to specific media items for automatic import

ALTER TABLE downloads ADD COLUMN IF NOT EXISTS media_item_id INTEGER REFERENCES media_items(id) ON DELETE SET NULL;

CREATE INDEX IF NOT EXISTS idx_downloads_media_item_id ON downloads(media_item_id);
