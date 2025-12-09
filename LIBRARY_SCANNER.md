# Library Scanner Service - Implementation Guide

## Overview

The Library Scanner Service is a complete media library management system for Nimbus that automatically discovers, parses, and imports media files from the filesystem into the database.

## Features

- **Automatic Media Discovery**: Recursively walks filesystem directories to find media files
- **Intelligent Filename Parsing**: Extracts metadata from filenames using pattern matching
- **Multi-Media Support**: Movies, TV shows, music, and books
- **Hierarchical Relationships**: Automatically creates parent-child relationships (Series → Season → Episode, Artist → Album → Track)
- **Real-time Progress Tracking**: Monitor scan progress with live updates
- **Error Handling**: Comprehensive error logging with file-level details
- **Idempotent Operations**: Safe to re-run scans - won't create duplicates
- **Admin Controls**: Start, stop, and reset scanner via REST API

## Supported Media Types

### Video Files
- **Extensions**: `.mkv`, `.mp4`, `.avi`, `.mov`, `.wmv`, `.flv`, `.webm`, `.m4v`, `.mpg`, `.mpeg`, `.m2ts`, `.ts`
- **Movies**: Standalone files with optional year
- **TV Shows**: Files with season/episode patterns

### Audio Files
- **Extensions**: `.mp3`, `.flac`, `.m4a`, `.aac`, `.ogg`, `.opus`, `.wma`, `.wav`, `.ape`, `.alac`
- **Organization**: Artist/Album/Track hierarchy from directory structure

### Books
- **Extensions**: `.epub`, `.mobi`, `.azw`, `.azw3`, `.pdf`, `.djvu`, `.fb2`, `.cbz`, `.cbr`
- **Format**: Optional author extraction from filename

## Filename Parsing Rules

### Movies

Supported patterns:
```
Movie.Name.2021.mkv
Movie Name (2021).mp4
Movie.Name[2021].1080p.BluRay.mkv
```

The parser:
1. Extracts the 4-digit year from anywhere in the filename
2. Uses everything before the year as the title
3. Removes quality tags (1080p, BluRay, WEB-DL, etc.)
4. Normalizes dots and underscores to spaces

### TV Shows

Supported patterns:
```
Show.Name.S01E02.mkv          (Standard S01E02 format)
Show Name - 1x02.mp4          (Alternative 1x02 format)
Show/Season 1/Episode 02.mkv  (Directory-based)
```

The parser:
1. Extracts season and episode numbers
2. Uses show name from filename or parent directory
3. Creates hierarchy: Series → Season → Episode
4. Links episodes to seasons via `media_relations`

### Music

Expected structure:
```
Artist Name/
  Album Name/
    01 Track Name.mp3
    02 Another Track.flac
```

The parser:
1. Extracts artist from parent directory (2 levels up)
2. Extracts album from parent directory (1 level up)
3. Parses track number from filename prefix
4. Creates hierarchy: Artist → Album → Track

### Books

Supported patterns:
```
Book Title - Author Name.epub
Just The Title.mobi
```

The parser:
1. Looks for " - " separator
2. Left side = title, right side = author
3. If no separator, entire filename is the title

## Database Schema

### New Tables

#### `media_relations`
Tracks hierarchical relationships between media items.

