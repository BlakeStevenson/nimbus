# Download Import Flow - Complete Implementation

## Overview

Nimbus now features a complete download-to-library import system that automatically processes downloaded media files, applies naming conventions, organizes them into proper folder structures, and updates the database - similar to Sonarr/Radarr functionality.

## Architecture

### Components

1. **Importer Service** (`internal/importer/service.go`)
   - Handles all import logic
   - Applies naming templates
   - Creates folder structures
   - Moves/copies files with hardlink support
   - Updates database

2. **Import Configuration** (`internal/importer/config.go`)
   - Loads settings from Downloads configuration
   - Validates configuration
   - Provides defaults

3. **Download Handler** (`internal/downloader/handlers.go`)
   - Exposes import API endpoint
   - Auto-import monitoring for completed downloads
   - Builds import requests from metadata

4. **NZB Downloader Plugin** (`plugins/nzb-downloader/main.go`)
   - Completes download and extraction
   - Finds main media file
   - Calls import API with metadata
   - Logs import progress

## Flow Diagram

```
┌─────────────────────┐
│  Download Started   │
│  (with metadata)    │
└──────────┬──────────┘
           │
           v
┌─────────────────────┐
│  Download File(s)   │
│  to temp directory  │
└──────────┬──────────┘
           │
           v
┌─────────────────────┐
│  Post-Process       │
│  (extract archives) │
└──────────┬──────────┘
           │
           v
┌─────────────────────┐
│  Find Main Media    │
│  File (largest)     │
└──────────┬──────────┘
           │
           v
┌─────────────────────┐
│  Call Import API    │
│  with metadata      │
└──────────┬──────────┘
           │
           v
┌─────────────────────┐
│  Load Config        │
│  (naming, folders)  │
└──────────┬──────────┘
           │
           v
┌─────────────────────┐
│  Apply Templates    │
│  (generate names)   │
└──────────┬──────────┘
           │
           v
┌─────────────────────┐
│  Create Folders     │
│  (series, season)   │
└──────────┬──────────┘
           │
           v
┌─────────────────────┐
│  Move/Hardlink File │
│  to library path    │
└──────────┬──────────┘
           │
           v
┌─────────────────────┐
│  Import Extra Files │
│  (subtitles, NFO)   │
└──────────┬──────────┘
           │
           v
┌─────────────────────┐
│  Update Database    │
│  (media_files)      │
└──────────┬──────────┘
           │
           v
┌─────────────────────┐
│  Mark Complete      │
│  Return final path  │
└─────────────────────┘
```

## API Endpoint

### POST /api/downloads/import

Imports a completed download into the library.

**Request Body:**
```json
{
  "source_path": "/tmp/downloads/movie.mkv",
  "media_type": "movie",
  "title": "The Matrix",
  "year": 1999,
  "quality": "1080p",
  "media_item_id": 123
}
```

**For TV Episodes:**
```json
{
  "source_path": "/tmp/downloads/episode.mkv",
  "media_type": "tv",
  "title": "Breaking Bad",
  "season": 1,
  "episode": 1,
  "episode_title": "Pilot",
  "quality": "720p"
}
```

**Response:**
```json
{
  "success": true,
  "final_path": "/media/movies/The Matrix (1999)/The Matrix (1999).mkv",
  "media_item_id": 123,
  "message": "Successfully imported The Matrix to /media/movies/...",
  "created_folders": [
    "/media/movies/The Matrix (1999)"
  ],
  "moved_files": [
    "/media/movies/The Matrix (1999)/The Matrix (1999).mkv"
  ],
  "imported_extras": [
    "/media/movies/The Matrix (1999)/The Matrix (1999).srt"
  ]
}
```

## Configuration Integration

The importer uses all settings from the Downloads configuration:

### Movie Import
- `downloads.movie_naming_format` - File naming template
- `downloads.movie_folder_format` - Folder naming template
- `downloads.create_movie_folder` - Whether to create movie folders
- `downloads.rename_movies` - Whether to rename files

