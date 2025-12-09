-- Add configuration for enabled media types
INSERT INTO config (key, value, metadata) VALUES
    ('library.enabled_media_types', '["tv", "movie", "music", "book"]', jsonb_build_object(
        'title', 'Enabled Media Types',
        'description', 'Media types that are enabled for scanning and indexing',
        'type', 'multi',
        'values', jsonb_build_array('tv', 'movie', 'music', 'book')
    ))
ON CONFLICT (key) DO NOTHING;
