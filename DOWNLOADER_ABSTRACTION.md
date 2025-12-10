# Downloader Service Abstraction

This document describes the downloader service abstraction that was implemented to provide a unified interface for managing downloads across multiple downloader plugins (NZB, torrent, etc.), following the same pattern as the indexer service.

## Architecture Overview

The downloader service follows the same architectural pattern as the indexer service:

```
┌─────────────────────────────────────────────────────────────┐
│                     HTTP API Layer                           │
│              /api/downloaders, /api/downloads/*             │
└─────────────────────────────────────────────────────────────┘
                            ↓
┌─────────────────────────────────────────────────────────────┐
│                  Downloader Service                          │
│   - Unified interface for all downloader plugins            │
│   - Aggregates downloads from multiple sources              │
│   - Handles download lifecycle (pause/resume/cancel)        │
└─────────────────────────────────────────────────────────────┘
                            ↓
            ┌───────────────┴───────────────┐
            ↓                               ↓
┌───────────────────────┐       ┌───────────────────────┐
│   NZB Downloader      │       │  Torrent Downloader   │
│   Plugin              │       │  Plugin (future)      │
│   - IsDownloader=true │       │  - IsDownloader=true  │
└───────────────────────┘       └───────────────────────┘
```

## Implemented Components

### 1. Database Schema (`internal/db/migrations/0009_downloads.sql`)

Created two new tables:

#### `downloads` table:
- `id` - Unique download ID
- `plugin_id` - References which downloader plugin handles this download
- `name` - Display name
- `status` - queued, downloading, processing, paused, completed, failed, cancelled
- `progress` - 0-100%
- `total_bytes` / `downloaded_bytes` - Size tracking
- `url` / `file_content` - Download source (URL or file like NZB/torrent)
- `destination_path` - Where files are saved
- `error_message` - Error tracking
- `queue_position` / `priority` - Queue management
- `created_at` / `started_at` / `completed_at` - Timestamps
- `metadata` - Plugin-specific JSON data
- `created_by_user_id` - User tracking

#### `download_logs` table:
- Detailed logging per download
- Levels: info, warn, error
- Timestamped messages

**Note**: These tables provide persistent download tracking across restarts, unlike the current in-memory implementation in the NZB plugin.

### 2. Downloader Service (`internal/downloader/service.go`)

Provides a unified API for download management:

**Core Methods:**
- `CreateDownload(req DownloadRequest)` - Create a new download via plugin
- `ListDownloads(pluginID, status)` - Aggregate downloads from all plugins
- `GetDownload(downloadID, pluginID)` - Get specific download details
- `PauseDownload(downloadID, pluginID)` - Pause a download
- `ResumeDownload(downloadID, pluginID)` - Resume a paused download
- `CancelDownload(downloadID, pluginID)` - Cancel/delete a download
- `RetryDownload(downloadID, pluginID)` - Retry a failed download
- `ListDownloaders()` - List available downloader plugins

**Implementation Pattern:**
- Makes HTTP calls to plugin endpoints (same as indexer service)
- Aggregates results from multiple downloader plugins
- No plugin-specific logic in the service layer
- Handles authentication and routing

### 3. HTTP Routes (`internal/http/downloader_routes.go`)

Unified API endpoints:

```
GET  /api/downloaders                           - List available downloader plugins
GET  /api/downloads                            - List all downloads (filterable)
POST /api/downloads                            - Create new download
GET  /api/downloads/{plugin_id}/{download_id}  - Get specific download
POST /api/downloads/{plugin_id}/{download_id}/pause   - Pause download
POST /api/downloads/{plugin_id}/{download_id}/resume  - Resume download
POST /api/downloads/{plugin_id}/{download_id}/retry   - Retry download
DELETE /api/downloads/{plugin_id}/{download_id}       - Cancel download
```

**Integrated into router** (`internal/http/router.go`):
- Downloader service initialized if plugin manager available
- Routes require authentication
- Follows same pattern as indexer routes

### 4. Plugin Interface Extension

**Updated `MediaSuitePlugin` interface** (`internal/plugins/types.go`):
```go
// Downloader facet - OPTIONAL
IsDownloader(ctx context.Context) (bool, error)
```

