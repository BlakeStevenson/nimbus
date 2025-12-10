-- Migration: Add media_item relation to downloads table
-- This allows associating downloads with specific media items in the library

ALTER TABLE downloads
ADD COLUMN media_item_id BIGINT REFERENCES media_items(id) ON DELETE SET NULL;

-- Index for querying downloads by media item
CREATE INDEX IF NOT EXISTS idx_downloads_media_item_id ON downloads(media_item_id);
