-- Add external_ids column to media_items table
-- This stores external IDs from TMDB (tvdb_id, imdb_id, etc.) as JSONB

ALTER TABLE media_items
ADD COLUMN external_ids JSONB DEFAULT '{}'::jsonb;

-- Create index on external_ids for faster lookups
CREATE INDEX media_items_external_ids_idx ON media_items USING GIN(external_ids);

-- Add comments for documentation
COMMENT ON COLUMN media_items.external_ids IS 'External IDs from TMDB (tvdb_id, imdb_id, tvrage_id, etc.) stored as JSONB';