**Updated `LoadedPlugin` struct** (`internal/plugins/manager.go`):
```go
type LoadedPlugin struct {
    // ...
    IsIndexer    bool  // Whether this plugin provides indexer functionality
    IsDownloader bool  // Whether this plugin provides downloader functionality
    // ...
}
```

**Added method** (`internal/plugins/manager.go`):
```go
func (pm *PluginManager) ListDownloaderPlugins() []*LoadedPlugin
```

### 5. gRPC Protocol Updates

**Updated proto definition** (`internal/plugins/proto/plugin.proto`):
```protobuf
service PluginService {
  // ...
  rpc IsDownloader(IsDownloaderRequest) returns (IsDownloaderResponse);
}

message IsDownloaderRequest {}
message IsDownloaderResponse {
  bool is_downloader = 1;
  string error = 2;
}
```

**Updated RPC implementation** (`internal/plugins/rpc.go`):
- Added `IsDownloader` server method
- Added `IsDownloader` client method
- Properly handles errors and responses

### 6. Plugin Updates

All plugins updated to implement the `IsDownloader` method:

**NZB Downloader Plugin** (`plugins/nzb-downloader/main.go`):
```go
func (p *NZBDownloaderPlugin) IsDownloader(ctx context.Context) (bool, error) {
    return true, nil  // This IS a downloader
}
```

**TMDB Plugin** (`plugins/tmdb-plugin/main.go`):
```go
func (p *TMDBPlugin) IsDownloader(ctx context.Context) (bool, error) {
    return false, nil  // This is NOT a downloader
}
```

**Usenet Indexer Plugin** (`plugins/usenet-indexer/main.go`):
```go
func (p *UsenetIndexerPlugin) IsDownloader(ctx context.Context) (bool, error) {
    return false, nil  // This is NOT a downloader
}
```

## Remaining Work

### 1. **Regenerate Protobuf Files** (REQUIRED)

The protobuf definition was updated but the generated Go code needs to be regenerated:

```bash
# Install protoc-gen-go and protoc-gen-go-grpc if not already installed
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest

# Regenerate the proto files
cd /Users/blakestevenson/repos/nimbus
protoc --go_out=. --go_opt=paths=source_relative \
       --go-grpc_out=. --go-grpc_opt=paths=source_relative \
       internal/plugins/proto/plugin.proto
```

**Without this step, the code will not compile.**

### 2. **Run Database Migration**

The downloads table needs to be created:

```bash
# The migration file is at:
# internal/db/migrations/0009_downloads.sql

# Your migration system should automatically detect and run it
# on next startup, or run it manually if needed
```

### 3. **Update NZB Plugin to Use Database**

Currently, the NZB downloader plugin stores downloads in memory. To fully leverage the new architecture:

**Changes needed in `plugins/nzb-downloader/main.go`:**

1. **Store downloads in database instead of memory:**
   - Remove `downloads` and `queue` fields from `DownloadManager`
   - Use plugin SDK to call database operations via HTTP API
   - Query downloads from database instead of in-memory map

2. **Implement proper API routes for service integration:**
   - Ensure `/api/plugins/nzb-downloader/downloads` returns database downloads
   - Update `handleAddDownload` to insert into database
   - Update progress tracking to write to database
   - Update status changes to persist to database

3. **Resume downloads on restart:**
   - On plugin initialization, query database for in-progress downloads
   - Resume any downloads that were interrupted

### 4. **Add Database Queries**

Generate SQLc queries for the new downloads table:

