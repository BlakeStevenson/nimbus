-- Migration: 0004_plugins.sql
-- Description: Add plugin system tables

-- =============================================================================
-- plugins - Track installed plugins and their metadata
-- =============================================================================
-- This table stores information about plugins that are installed in Nimbus.
-- Plugins can provide:
--   - API routes (new REST endpoints)
--   - UI extensions (React components and navigation items)
--   - Event handlers (respond to system events)
--   - Compatibility layers (e.g., Sonarr/Radarr API compatibility)
--
-- The enabled field controls whether a plugin is active and loaded at runtime.
-- The capabilities array indicates what features the plugin provides.
-- =============================================================================

CREATE TABLE plugins (
    id TEXT PRIMARY KEY,                  -- Unique plugin identifier (e.g., "sonarr-compat", "bittorrent")
    name TEXT NOT NULL,                   -- Human-readable name (e.g., "Sonarr Compatibility")
    description TEXT NOT NULL DEFAULT '', -- Brief description of plugin functionality
    version TEXT NOT NULL DEFAULT '0.1.0', -- Semantic version (e.g., "1.2.3")
    enabled BOOLEAN NOT NULL DEFAULT TRUE, -- Whether plugin is active
    capabilities JSONB NOT NULL DEFAULT '[]'::jsonb, -- Array of strings: ["api", "ui", "events", "compat:sonarr"]
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Index for efficiently finding enabled plugins (most common query)
CREATE INDEX plugins_enabled_idx ON plugins(enabled) WHERE enabled = TRUE;

-- Index for searching plugins by capability
CREATE INDEX plugins_capabilities_idx ON plugins USING GIN(capabilities);

-- Auto-update timestamp trigger
CREATE TRIGGER update_plugins_updated_at
    BEFORE UPDATE ON plugins
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- =============================================================================
-- Example plugins for reference
-- =============================================================================
-- These are commented out - plugins should self-register at runtime
-- INSERT INTO plugins (id, name, description, version, enabled, capabilities) VALUES
-- ('sonarr-compat', 'Sonarr Compatibility', 'Provides Sonarr-compatible API endpoints for TV show management', '0.1.0', true, '["api", "ui", "compat:sonarr"]'::jsonb),
-- ('radarr-compat', 'Radarr Compatibility', 'Provides Radarr-compatible API endpoints for movie management', '0.1.0', true, '["api", "ui", "compat:radarr"]'::jsonb),
-- ('bittorrent', 'BitTorrent Client', 'Manages torrent downloads with qBittorrent/Transmission', '0.1.0', true, '["api", "ui", "events"]'::jsonb);
