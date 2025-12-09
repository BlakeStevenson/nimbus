-- Add metadata column to config table
ALTER TABLE config ADD COLUMN metadata JSONB DEFAULT '{}'::jsonb;

-- Update existing config entries with metadata
UPDATE config SET metadata = jsonb_build_object(
    'title', 'Library Root Path',
    'description', 'The root directory for your media library',
    'type', 'text'
) WHERE key = 'library.root_path';

UPDATE config SET metadata = jsonb_build_object(
    'title', 'Download Temporary Path',
    'description', 'Temporary directory for downloads and processing',
    'type', 'text'
) WHERE key = 'download.tmp_path';

UPDATE config SET metadata = jsonb_build_object(
    'title', 'Default Language',
    'description', 'Default language for metadata indexing',
    'type', 'text'
) WHERE key = 'indexer.default_language';

UPDATE config SET metadata = jsonb_build_object(
    'title', 'Server Version',
    'description', 'Current server version',
    'type', 'text'
) WHERE key = 'server.version';
