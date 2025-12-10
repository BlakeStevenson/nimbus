CREATE TABLE media_items (
    id              BIGSERIAL PRIMARY KEY,
    kind            TEXT NOT NULL,
    title           TEXT NOT NULL,
    sort_title      TEXT NOT NULL,
    year            INTEGER,
    external_ids    JSONB DEFAULT '{}'::jsonb,
    metadata        JSONB DEFAULT '{}'::jsonb,
    parent_id       BIGINT REFERENCES media_items(id) ON DELETE CASCADE,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Indexes for performance
CREATE INDEX idx_media_items_kind ON media_items(kind);
CREATE INDEX idx_media_items_title ON media_items(title);
CREATE INDEX idx_media_items_sort_title ON media_items(sort_title);
CREATE INDEX idx_media_items_parent_id ON media_items(parent_id) WHERE parent_id IS NOT NULL;
CREATE INDEX idx_media_items_year ON media_items(year) WHERE year IS NOT NULL;
CREATE INDEX idx_media_items_external_ids ON media_items USING GIN(external_ids);
CREATE INDEX idx_media_items_metadata ON media_items USING GIN(metadata);

-- Unique constraint for upsert operations
CREATE UNIQUE INDEX media_items_natural_key_idx
ON media_items(kind, title, COALESCE(year, -1), COALESCE(parent_id, -1));

-- Global configuration table
CREATE TABLE config (
    key         TEXT PRIMARY KEY,
    value       JSONB NOT NULL,
    metadata    JSONB DEFAULT '{}'::jsonb,
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- =============================================================================
-- User Authentication Tables
-- =============================================================================

-- Users table
CREATE TABLE users (
    id BIGSERIAL PRIMARY KEY,
    username TEXT NOT NULL UNIQUE,
    email TEXT NOT NULL UNIQUE,
    is_active BOOLEAN NOT NULL DEFAULT true,
    is_admin BOOLEAN NOT NULL DEFAULT false,
    metadata JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Auth providers table to support multiple auth methods
CREATE TABLE auth_providers (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    provider_type TEXT NOT NULL, -- 'password', 'oauth', 'saml', etc.
    provider_id TEXT, -- External provider identifier (null for password auth)
    credentials JSONB NOT NULL DEFAULT '{}', -- Encrypted credentials/password hash
    metadata JSONB NOT NULL DEFAULT '{}', -- Provider-specific metadata
    is_primary BOOLEAN NOT NULL DEFAULT false, -- Primary authentication method
    last_used_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(user_id, provider_type, provider_id)
);

-- Refresh tokens table for JWT refresh tokens
CREATE TABLE refresh_tokens (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash TEXT NOT NULL UNIQUE,
    expires_at TIMESTAMPTZ NOT NULL,
    revoked BOOLEAN NOT NULL DEFAULT false,
    revoked_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_used_at TIMESTAMPTZ
);

-- Indexes for users
CREATE INDEX idx_users_username ON users(username);
CREATE INDEX idx_users_email ON users(email);
CREATE INDEX idx_users_active ON users(is_active) WHERE is_active = true;

-- Indexes for auth_providers
CREATE INDEX idx_auth_providers_user_id ON auth_providers(user_id);
CREATE INDEX idx_auth_providers_type ON auth_providers(provider_type);
CREATE INDEX idx_auth_providers_primary ON auth_providers(user_id, is_primary) WHERE is_primary = true;

-- Indexes for refresh_tokens
CREATE INDEX idx_refresh_tokens_user_id ON refresh_tokens(user_id);
CREATE INDEX idx_refresh_tokens_hash ON refresh_tokens(token_hash);
CREATE INDEX idx_refresh_tokens_expires ON refresh_tokens(expires_at) WHERE NOT revoked;

-- =============================================================================
-- Media Library Management Tables
-- =============================================================================

-- Media relations - Track hierarchical relationships between media items
CREATE TABLE media_relations (
    id BIGSERIAL PRIMARY KEY,
    parent_id BIGINT NOT NULL REFERENCES media_items(id) ON DELETE CASCADE,
    child_id BIGINT NOT NULL REFERENCES media_items(id) ON DELETE CASCADE,
    relation TEXT NOT NULL,        -- Type of relationship (e.g., "series-season", "season-episode")
    sort_index NUMERIC,             -- Ordering within relationship (season_number, episode_number, track_number)
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Ensure each parent-child-relation combination is unique
CREATE UNIQUE INDEX media_relations_unique_idx ON media_relations(parent_id, child_id, relation);
CREATE INDEX media_relations_parent_idx ON media_relations(parent_id);
CREATE INDEX media_relations_child_idx ON media_relations(child_id);
CREATE INDEX media_relations_relation_idx ON media_relations(relation);

-- Media files - Track physical files associated with media items
CREATE TABLE media_files (
    id BIGSERIAL PRIMARY KEY,
    media_item_id BIGINT REFERENCES media_items(id) ON DELETE CASCADE,
    path TEXT NOT NULL UNIQUE,
    size BIGINT,
    hash TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX media_files_item_idx ON media_files(media_item_id);
CREATE INDEX media_files_path_idx ON media_files(path text_pattern_ops);

-- Scanner state - Track library scanner status and progress
CREATE TABLE scanner_state (
    id INT PRIMARY KEY DEFAULT 1,
    running BOOLEAN NOT NULL DEFAULT FALSE,
    started_at TIMESTAMPTZ,
    finished_at TIMESTAMPTZ,
    files_scanned INT NOT NULL DEFAULT 0,
    items_created INT NOT NULL DEFAULT 0,
    items_updated INT NOT NULL DEFAULT 0,
    errors JSONB NOT NULL DEFAULT '[]'::jsonb,
    log JSONB NOT NULL DEFAULT '[]'::jsonb,
    CONSTRAINT single_row_check CHECK (id = 1)
);

CREATE INDEX scanner_state_errors_idx ON scanner_state USING GIN(errors);
CREATE INDEX scanner_state_log_idx ON scanner_state USING GIN(log);

-- Insert the single row with default values
INSERT INTO scanner_state (id, running) VALUES (1, FALSE)
ON CONFLICT (id) DO NOTHING;

-- =============================================================================
-- Plugin System Tables
-- =============================================================================

-- Plugins - Track installed plugins and their metadata
CREATE TABLE plugins (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    version TEXT NOT NULL DEFAULT '0.1.0',
    enabled BOOLEAN NOT NULL DEFAULT TRUE,
    capabilities JSONB NOT NULL DEFAULT '[]'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX plugins_enabled_idx ON plugins(enabled) WHERE enabled = TRUE;
CREATE INDEX plugins_capabilities_idx ON plugins USING GIN(capabilities);

-- =============================================================================
-- Download Management Tables
-- =============================================================================

-- Downloads - Persistent download tracking
CREATE TABLE downloads (
    id TEXT PRIMARY KEY DEFAULT ('dl_' || md5(random()::text)),
    plugin_id TEXT NOT NULL REFERENCES plugins(id) ON DELETE CASCADE,
    media_item_id BIGINT REFERENCES media_items(id) ON DELETE SET NULL,

    -- Download metadata
    name TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'queued',
    progress INTEGER NOT NULL DEFAULT 0 CHECK (progress >= 0 AND progress <= 100),

    -- Size tracking
    total_bytes BIGINT,
    downloaded_bytes BIGINT DEFAULT 0,

    -- Download source
    url TEXT,
    file_content BYTEA,
    file_name TEXT,

    -- Destination
    destination_path TEXT,

    -- Error tracking
    error_message TEXT,
    retry_count INTEGER DEFAULT 0,

    -- Queue management
    queue_position INTEGER,
    priority INTEGER DEFAULT 0,

    -- Timestamps
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    started_at TIMESTAMP,
    completed_at TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,

    -- Plugin-specific metadata
    metadata JSONB DEFAULT '{}',

    -- User tracking
    created_by_user_id INTEGER REFERENCES users(id) ON DELETE SET NULL
);

-- Indexes for downloads
CREATE INDEX idx_downloads_status ON downloads(status);
CREATE INDEX idx_downloads_plugin_id ON downloads(plugin_id);
CREATE INDEX idx_downloads_media_item_id ON downloads(media_item_id);
CREATE INDEX idx_downloads_created_by ON downloads(created_by_user_id);
CREATE INDEX idx_downloads_queue_position ON downloads(queue_position) WHERE status IN ('queued', 'downloading');
CREATE INDEX idx_downloads_created_at ON downloads(created_at DESC);

-- Download logs - Detailed history
CREATE TABLE download_logs (
    id SERIAL PRIMARY KEY,
    download_id TEXT NOT NULL REFERENCES downloads(id) ON DELETE CASCADE,
    level TEXT NOT NULL DEFAULT 'info',
    message TEXT NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_download_logs_download_id ON download_logs(download_id);
CREATE INDEX idx_download_logs_created_at ON download_logs(created_at DESC);

-- =============================================================================
-- Triggers
-- =============================================================================

-- Function to auto-update updated_at column
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

-- Apply triggers
CREATE TRIGGER update_media_items_updated_at
    BEFORE UPDATE ON media_items
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_config_updated_at
    BEFORE UPDATE ON config
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_users_updated_at
    BEFORE UPDATE ON users
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_auth_providers_updated_at
    BEFORE UPDATE ON auth_providers
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_media_relations_updated_at
    BEFORE UPDATE ON media_relations
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_media_files_updated_at
    BEFORE UPDATE ON media_files
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_plugins_updated_at
    BEFORE UPDATE ON plugins
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- =============================================================================
-- Helper Functions
-- =============================================================================

-- Get or create scanner state
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

-- =============================================================================
-- Default Configuration Values
-- =============================================================================

INSERT INTO config (key, value, metadata) VALUES
    -- General
    ('server.version', '"0.1.0"', jsonb_build_object(
        'title', 'Server Version',
        'description', 'Current server version (read-only)',
        'type', 'text'
    )),

    -- Library paths
    ('library.root_path', '"/media"', jsonb_build_object(
        'title', 'Library Root Path',
        'description', 'Legacy root directory for all media (deprecated - use media-specific paths instead)',
        'type', 'text',
        'deprecated', true
    )),
    ('library.movie_path', '"/media/movies"', jsonb_build_object(
        'title', 'Movie Library Path',
        'description', 'Root directory for movie files',
        'type', 'text'
    )),
    ('library.tv_path', '"/media/tv"', jsonb_build_object(
        'title', 'TV Shows Library Path',
        'description', 'Root directory for TV show files',
        'type', 'text'
    )),
    ('library.music_path', '"/media/music"', jsonb_build_object(
        'title', 'Music Library Path',
        'description', 'Root directory for music files',
        'type', 'text'
    )),
    ('library.book_path', '"/media/books"', jsonb_build_object(
        'title', 'Books Library Path',
        'description', 'Root directory for book files',
        'type', 'text'
    )),
    ('library.enabled_media_types', '["tv", "movie", "music", "book"]', jsonb_build_object(
        'title', 'Enabled Media Types',
        'description', 'Media types that are enabled for scanning and indexing',
        'type', 'multi',
        'values', jsonb_build_array('tv', 'movie', 'music', 'book')
    )),

    -- Downloads
    ('download.tmp_path', '"/tmp/downloads"', jsonb_build_object(
        'title', 'Download Temporary Path',
        'description', 'Temporary directory for downloads and processing',
        'type', 'text'
    )),

    -- Plugins
    ('plugins.enabled', 'false', jsonb_build_object(
        'title', 'Enable Plugins',
        'description', 'Enable or disable the plugin system',
        'type', 'boolean'
    )),
    ('plugins.directory', '"/var/lib/nimbus/plugins"', jsonb_build_object(
        'title', 'Plugins Directory',
        'description', 'Directory where plugins are stored',
        'type', 'text'
    )),

    -- Indexer
    ('indexer.default_language', '"en"', jsonb_build_object(
        'title', 'Default Language',
        'description', 'Default language code for metadata indexing (e.g., en, es, fr)',
        'type', 'text'
    )),

    -- Download naming - Movies
    ('downloads.movie_naming_format', '"{Movie Title} ({Release Year})"', jsonb_build_object(
        'title', 'Movie File Naming Format',
        'description', 'Template for naming movie files. Available tokens: {Movie Title}, {Release Year}, {Quality}, {Edition}, {IMDb ID}',
        'type', 'text',
        'category', 'downloads',
        'section', 'Movie Naming'
    )),
    ('downloads.movie_folder_format', '"{Movie Title} ({Release Year})"', jsonb_build_object(
        'title', 'Movie Folder Format',
        'description', 'Template for movie folder names. Available tokens: {Movie Title}, {Release Year}, {IMDb ID}',
        'type', 'text',
        'category', 'downloads',
        'section', 'Movie Naming'
    )),

    -- Download naming - TV Shows
    ('downloads.tv_naming_format', '"{Series Title} - S{season:00}E{episode:00} - {Episode Title}"', jsonb_build_object(
        'title', 'TV Episode Naming Format',
        'description', 'Template for TV episode files. Available tokens: {Series Title}, {Season}, {Episode}, {Episode Title}, {Quality}, {Release Date}',
        'type', 'text',
        'category', 'downloads',
        'section', 'TV Naming'
    )),
    ('downloads.tv_folder_format', '"{Series Title}"', jsonb_build_object(
        'title', 'TV Series Folder Format',
        'description', 'Template for TV series folder names. Available tokens: {Series Title}, {Year}, {TVDb ID}',
        'type', 'text',
        'category', 'downloads',
        'section', 'TV Naming'
    )),
    ('downloads.tv_season_folder_format', '"Season {season:00}"', jsonb_build_object(
        'title', 'TV Season Folder Format',
        'description', 'Template for season folder names within series folder. Available tokens: {Season}',
        'type', 'text',
        'category', 'downloads',
        'section', 'TV Naming'
    )),
    ('downloads.tv_use_season_folders', 'true', jsonb_build_object(
        'title', 'Use Season Folders',
        'description', 'Create separate folders for each season within the series folder',
        'type', 'boolean',
        'category', 'downloads',
        'section', 'TV Naming'
    )),

    -- Folder structure
    ('downloads.create_series_folder', 'true', jsonb_build_object(
        'title', 'Create Series Folder',
        'description', 'Automatically create a folder for each TV series',
        'type', 'boolean',
        'category', 'downloads',
        'section', 'Folder Structure'
    )),
    ('downloads.create_movie_folder', 'true', jsonb_build_object(
        'title', 'Create Movie Folder',
        'description', 'Automatically create a folder for each movie',
        'type', 'boolean',
        'category', 'downloads',
        'section', 'Folder Structure'
    )),

    -- File management
    ('downloads.rename_episodes', 'true', jsonb_build_object(
        'title', 'Rename Episodes',
        'description', 'Automatically rename downloaded episodes to match naming format',
        'type', 'boolean',
        'category', 'downloads',
        'section', 'File Management'
    )),
    ('downloads.rename_movies', 'true', jsonb_build_object(
        'title', 'Rename Movies',
        'description', 'Automatically rename downloaded movies to match naming format',
        'type', 'boolean',
        'category', 'downloads',
        'section', 'File Management'
    )),
    ('downloads.replace_illegal_characters', 'true', jsonb_build_object(
        'title', 'Replace Illegal Characters',
        'description', 'Replace characters that are illegal in filenames with legal alternatives',
        'type', 'boolean',
        'category', 'downloads',
        'section', 'File Management'
    )),
    ('downloads.colon_replacement', '"dash"', jsonb_build_object(
        'title', 'Colon Replacement',
        'description', 'How to handle colons in filenames',
        'type', 'select',
        'values', jsonb_build_array('delete', 'dash', 'space', 'spacedash'),
        'category', 'downloads',
        'section', 'File Management'
    )),

    -- Quality & upgrades
    ('downloads.quality_profiles', '["Any", "SD", "720p", "1080p", "2160p"]', jsonb_build_object(
        'title', 'Quality Profiles',
        'description', 'Available quality profiles for downloads (ordered by preference)',
        'type', 'array',
        'category', 'downloads',
        'section', 'Quality'
    )),
    ('downloads.preferred_quality', '"1080p"', jsonb_build_object(
        'title', 'Preferred Quality',
        'description', 'Preferred quality for downloads',
        'type', 'select',
        'values', jsonb_build_array('Any', 'SD', '720p', '1080p', '2160p'),
        'category', 'downloads',
        'section', 'Quality'
    )),
    ('downloads.enable_quality_upgrades', 'true', jsonb_build_object(
        'title', 'Enable Quality Upgrades',
        'description', 'Automatically upgrade to higher quality releases when available',
        'type', 'boolean',
        'category', 'downloads',
        'section', 'Quality'
    )),
    ('downloads.upgrade_until_quality', '"1080p"', jsonb_build_object(
        'title', 'Upgrade Until Quality',
        'description', 'Stop upgrading once this quality is reached',
        'type', 'select',
        'values', jsonb_build_array('SD', '720p', '1080p', '2160p'),
        'category', 'downloads',
        'section', 'Quality'
    )),

    -- Download client
    ('downloads.completed_download_handling', 'true', jsonb_build_object(
        'title', 'Enable Completed Download Handling',
        'description', 'Automatically import completed downloads and move to library',
        'type', 'boolean',
        'category', 'downloads',
        'section', 'Download Client'
    )),
    ('downloads.remove_completed_downloads', 'false', jsonb_build_object(
        'title', 'Remove Completed Downloads',
        'description', 'Remove downloads from download client after import',
        'type', 'boolean',
        'category', 'downloads',
        'section', 'Download Client'
    )),
    ('downloads.check_for_finished_download_interval', '1', jsonb_build_object(
        'title', 'Check For Finished Downloads Interval',
        'description', 'Interval in minutes to check for completed downloads',
        'type', 'number',
        'category', 'downloads',
        'section', 'Download Client'
    )),

    -- Importing
    ('downloads.skip_free_space_check', 'false', jsonb_build_object(
        'title', 'Skip Free Space Check',
        'description', 'Skip checking free space before importing',
        'type', 'boolean',
        'category', 'downloads',
        'section', 'Importing'
    )),
    ('downloads.minimum_free_space', '100', jsonb_build_object(
        'title', 'Minimum Free Space (MB)',
        'description', 'Minimum free space required before importing (in megabytes)',
        'type', 'number',
        'category', 'downloads',
        'section', 'Importing'
    )),
    ('downloads.use_hardlinks', 'true', jsonb_build_object(
        'title', 'Use Hardlinks Instead of Copy',
        'description', 'Use hardlinks when possible to save disk space (requires same filesystem)',
        'type', 'boolean',
        'category', 'downloads',
        'section', 'Importing'
    )),
    ('downloads.import_extra_files', 'true', jsonb_build_object(
        'title', 'Import Extra Files',
        'description', 'Import extra files found with media (subtitles, nfo, etc)',
        'type', 'boolean',
        'category', 'downloads',
        'section', 'Importing'
    )),
    ('downloads.extra_file_extensions', '"srt,nfo,txt"', jsonb_build_object(
        'title', 'Extra File Extensions',
        'description', 'Comma-separated list of extra file extensions to import',
        'type', 'text',
        'category', 'downloads',
        'section', 'Importing'
    )),

    -- Advanced
    ('downloads.set_permissions', 'false', jsonb_build_object(
        'title', 'Set Permissions',
        'description', 'Set permissions on imported files and folders',
        'type', 'boolean',
        'category', 'downloads',
        'section', 'Advanced'
    )),
    ('downloads.chmod_folder', '"755"', jsonb_build_object(
        'title', 'Folder Permissions',
        'description', 'Permissions for created folders (octal format)',
        'type', 'text',
        'category', 'downloads',
        'section', 'Advanced'
    )),
    ('downloads.chmod_file', '"644"', jsonb_build_object(
        'title', 'File Permissions',
        'description', 'Permissions for imported files (octal format)',
        'type', 'text',
        'category', 'downloads',
        'section', 'Advanced'
    )),
    ('downloads.recycle_bin', '""', jsonb_build_object(
        'title', 'Recycle Bin Path',
        'description', 'Move deleted files to this location instead of permanently deleting (empty = disabled)',
        'type', 'text',
        'category', 'downloads',
        'section', 'Advanced'
    )),
    ('downloads.recycle_bin_cleanup_days', '7', jsonb_build_object(
        'title', 'Recycle Bin Cleanup Days',
        'description', 'Days to keep files in recycle bin before permanent deletion (0 = keep forever)',
        'type', 'number',
        'category', 'downloads',
        'section', 'Advanced'
    ))
ON CONFLICT (key) DO NOTHING;