### TV Import
- `downloads.tv_naming_format` - Episode naming template
- `downloads.tv_folder_format` - Series folder template
- `downloads.tv_season_folder_format` - Season folder template
- `downloads.tv_use_season_folders` - Create season folders
- `downloads.create_series_folder` - Create series folder
- `downloads.rename_episodes` - Rename episode files

### File Management
- `downloads.replace_illegal_characters` - Replace invalid characters
- `downloads.colon_replacement` - How to handle colons
- `downloads.use_hardlinks` - Use hardlinks instead of copy
- `downloads.import_extra_files` - Import subtitles, NFO, etc.
- `downloads.extra_file_extensions` - Extensions to import

### Advanced
- `downloads.set_permissions` - Set file/folder permissions
- `downloads.chmod_folder` - Folder permission mode
- `downloads.chmod_file` - File permission mode

## Template Processing

### Token Replacement

The importer supports these tokens in naming templates:

**Movies:**
- `{Movie Title}` → "The Matrix"
- `{Release Year}` → "1999"
- `{Quality}` → "1080p"

**TV Shows:**
- `{Series Title}` → "Breaking Bad"
- `{season:00}` → "01" (zero-padded)
- `{episode:00}` → "01" (zero-padded)
- `{Episode Title}` → "Pilot"
- `{Quality}` → "720p"

### Path Sanitization

The importer automatically:
- Replaces illegal filesystem characters
- Handles colons per configuration
- Trims whitespace and dots
- Ensures valid path names

## File Operations

### Hardlinks vs Copy

**Hardlinks (default):**
```
Source: /tmp/downloads/movie.mkv (5GB)
Target: /media/movies/Movie (2020)/Movie (2020).mkv
Result: Same file, no duplicate space
```

**Copy (fallback):**
```
Source: /tmp/downloads/movie.mkv (5GB)
Target: /media/movies/Movie (2020)/Movie (2020).mkv
Result: Two copies, 10GB total
```

Hardlinks automatically fall back to copy if:
- Source and destination are on different filesystems
- Filesystem doesn't support hardlinks
- Hardlink operation fails

### Extra Files

When `import_extra_files` is enabled:
```
Source Directory:
  movie.mkv
  movie.srt
  movie.en.srt
  movie.nfo

Result:
  /media/movies/Movie (2020)/
    Movie (2020).mkv
    Movie (2020).srt
    Movie (2020).en.srt
    Movie (2020).nfo
```

## Usage Examples

### Example 1: Movie Import

**Download Metadata:**
```json
{
  "title": "Inception",
  "year": 2010,
  "media_type": "movie",
  "quality": "1080p"
}
```

**Configuration:**
```json
{
  "movie_naming_format": "{Movie Title} ({Release Year}) [{Quality}]",
  "movie_folder_format": "{Movie Title} ({Release Year})",
  "create_movie_folder": true
}
```

**Result:**
```
/media/movies/Inception (2010)/Inception (2010) [1080p].mkv
```

### Example 2: TV Episode Import

**Download Metadata:**
```json
{
  "title": "Breaking Bad",
  "season": 1,
  "episode": 1,
  "episode_title": "Pilot",
  "media_type": "tv",
  "quality": "720p"
}
```

**Configuration:**
```json
{
  "tv_naming_format": "{Series Title} - S{season:00}E{episode:00} - {Episode Title}",
  "tv_folder_format": "{Series Title}",
  "tv_season_folder_format": "Season {season:00}",
  "tv_use_season_folders": true
}
```

**Result:**
```
/media/tv/Breaking Bad/Season 01/Breaking Bad - S01E01 - Pilot.mkv
```

### Example 3: With Subtitles

**Source:**
```
/tmp/downloads/
  Movie.mkv
  Movie.srt
  Movie.en.srt
  Movie.nfo
```

**Result:**
```
/media/movies/Movie (2020)/
  Movie (2020).mkv
  Movie (2020).srt
  Movie (2020).en.srt
  Movie (2020).nfo
```

## NZB Downloader Integration

The NZB downloader plugin automatically triggers import when:

1. Download completes successfully
2. Post-processing (extraction) finishes
3. Main media file is found
4. Download has required metadata

