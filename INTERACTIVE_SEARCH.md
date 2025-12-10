# Interactive Search Feature

## Overview

The Interactive Search feature provides a Sonarr-style manual search interface for media items. Users can manually search for and review available releases across all configured indexer plugins, then select specific releases for download.

## Features

✅ **One-Click Search**: Search button on every media detail page  
✅ **Multi-Indexer Support**: Searches across all enabled indexer plugins simultaneously  
✅ **Rich Release Information**: Shows quality, codecs, file size, age, and source indexer  
✅ **Smart Filtering**: Real-time client-side filtering of search results  
✅ **Quality Badges**: Automatically detects and displays quality (4K, 1080p, 720p, etc.)  
✅ **Codec Detection**: Shows video and audio codec information (x265, DTS, DD5.1, etc.)  
✅ **Episode/Season Tags**: Displays season and episode numbers for TV content  
✅ **Source Attribution**: Each release shows which indexer provided it  
✅ **External Links**: Direct links to indexer detail pages  
✅ **Download Integration**: Ready for integration with download clients

## Architecture

### Backend Components

#### 1. Search Routes (`internal/http/search_routes.go`)
- **Endpoint**: `GET /api/media/{id}/search`
- **Authentication**: Required (session-based)
- **Functionality**: 
  - Fetches media item metadata from database
  - Builds appropriate search request based on media kind
  - Delegates to indexer service for multi-indexer search
  - Returns aggregated results with metadata

#### 2. Search Request Builder
Intelligently constructs search queries based on media type:

**Movies**:
- Query: Media title
- Type: "movie"
- Metadata: IMDb ID, TMDB ID (if available)

**TV Episodes**:
- Query: Series title
- Type: "tv"
- Season & Episode numbers
- Metadata: TVDB ID (if available)

**TV Seasons**:
- Query: Series title
- Type: "tv"
- Season number
- Metadata: TVDB ID (if available)

**TV Series**:
- Query: Series title
- Type: "tv"
- Metadata: TVDB ID (if available)

### Frontend Components

#### 1. API Client (`frontend/src/lib/api/media.ts`)

**New Types**:
```typescript
interface IndexerRelease {
  guid: string;
  title: string;
  link?: string;
  publish_date: string;
  size: number;
  download_url: string;
  attributes?: Record<string, string>;
  indexer_id: string;
  indexer_name: string;
}

interface InteractiveSearchResponse {
  media_id: number;
  releases: IndexerRelease[];
  total: number;
  sources: string[];
  metadata: { kind, title, year };
}
```

**New Hook**:
```typescript
useInteractiveSearch(mediaId: string | number)
```
- Disabled by default (manual trigger only)
- 5-minute cache time
- Returns search results and loading/error states

#### 2. InteractiveSearchDialog Component

**Location**: `frontend/src/components/media/InteractiveSearchDialog.tsx`

**Props**:
- `mediaId`: The media item to search for
- `mediaTitle`: Display title
- `mediaKind`: Media type (movie, tv_episode, etc.)
- `open`: Dialog visibility state
- `onOpenChange`: Callback for dialog state changes
- `onSelectRelease`: Callback when user selects a release

**Features**:
- Auto-triggers search when opened
- Real-time filtering by release title
- Sortable table of results
- Quality and codec detection
- File size formatting
- Relative age display ("2 days ago")
- External link buttons
- Download/Select buttons

#### 3. MediaDetailPage Integration

**Changes**:
- Added "Search Releases" button to action bar
- Integrated InteractiveSearchDialog component
- Added release selection handler (placeholder for download integration)

## Usage

### For End Users

1. **Navigate to Media Detail Page**
   - Click on any movie, TV series, season, or episode

2. **Click "Search Releases"**
   - Button appears in the top action bar
   - Dialog opens and automatically searches all configured indexers

3. **Review Results**
   - Browse available releases in table format
   - Filter by typing in the search box
   - See quality badges (4K, 1080p, 720p)
   - See codec info (x265, DTS, etc.)
   - Check file sizes and age
   - View which indexer provided each release

4. **Select a Release**
   - Click external link icon to view details on indexer site
   - Click "Download" button to select release
   - (Download integration coming soon)

### API Usage

**Interactive Search Endpoint**:
```bash
# Search for releases for a specific media item
curl -H "Cookie: session=..." \
  http://localhost:8080/api/media/123/search
```

**Response**:
```json
{
  "media_id": 123,
  "releases": [
    {
      "guid": "unique-id",
      "title": "Movie.Title.2024.1080p.BluRay.x264-GROUP",
      "size": 1073741824,
      "download_url": "http://indexer.com/download/...",
      "publish_date": "2025-12-09T12:00:00Z",
      "attributes": {
        "season": "1",
        "episode": "1"
      },
      "indexer_id": "usenet-indexer",
      "indexer_name": "Usenet Indexer"
    }
  ],
  "total": 10,
  "sources": ["usenet-indexer"],
  "metadata": {
    "kind": "movie",
    "title": "Movie Title",
    "year": 2024
  }
}
```

## Smart Features

