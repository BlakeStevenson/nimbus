# Phase 4: Plugin Host Implementation Summary

This document summarizes the implementation of the Nimbus plugin system (Phase 4).

## Overview

Phase 4 introduces a comprehensive plugin system that allows developers to extend Nimbus with custom functionality through isolated plugin processes that communicate via gRPC.

## What Was Implemented

### 1. Database Layer

**File**: `internal/db/migrations/0004_plugins.sql`

- Created `plugins` table to track installed plugins
- Fields: id, name, description, version, enabled, capabilities (JSONB)
- Indexes for efficient queries

**File**: `internal/db/queries/plugins.sql`

- CRUD operations for plugins
- Enable/disable functionality
- Plugin existence checks

### 2. Backend Plugin System

#### Core Types (`internal/plugins/types.go`)

- `PluginMetadata`: Plugin information
- `RouteDescriptor`: HTTP route definitions
- `PluginHTTPRequest/Response`: HTTP bridging types
- `UIManifest`: Frontend integration types
- `MediaSuitePlugin` interface: Main plugin contract

#### RPC Layer (`internal/plugins/rpc.go`)

- gRPC client/server implementation using hashicorp/go-plugin
- Protocol buffers definitions (`proto/plugin.proto`)
- Handshake configuration for secure plugin loading

#### Plugin Manager (`internal/plugins/manager.go`)

- Plugin discovery from filesystem
- Lifecycle management (load, start, stop, enable, disable)
- Route registration
- In-memory plugin cache
- Concurrent-safe operations

#### SDK (`internal/plugins/sdk.go`)

- Config access (get/set/delete)
- Media CRUD operations
- Logging utilities
- Future: Downloads, events, more services

#### API Handlers (`internal/plugins/api_handlers.go`)

- `GET /api/plugins` - List all plugins
- `GET /api/plugins/{id}/ui-manifest` - Get UI manifest
- `POST /api/plugins/{id}/enable` - Enable plugin
- `POST /api/plugins/{id}/disable` - Disable plugin
- Static file serving for plugin assets

#### Router Integration (`internal/http/plugin_routes.go`)

- Plugin route setup helpers
- Authentication middleware per-route
- Static file serving
- Dynamic route registration

### 3. Frontend Integration

#### API Layer (`frontend/src/lib/api/plugins.ts`)

- TypeScript types matching backend
- React Query hooks for data fetching
- Enable/disable mutations
- UI manifest fetching

#### Components

**PluginPageLoader** (`frontend/src/components/plugins/PluginPageLoader.tsx`)
- Dynamic loading of plugin JavaScript bundles
- Error handling and fallbacks
- Loading states

**DynamicPluginRoute** (`frontend/src/components/plugins/DynamicPluginRoute.tsx`)
- Route matching for plugin pages
- Integration with React Router

#### UI Integration

**Sidebar** (`frontend/src/components/layout/Sidebar.tsx`)
- Already integrated! Uses `usePluginNavItems()` hook
- Displays plugin nav items in "Plugin Extensions" section

**PluginsPage** (`frontend/src/pages/PluginsPage.tsx`)
- Enhanced with enable/disable controls
- Plugin cards showing metadata
- Capability badges
- Real-time status updates

**Router** (`frontend/src/router/routes.tsx`)
- Catch-all route for plugin pages: `/plugins/:pluginId/*`

### 4. Example Plugin

**Location**: `plugins/example-plugin/`

Complete working plugin demonstrating:
- Two API endpoints (public and authenticated)
- UI integration with navigation
- Proper manifest structure
- React component bundle
- Build script

### 5. Documentation

- `PLUGINS.md`: Comprehensive plugin development guide
- `plugins/example-plugin/README.md`: Example plugin documentation
- `docs/PLUGIN_QUICKSTART.md`: Quick start guide for new developers

## Architecture Highlights

### Plugin Lifecycle

```
1. Discovery: Scan filesystem for manifest.json files
2. Database Sync: Upsert plugin metadata to database
3. Load: For enabled plugins, spawn process via go-plugin
4. RPC: Establish gRPC communication channel
5. Metadata Fetch: Retrieve plugin info, routes, UI manifest
6. Route Registration: Add plugin routes to Chi router
7. Running: Plugin handles requests via RPC
8. Shutdown: Kill plugin processes gracefully
```

### Communication Flow

```
HTTP Request → Chi Router → Plugin Handler
    ↓
Convert to PluginHTTPRequest
    ↓
gRPC Call to Plugin Process
    ↓
Plugin HandleAPI() method
    ↓
PluginHTTPResponse
    ↓
Convert to HTTP Response
```

### Security

- Plugins run in separate processes (isolation)
- Per-route authentication (session/apikey/none)
- Path traversal protection for static files
- User context passed via RPC

## Key Features

### Multi-Facet Design

