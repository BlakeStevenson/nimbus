# Downloads Configuration Guide

Nimbus includes a comprehensive Downloads configuration system similar to Sonarr/Radarr, allowing you to control how downloaded media is organized, named, and imported into your library.

## Table of Contents

1. [Overview](#overview)
2. [Configuration Sections](#configuration-sections)
3. [Naming Templates](#naming-templates)
4. [Configuration Options](#configuration-options)
5. [Examples](#examples)

## Overview

The Downloads configuration is organized into logical sections, each controlling a different aspect of the download and import process:

- **Movie Naming** - File and folder naming for movies
- **TV Naming** - File and folder naming for TV shows
- **Folder Structure** - How folders are organized
- **File Management** - File handling and character replacement
- **Quality** - Quality profiles and upgrade settings
- **Download Client** - Integration with download clients
- **Importing** - Import behavior and extra files
- **Advanced** - Permissions and cleanup settings

## Configuration Sections

### Movie Naming

Control how movie files and folders are named:

- **Movie File Naming Format** - Template for movie filenames
- **Movie Folder Format** - Template for movie folders

**Available Tokens:**
- `{Movie Title}` - Movie title
- `{Release Year}` - Year of release
- `{Quality}` - Quality profile (e.g., 1080p)
- `{Edition}` - Edition info (Director's Cut, Extended, etc.)
- `{IMDb ID}` - IMDb identifier

**Default:** `{Movie Title} ({Release Year})`

### TV Naming

Control how TV show files and folders are named:

- **TV Episode Naming Format** - Template for episode filenames
- **TV Series Folder Format** - Template for series folders
- **TV Season Folder Format** - Template for season folders
- **Use Season Folders** - Create separate folders for each season

**Available Tokens:**
- `{Series Title}` - TV series title
- `{Season}` or `{season:00}` - Season number (with zero-padding)
- `{Episode}` or `{episode:00}` - Episode number (with zero-padding)
- `{Episode Title}` - Episode title
- `{Quality}` - Quality profile
- `{Release Date}` - Air date
- `{TVDb ID}` - TVDb identifier

**Defaults:**
- Episode: `{Series Title} - S{season:00}E{episode:00} - {Episode Title}`
- Series Folder: `{Series Title}`
- Season Folder: `Season {season:00}`

### Folder Structure

- **Create Series Folder** - Automatically create a folder for each TV series
- **Create Movie Folder** - Automatically create a folder for each movie

### File Management

- **Rename Episodes** - Automatically rename downloaded episodes
- **Rename Movies** - Automatically rename downloaded movies
- **Replace Illegal Characters** - Replace invalid filename characters
- **Colon Replacement** - How to handle colons in filenames
  - `delete` - Remove colons entirely
  - `dash` - Replace with dash (-)
  - `space` - Replace with space
  - `spacedash` - Replace with space-dash ( - )

### Quality

- **Quality Profiles** - Available quality profiles (ordered by preference)
- **Preferred Quality** - Preferred quality for downloads
- **Enable Quality Upgrades** - Automatically upgrade to higher quality
- **Upgrade Until Quality** - Stop upgrading at this quality level

**Quality Options:** `Any`, `SD`, `720p`, `1080p`, `2160p`

### Download Client

- **Enable Completed Download Handling** - Auto-import completed downloads
- **Remove Completed Downloads** - Remove from download client after import
- **Check For Finished Downloads Interval** - Check interval in minutes

### Importing

- **Skip Free Space Check** - Skip free space verification
- **Minimum Free Space (MB)** - Required free space before importing
- **Use Hardlinks Instead of Copy** - Save disk space with hardlinks
- **Import Extra Files** - Import subtitles, NFO files, etc.
- **Extra File Extensions** - Comma-separated list of extensions to import

**Default Extra Files:** `srt,nfo,txt`

### Advanced

- **Set Permissions** - Set permissions on imported files/folders
- **Folder Permissions** - Permissions for folders (octal)
- **File Permissions** - Permissions for files (octal)
- **Recycle Bin Path** - Move deleted files here instead of permanent deletion
- **Recycle Bin Cleanup Days** - Days to keep files in recycle bin

## Naming Templates

### Template Syntax

- Use curly braces `{}` to denote tokens
- Add format specifiers with colon: `{season:00}` (zero-padded to 2 digits)
- Combine tokens with literal text and separators

### Movie Examples

**Simple:**
```
{Movie Title} ({Release Year})
→ The Matrix (1999)
```

**With Quality:**
```
{Movie Title} ({Release Year}) [{Quality}]
→ Inception (2010) [1080p]
```

**With IMDb ID:**
```
{Movie Title} ({Release Year}) {IMDb ID}
→ The Dark Knight (2008) tt0468569
```

### TV Show Examples

**Standard (Sonarr-style):**
```
{Series Title} - S{season:00}E{episode:00} - {Episode Title}
→ Breaking Bad - S01E01 - Pilot
```

**Simple:**
```
{Series Title} {season:00}x{episode:00}
→ Game of Thrones 01x01
```

**With Quality:**
```
{Series Title} - S{season:00}E{episode:00} - {Episode Title} [{Quality}]
→ The Wire - S03E05 - Straight and True [720p]
```

**Date-based:**
```
{Series Title} - {Release Date} - {Episode Title}
→ Last Week Tonight - 2024-01-15 - Episode Title
```

## Configuration Options

### API Access

All download settings can be accessed via the config API:

```bash
# Get a setting
curl http://localhost:8080/api/config/downloads.movie_naming_format

# Update a setting
curl -X POST http://localhost:8080/api/config \
  -H "Content-Type: application/json" \
  -d '{
    "key": "downloads.movie_naming_format",
    "value": "{Movie Title} ({Release Year}) [{Quality}]"
  }'
```

### Configuration UI

Access the Downloads configuration through the web interface:

1. Navigate to **Configuration** page
2. Click on the **Downloads** tab
3. Configure settings organized by section
4. Click **Save Changes** to apply

## Examples

### Example 1: Plex-Optimized Structure

For a Plex-friendly library structure:

**Movies:**
- Movie Folder Format: `{Movie Title} ({Release Year})`
- Movie File Format: `{Movie Title} ({Release Year})`
- Create Movie Folder: `true`

Result:
```
/media/movies/
├── The Matrix (1999)/
│   └── The Matrix (1999).mkv
└── Inception (2010)/
    └── Inception (2010).mkv
```

**TV Shows:**
- Series Folder Format: `{Series Title}`
- Season Folder Format: `Season {season:00}`
- Episode Format: `{Series Title} - S{season:00}E{episode:00} - {Episode Title}`
- Use Season Folders: `true`

Result:
```
/media/tv/
└── Breaking Bad/
    ├── Season 01/
    │   ├── Breaking Bad - S01E01 - Pilot.mkv
    │   └── Breaking Bad - S01E02 - Cat's in the Bag....mkv
    └── Season 02/
        └── Breaking Bad - S02E01 - Seven Thirty-Seven.mkv
```

### Example 2: Compact Structure

For minimal folder depth:

**Movies:**
- Movie Folder Format: *(empty)*
- Movie File Format: `{Movie Title} ({Release Year})`
- Create Movie Folder: `false`

Result:
```
/media/movies/
├── The Matrix (1999).mkv
└── Inception (2010).mkv
```

**TV Shows:**
- Episode Format: `{Series Title} {season:00}x{episode:00} - {Episode Title}`
- Use Season Folders: `false`

Result:
```
/media/tv/
└── Breaking Bad/
    ├── Breaking Bad 01x01 - Pilot.mkv
    ├── Breaking Bad 01x02 - Cat's in the Bag....mkv
    └── Breaking Bad 02x01 - Seven Thirty-Seven.mkv
```

### Example 3: Quality Tracking

Track quality in filenames for easy identification:

**Movies:**
- Movie File Format: `{Movie Title} ({Release Year}) [{Quality}]`

**TV Shows:**
- Episode Format: `{Series Title} - S{season:00}E{episode:00} - {Episode Title} [{Quality}]`

Result:
```
The Matrix (1999) [1080p].mkv
Breaking Bad - S01E01 - Pilot [720p].mkv
```

## Quality Management

### Quality Profiles

Define which qualities are acceptable:

```json
["Any", "SD", "720p", "1080p", "2160p"]
```

### Upgrade Strategy

Configure automatic quality upgrades:

1. **Enable Quality Upgrades**: `true`
2. **Preferred Quality**: `1080p`
3. **Upgrade Until Quality**: `1080p`

This will automatically replace lower quality releases with 1080p versions when available.

## File Management

### Hardlinks vs Copy

**Hardlinks** (recommended):
- Same file referenced in multiple locations
- No duplicate disk usage
- Requires same filesystem
- Enable with: `Use Hardlinks Instead of Copy: true`

**Copy**:
- Full file duplication
- Works across filesystems
- Uses more disk space
- Use when: Source and destination are on different filesystems

### Extra Files

Import subtitle files, NFO metadata, and other extras alongside your media:

1. **Import Extra Files**: `true`
2. **Extra File Extensions**: `srt,nfo,txt,jpg,png`

Result:
```
/media/movies/The Matrix (1999)/
├── The Matrix (1999).mkv
├── The Matrix (1999).srt
├── The Matrix (1999).nfo
└── poster.jpg
```

## Permissions

### Setting Permissions

Control file/folder permissions for multi-user environments:

1. **Set Permissions**: `true`
2. **Folder Permissions**: `755` (rwxr-xr-x)
3. **File Permissions**: `644` (rw-r--r--)

### Common Permission Values

- `755` - Owner: rwx, Group/Others: rx (standard for folders)
- `644` - Owner: rw, Group/Others: r (standard for files)
- `775` - Owner/Group: rwx, Others: rx (shared group access)
- `666` - All: rw (not recommended for security)

## Recycle Bin

Move deleted files to a recycle bin instead of permanent deletion:

1. **Recycle Bin Path**: `/mnt/recycle`
2. **Recycle Bin Cleanup Days**: `7`

Files deleted from Nimbus will be moved to the recycle bin and automatically removed after 7 days.

## Best Practices

1. **Test naming formats** with a single download before applying to your entire library
2. **Use season folders** for TV shows to keep them organized
3. **Enable hardlinks** to save disk space when possible
4. **Import extra files** to preserve subtitles and metadata
5. **Set up recycle bin** to protect against accidental deletions
6. **Enable quality upgrades** to automatically improve your library over time
7. **Configure permissions** if running in a multi-user environment

## Migration from Sonarr/Radarr

If you're migrating from Sonarr or Radarr, Nimbus supports similar naming formats and structures. Simply copy your naming templates from Sonarr/Radarr to Nimbus configuration.

### Token Mapping

Most Sonarr/Radarr tokens work directly in Nimbus:

| Sonarr/Radarr | Nimbus |
|---------------|--------|
| `{Series Title}` | `{Series Title}` |
| `{season:00}` | `{season:00}` |
| `{episode:00}` | `{episode:00}` |
| `{Movie Title}` | `{Movie Title}` |
| `{Release Year}` | `{Release Year}` |
| `{Quality Full}` | `{Quality}` |

## Troubleshooting

### Files not importing

1. Check **Minimum Free Space** settings
2. Verify **Download Client** connection
3. Review scanner logs for errors
4. Ensure paths are accessible and have correct permissions

### Naming not applying

1. Verify **Rename Episodes/Movies** is enabled
2. Check naming format for syntax errors
3. Ensure tokens are spelled correctly (case-sensitive)
4. Save changes before triggering import

### Hardlinks failing

1. Verify source and destination are on same filesystem
2. Fall back to **Use Hardlinks: false** if needed
3. Check filesystem supports hardlinks (not FAT32)

## Related Documentation

- [Library Scanner](LIBRARY_SCANNER.md) - How media is discovered and indexed
- [Media Library Paths](MEDIA_LIBRARY_PATHS.md) - Organizing media by type
- [Plugin System](docs/PLUGIN_SYSTEM.md) - Download client plugins
