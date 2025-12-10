# Nimbus Indexer System - Implementation Summary

## Overview

This document summarizes the complete implementation of the generic indexer system and Sonarr-style interactive search feature for Nimbus.

## Part 1: Generic Indexer System

### What Was Built

A plugin-based indexer system that allows the community to build custom content indexers without modifying the core application.

### Key Components

1. **Plugin Interface** (`internal/plugins/types.go`)
   - Added `IsIndexer()` and `Search()` methods to plugin interface
   - Created standardized types: `IndexerSearchRequest`, `IndexerSearchResponse`, `IndexerRelease`

2. **Protocol Buffers** (`internal/plugins/proto/plugin.proto`)
   - Added gRPC service definitions for indexer methods
   - Generated Go code for RPC communication

3. **Plugin Manager** (`internal/plugins/manager.go`)
   - Auto-detects indexer capabilities on plugin load
   - Tracks which plugins provide indexer functionality
   - `ListIndexerPlugins()` method to get all indexers

4. **Indexer Service** (`internal/indexer/service.go`)
   - Unified interface for searching across all indexer plugins
   - Parallel HTTP-based searches for better performance
   - Deduplicates and aggregates results from multiple sources
   - Forwards authentication cookies for secure API access

5. **HTTP API Endpoints** (`internal/http/indexer_routes.go`)
   - `GET /api/indexers` - List available indexers
   - `GET /api/indexers/search` - General search
   - `GET /api/indexers/search/tv` - TV show search
   - `GET /api/indexers/search/movie` - Movie search

6. **Updated Plugins**
   - `usenet-indexer` plugin implements the indexer interface
   - Returns `true` from `IsIndexer()` method

### Architecture Decision: HTTP vs RPC

**Initial Approach**: Direct RPC calls to plugin `Search()` method

**Issue Discovered**: The RPC `Search()` method doesn't have access to the SDK (configuration, database) needed to perform searches.

**Solution Implemented**: HTTP-based internal API calls
- Indexer service makes HTTP requests to plugin endpoints
- Forwards authentication cookies for security
- Plugins have full SDK access via their HTTP handlers
- More reliable and mirrors external API usage

### Benefits

✅ **Extensible**: New indexers don't require core changes  
✅ **Unified API**: Single interface for all indexers  
✅ **Parallel Search**: Searches multiple indexers simultaneously  
✅ **Deduplication**: Automatic removal of duplicate releases  
✅ **Source Attribution**: Each release tagged with its indexer  

## Part 2: Interactive Search Feature

### What Was Built

A Sonarr-style manual search interface that allows users to browse and select specific releases for their media items.

### Key Components

1. **Search Route Handler** (`internal/http/search_routes.go`)
   - `GET /api/media/{id}/search` endpoint
   - Analyzes media metadata to build optimal search queries
   - Supports movies, TV episodes, seasons, and series
   - Extracts IMDb, TMDB, and TVDB IDs when available

2. **Smart Search Request Builder**
   - **Movies**: Uses title, IMDb ID, TMDB ID
   - **TV Episodes**: Uses series title, season/episode numbers, TVDB ID
   - **TV Seasons**: Uses series title, season number, TVDB ID
   - **TV Series**: Uses series title, TVDB ID

3. **Frontend API Client** (`frontend/src/lib/api/media.ts`)
   - `useInteractiveSearch()` hook
   - TypeScript types for `IndexerRelease` and `InteractiveSearchResponse`
   - Manual trigger only (disabled by default)

4. **Interactive Search Dialog** (`frontend/src/components/media/InteractiveSearchDialog.tsx`)
   - Auto-triggers search when opened
   - Real-time filtering by release title
   - Quality badge detection (4K, 1080p, 720p, 480p)
   - Codec detection (x265, x264, DTS, DD5.1)
   - Season/episode tags for TV content
   - File size formatting and relative age display
   - External links to indexer detail pages
   - Download/select buttons

5. **MediaDetailPage Integration** (`frontend/src/pages/MediaDetailPage.tsx`)
   - "Search Releases" button in action bar
   - Dialog state management
   - Release selection handler (placeholder for download integration)

### UI Features

