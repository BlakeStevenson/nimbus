-- Core media items table (generic, extensible)
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

-- Global configuration table
CREATE TABLE config (
    key         TEXT PRIMARY KEY,
    value       JSONB NOT NULL,
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Trigger to auto-update updated_at on media_items
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

CREATE TRIGGER update_media_items_updated_at
    BEFORE UPDATE ON media_items
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_config_updated_at
    BEFORE UPDATE ON config
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- Insert some default configuration values
INSERT INTO config (key, value) VALUES
    ('library.root_path', '"/media"'),
    ('download.tmp_path', '"/tmp/downloads"'),
    ('indexer.default_language', '"en"'),
    ('server.version', '"0.1.0"');
