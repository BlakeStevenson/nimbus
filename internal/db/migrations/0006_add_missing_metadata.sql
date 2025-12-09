-- Add metadata for all existing config entries

UPDATE config SET metadata = jsonb_build_object(
    'title', 'Download Temporary Path',
    'description', 'Temporary directory for downloads and processing',
    'type', 'text'
) WHERE key = 'download.tmp_path' AND metadata = '{}'::jsonb;

UPDATE config SET metadata = jsonb_build_object(
    'title', 'Default Language',
    'description', 'Default language code for metadata indexing (e.g., en, es, fr)',
    'type', 'text'
) WHERE key = 'indexer.default_language' AND metadata = '{}'::jsonb;

UPDATE config SET metadata = jsonb_build_object(
    'title', 'Enable Plugins',
    'description', 'Enable or disable the plugin system',
    'type', 'boolean'
) WHERE key = 'plugins.enabled' AND metadata = '{}'::jsonb;

UPDATE config SET metadata = jsonb_build_object(
    'title', 'Plugins Directory',
    'description', 'Directory where plugins are stored',
    'type', 'text'
) WHERE key = 'plugins.directory' AND metadata = '{}'::jsonb;

UPDATE config SET metadata = jsonb_build_object(
    'title', 'Library Root Path',
    'description', 'The root directory for your media library',
    'type', 'text'
) WHERE key = 'library.root_path' AND metadata = '{}'::jsonb;

UPDATE config SET metadata = jsonb_build_object(
    'title', 'TMDB API Key',
    'description', 'API key for The Movie Database (TMDB) integration',
    'type', 'text'
) WHERE key = 'plugins.tmdb.api_key' AND metadata = '{}'::jsonb;

UPDATE config SET metadata = jsonb_build_object(
    'title', 'Server Version',
    'description', 'Current server version (read-only)',
    'type', 'text'
) WHERE key = 'server.version' AND metadata = '{}'::jsonb;