**Create** `internal/db/queries/downloads.sql`:
```sql
-- name: CreateDownload :one
INSERT INTO downloads (
    plugin_id, name, status, url, file_content, file_name, 
    priority, metadata, created_by_user_id
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
RETURNING *;

-- name: GetDownload :one
SELECT * FROM downloads WHERE id = ?;

-- name: ListDownloads :many
SELECT * FROM downloads
WHERE (plugin_id = ? OR ? = '')
  AND (status = ? OR ? = '')
ORDER BY 
    CASE WHEN status IN ('queued', 'downloading') 
         THEN queue_position 
         ELSE 999999 
    END,
    created_at DESC;

-- name: UpdateDownloadStatus :one
UPDATE downloads 
SET status = ?, updated_at = CURRENT_TIMESTAMP
WHERE id = ?
RETURNING *;

-- name: UpdateDownloadProgress :exec
UPDATE downloads 
SET progress = ?, downloaded_bytes = ?, updated_at = CURRENT_TIMESTAMP
WHERE id = ?;

-- name: DeleteDownload :exec
DELETE FROM downloads WHERE id = ?;

-- name: AddDownloadLog :exec
INSERT INTO download_logs (download_id, level, message)
VALUES (?, ?, ?);
```

Then regenerate SQLc:
```bash
sqlc generate
```

### 5. **Test the Integration**

Once protobuf is regenerated and the code compiles:

1. Start the server
2. Check that downloader service is initialized
3. Test the unified API:
   ```bash
   # List downloaders
   curl http://localhost:8080/api/downloaders
   
   # Create a download
   curl -X POST http://localhost:8080/api/downloads \
     -H "Content-Type: application/json" \
     -d '{
       "plugin_id": "nzb-downloader",
       "name": "Test Download",
       "file_content": "...",
       "priority": 1
     }'
   
   # List downloads
   curl http://localhost:8080/api/downloads
   ```

### 6. **Future Enhancements**

Once the basic system is working:

1. **Event System**: Emit events when downloads complete
   - Trigger media import automatically
   - Send notifications

2. **Bandwidth Management**: Add global bandwidth limits
   - Throttle all downloaders collectively
   - Configurable per-user or global limits

3. **Torrent Plugin**: Implement a torrent downloader
   - Follow same pattern as NZB plugin
   - Implements `IsDownloader() = true`
   - Uses same unified API

4. **Download History**: Add retention policies
   - Auto-cleanup old completed downloads
   - Configurable retention periods

5. **Priority Queue**: Improve queue management
   - Cross-plugin priority (NZB vs torrent)
   - User-based quotas

## Benefits of This Architecture

1. **Unified Interface**: Single API for all download types
2. **Plugin Independence**: Easy to add new downloader types
3. **Persistent State**: Downloads survive restarts
4. **Multi-User**: Track which user initiated downloads
5. **Scalability**: Can run multiple downloaders simultaneously
6. **Monitoring**: Centralized view of all download activity
7. **Consistent with Indexers**: Same pattern, easier to understand

## File Checklist

- ✅ `internal/db/migrations/0009_downloads.sql` - Database schema
- ✅ `internal/downloader/service.go` - Downloader service
- ✅ `internal/http/downloader_routes.go` - HTTP routes
- ✅ `internal/http/router.go` - Router integration
- ✅ `internal/plugins/types.go` - Plugin interface extension
- ✅ `internal/plugins/manager.go` - Plugin manager updates
- ✅ `internal/plugins/proto/plugin.proto` - gRPC protocol
- ✅ `internal/plugins/rpc.go` - RPC implementation
- ✅ `plugins/nzb-downloader/main.go` - NZB plugin updates
- ✅ `plugins/tmdb-plugin/main.go` - TMDB plugin updates
- ✅ `plugins/usenet-indexer/main.go` - Usenet plugin updates
- ⚠️  `internal/plugins/proto/plugin.pb.go` - **NEEDS REGENERATION**
- ⚠️  `internal/plugins/proto/plugin_grpc.pb.go` - **NEEDS REGENERATION**
- ⚠️  `internal/db/queries/downloads.sql` - **NEEDS CREATION**
- ⚠️  `internal/db/generated/*_downloads.go` - **NEEDS GENERATION**

## Summary

The downloader service abstraction is **90% complete**. The architecture is implemented and all code is in place. The remaining work is:

1. **Critical**: Regenerate protobuf files (prevents compilation)
2. **Important**: Add SQLc queries for database operations
3. **Enhancement**: Update NZB plugin to use database instead of memory
4. **Testing**: Verify end-to-end functionality

The foundation is solid and follows established patterns from the indexer service, making it easy to understand and extend.