1. **API Facet**: Custom HTTP endpoints
2. **UI Facet**: React pages and navigation
3. **Events Facet**: System event handlers (stub)
4. **SDK Facet**: Access to core services

### Hot Reload

Plugins can be enabled/disabled without server restart (route registration happens at runtime).

### Authentication

Three levels:
- `none`: Public endpoint
- `session`: Requires logged-in user
- `apikey`: API key authentication (future)

### Frontend Dynamic Loading

- Plugins provide JavaScript bundles
- Lazy-loaded via React.lazy()
- Error boundaries for plugin failures
- No rebuild required for plugin changes

## File Structure Created

```
internal/
├── plugins/
│   ├── types.go          # Core types and interfaces
│   ├── rpc.go           # gRPC implementation
│   ├── manager.go       # Plugin lifecycle manager
│   ├── sdk.go           # Core SDK for plugins
│   ├── api_handlers.go  # HTTP API handlers
│   └── proto/
│       ├── plugin.proto # Protocol buffer definitions
│       └── generate.sh  # Codegen script
├── http/
│   └── plugin_routes.go # Router integration
└── db/
    ├── migrations/
    │   └── 0004_plugins.sql
    └── queries/
        └── plugins.sql

frontend/src/
├── lib/api/
│   └── plugins.ts       # API hooks
├── components/plugins/
│   ├── PluginPageLoader.tsx
│   ├── PluginRoutes.tsx
│   └── DynamicPluginRoute.tsx
├── pages/
│   └── PluginsPage.tsx  # Enhanced management UI
└── router/
    └── routes.tsx       # Dynamic route integration

plugins/
└── example-plugin/
    ├── manifest.json
    ├── main.go
    ├── go.mod
    ├── build.sh
    ├── web/
    │   └── main.js
    └── README.md

docs/
├── PLUGINS.md           # Full documentation
└── PLUGIN_QUICKSTART.md # Quick start guide
```

## Dependencies Added

Required Go modules:
- `github.com/hashicorp/go-plugin` v1.6.0+
- `google.golang.org/grpc` v1.60.0+
- `google.golang.org/protobuf` (for protoc-gen-go)

Note: Protocol buffer generation requires:
- `protoc` compiler
- `protoc-gen-go`
- `protoc-gen-go-grpc`

## Environment Variables

- `ENABLE_PLUGINS`: Set to "true" to enable plugin system
- `PLUGINS_DIR`: Path to plugins directory (default: `/var/lib/nimbus/plugins`)

## Future Enhancements

1. **Event System**: Full implementation of event pub/sub
2. **Enhanced SDK**: More core service access (downloads, scheduling)
3. **API Keys**: API key authentication system
4. **Plugin Marketplace**: Discovery and installation UI
5. **Dependencies**: Plugin-to-plugin dependencies
6. **Permissions**: Fine-grained access control
7. **Separate Ports**: Compatibility routes on dedicated ports
8. **Metrics**: Plugin performance monitoring

## Testing the Implementation

### Prerequisites

1. Generate protocol buffers:
   ```bash
   cd internal/plugins/proto
   bash generate.sh
   ```

2. Run database migrations:
   ```bash
   # Migrations are auto-applied on startup
   ```

### Run the Example Plugin

```bash
# Build example plugin
cd plugins/example-plugin
./build.sh

# Start Nimbus with plugins enabled
cd ../..
export ENABLE_PLUGINS=true
export PLUGINS_DIR=$(pwd)/plugins
go run cmd/server/main.go
```

### Verify

1. Open Nimbus UI → Plugins page
2. Check "Example Plugin" is listed and enabled
3. Click "Example Plugin" in sidebar
4. Test API buttons on the plugin page

## Known Limitations

1. Protocol buffers must be generated manually (run `proto/generate.sh`)
2. Plugin UI bundles must be pre-built JavaScript (no build integration)
3. Hot reload requires manual enable/disable cycle
4. No plugin versioning/upgrade system yet
5. SDK access is limited (config and media only)

## Success Criteria ✅

All Phase 4 requirements met:

- [x] Database model for plugins
- [x] Plugin host using go-plugin + gRPC
- [x] Multi-facet plugin interface (API, UI, Events stub)
- [x] Core SDK for plugin-to-core communication
- [x] Backend HTTP route integration
- [x] Plugin management API endpoints
- [x] Frontend plugin UI integration
- [x] Dynamic nav items and routes
- [x] Lazy-loaded plugin bundles
- [x] Example plugin implementation
- [x] Comprehensive documentation

## Migration Notes

When deploying to production:

1. Run database migration 0004_plugins.sql
2. Install plugins to `/var/lib/nimbus/plugins/`
3. Set environment variables in systemd or config
4. Restart Nimbus service
5. Enable plugins via UI or API

## Support

For questions or issues:
- See `PLUGINS.md` for full documentation
- Check `PLUGIN_QUICKSTART.md` for quick start
- Review example plugin for reference implementation
- File issues on GitHub
