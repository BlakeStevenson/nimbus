-- Add independent library paths for each media type
-- This allows users to specify different root directories for movies, TV shows, music, and books

INSERT INTO config (key, value, metadata) VALUES
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
    ))
ON CONFLICT (key) DO NOTHING;

-- Update metadata for library.root_path to indicate it's deprecated but still used as fallback
UPDATE config
SET metadata = metadata || jsonb_build_object(
    'description', 'Legacy root directory for all media (deprecated - use media-specific paths instead)',
    'deprecated', true
)
WHERE key = 'library.root_path';
