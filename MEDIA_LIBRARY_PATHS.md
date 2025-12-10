# Media-Specific Library Paths

## Overview

Nimbus now supports independent library paths for each media type (movies, TV shows, music, and books). This allows you to organize your media across different directories or storage volumes.

## Configuration

The following configuration keys are available:

- `library.movie_path` - Root directory for movie files (default: `/media/movies`)
- `library.tv_path` - Root directory for TV show files (default: `/media/tv`)
- `library.music_path` - Root directory for music files (default: `/media/music`)
- `library.book_path` - Root directory for book files (default: `/media/books`)

### Legacy Configuration

The original `library.root_path` configuration is still supported as a fallback. If media-specific paths are not configured, the scanner will use the root path.

## Database Migration

The configuration keys are added via migration `0011_media_type_library_paths.sql`:

```sql
INSERT INTO config (key, value, metadata) VALUES
    ('library.movie_path', '"/media/movies"', ...),
    ('library.tv_path', '"/media/tv"', ...),
    ('library.music_path', '"/media/music"', ...),
    ('library.book_path', '"/media/books"', ...)
ON CONFLICT (key) DO NOTHING;
```

## Scanner Behavior

When a library scan is triggered, the scanner will:

1. Check for media-specific paths in the configuration
2. Scan each configured path separately
3. Aggregate all discovered files from all paths
4. Process files normally, detecting media type from filename patterns

### Example Scan Log

```
[INFO] starting library scan
[INFO] scanning media-specific path: media_type=movie path=/mnt/movies
[INFO] scanning media-specific path: media_type=tv path=/mnt/tv-shows
[INFO] walking filesystem: path=/mnt/movies
[INFO] walking filesystem: path=/mnt/tv-shows
[INFO] found media files: count=1523 across all library paths
```

## API Usage

### Setting Media Paths

Use the standard config API to set media-specific paths:

```bash
# Set movie library path
curl -X POST http://localhost:8080/api/config \
  -H "Content-Type: application/json" \
  -d '{
    "key": "library.movie_path",
    "value": "/mnt/storage/movies"
  }'

# Set TV library path
curl -X POST http://localhost:8080/api/config \
  -H "Content-Type: application/json" \
  -d '{
    "key": "library.tv_path",
    "value": "/mnt/storage/tv-shows"
  }'
```

### Getting Current Paths

```bash
# Get movie path
curl http://localhost:8080/api/config/library.movie_path

# Get TV path
curl http://localhost:8080/api/config/library.tv_path
```

### Triggering a Scan

After updating paths, trigger a new scan:

```bash
curl -X POST http://localhost:8080/api/library/scan
```

## Configuration UI

The media-specific paths are automatically displayed in the Configuration page under the "General" tab. Each media type has its own text input field where you can specify the directory path.

## Implementation Details

### Scanner Changes

The `Scanner` struct in `internal/library/scanner.go` now includes:

- `mediaPaths map[string]string` - Stores media-specific paths
- `SetMediaPath(mediaType, path string)` - Sets a media-specific path
- `GetMediaPath(mediaType string) string` - Gets path with fallback to rootDir

The `Run()` method collects all configured paths and walks each directory, aggregating files before processing.

### Router Initialization

The HTTP router (`internal/http/router.go`) loads media-specific paths from the config store during initialization and configures the library handler accordingly.

## Benefits

1. **Flexible Storage** - Store different media types on different volumes/directories
2. **Organization** - Keep your media organized by type
3. **Performance** - Only scan relevant directories for specific media types
4. **Backward Compatible** - Falls back to `library.root_path` if specific paths aren't set

## Migration Path

Existing installations will continue to work:

1. The migration adds default media-specific paths but doesn't affect existing scans
2. `library.root_path` is marked as deprecated but still functional
3. You can gradually migrate to media-specific paths by updating the config
4. The scanner automatically uses specific paths when configured, otherwise falls back to root path

## Example Directory Structure

```
/mnt/storage/
├── movies/
│   ├── The Matrix (1999)/
│   │   └── The Matrix (1999).mkv
│   └── Inception (2010)/
│       └── Inception (2010).mkv
├── tv-shows/
│   ├── Breaking Bad/
│   │   ├── Season 01/
│   │   │   └── Breaking Bad S01E01.mkv
│   │   └── Season 02/
│   │       └── Breaking Bad S02E01.mkv
├── music/
│   └── [music files]
└── books/
    └── [ebook files]
```

Configure each path separately:
- `library.movie_path` = `/mnt/storage/movies`
- `library.tv_path` = `/mnt/storage/tv-shows`
- `library.music_path` = `/mnt/storage/music`
- `library.book_path` = `/mnt/storage/books`