- **Quality Badges**: Automatically detects and displays quality levels
- **Codec Information**: Shows video (x265, x264) and audio (DTS, DD5.1) codecs
- **Smart Filtering**: Real-time client-side search
- **Source Attribution**: Each release shows which indexer provided it
- **Age Display**: "2 days ago" style relative timestamps
- **Size Formatting**: Human-readable file sizes (GB, MB, etc.)
- **Result Count**: Shows filtered vs total results

### Technical Details

**Authentication Flow**:
1. User makes request with session cookie
2. Interactive search endpoint receives authenticated request
3. Cookies forwarded to indexer service
4. HTTP client adds cookies to internal plugin API calls
5. Plugins validate session and perform search

**Search Flow**:
1. User clicks "Search Releases" on media detail page
2. Dialog opens and auto-triggers search
3. Frontend calls `GET /api/media/{id}/search`
4. Backend fetches media metadata from database
5. Constructs appropriate search request based on media kind
6. Indexer service makes parallel HTTP calls to all indexer plugins
7. Results aggregated, deduplicated, and sorted by date
8. Frontend displays results in sortable table

## Files Created

### Backend
- `internal/indexer/service.go` - Indexer service with HTTP-based search
- `internal/http/indexer_routes.go` - Unified indexer API endpoints
- `internal/http/search_routes.go` - Interactive search endpoints
- `INDEXER_SYSTEM.md` - Generic indexer system documentation
- `INTERACTIVE_SEARCH.md` - Interactive search feature documentation

### Frontend
- `frontend/src/components/media/InteractiveSearchDialog.tsx` - Search dialog component

### Modified
- `internal/plugins/types.go` - Added indexer interface
- `internal/plugins/proto/plugin.proto` - Added indexer RPC definitions
- `internal/plugins/rpc.go` - Implemented indexer RPC methods
- `internal/plugins/manager.go` - Added indexer detection
- `internal/http/router.go` - Registered new routes
- `frontend/src/lib/api/media.ts` - Added search hooks
- `frontend/src/pages/MediaDetailPage.tsx` - Integrated search dialog
- `plugins/usenet-indexer/main.go` - Implemented indexer interface

## API Endpoints

### Unified Indexer API
```
GET  /api/indexers                 - List available indexers
GET  /api/indexers/search           - General search
GET  /api/indexers/search/tv        - TV show search
GET  /api/indexers/search/movie     - Movie search
```

### Interactive Search API
```
GET  /api/media/{id}/search         - Search for specific media item
```

### Plugin-Specific APIs (Still Available)
```
GET  /api/plugins/{plugin-id}/search
GET  /api/plugins/{plugin-id}/search/tv
GET  /api/plugins/{plugin-id}/search/movie
```

## Usage Examples

### List Available Indexers
```bash
curl -H "Cookie: session=..." \
  http://localhost:8080/api/indexers
```

### Search Across All Indexers
```bash
curl -H "Cookie: session=..." \
  "http://localhost:8080/api/indexers/search?q=Breaking+Bad&type=tv"
```

### Interactive Search for Media Item
```bash
curl -H "Cookie: session=..." \
  http://localhost:8080/api/media/108/search
```

## Future Enhancements

### Priority 1: Download Integration
- Connect to nzb-downloader plugin
- Auto-associate downloaded files with media items
- Track download status

### Priority 2: Quality Profiles
- User-configurable quality preferences
- Auto-select best matching releases
- Preferred/blocked release groups

### Priority 3: Advanced Features
- Release scoring and ranking
- Batch operations (download entire seasons)
- Search history and statistics
- Custom filters and sorting

## Dependencies Added

**Frontend**:
- `date-fns` - For relative time formatting ("2 days ago")

## Testing Checklist

✅ Backend builds successfully  
✅ Frontend builds successfully  
✅ Indexer plugins detected on startup  
✅ Interactive search button appears on media pages  
✅ Search dialog opens and auto-searches  
✅ Results display with quality badges and codec info  
✅ Filtering works in real-time  
✅ Authentication cookies forwarded correctly  
⏳ Download integration (pending - placeholder implemented)

## Conclusion

The generic indexer system and interactive search feature provide a powerful, extensible platform for content discovery in Nimbus. The HTTP-based architecture ensures reliability and proper SDK access, while the Sonarr-style UI gives users familiar and intuitive control over their content acquisition.

Community developers can now create custom indexer plugins for any content source (torrents, specialized indexers, regional sources, etc.) without modifying the core application. Users benefit from a unified search interface that works consistently across all indexers.
