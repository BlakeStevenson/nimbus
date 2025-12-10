-- Migration: Add downloads table for persistent download tracking
-- This supports multiple downloader plugins (NZB, torrent, etc.)

CREATE TABLE IF NOT EXISTS downloads (
    id TEXT PRIMARY KEY DEFAULT ('dl_' || md5(random()::text)),
    plugin_id TEXT NOT NULL REFERENCES plugins(id) ON DELETE CASCADE,

    -- Download metadata
    name TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'queued', -- queued, downloading, processing, paused, completed, failed, cancelled
    progress INTEGER NOT NULL DEFAULT 0 CHECK (progress >= 0 AND progress <= 100),

    -- Size tracking
    total_bytes BIGINT,
    downloaded_bytes BIGINT DEFAULT 0,

    -- Download source (either URL or file content)
    url TEXT,
    file_content BYTEA, -- For storing NZB/torrent files
    file_name TEXT, -- Original filename

    -- Destination
    destination_path TEXT,

    -- Error tracking
    error_message TEXT,
    retry_count INTEGER DEFAULT 0,

    -- Queue management
    queue_position INTEGER,
    priority INTEGER DEFAULT 0, -- Higher priority downloads first

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

-- Indexes for common queries
CREATE INDEX IF NOT EXISTS idx_downloads_status ON downloads(status);
CREATE INDEX IF NOT EXISTS idx_downloads_plugin_id ON downloads(plugin_id);
CREATE INDEX IF NOT EXISTS idx_downloads_created_by ON downloads(created_by_user_id);
CREATE INDEX IF NOT EXISTS idx_downloads_queue_position ON downloads(queue_position) WHERE status IN ('queued', 'downloading');
CREATE INDEX IF NOT EXISTS idx_downloads_created_at ON downloads(created_at DESC);

-- Download logs table for detailed history
CREATE TABLE IF NOT EXISTS download_logs (
    id SERIAL PRIMARY KEY,
    download_id TEXT NOT NULL REFERENCES downloads(id) ON DELETE CASCADE,
    level TEXT NOT NULL DEFAULT 'info', -- info, warn, error
    message TEXT NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_download_logs_download_id ON download_logs(download_id);
CREATE INDEX IF NOT EXISTS idx_download_logs_created_at ON download_logs(created_at DESC);
