# Nimbus Development Roadmap

This roadmap outlines planned features, plugins, and improvements for Nimbus. Items are organized by priority and category.

## Table of Contents

- [Phase 1: Core Functionality Enhancement](#phase-1-core-functionality-enhancement)
- [Phase 2: Torrent Ecosystem](#phase-2-torrent-ecosystem)
- [Phase 3: Sonarr/Radarr Feature Parity](#phase-3-sonarrradarr-feature-parity)
- [Phase 4: Books & E-Reading](#phase-4-books--e-reading)
- [Phase 5: Music & Audio](#phase-5-music--audio)
- [Phase 6: Advanced Features](#phase-6-advanced-features)
- [Phase 7: Community & Ecosystem](#phase-7-community--ecosystem)
- [Plugin Ideas](#plugin-ideas)

---

## Phase 1: Core Functionality Enhancement

**Priority**: High  
**Timeline**: Q1 2026

### 1.1 Download System Improvements

- [ ] **Automatic Post-Processing**
  - Automatic import of completed downloads to library
  - File verification (checksum validation)
  - Automatic extraction of archives (RAR, ZIP, 7z)
  - Sample file detection and removal
  - Failed download handling and cleanup

- [ ] **Advanced Queue Management**
  - Drag-and-drop priority reordering
  - Download categories and tagging
  - Bandwidth throttling per download
  - Scheduled download times (download only during off-peak hours)
  - Download history with statistics

- [ ] **Download Client Health Monitoring**
  - Connection health checks
  - Automatic reconnection on failure
  - Performance metrics (speed, success rate)
  - Alerting for failed downloads

### 1.2 Library Management Enhancements

- [ ] **Bulk Operations**
  - Bulk metadata refresh
  - Bulk file renaming
  - Bulk media item editing
  - Bulk quality upgrades
  - Export/import library data

- [ ] **Library Statistics Dashboard**
  - Total media counts by type
  - Storage usage breakdown
  - Recent additions timeline
  - Quality distribution charts
  - Missing/wanted items summary

- [ ] **Advanced Search & Filtering**
  - Full-text search across titles and metadata
  - Multi-field filtering (genre, year, quality, codec)
  - Saved search filters
  - Custom collections/playlists

- [ ] **File Management**
  - Duplicate file detection
  - Orphaned file cleanup
  - Storage path migration tools
  - Symbolic link support

### 1.3 Metadata Improvements

- [ ] **Multiple Metadata Providers**
  - TVDB integration
  - IMDB integration (via OMDb API)
  - MusicBrainz for music metadata
  - Goodreads/OpenLibrary for books
  - Metadata provider priority/fallback

- [ ] **Custom Metadata Fields**
  - User-defined custom fields
  - Tags and labels
  - Personal ratings and reviews
  - Watch status tracking

- [ ] **Artwork Management**
  - Multiple poster/backdrop selection
  - Custom artwork upload
  - Fanart.tv integration
  - Automatic artwork download and caching

### 1.4 User Experience

- [ ] **Mobile-Responsive UI**
  - Touch-optimized interface
  - Progressive Web App (PWA) support
  - Responsive grid layouts
  - Mobile-friendly navigation

- [ ] **Dark/Light Theme Refinements**
  - Custom theme colors
  - Accent color customization
  - Contrast adjustments

- [ ] **Notifications System**
  - In-app notifications
  - Email notifications (download complete, failed, etc.)
  - Push notifications (via web push API)
  - Webhook support for custom integrations

- [ ] **Activity Log**
  - User activity tracking
  - System event log
  - Download history
  - Import history

---

## Phase 2: Torrent Ecosystem

**Priority**: High  
**Timeline**: Q2 2026

### 2.1 Torrent Downloader Plugin

**Plugin ID**: `torrent-downloader`  
**Capabilities**: API, UI, Downloader

**Features**:
- [ ] Built-in BitTorrent client (using anacrolix/torrent library)
- [ ] Support for magnet links and .torrent files
- [ ] DHT, PEX, and tracker support
- [ ] Sequential downloading for streaming
- [ ] Bandwidth limiting (per-torrent and global)
- [ ] Seeding ratio management
- [ ] Port forwarding (UPnP/NAT-PMP)
- [ ] Connection encryption
- [ ] IP filtering/blocklists

**Configuration**:
- [ ] Download directory
- [ ] Max active downloads
- [ ] Upload/download speed limits
- [ ] Seeding limits (ratio, time)
- [ ] Port configuration
- [ ] Proxy support (SOCKS5, HTTP)

**UI Features**:
- [ ] Torrent queue management
- [ ] Peer list viewer
- [ ] Tracker management
- [ ] Speed graphs
- [ ] File priority selection

### 2.2 External Torrent Client Integrations

**qBittorrent Plugin** (`qbittorrent-client`)
- [ ] API integration with qBittorrent
- [ ] Category management
- [ ] Automatic labeling
- [ ] RSS support

**Transmission Plugin** (`transmission-client`)
- [ ] API integration with Transmission
- [ ] Remote client support
- [ ] Bandwidth management

**Deluge Plugin** (`deluge-client`)
- [ ] API integration with Deluge
- [ ] Label/plugin support

**rTorrent Plugin** (`rtorrent-client`)
- [ ] XMLRPC integration
- [ ] ruTorrent compatibility

### 2.3 Torrent Indexer Plugins

**Jackett Bridge Plugin** (`jackett-indexer`)
- [ ] Integration with Jackett
- [ ] Support for 500+ torrent trackers
- [ ] Unified search across all configured indexers
- [ ] Automatic indexer discovery

**Prowlarr Integration** (`prowlarr-indexer`)
- [ ] Native Prowlarr API integration
- [ ] Indexer synchronization
- [ ] Health monitoring

**Public Tracker Plugins**
- [ ] The Pirate Bay indexer
- [ ] 1337x indexer
- [ ] RARBG indexer (if available)
- [ ] EZTV indexer

**Private Tracker Templates**
- [ ] Generic tracker plugin template
- [ ] Cookie/token authentication
- [ ] Ratio tracking
- [ ] Freeleech detection

### 2.4 Torrent Features

- [ ] **Automatic Torrent Selection**
  - Prefer torrents with high seed count
  - Minimum seeder threshold
  - Trusted uploader detection
  - Prefer scene releases

- [ ] **Torrent Health Monitoring**
  - Stalled download detection
  - Dead torrent removal
  - Automatic re-searching for failed downloads

- [ ] **Seeding Management**
  - Automatic seeding based on ratio goals
  - Seeding time limits
  - Storage-based seeding limits
  - Selective seeding (by quality, rarity)

---

## Phase 3: Sonarr/Radarr Feature Parity

**Priority**: High  
**Timeline**: Q2-Q3 2026

### 3.1 Quality Profiles & Management

- [ ] **Quality Profile System**
  - Custom quality profiles (SD, 720p, 1080p, 4K, etc.)
  - Preferred quality ordering
  - Upgrade until quality threshold
  - Cutoff quality configuration

- [ ] **Quality Detection**
  - Automatic quality detection from release names
  - Resolution parsing
  - Source detection (WEB-DL, BluRay, HDTV, etc.)
  - Codec detection (H.264, H.265, AV1, etc.)

- [ ] **Quality Upgrades**
  - Automatic quality upgrade searching
  - Upgrade history tracking
  - Rollback to previous version
  - Upgrade blackout periods

### 3.2 Monitoring & Automation

- [ ] **Series/Movie Monitoring**
  - Monitor specific series/movies for new releases
  - Episode monitoring (all, future, missing, first season, latest season)
  - Season pack vs. individual episode preferences
  - Backlog search for missing episodes

- [ ] **Automatic Searching**
  - RSS feed monitoring for new releases
  - Scheduled search intervals
  - Search on add (search immediately when new item added)
  - Smart search (avoid searching too frequently)
  - Failed download auto-retry with different release

- [ ] **Calendar View**
  - Upcoming episodes/releases calendar
  - Past episodes view (missing items highlighted)
  - iCal feed export
  - Filter by monitored status

- [ ] **Wanted/Missing Management**
  - Wanted episodes list
  - Missing episodes report
  - Cutoff unmet report
  - Manual search for missing items

### 3.3 Release Management

- [ ] **Release Profiles**
  - Must contain keywords
  - Must not contain keywords
  - Preferred word scoring
  - Release tags (proper, repack, edition)
  - Language preferences
  - Release group preferences

- [ ] **Blocklist/Blacklist**
  - Automatic blocklist of failed releases
  - Manual blocklist additions
  - Blocklist reasons (quality, encoding issues, fake, etc.)
  - Temporary vs. permanent blocks
  - Block by release group
  - Block by indexer

- [ ] **Release Restrictions**
  - Size limits (min/max file size)
  - Age restrictions
  - Indexer priority
  - Required/ignored keywords
  - Scene release detection

### 3.4 Import & File Management

- [ ] **Naming Templates**
  - Standard/daily/anime episode formats
  - Movie naming formats
  - Season folder formats
  - Tokens: {Series Title}, {Episode Title}, {Quality}, {Release Group}, etc.
  - Example preview in UI

- [ ] **Advanced Renaming**
  - Replace illegal characters
  - Colon replacement options
  - Folder structure customization
  - Multi-episode naming

- [ ] **Import Decisions**
  - Automatic import of completed downloads
  - Import existing files from folders
  - Interactive import (choose import options)
  - Import failed handling

- [ ] **File Management**
  - Hardlink vs. copy configuration
  - Recycle bin for deleted files
  - Cleanup empty folders
  - Set file permissions
  - Create series folders automatically

### 3.5 Lists & Automation

- [ ] **List Integration**
  - IMDB lists
  - Trakt.tv lists
  - TMDb lists
  - Custom lists
  - Automatic adding from lists

- [ ] **List Sync**
  - Periodic sync interval
  - Add/remove based on list changes
  - Monitor new additions automatically
  - Exclude certain items from lists

### 3.6 Custom Scripts & Hooks

- [ ] **Event Scripts**
  - On download (before/after)
  - On import (before/after)
  - On upgrade
  - On rename
  - On delete

- [ ] **Script Types**
  - Bash/shell scripts
  - Python scripts
  - Custom executables
  - Webhook calls

### 3.7 Indexer Management

- [ ] **Indexer Configuration**
  - Per-indexer settings (API key, URL, categories)
  - Enable/disable indexers
  - Indexer priority
  - Test indexer connection

- [ ] **Indexer Health**
  - Success/failure rate tracking
  - Response time monitoring
  - Automatic disable on repeated failures
  - Indexer capability detection

- [ ] **Search Throttling**
  - Rate limiting per indexer
  - API call tracking
  - Backoff on rate limit errors

---

## Phase 4: Books & E-Reading

**Priority**: Medium  
**Timeline**: Q3-Q4 2026

### 4.1 Book Library Management

- [ ] **Book Metadata**
  - Author, series, publisher information
  - ISBN tracking
  - Publication date
  - Book covers and descriptions
  - Genres and tags
  - Reading status (want to read, reading, finished)

- [ ] **Book Organization**
  - Author → Series → Books hierarchy
  - Standalone books
  - Anthologies and collections
  - Multiple editions tracking

### 4.2 Anna's Archive Integration

**Plugin ID**: `annas-archive-indexer`  
**Capabilities**: API, Indexer

**Features**:
- [ ] Search Anna's Archive database
- [ ] Multiple format support (EPUB, PDF, MOBI, AZW3)
- [ ] Language filtering
- [ ] Quality/source preference (LibGen, Z-Library, etc.)
- [ ] Metadata enrichment from multiple sources
- [ ] Direct download links

**Advanced Search**:
- [ ] Search by ISBN
- [ ] Search by author
- [ ] Search by series
- [ ] Filter by language, year, format
- [ ] Full-text search in descriptions

### 4.3 Book Downloaders

**HTTP Downloader Plugin** (`http-downloader`)
- [ ] Direct HTTP/HTTPS downloads
- [ ] Resume support
- [ ] Parallel chunk downloading
- [ ] Retry on failure
- [ ] Support for various authentication methods

**LibGen Downloader** (`libgen-downloader`)
- [ ] LibGen-specific download handling
- [ ] Mirror selection and fallback
- [ ] Format conversion queue

### 4.4 E-Reader Integrations

**Send to Kindle Plugin** (`send-to-kindle`)
- [ ] Email delivery to Kindle
- [ ] Format conversion (EPUB → MOBI/AZW3)
- [ ] Send to specific Kindle device
- [ ] Batch sending
- [ ] Send with metadata preservation

**Calibre Integration** (`calibre-integration`)
- [ ] Import books to Calibre library
- [ ] Use Calibre for format conversion
- [ ] Sync reading progress
- [ ] Calibre Web integration
- [ ] OPDS catalog exposure

**E-Reader Device Sync**
- [ ] Kobo integration
- [ ] PocketBook integration
- [ ] Generic USB device transfer
- [ ] WiFi transfer protocols

### 4.5 Book Reading Features

- [ ] **Web-Based Reader**
  - EPUB reader in browser
  - PDF viewer
  - Reading progress tracking
  - Bookmarks and highlights
  - Adjustable font size and styles

- [ ] **Reading Lists**
  - Custom reading lists
  - Currently reading
  - Want to read
  - Finished books with ratings

- [ ] **Goodreads Integration** (`goodreads-sync`)
  - Sync reading status
  - Import shelves
  - Sync ratings and reviews
  - Social features

### 4.6 Audiobook Support

- [ ] **Audiobook Library**
  - Audiobook-specific metadata (narrator, duration)
  - Chapter markers
  - Multi-part audiobook handling

- [ ] **Audiobook Indexers**
  - AudiobookBay integration
  - MyAnonamouse integration
  - MAM freeleech monitoring

- [ ] **Audiobook Player**
  - Web-based audiobook player
  - Playback speed control
  - Sleep timer
  - Progress synchronization

---

## Phase 5: Music & Audio

**Priority**: Medium  
**Timeline**: Q4 2026

### 5.1 Music Library Enhancements

- [ ] **Advanced Music Metadata**
  - MusicBrainz integration
  - Discogs integration
  - LastFM scrobbling
  - Lyrics integration
  - Album reviews and ratings

- [ ] **Music Organization**
  - Genre classification
  - Mood/style tags
  - Smart playlists
  - Artist similarity

### 5.2 Music Indexers

**Redacted Integration** (`redacted-indexer`)
- [ ] API integration with Redacted (RED)
- [ ] Format/quality filtering
- [ ] Torrent group handling
- [ ] Freeleech detection

**Orpheus Network** (`orpheus-indexer`)
- [ ] Orpheus API integration
- [ ] Similar features to Redacted plugin

**Soulseek Plugin** (`soulseek-client`)
- [ ] Soulseek P2P integration
- [ ] Search and download
- [ ] Share library option

### 5.3 Music Features

- [ ] **Audio Quality Management**
  - Lossless vs. lossy preference
  - Bitrate preferences
  - Format preferences (FLAC, MP3, OGG, etc.)
  - Automatic transcoding

- [ ] **Music Player**
  - Web-based audio player
  - Playlist management
  - Queue system
  - Gapless playback

- [ ] **Subsonic API**
  - Subsonic-compatible API endpoint
  - Support for Subsonic clients (DSub, Ultrasonic, etc.)

---

## Phase 6: Advanced Features

**Priority**: Medium-Low  
**Timeline**: 2027+

### 6.1 Media Server Integration

**Plex Plugin** (`plex-integration`)
- [ ] Library sync
- [ ] Automatic library updates on import
- [ ] Watch status sync
- [ ] Plex authentication

**Jellyfin Plugin** (`jellyfin-integration`)
- [ ] Library sync
- [ ] Webhook integration
- [ ] Watch status sync
- [ ] Jellyfin API integration

**Emby Plugin** (`emby-integration`)
- [ ] Similar to Jellyfin integration

### 6.2 Request System

- [ ] **Media Requests**
  - User request system (like Overseerr/Ombi)
  - Approval workflow
  - Request quotas
  - Automatic searching on approval
  - Request notifications

- [ ] **Request Voting**
  - Users can vote on requests
  - Priority based on votes

### 6.3 Multi-User Features

- [ ] **User Permissions**
  - Granular permission system
  - Library access control
  - Feature restrictions
  - Request limits per user

- [ ] **Shared Libraries**
  - Share library with specific users
  - Read-only vs. read-write access

- [ ] **User Statistics**
  - Per-user download statistics
  - Storage usage per user
  - Activity tracking

### 6.4 Advanced Automation

- [ ] **Automatic Tagging**
  - Auto-tag based on metadata
  - Custom tagging rules
  - Tag-based organization

- [ ] **Content Rating Filters**
  - Age rating filters
  - Content warning detection
  - Parental controls

- [ ] **Duplicate Detection**
  - Find duplicate media items
  - Merge duplicates
  - Prefer specific versions

- [ ] **Related Content Discovery**
  - "Customers who watched X also watched Y"
  - Actor/director filmography
  - Similar items recommendations

### 6.5 Backup & Sync

- [ ] **Database Backup**
  - Automated database backups
  - Configuration export/import
  - Disaster recovery tools

- [ ] **Cloud Sync**
  - Sync library to cloud storage
  - Remote library access
  - Multi-instance sync

### 6.6 Performance & Scalability

- [ ] **Caching Layer**
  - Redis integration for caching
  - Image/thumbnail caching
  - API response caching

- [ ] **Distributed Architecture**
  - Multiple backend instances
  - Load balancing
  - Horizontal scaling

- [ ] **CDN Integration**
  - CDN support for static assets
  - Image optimization

---

## Phase 7: Community & Ecosystem

**Priority**: Low  
**Timeline**: Ongoing

### 7.1 Documentation

- [ ] **User Documentation**
  - Getting started guide
  - Feature tutorials
  - FAQ
  - Troubleshooting guide

- [ ] **Developer Documentation**
  - API documentation (OpenAPI/Swagger)
  - Plugin development guide
  - Architecture overview
  - Contributing guidelines

- [ ] **Video Tutorials**
  - Installation walkthrough
  - Feature demonstrations
  - Plugin development tutorials

### 7.2 Community Features

- [ ] **Plugin Marketplace**
  - Community plugin repository
  - Plugin ratings and reviews
  - One-click plugin installation
  - Plugin update notifications

- [ ] **Themes & Customization**
  - Community themes
  - Custom CSS support
  - Layout customization

- [ ] **Translation/Localization**
  - Multi-language support
  - Community translations
  - i18n infrastructure

### 7.3 Ecosystem Integrations

- [ ] **Discord Bot**
  - Media search from Discord
  - Request management
  - Notifications in Discord

- [ ] **Telegram Bot**
  - Similar to Discord bot

- [ ] **Home Assistant Integration**
  - Home Assistant addon
  - Entity exposure (sensors, switches)
  - Automation triggers

- [ ] **IFTTT/Zapier Integration**
  - Webhook support
  - Trigger actions on events

---

## Plugin Ideas

### Indexers

- [ ] **IMDb Watchlist Sync** - Automatically monitor and download items from IMDb watchlist
- [ ] **Letterboxd Integration** - Sync watchlist and ratings from Letterboxd
- [ ] **Netflix/Hulu Expiring** - Track expiring content on streaming services
- [ ] **Trakt.tv Sync** - Full Trakt integration (watchlist, history, scrobbling)

### Metadata Providers

- [ ] **Fanart.tv Plugin** - High-quality artwork provider
- [ ] **TVDB Plugin** - TV show metadata
- [ ] **OMDb Plugin** - IMDb data via OMDb API
- [ ] **AniDB Plugin** - Anime metadata
- [ ] **OpenSubtitles Plugin** - Automatic subtitle downloading

### Downloaders

- [ ] **Direct Download (Debrid)** - Real-Debrid, AllDebrid, Premiumize integration
- [ ] **MEGA Downloader** - MEGA.nz downloads
- [ ] **Google Drive Downloader** - Download from shared Google Drive links
- [ ] **YouTube-DL Integration** - Download from YouTube and other video sites

### Utilities

- [ ] **Tautulli Integration** - Plex analytics and monitoring
- [ ] **MediaInfo Plugin** - Extract detailed media file information
- [ ] **Subtitle Manager** - Automatic subtitle search and download
- [ ] **Trailer Downloader** - Download movie/TV show trailers
- [ ] **Custom Scripts Runner** - Execute custom scripts on events
- [ ] **Notification Aggregator** - Send notifications to multiple services (Discord, Telegram, Email, Pushover, etc.)

### Content Discovery

- [ ] **IMDb Top 250 Monitor** - Track and download top-rated content
- [ ] **Rotten Tomatoes Integration** - Filter by RT scores
- [ ] **Trending Content** - Discover trending movies/TV shows
- [ ] **Awards Tracker** - Track Oscar/Emmy nominees and winners

### Specialized

- [ ] **Anime Downloader** - Nyaa.si integration, AniDB metadata
- [ ] **Sports Recorder** - Track and download sports events
- [ ] **Podcast Manager** - Subscribe to and download podcasts
- [ ] **Comic Book Manager** - Comic/manga library with Calibre-like features
- [ ] **Game ROM Manager** - Video game ROM library (for emulation)
- [ ] **Software Library** - Track and organize software/ISO images

---

## Priority Legend

- **High**: Core features needed for basic usability and Sonarr/Radarr parity
- **Medium**: Nice-to-have features that enhance user experience
- **Low**: Future enhancements and community-driven features

## Contributing

We welcome contributions! If you'd like to work on any of these features:

1. Check the [GitHub Issues](https://github.com/blakestevenson/nimbus/issues) for existing discussions
2. Open a new issue to discuss your approach
3. Fork the repository and create a feature branch
4. Submit a pull request with your changes

For plugin development, see the [Plugin Development Guide](PLUGIN_QUICKSTART.md).

---

## Feedback

Have ideas for features not listed here? Open an issue on GitHub with the `feature-request` label!
