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

CREATE TRIGGER update_quality_definitions_updated_at
    BEFORE UPDATE ON quality_definitions
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_quality_profiles_updated_at
    BEFORE UPDATE ON quality_profiles
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_media_quality_updated_at
    BEFORE UPDATE ON media_quality
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_monitoring_rules_updated_at
    BEFORE UPDATE ON monitoring_rules
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_episode_monitoring_updated_at
    BEFORE UPDATE ON episode_monitoring
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_rss_sync_state_updated_at
    BEFORE UPDATE ON rss_sync_state
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_calendar_events_updated_at
    BEFORE UPDATE ON calendar_events
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_scheduler_jobs_updated_at
    BEFORE UPDATE ON scheduler_jobs
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- =============================================================================
-- Quality Profile System Tables
-- =============================================================================

-- Quality definitions - Define available quality levels with detailed specifications
CREATE TABLE quality_definitions (
    id SERIAL PRIMARY KEY,
    name TEXT NOT NULL UNIQUE,
    title TEXT NOT NULL,
    resolution INTEGER,  -- Vertical resolution (480, 720, 1080, 2160, etc.)
    source TEXT,         -- Source type (WEBDL, BluRay, HDTV, DVD, etc.)
    modifier TEXT,       -- Additional modifier (Remux, Proper, Repack, etc.)
    min_size BIGINT,     -- Minimum file size in bytes (for preferred size ranges)
    max_size BIGINT,     -- Maximum file size in bytes (for preferred size ranges)
    weight INTEGER NOT NULL DEFAULT 0, -- Weight for sorting (higher = better quality)
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Indexes for quality definitions
CREATE INDEX idx_quality_definitions_name ON quality_definitions(name);
CREATE INDEX idx_quality_definitions_weight ON quality_definitions(weight);

-- Quality profiles - User-defined profiles with ordered quality preferences
CREATE TABLE quality_profiles (
    id SERIAL PRIMARY KEY,
    name TEXT NOT NULL UNIQUE,
    description TEXT,
    cutoff_quality_id INTEGER REFERENCES quality_definitions(id) ON DELETE RESTRICT, -- Stop upgrading at this quality
    upgrade_allowed BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Indexes for quality profiles
CREATE INDEX idx_quality_profiles_name ON quality_profiles(name);

-- Quality profile items - Ordered list of qualities in a profile
CREATE TABLE quality_profile_items (
    id SERIAL PRIMARY KEY,
    profile_id INTEGER NOT NULL REFERENCES quality_profiles(id) ON DELETE CASCADE,
    quality_id INTEGER NOT NULL REFERENCES quality_definitions(id) ON DELETE CASCADE,
    allowed BOOLEAN NOT NULL DEFAULT true,
    sort_order INTEGER NOT NULL, -- Order in the profile (lower = higher priority)
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(profile_id, quality_id)
);

-- Indexes for quality profile items
CREATE INDEX idx_quality_profile_items_profile ON quality_profile_items(profile_id, sort_order);
CREATE INDEX idx_quality_profile_items_quality ON quality_profile_items(quality_id);

-- Media quality - Track current quality of media items
CREATE TABLE media_quality (
    id BIGSERIAL PRIMARY KEY,
    media_item_id BIGINT NOT NULL REFERENCES media_items(id) ON DELETE CASCADE,
    media_file_id BIGINT REFERENCES media_files(id) ON DELETE CASCADE,
    quality_id INTEGER REFERENCES quality_definitions(id) ON DELETE SET NULL,
    profile_id INTEGER REFERENCES quality_profiles(id) ON DELETE SET NULL,

    -- Detected quality information
    detected_quality TEXT, -- Raw quality string detected from filename
    resolution INTEGER,
    source TEXT,
    codec_video TEXT,      -- Video codec (H.264, H.265, AV1, etc.)
    codec_audio TEXT,      -- Audio codec (AAC, DTS, Atmos, etc.)

    -- Quality metadata
    is_proper BOOLEAN DEFAULT false,
    is_repack BOOLEAN DEFAULT false,
    is_remux BOOLEAN DEFAULT false,
    revision_version INTEGER DEFAULT 1,

    -- Upgrade tracking
    upgrade_allowed BOOLEAN DEFAULT true,
    cutoff_met BOOLEAN DEFAULT false, -- Has the cutoff quality been reached?

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    UNIQUE(media_item_id, media_file_id)
);

-- Indexes for media quality
CREATE INDEX idx_media_quality_media_item ON media_quality(media_item_id);
CREATE INDEX idx_media_quality_media_file ON media_quality(media_file_id);
CREATE INDEX idx_media_quality_quality ON media_quality(quality_id);
CREATE INDEX idx_media_quality_profile ON media_quality(profile_id);
CREATE INDEX idx_media_quality_upgrade ON media_quality(upgrade_allowed, cutoff_met)
    WHERE upgrade_allowed = true AND cutoff_met = false;

-- Quality upgrade history - Track quality upgrades over time
CREATE TABLE quality_upgrade_history (
    id BIGSERIAL PRIMARY KEY,
    media_item_id BIGINT NOT NULL REFERENCES media_items(id) ON DELETE CASCADE,
    old_quality_id INTEGER REFERENCES quality_definitions(id) ON DELETE SET NULL,
    new_quality_id INTEGER REFERENCES quality_definitions(id) ON DELETE SET NULL,
    old_file_id BIGINT,     -- Old file may have been deleted
    new_file_id BIGINT REFERENCES media_files(id) ON DELETE SET NULL,
    download_id TEXT REFERENCES downloads(id) ON DELETE SET NULL,

    -- Details
    reason TEXT,            -- upgrade, manual_replacement, etc.
    old_file_size BIGINT,
    new_file_size BIGINT,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by_user_id BIGINT REFERENCES users(id) ON DELETE SET NULL
);

-- Indexes for quality upgrade history
CREATE INDEX idx_quality_upgrade_history_media ON quality_upgrade_history(media_item_id, created_at DESC);
CREATE INDEX idx_quality_upgrade_history_download ON quality_upgrade_history(download_id);
CREATE INDEX idx_quality_upgrade_history_created_at ON quality_upgrade_history(created_at DESC);

-- =============================================================================
-- Monitoring & Automation Tables
-- =============================================================================

-- Monitoring rules - Define what media items to monitor and automatically search for
CREATE TABLE monitoring_rules (
    id BIGSERIAL PRIMARY KEY,
    media_item_id BIGINT NOT NULL REFERENCES media_items(id) ON DELETE CASCADE,

    -- Monitoring settings
    enabled BOOLEAN NOT NULL DEFAULT true,
    quality_profile_id INTEGER REFERENCES quality_profiles(id) ON DELETE SET NULL,

    -- Monitoring mode for series/seasons
    monitor_mode TEXT NOT NULL DEFAULT 'all', -- all, future, missing, existing, first_season, latest_season, pilot, none

    -- Search settings
    search_on_add BOOLEAN NOT NULL DEFAULT true,          -- Search immediately when monitoring is enabled
    automatic_search BOOLEAN NOT NULL DEFAULT true,       -- Search automatically for new releases
    backlog_search BOOLEAN NOT NULL DEFAULT true,         -- Search for missing episodes in backlog

    -- Release preferences
    prefer_season_packs BOOLEAN NOT NULL DEFAULT false,   -- Prefer season packs over individual episodes
    minimum_seeders INTEGER DEFAULT 1,                    -- Minimum seeders for torrents
    tags TEXT[] DEFAULT '{}',                             -- Tags for organization/filtering

    -- Schedule
    search_interval_minutes INTEGER DEFAULT 60,           -- How often to search (RSS sync interval)
    last_search_at TIMESTAMPTZ,                          -- Last automatic search time
    next_search_at TIMESTAMPTZ,                          -- Next scheduled search time

    -- Statistics
    search_count INTEGER DEFAULT 0,                       -- Total number of searches performed
    items_found_count INTEGER DEFAULT 0,                  -- Total number of items found
    items_grabbed_count INTEGER DEFAULT 0,                -- Total number of items downloaded

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by_user_id BIGINT REFERENCES users(id) ON DELETE SET NULL,

    UNIQUE(media_item_id)
);

-- Indexes for monitoring rules
CREATE INDEX idx_monitoring_rules_enabled ON monitoring_rules(enabled) WHERE enabled = true;
CREATE INDEX idx_monitoring_rules_next_search ON monitoring_rules(next_search_at) WHERE enabled = true AND next_search_at IS NOT NULL;
CREATE INDEX idx_monitoring_rules_media_item ON monitoring_rules(media_item_id);
CREATE INDEX idx_monitoring_rules_quality_profile ON monitoring_rules(quality_profile_id);
CREATE INDEX idx_monitoring_rules_tags ON monitoring_rules USING GIN(tags);

-- Episode monitoring - Fine-grained episode-level monitoring for TV series
CREATE TABLE episode_monitoring (
    id BIGSERIAL PRIMARY KEY,
    media_item_id BIGINT NOT NULL REFERENCES media_items(id) ON DELETE CASCADE,
    monitored BOOLEAN NOT NULL DEFAULT true,

    -- Episode status
    has_file BOOLEAN NOT NULL DEFAULT false,              -- Does this episode have a file?
    file_id BIGINT REFERENCES media_files(id) ON DELETE SET NULL,

    -- Air date tracking
    air_date DATE,
    air_date_utc TIMESTAMPTZ,

    -- Statistics
    search_count INTEGER DEFAULT 0,
    last_search_at TIMESTAMPTZ,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    UNIQUE(media_item_id)
);

-- Indexes for episode monitoring
CREATE INDEX idx_episode_monitoring_media_item ON episode_monitoring(media_item_id);
CREATE INDEX idx_episode_monitoring_monitored ON episode_monitoring(monitored, has_file) WHERE monitored = true;
CREATE INDEX idx_episode_monitoring_missing ON episode_monitoring(monitored, has_file, air_date) WHERE monitored = true AND has_file = false;
CREATE INDEX idx_episode_monitoring_air_date ON episode_monitoring(air_date) WHERE air_date IS NOT NULL;

-- Search history - Track all automatic and manual searches
CREATE TABLE search_history (
    id BIGSERIAL PRIMARY KEY,
    monitoring_rule_id BIGINT REFERENCES monitoring_rules(id) ON DELETE SET NULL,
    media_item_id BIGINT REFERENCES media_items(id) ON DELETE CASCADE,

    -- Search details
    search_type TEXT NOT NULL,                            -- automatic, manual, rss, backlog
    trigger_source TEXT,                                  -- user, scheduler, rss_sync, missing_check
    query TEXT,                                           -- Search query used

    -- Results
    results_found INTEGER DEFAULT 0,
    results_approved INTEGER DEFAULT 0,                   -- Results that met quality criteria
    results_rejected INTEGER DEFAULT 0,                   -- Results that didn't meet criteria
    download_grabbed BOOLEAN DEFAULT false,               -- Was a download initiated?
    download_id TEXT REFERENCES downloads(id) ON DELETE SET NULL,

    -- Timing
    search_duration_ms INTEGER,                           -- How long the search took

    -- Status and errors
    status TEXT NOT NULL DEFAULT 'pending',               -- pending, completed, failed
    error_message TEXT,

    metadata JSONB DEFAULT '{}'::jsonb,                   -- Additional search metadata

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by_user_id BIGINT REFERENCES users(id) ON DELETE SET NULL
);

-- Indexes for search history
CREATE INDEX idx_search_history_monitoring_rule ON search_history(monitoring_rule_id, created_at DESC);
CREATE INDEX idx_search_history_media_item ON search_history(media_item_id, created_at DESC);
CREATE INDEX idx_search_history_created_at ON search_history(created_at DESC);
CREATE INDEX idx_search_history_download ON search_history(download_id);
CREATE INDEX idx_search_history_status ON search_history(status);

-- Blocklist - Track rejected/blocked releases
CREATE TABLE blocklist (
    id BIGSERIAL PRIMARY KEY,
    media_item_id BIGINT REFERENCES media_items(id) ON DELETE CASCADE,

    -- Release identification
    release_hash TEXT NOT NULL,                           -- Hash/GUID of the release
    release_title TEXT NOT NULL,                          -- Full release title
    indexer_id TEXT,                                      -- Which indexer it came from

    -- Block reason
    reason TEXT NOT NULL,                                 -- quality, fake, corrupted, failed_download, manual, etc.
    message TEXT,                                         -- Additional details

    -- Block type
    permanent BOOLEAN NOT NULL DEFAULT true,              -- Permanent or temporary block
    expires_at TIMESTAMPTZ,                              -- For temporary blocks

    -- Context
    download_id TEXT REFERENCES downloads(id) ON DELETE SET NULL,
    search_history_id BIGINT REFERENCES search_history(id) ON DELETE SET NULL,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by_user_id BIGINT REFERENCES users(id) ON DELETE SET NULL,

    UNIQUE(release_hash, media_item_id)
);

-- Indexes for blocklist
CREATE INDEX idx_blocklist_media_item ON blocklist(media_item_id);
CREATE INDEX idx_blocklist_release_hash ON blocklist(release_hash);
CREATE INDEX idx_blocklist_expires ON blocklist(expires_at) WHERE expires_at IS NOT NULL AND permanent = false;
CREATE INDEX idx_blocklist_created_at ON blocklist(created_at DESC);

-- RSS sync state - Track RSS feed synchronization for automatic detection
CREATE TABLE rss_sync_state (
    id BIGSERIAL PRIMARY KEY,
    indexer_id TEXT NOT NULL,                             -- Plugin ID of the indexer

    -- Sync tracking
    last_sync_at TIMESTAMPTZ,
    next_sync_at TIMESTAMPTZ,
    sync_interval_minutes INTEGER DEFAULT 15,             -- How often to sync RSS feeds

    -- Statistics
    total_syncs INTEGER DEFAULT 0,
    total_items_found INTEGER DEFAULT 0,
    total_items_grabbed INTEGER DEFAULT 0,
    consecutive_failures INTEGER DEFAULT 0,
    last_error TEXT,

    -- State
    enabled BOOLEAN NOT NULL DEFAULT true,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    UNIQUE(indexer_id)
);

-- Indexes for RSS sync state
CREATE INDEX idx_rss_sync_state_next_sync ON rss_sync_state(next_sync_at) WHERE enabled = true;
CREATE INDEX idx_rss_sync_state_enabled ON rss_sync_state(enabled) WHERE enabled = true;

-- Calendar events - Upcoming and recent releases for calendar view
CREATE TABLE calendar_events (
    id BIGSERIAL PRIMARY KEY,
    media_item_id BIGINT NOT NULL REFERENCES media_items(id) ON DELETE CASCADE,

    -- Event details
    event_type TEXT NOT NULL,                             -- air_date, digital_release, physical_release
    event_date DATE NOT NULL,
    event_datetime_utc TIMESTAMPTZ,

    -- Status
    monitored BOOLEAN NOT NULL DEFAULT false,
    has_file BOOLEAN NOT NULL DEFAULT false,
    downloaded BOOLEAN NOT NULL DEFAULT false,

    -- Metadata
    title TEXT NOT NULL,                                  -- Episode/movie title for display
    parent_title TEXT,                                    -- Series title for episodes

    metadata JSONB DEFAULT '{}'::jsonb,                   -- Additional event data

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Indexes for calendar events
CREATE INDEX idx_calendar_events_media_item ON calendar_events(media_item_id);
CREATE INDEX idx_calendar_events_date ON calendar_events(event_date);
CREATE INDEX idx_calendar_events_monitored ON calendar_events(monitored, event_date) WHERE monitored = true;
CREATE INDEX idx_calendar_events_missing ON calendar_events(monitored, has_file, event_date) WHERE monitored = true AND has_file = false;

-- Scheduler jobs - Track background job execution
CREATE TABLE scheduler_jobs (
    id BIGSERIAL PRIMARY KEY,

    -- Job identification
    job_name TEXT NOT NULL UNIQUE,                        -- rss_sync, backlog_search, calendar_update, etc.
    job_type TEXT NOT NULL,                               -- recurring, one_time

    -- Schedule
    cron_expression TEXT,                                 -- Cron expression for recurring jobs
    interval_minutes INTEGER,                             -- Alternative: simple interval
    next_run_at TIMESTAMPTZ,
    last_run_at TIMESTAMPTZ,
    last_run_duration_ms INTEGER,

    -- Status
    enabled BOOLEAN NOT NULL DEFAULT true,
    running BOOLEAN NOT NULL DEFAULT false,

    -- Statistics
    total_runs INTEGER DEFAULT 0,
    consecutive_failures INTEGER DEFAULT 0,
    last_status TEXT,                                     -- success, failed, skipped
    last_error TEXT,

    -- Configuration
    config JSONB DEFAULT '{}'::jsonb,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Indexes for scheduler jobs
CREATE INDEX idx_scheduler_jobs_next_run ON scheduler_jobs(next_run_at) WHERE enabled = true AND running = false;
CREATE INDEX idx_scheduler_jobs_enabled ON scheduler_jobs(enabled) WHERE enabled = true;
CREATE INDEX idx_scheduler_jobs_running ON scheduler_jobs(running) WHERE running = true;

-- Scheduler job history - Audit trail of job executions
CREATE TABLE scheduler_job_history (
    id BIGSERIAL PRIMARY KEY,
    job_id BIGINT NOT NULL REFERENCES scheduler_jobs(id) ON DELETE CASCADE,

    -- Execution details
    started_at TIMESTAMPTZ NOT NULL,
    finished_at TIMESTAMPTZ,
    duration_ms INTEGER,

    -- Results
    status TEXT NOT NULL,                                 -- success, failed, skipped
    error_message TEXT,
    items_processed INTEGER DEFAULT 0,

    -- Output
    log_entries JSONB DEFAULT '[]'::jsonb,
    metadata JSONB DEFAULT '{}'::jsonb,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Indexes for scheduler job history
CREATE INDEX idx_scheduler_job_history_job ON scheduler_job_history(job_id, created_at DESC);
CREATE INDEX idx_scheduler_job_history_created_at ON scheduler_job_history(created_at DESC);

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

-- =============================================================================
-- Default Quality Definitions
-- =============================================================================

INSERT INTO quality_definitions (name, title, resolution, source, weight) VALUES
    -- Unknown/Any
    ('Unknown', 'Unknown', NULL, NULL, 0),

    -- SD Qualities
    ('SDTV', 'SDTV', 480, 'TV', 10),
    ('DVD', 'DVD', 480, 'DVD', 20),
    ('WEBDL-480p', 'WEB-DL 480p', 480, 'WEBDL', 30),
    ('WEBRip-480p', 'WEBRip 480p', 480, 'WEBRIP', 25),
    ('Bluray-480p', 'Bluray 480p', 480, 'BLURAY', 35),

    -- 720p Qualities
    ('HDTV-720p', 'HDTV 720p', 720, 'HDTV', 40),
    ('WEBDL-720p', 'WEB-DL 720p', 720, 'WEBDL', 60),
    ('WEBRip-720p', 'WEBRip 720p', 720, 'WEBRIP', 55),
    ('Bluray-720p', 'Bluray 720p', 720, 'BLURAY', 70),

    -- 1080p Qualities
    ('HDTV-1080p', 'HDTV 1080p', 1080, 'HDTV', 80),
    ('WEBDL-1080p', 'WEB-DL 1080p', 1080, 'WEBDL', 100),
    ('WEBRip-1080p', 'WEBRip 1080p', 1080, 'WEBRIP', 95),
    ('Bluray-1080p', 'Bluray 1080p', 1080, 'BLURAY', 110),
    ('Remux-1080p', 'Remux 1080p', 1080, 'BLURAY', 120),

    -- 2160p/4K Qualities
    ('HDTV-2160p', 'HDTV 2160p', 2160, 'HDTV', 130),
    ('WEBDL-2160p', 'WEB-DL 2160p', 2160, 'WEBDL', 150),
    ('WEBRip-2160p', 'WEBRip 2160p', 2160, 'WEBRIP', 145),
    ('Bluray-2160p', 'Bluray 2160p', 2160, 'BLURAY', 160),
    ('Remux-2160p', 'Remux 2160p', 2160, 'BLURAY', 170)
ON CONFLICT (name) DO NOTHING;

-- =============================================================================
-- Default Quality Profiles
-- =============================================================================

-- HD-1080p Profile (Most common)
INSERT INTO quality_profiles (name, description, upgrade_allowed) VALUES
    ('HD-1080p', 'Prefer 1080p WEB-DL or Bluray with upgrades allowed', true)
ON CONFLICT (name) DO NOTHING;

-- Update cutoff to WEBDL-1080p
UPDATE quality_profiles
SET cutoff_quality_id = (SELECT id FROM quality_definitions WHERE name = 'WEBDL-1080p')
WHERE name = 'HD-1080p' AND cutoff_quality_id IS NULL;

-- Add allowed qualities to HD-1080p profile
INSERT INTO quality_profile_items (profile_id, quality_id, allowed, sort_order)
SELECT
    p.id,
    q.id,
    true,
    CASE q.name
        WHEN 'Bluray-1080p' THEN 1
        WHEN 'Remux-1080p' THEN 2
        WHEN 'WEBDL-1080p' THEN 3
        WHEN 'WEBRip-1080p' THEN 4
        WHEN 'HDTV-1080p' THEN 5
        WHEN 'Bluray-720p' THEN 6
        WHEN 'WEBDL-720p' THEN 7
        WHEN 'WEBRip-720p' THEN 8
        WHEN 'HDTV-720p' THEN 9
    END
FROM quality_profiles p
CROSS JOIN quality_definitions q
WHERE p.name = 'HD-1080p'
AND q.name IN ('Bluray-1080p', 'Remux-1080p', 'WEBDL-1080p', 'WEBRip-1080p', 'HDTV-1080p',
               'Bluray-720p', 'WEBDL-720p', 'WEBRip-720p', 'HDTV-720p')
ON CONFLICT (profile_id, quality_id) DO NOTHING;

-- Ultra-HD Profile
INSERT INTO quality_profiles (name, description, upgrade_allowed) VALUES
    ('Ultra-HD', 'Prefer 4K/2160p quality with upgrades allowed', true)
ON CONFLICT (name) DO NOTHING;

UPDATE quality_profiles
SET cutoff_quality_id = (SELECT id FROM quality_definitions WHERE name = 'WEBDL-2160p')
WHERE name = 'Ultra-HD' AND cutoff_quality_id IS NULL;

INSERT INTO quality_profile_items (profile_id, quality_id, allowed, sort_order)
SELECT
    p.id,
    q.id,
    true,
    CASE q.name
        WHEN 'Remux-2160p' THEN 1
        WHEN 'Bluray-2160p' THEN 2
        WHEN 'WEBDL-2160p' THEN 3
        WHEN 'WEBRip-2160p' THEN 4
        WHEN 'HDTV-2160p' THEN 5
        WHEN 'Remux-1080p' THEN 6
        WHEN 'Bluray-1080p' THEN 7
        WHEN 'WEBDL-1080p' THEN 8
    END
FROM quality_profiles p
CROSS JOIN quality_definitions q
WHERE p.name = 'Ultra-HD'
AND q.name IN ('Remux-2160p', 'Bluray-2160p', 'WEBDL-2160p', 'WEBRip-2160p', 'HDTV-2160p',
               'Remux-1080p', 'Bluray-1080p', 'WEBDL-1080p')
ON CONFLICT (profile_id, quality_id) DO NOTHING;

-- SD Profile
INSERT INTO quality_profiles (name, description, upgrade_allowed) VALUES
    ('SD', 'Standard definition quality for limited bandwidth', true)
ON CONFLICT (name) DO NOTHING;

UPDATE quality_profiles
SET cutoff_quality_id = (SELECT id FROM quality_definitions WHERE name = 'DVD')
WHERE name = 'SD' AND cutoff_quality_id IS NULL;

INSERT INTO quality_profile_items (profile_id, quality_id, allowed, sort_order)
SELECT
    p.id,
    q.id,
    true,
    CASE q.name
        WHEN 'DVD' THEN 1
        WHEN 'Bluray-480p' THEN 2
        WHEN 'WEBDL-480p' THEN 3
        WHEN 'WEBRip-480p' THEN 4
        WHEN 'SDTV' THEN 5
    END
FROM quality_profiles p
CROSS JOIN quality_definitions q
WHERE p.name = 'SD'
AND q.name IN ('DVD', 'Bluray-480p', 'WEBDL-480p', 'WEBRip-480p', 'SDTV')
ON CONFLICT (profile_id, quality_id) DO NOTHING;

-- Any Quality Profile
INSERT INTO quality_profiles (name, description, upgrade_allowed) VALUES
    ('Any', 'Accept any quality (no restrictions)', false)
ON CONFLICT (name) DO NOTHING;

UPDATE quality_profiles
SET cutoff_quality_id = (SELECT id FROM quality_definitions WHERE name = 'Unknown')
WHERE name = 'Any' AND cutoff_quality_id IS NULL;

INSERT INTO quality_profile_items (profile_id, quality_id, allowed, sort_order)
SELECT
    p.id,
    q.id,
    true,
    q.weight DESC
FROM quality_profiles p
CROSS JOIN quality_definitions q
WHERE p.name = 'Any'
ON CONFLICT (profile_id, quality_id) DO NOTHING;

-- =============================================================================
-- Default Scheduler Jobs
-- =============================================================================

INSERT INTO scheduler_jobs (job_name, job_type, interval_minutes, enabled, config) VALUES
    -- RSS sync job - Check RSS feeds for new releases every 15 minutes
    ('rss_sync', 'recurring', 15, true, jsonb_build_object(
        'description', 'Synchronize RSS feeds from all enabled indexers',
        'max_items_per_sync', 100
    )),

    -- Backlog search job - Search for missing/wanted items hourly
    ('backlog_search', 'recurring', 60, true, jsonb_build_object(
        'description', 'Search for missing monitored items in backlog',
        'max_items_per_run', 50,
        'prioritize_recent', true
    )),

    -- Calendar update job - Update calendar events daily
    ('calendar_update', 'recurring', 1440, true, jsonb_build_object(
        'description', 'Update calendar with upcoming releases',
        'days_ahead', 30,
        'days_behind', 7
    )),

    -- Monitoring check job - Check monitored items for new episodes/releases
    ('monitoring_check', 'recurring', 30, true, jsonb_build_object(
        'description', 'Check for new episodes/releases for monitored items',
        'use_metadata_apis', true
    )),

    -- Download cleanup job - Clean up old completed/failed downloads
    ('download_cleanup', 'recurring', 1440, true, jsonb_build_object(
        'description', 'Remove old download records and clean up temporary files',
        'keep_completed_days', 30,
        'keep_failed_days', 7
    )),

    -- Blocklist cleanup job - Remove expired temporary blocks
    ('blocklist_cleanup', 'recurring', 360, true, jsonb_build_object(
        'description', 'Remove expired temporary blocklist entries',
        'cleanup_threshold_days', 30
    )),

    -- Quality upgrade search - Search for quality upgrades for existing media
    ('quality_upgrade_search', 'recurring', 720, true, jsonb_build_object(
        'description', 'Search for quality upgrades for media below cutoff',
        'max_items_per_run', 25,
        'min_age_days', 7
    ))
ON CONFLICT (job_name) DO NOTHING;