```sql
CREATE TABLE media_relations (
    id BIGSERIAL PRIMARY KEY,
    parent_id BIGINT NOT NULL REFERENCES media_items(id) ON DELETE CASCADE,
    child_id BIGINT NOT NULL REFERENCES media_items(id) ON DELETE CASCADE,
    relation TEXT NOT NULL,        -- e.g., "series-season", "season-episode"
    sort_index NUMERIC,             -- Season/episode/track number
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

**Relation Types**:
- `series-season`: TV series to season
- `season-episode`: TV season to episode
- `artist-album`: Music artist to album
- `album-track`: Music album to track

#### `media_files`
Maps filesystem paths to media items.

```sql
CREATE TABLE media_files (
    id BIGSERIAL PRIMARY KEY,
    media_item_id BIGINT REFERENCES media_items(id) ON DELETE CASCADE,
    path TEXT NOT NULL UNIQUE,
    size BIGINT,
    hash TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

#### `scanner_state`
Single-row table tracking scanner status.

```sql
CREATE TABLE scanner_state (
    id INT PRIMARY KEY DEFAULT 1,
    running BOOLEAN NOT NULL DEFAULT FALSE,
    started_at TIMESTAMPTZ,
    finished_at TIMESTAMPTZ,
    files_scanned INT NOT NULL DEFAULT 0,
    items_created INT NOT NULL DEFAULT 0,
    items_updated INT NOT NULL DEFAULT 0,
    errors JSONB NOT NULL DEFAULT '[]'::jsonb,
    log JSONB NOT NULL DEFAULT '[]'::jsonb
);
```

## Backend Architecture

### Package Structure

```
internal/library/
├── parser.go      # Filename parsing logic
├── walker.go      # Filesystem traversal
├── service.go     # Media item upsert logic
├── scanner.go     # Main scanner loop
└── handlers.go    # HTTP API handlers
```

### Key Components

#### 1. Parser (`parser.go`)

**Function**: `ParseFilename(path string) *ParsedMedia`

Analyzes a file path and returns structured metadata:
```go
type ParsedMedia struct {
    Kind    string  // "movie", "tv_episode", "music_track", "book"
    Title   string
    Year    int
    Season  int     // TV only
    Episode int     // TV only
    Artist  string  // Music only
    Album   string  // Music only
    Track   int     // Music only
    Author  string  // Books only
}
```

#### 2. Walker (`walker.go`)

**Function**: `WalkMediaFiles(root string) ([]string, error)`

Recursively walks a directory tree and returns paths to all media files. Automatically skips:
- Hidden files/directories (starting with `.`)
- System directories (`@eaDir`, `$RECYCLE.BIN`, etc.)
- Thumbnail caches

#### 3. Service (`service.go`)

Contains upsert logic for each media type:
- `UpsertMovie()`: Insert/update movies
- `UpsertTVEpisode()`: Insert/update TV episodes (creates series/season hierarchy)
- `UpsertMusicTrack()`: Insert/update music tracks (creates artist/album hierarchy)
- `UpsertBook()`: Insert/update books

All operations are **idempotent** - running the scanner multiple times won't create duplicates.

#### 4. Scanner (`scanner.go`)

**Main Loop**: `Run(ctx context.Context) error`

1. Checks if scan is already running (prevents concurrent scans)
2. Marks scan as started in `scanner_state`
3. Walks filesystem to find all media files
4. Processes each file:
   - Parses filename
   - Upserts media item
   - Links file to item in `media_files`
   - Updates progress counters
5. Logs errors and activity
6. Marks scan as finished

Progress is updated every 10 files for efficiency.

#### 5. Handlers (`handlers.go`)

HTTP API endpoints:

**POST `/api/library/scan`** (Admin only)
- Start a new scan in the background
- Returns 409 if scan already running

**GET `/api/library/scan/status`** (Authenticated)
- Get current scanner state
- Polls every 5 seconds for real-time updates

**POST `/api/library/scan/stop`** (Admin only)
- Stop a running scan

**POST `/api/library/scan/reset`** (Admin only)
- Reset scanner state (clears logs and counters)

## Frontend Architecture

### Package Structure

```
frontend/src/
├── lib/api/library.ts        # API client and React Query hooks
└── pages/LibraryPage.tsx     # Scanner UI
```

### React Query Hooks

```typescript
// Fetch scan status (auto-polls every 5 seconds)
const { data: status } = useLibraryScanStatus();

// Start a scan
const startScan = useStartLibraryScan();
startScan.mutate();

// Stop a scan
const stopScan = useStopLibraryScan();
stopScan.mutate();

// Reset scanner state
const resetState = useResetScannerState();
resetState.mutate();
```

### UI Features

The Library Page (`/library`) provides:

1. **Status Dashboard**
   - Real-time scan status badge (Running/Idle)
   - Progress counters (files scanned, items created/updated)
   - Start/finish timestamps

2. **Control Panel**
   - Start Scan button
   - Stop Scan button (when running)
   - Reset State button

3. **Error Display**
   - Collapsible error log
   - Timestamp and message for each error
   - Scroll area for long lists

4. **Activity Log**
   - Collapsible activity log
   - Info/warn/error level indicators
   - Timestamps for each entry

5. **Help Section**
   - Supported file types
   - Filename pattern examples
   - How the scanner works

## How the Scanner Works

### Workflow

```
┌─────────────────────────────────────────────────┐
│ 1. User clicks "Start Scan"                     │
└─────────────────────────────────────────────────┘
                      ↓
┌─────────────────────────────────────────────────┐
│ 2. Backend starts goroutine                     │
│    - Marks scanner_state.running = true         │
│    - Sets started_at timestamp                  │
└─────────────────────────────────────────────────┘
                      ↓
┌─────────────────────────────────────────────────┐
│ 3. Walk filesystem                              │
│    - Recursively find all media files           │
│    - Skip hidden/system directories             │
└─────────────────────────────────────────────────┘
                      ↓
┌─────────────────────────────────────────────────┐
│ 4. Process each file                            │
│    ├─ Parse filename → extract metadata         │
│    ├─ Upsert media_items                        │
│    ├─ Upsert media_files                        │
│    ├─ Create media_relations (if hierarchical)  │
│    └─ Update progress counters                  │
└─────────────────────────────────────────────────┘
                      ↓
┌─────────────────────────────────────────────────┐
│ 5. Frontend polls status every 5 seconds        │
│    - Updates progress bars                      │
│    - Shows activity log                         │
│    - Displays errors                            │
└─────────────────────────────────────────────────┘
                      ↓
┌─────────────────────────────────────────────────┐
│ 6. Scan completes                               │
│    - Marks scanner_state.running = false        │
│    - Sets finished_at timestamp                 │
│    - Final log entry                            │
└─────────────────────────────────────────────────┘
```

### Media Hierarchy Creation

#### TV Shows
```
Breaking Bad (tv_series)
└── Season 1 (tv_season) ──[series-season, sort_index=1]
    ├── S01E01 (tv_episode) ──[season-episode, sort_index=1]
    ├── S01E02 (tv_episode) ──[season-episode, sort_index=2]
    └── S01E03 (tv_episode) ──[season-episode, sort_index=3]
```

#### Music
```
Pink Floyd (music_artist)
└── The Dark Side of the Moon (music_album) ──[artist-album]
    ├── 01 Speak to Me (music_track) ──[album-track, sort_index=1]
    ├── 02 Breathe (music_track) ──[album-track, sort_index=2]
    └── 03 On the Run (music_track) ──[album-track, sort_index=3]
```

### Deduplication Strategy

The scanner uses a **natural key** approach to prevent duplicates:

```sql
CREATE UNIQUE INDEX media_items_natural_key_idx 
ON media_items(kind, title, COALESCE(year, -1), COALESCE(parent_id, -1));
```

This ensures:
- Movies with same title + year are not duplicated
- TV episodes with same title + season are not duplicated
- Music tracks with same title + album are not duplicated

The `UpsertMediaItem` query uses `ON CONFLICT ... DO UPDATE` to update existing items instead of creating duplicates.

## Configuration

The scanner uses the `library.root_path` configuration value:

```sql
-- Default value in migration
INSERT INTO config (key, value) VALUES
    ('library.root_path', '"/media"');
```

To change the library path:
1. Go to Configuration page
2. Update `library.root_path` to your media directory
3. Restart the server for changes to take effect

## Extending the Scanner

### Adding New Media Types

1. **Update `parser.go`**:
   - Add new media type constant
   - Implement parsing logic
   - Add file extensions to appropriate map

2. **Update `service.go`**:
   - Add `Upsert<MediaType>()` function
   - Implement hierarchy creation if needed

3. **Update `scanner.go`**:
   - Add case in `processFile()` switch statement

4. **Update Database**:
   - Add new `MediaKind` constant in `internal/media/types.go`

### Customizing Filename Patterns

Edit the regex patterns in `parser.go`:

```go
var (
    // Add your custom patterns here
    myCustomPattern = regexp.MustCompile(`your-regex-here`)
)
```

Then update the respective parsing function to use the new pattern.

## Performance Considerations

### Batch Updates
Progress counters are updated every 10 files (not after each file) to reduce database round-trips.

### Concurrent Scans
The scanner prevents concurrent scans using the `scanner_state.running` flag. Attempting to start a second scan returns `409 Conflict`.

### Memory Usage
The scanner loads all file paths into memory before processing. For very large libraries (>100,000 files), consider using the channel-based `WalkMediaFilesChan()` function for streaming.

### Database Connections
The scanner uses the main database connection pool and doesn't require separate connections.

## Troubleshooting

### Scanner Won't Start

**Problem**: Clicking "Start Scan" returns an error

**Solutions**:
1. Check that no scan is currently running
2. Verify `library.root_path` exists and is readable
3. Check server logs for permission errors
4. Try resetting scanner state

### Files Not Being Imported

**Problem**: Scan completes but no items created

**Solutions**:
1. Verify files have supported extensions
2. Check filename patterns match expected format
3. Review error log for parsing failures
4. Ensure files are not in skipped directories (hidden, system)

### Duplicate Items

**Problem**: Same media appears multiple times

**Solutions**:
1. Check that filenames are consistent
2. Verify natural key index is present
3. Review logs for upsert errors
4. Manually remove duplicates and re-scan

### Scan Appears Stuck

**Problem**: Progress stops updating

**Solutions**:
1. Check backend logs for errors
2. Verify server is running
3. Try stopping and restarting scan
4. Reset scanner state if needed

## API Reference

### Start Scan

```http
POST /api/library/scan
Authorization: Bearer <admin-token>
```

**Response**:
```json
{
  "status": "started",
  "message": "Library scan started in background"
}
```

**Error Codes**:
- `401`: Not authenticated
- `403`: Not admin
- `409`: Scan already running
- `500`: Server error

### Get Scan Status

```http
GET /api/library/scan/status
Authorization: Bearer <token>
```

**Response**:
```json
{
  "running": true,
  "started_at": "2024-01-15T10:30:00Z",
  "finished_at": null,
  "files_scanned": 150,
  "items_created": 120,
  "items_updated": 30,
  "errors": [
    {
      "timestamp": "2024-01-15T10:31:00Z",
      "message": "Failed to parse file.mkv"
    }
  ],
  "log": [
    {
      "timestamp": "2024-01-15T10:30:00Z",
      "level": "info",
      "message": "Scan started"
    }
  ]
}
```

### Stop Scan

```http
POST /api/library/scan/stop
Authorization: Bearer <admin-token>
```

**Response**:
```json
{
  "status": "stopped",
  "message": "Scanner stopped"
}
```

### Reset Scanner State

```http
POST /api/library/scan/reset
Authorization: Bearer <admin-token>
```

**Response**:
```json
{
  "status": "reset",
  "message": "Scanner state reset"
}
```

## Future Enhancements

Potential improvements for future versions:

1. **Metadata Fetching**: Integrate with TMDB/OMDB/MusicBrainz to fetch rich metadata
2. **File Hashing**: Implement SHA-256 hashing to detect file changes
3. **Incremental Scans**: Only scan new/modified files
4. **Orphan Cleanup**: Remove media items whose files no longer exist
5. **Parallel Processing**: Process multiple files concurrently
6. **Custom Parsers**: Plugin system for user-defined parsing rules
7. **Scan Scheduling**: Automatic periodic scans (cron-like)
8. **Directory Watching**: Real-time updates using filesystem watchers
9. **Scan Profiles**: Different parsing rules for different directories
10. **Subtitle Support**: Parse and import subtitle files

## License

Part of the Nimbus media management system.