### Quality Detection
The UI automatically detects and displays quality badges from release titles:
- **4K/2160p** → 4K badge
- **1080p** → 1080p badge
- **720p** → 720p badge
- **480p** → 480p badge

### Codec Detection
Automatically identifies video and audio codecs:
- **Video**: x265/HEVC, x264/AVC
- **Audio**: DTS, DD5.1/AC3

### Season/Episode Tags
For TV content, automatically extracts and displays:
- Season numbers (S01)
- Episode numbers (E05)
- Combined tags (S01E05)

## Integration Points

### Current State
The interactive search feature is **fully functional** for:
- ✅ Searching across all indexer plugins
- ✅ Displaying results with rich metadata
- ✅ Filtering and sorting results
- ✅ Viewing release details

### Pending Integrations
To complete the workflow, integrate with:

1. **Download Clients**
   - NZB Downloader plugin (already available)
   - Future torrent clients
   - Direct download handlers

2. **Integration Example**:
```typescript
const handleSelectRelease = async (release: IndexerRelease) => {
  // Send to nzb-downloader plugin
  await fetch('/api/plugins/nzb-downloader/download', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({
      url: release.download_url,
      title: release.title,
      media_id: mediaId,
    }),
  });
};
```

3. **Quality Profiles** (Future Enhancement)
   - Allow users to set preferred qualities
   - Auto-select best matching release
   - Automatic download on match

4. **Release Scoring** (Future Enhancement)
   - Score releases based on quality, size, seeders, etc.
   - Show score/ranking in UI
   - Sort by score by default

## Technical Details

### Search Request Construction

The backend analyzes the media item's metadata to construct optimal search requests:

```go
// Example for TV Episode
{
  Query: "Breaking Bad",      // From media.title
  Type: "tv",                 // From media.kind
  Season: 1,                  // From metadata.season
  Episode: 5,                 // From metadata.episode
  TVDBID: "81189",           // From metadata.tvdb_id
  Limit: 100,
}
```

### Multi-Indexer Parallel Search

The indexer service searches all indexers simultaneously:
1. Get list of all indexer plugins
2. Launch goroutines for each indexer
3. Collect results via channels
4. Deduplicate by GUID
5. Sort by publish date (newest first)
6. Apply result limits

### Error Handling

- If all indexers fail: Returns error to user
- If some indexers fail: Returns successful results with warning
- Network timeouts: 30-second timeout per indexer
- Empty results: Displays "No releases found" message

## UI/UX Decisions

### Why Manual Search?
- Gives users full control over what gets downloaded
- Allows quality/codec preference selection
- Enables review of available options before committing
- Useful for finding specific releases or qualities

### Why Table View?
- Displays maximum information density
- Easy to scan and compare releases
- Familiar to users of Sonarr/Radarr
- Sortable by any column (future enhancement)

### Why Auto-Trigger Search?
- Reduces clicks (no separate "Search" button in dialog)
- Immediate feedback
- Better perceived performance
- Matches Sonarr/Radarr behavior

## Future Enhancements

### Priority 1 - Download Integration
- Connect to nzb-downloader plugin
- Send selected releases to download queue
- Show download status/progress
- Auto-associate downloaded files with media item

### Priority 2 - Advanced Filtering
- Filter by quality (checkbox list)
- Filter by codec
- Filter by size range
- Filter by indexer
- Save filter preferences

### Priority 3 - Smart Selection
- "Best Match" auto-selection
- Quality preference profiles
- Size preference (e.g., prefer smaller files)
- Preferred/blocked groups

### Priority 4 - Batch Operations
- Search multiple episodes/seasons at once
- Bulk download selections
- "Download Season" action
- "Download Series" action

### Priority 5 - Statistics & History
- Track which releases were downloaded
- Success/failure rates per indexer
- Average response times
- Popular quality distributions

## Compatibility

### Browser Support
- Modern browsers with ES2015+ support
- Tested on Chrome, Firefox, Safari, Edge
- Mobile responsive design

### Indexer Support
- Works with any plugin implementing the indexer interface
- Currently supported: Usenet indexers (via usenet-indexer plugin)
- Future: Torrent indexers, specialized content sources

### Media Types
- ✅ Movies
- ✅ TV Episodes
- ✅ TV Seasons
- ✅ TV Series
- ⚠️ Books (basic support, may need enhancement)
- ⚠️ Music (basic support, may need enhancement)

## Troubleshooting

### No Results Found
- Check that indexer plugins are enabled and configured
- Verify indexer API keys are valid
- Check indexer category settings
- Try manual search on indexer website to verify content exists

### Search is Slow
- Check indexer response times (30s timeout per indexer)
- Consider reducing number of concurrent indexers
- Check network connectivity to indexer APIs

### Releases Missing Metadata
- Some indexers provide limited metadata
- Quality/codec detection relies on release titles
- Consider using indexers with richer metadata

### Download Button Does Nothing
- Download integration is not yet implemented
- Currently shows placeholder alert
- Will be connected to download client plugins in future update

## Conclusion

The Interactive Search feature brings powerful Sonarr-style manual search capabilities to Nimbus, allowing users to manually review and select specific releases for their media items. Combined with the generic indexer system, this creates a flexible, extensible platform for content discovery and acquisition.