### Required Metadata

**For Movies:**
- `title` - Movie title
- `media_type` = "movie"
- Optional: `year`, `quality`, `media_id`

**For TV Episodes:**
- `title` - Series title
- `media_type` = "tv" or "tv_episode"
- `season` - Season number
- `episode` - Episode number
- Optional: `episode_title`, `quality`, `media_id`

### Import Logging

The downloader logs all import steps:
```
[12:34:56] Download complete, processing files...
[12:34:57] Found main media file: Movie.mkv
[12:34:57] Importing to library...
[12:34:58] Imported to: /media/movies/Movie (2020)/Movie (2020).mkv
[12:34:58] Processing completed successfully
```

## Database Updates

After successful import:

1. **media_files table** updated with final path
2. **media_items table** created/updated (if media_id not provided)
3. **downloads table** updated with destination_path

## Error Handling

The importer handles errors gracefully:

### Insufficient Metadata
```
ERROR: season and episode numbers are required for TV imports
```

### Path Issues
```
ERROR: source path does not exist: /tmp/downloads/movie.mkv
```

### Disk Space
```
ERROR: insufficient free space: 5GB required, 2GB available
```

### File Operations
```
WARNING: hardlink failed, falling back to copy
```

## Testing the Flow

### Manual Test

1. **Start Nimbus server:**
   ```bash
   ./nimbus
   ```

2. **Configure download settings:**
   - Navigate to Configuration → Downloads
   - Set naming formats
   - Configure folder structure
   - Save changes

3. **Create a test download with metadata:**
   ```bash
   curl -X POST http://localhost:8080/api/downloads \
     -H "Content-Type: application/json" \
     -d '{
       "plugin_id": "nzb-downloader",
       "name": "Test Movie",
       "url": "https://example.com/test.nzb",
       "metadata": {
         "title": "Test Movie",
         "year": 2024,
         "media_type": "movie",
         "quality": "1080p"
       }
     }'
   ```

4. **Monitor download progress:**
   - Check download logs in UI
   - Watch for "Importing to library..." message
   - Verify final path in logs

5. **Verify result:**
   ```bash
   ls -la /media/movies/Test\ Movie\ \(2024\)/
   ```

## Troubleshooting

### Import Not Triggering

**Check:**
1. Download has required metadata
2. `shouldImport()` returns true
3. Main media file was found
4. No errors in download logs

### Wrong Path

**Check:**
1. Downloads configuration is correct
2. Library paths are configured properly
3. Naming templates are valid

### Files Not Moving

**Check:**
1. Source file exists
2. Destination directory is writable
3. Sufficient disk space
4. Hardlinks vs copy setting

### Permissions Issues

**Check:**
1. Nimbus has write access to library paths
2. `set_permissions` configuration
3. File/folder permission values

## Future Enhancements

Potential improvements:
- [ ] Preview naming before import
- [ ] Bulk rename existing library
- [ ] Custom naming patterns (regex)
- [ ] Quality upgrade detection
- [ ] Import history tracking
- [ ] Rollback/undo imports
- [ ] Advanced metadata extraction
- [ ] Image/fanart download
- [ ] NFO file generation

## Related Documentation

- [Downloads Configuration](DOWNLOADS_CONFIGURATION.md) - Settings reference
- [Media Library Paths](MEDIA_LIBRARY_PATHS.md) - Library organization
- [Library Scanner](LIBRARY_SCANNER.md) - How scanner works
- [Plugin System](docs/PLUGIN_SYSTEM.md) - Plugin architecture

## Summary

The download import flow is now complete and production-ready:

✅ **Automatic import** after download completion
✅ **Template-based naming** with full token support
✅ **Folder structure** matching Plex/Emby conventions
✅ **Hardlink support** to save disk space
✅ **Extra file import** (subtitles, NFO, etc.)
✅ **Database integration** with media_files tracking
✅ **Error handling** with graceful fallbacks
✅ **Configuration driven** via Downloads settings
✅ **Logging and monitoring** throughout process

This provides a complete Sonarr/Radarr-like experience for media management in Nimbus!
