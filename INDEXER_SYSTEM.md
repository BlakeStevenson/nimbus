# Generic Indexer System

## Overview

The Nimbus media suite now includes a generic indexer system that allows plugins to provide content indexing capabilities. This enables the community to build custom indexer plugins without requiring changes to the core application.

## Architecture

### Plugin Interface

Plugins can now implement the `IndexerPlugin` interface to provide indexer functionality:

```go
type MediaSuitePlugin interface {
    // ... existing methods ...
    
    // Indexer facet - OPTIONAL
    IsIndexer(ctx context.Context) (bool, error)
    Search(ctx context.Context, req *IndexerSearchRequest) (*IndexerSearchResponse, error)
}
```

### Key Components

1. **Plugin Types** (`internal/plugins/types.go`)
   - `IndexerSearchRequest`: Unified search request format
   - `IndexerSearchResponse`: Unified search response format
   - `IndexerRelease`: Standardized release information

2. **Protocol Buffers** (`internal/plugins/proto/plugin.proto`)
   - gRPC service definitions for indexer methods
   - Message types for search requests and responses

3. **Plugin Manager** (`internal/plugins/manager.go`)
   - Automatically detects indexer plugins on load
   - Tracks which plugins provide indexer functionality
   - Provides `ListIndexerPlugins()` method

4. **Indexer Service** (`internal/indexer/service.go`)
   - Unified interface for searching across all indexer plugins
   - Handles parallel searches across multiple indexers
   - Deduplicates results and aggregates responses

5. **HTTP API** (`internal/http/indexer_routes.go`)
   - Unified API endpoints for indexer functionality
   - Routes:
     - `GET /api/indexers` - List available indexers
     - `GET /api/indexers/search` - General search
     - `GET /api/indexers/search/tv` - TV show search
     - `GET /api/indexers/search/movie` - Movie search

## Usage

### For Plugin Developers

To create an indexer plugin:

1. Implement the `MediaSuitePlugin` interface
2. Return `true` from `IsIndexer(ctx)` method
3. Implement the `Search(ctx, req)` method to perform searches
4. Return results in the standardized `IndexerSearchResponse` format

Example:

```go
type MyIndexerPlugin struct{}

func (p *MyIndexerPlugin) IsIndexer(ctx context.Context) (bool, error) {
    return true, nil
}

func (p *MyIndexerPlugin) Search(ctx context.Context, req *plugins.IndexerSearchRequest) (*plugins.IndexerSearchResponse, error) {
    // Perform search based on req.Type ("general", "tv", "movie")
    // Return standardized releases
    return &plugins.IndexerSearchResponse{
        Releases: []plugins.IndexerRelease{
            {
                GUID:        "unique-id",
                Title:       "Content Title",
                DownloadURL: "http://example.com/download",
                PublishDate: time.Now(),
                Size:        1024 * 1024 * 100, // 100 MB
                // ... other fields
            },
        },
        Total:       1,
        IndexerID:   "my-indexer",
        IndexerName: "My Indexer",
    }, nil
}
```

### For Application Users

The unified indexer system provides a single API for searching across all installed indexer plugins:

```bash
# List available indexers
curl http://localhost:8080/api/indexers

# Search for content
curl "http://localhost:8080/api/indexers/search?q=breaking+bad"

# Search for TV shows
curl "http://localhost:8080/api/indexers/search/tv?q=breaking+bad&season=1&episode=1"

# Search for movies
curl "http://localhost:8080/api/indexers/search/movie?q=inception&imdbid=tt1375666"
```

## Search Request Parameters

### Common Parameters
- `q` - Search query string
- `categories` - Comma-separated category IDs (indexer-specific)
- `limit` - Maximum number of results (default: 100)
- `offset` - Pagination offset

### TV-Specific Parameters
- `tvdbid` - TheTVDB ID
- `tvrageid` - TVRage ID
- `season` - Season number
- `episode` - Episode number

### Movie-Specific Parameters
- `imdbid` - IMDb ID (e.g., tt1234567)
- `tmdbid` - The Movie Database ID

## Response Format

All indexer search endpoints return a unified response:

```json
{
  "releases": [
    {
      "guid": "unique-release-id",
      "title": "Content Title S01E01",
      "link": "http://indexer.com/details/123",
      "publish_date": "2025-12-09T12:00:00Z",
      "size": 1073741824,
      "download_url": "http://indexer.com/download/123",
      "category": "5030",
      "attributes": {
        "season": "1",
        "episode": "1",
        "tvdbid": "12345"
      },
      "indexer_id": "usenet-indexer",
      "indexer_name": "Usenet Indexer"
    }
  ],
  "total": 1,
  "sources": ["usenet-indexer", "another-indexer"]
}
```

## Migration from Plugin-Specific APIs

The existing plugin-specific APIs (e.g., `/api/plugins/usenet-indexer/search`) continue to work for backward compatibility. However, new integrations should use the unified indexer API endpoints for a consistent experience across all indexer plugins.

### Benefits of the Unified API

1. **Aggregated Results**: Search across multiple indexers simultaneously
2. **Deduplication**: Automatically removes duplicate releases
3. **Consistent Format**: All indexers return data in the same structure
4. **Simplified Integration**: One API to support all current and future indexers
5. **No Core Changes**: New indexers can be added without modifying core application code

## Examples of Potential Indexer Plugins

The community can now build indexer plugins for:

- **Torrent Indexers**: Jackett, Prowlarr integration
- **Alternative Usenet Indexers**: NZBHydra2, different Newznab providers
- **Specialized Content**: Audiobook indexers, comic/manga indexers
- **Regional Indexers**: Country or language-specific content sources
- **Custom Sources**: Proprietary or internal content management systems

Each plugin implements the same interface, making them all work seamlessly with the core application and any clients that use the unified API.

## Technical Details

### Plugin Detection

When a plugin is loaded, the plugin manager calls `IsIndexer(ctx)` on each plugin. If it returns `true`, the plugin is registered as an indexer and made available through the unified indexer service.

### Parallel Search

The indexer service searches all available indexer plugins in parallel using goroutines, collecting results via channels. This ensures fast response times even when querying multiple indexers.

### Result Aggregation

Results from all indexers are:
1. Collected and merged
2. Deduplicated by GUID
3. Sorted by publish date (newest first)
4. Limited to the requested number of results

### Error Handling

If some indexers fail while others succeed, the service returns the successful results along with a list of sources that were queried. If all indexers fail, an error is returned.

## Future Enhancements

Possible future improvements to the indexer system:

1. **Direct RPC Search**: Currently, the Search() method requires SDK access for configuration. Future versions could pass SDK context through RPC.
2. **Priority/Ranking**: Allow users to configure indexer priority for result ordering
3. **Caching**: Add response caching to reduce duplicate searches
4. **Statistics**: Track indexer performance, success rates, and response times
5. **Capabilities Discovery**: Allow indexers to advertise their supported features
6. **Filtering**: Add advanced filtering options for search results

## Conclusion

The generic indexer system makes Nimbus extensible and community-friendly. Developers can create custom indexers without any changes to the core application, and users benefit from a unified, consistent API regardless of which indexers they choose to use.
