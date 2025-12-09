# Nimbus Plugin System

This document describes the Nimbus plugin system and how to develop plugins.

## Overview

Nimbus uses [hashicorp/go-plugin](https://github.com/hashicorp/go-plugin) with gRPC to provide a robust plugin architecture. Plugins run as separate processes and communicate with the Nimbus core via RPC.

**Plugin Capabilities:**
- **API Routes**: Expose custom HTTP endpoints (e.g., Sonarr/Radarr compatibility)
- **UI Extensions**: Add React pages and navigation items to the frontend
- **Event Handling**: Respond to system events (future feature)
- **Core SDK**: Access Nimbus core services (config, media, logging)

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                     Nimbus Core (Host)                      │
│  ┌───────────────────────────────────────────────────────┐  │
│  │              Plugin Manager                           │  │
│  │  - Discovers plugins from filesystem                  │  │
│  │  - Spawns plugin processes (go-plugin)               │  │
│  │  - Manages lifecycle (start/stop/enable/disable)     │  │
│  │  - Registers plugin routes into Chi router           │  │
│  └───────────────────────────────────────────────────────┘  │
│                            │                                 │
│                       gRPC over stdio                       │
│                            │                                 │
└─────────────────────────────┼───────────────────────────────┘
                              │
         ┌────────────────────┴────────────────────┐
         │                                         │
    ┌────▼────┐                             ┌─────▼────┐
    │ Plugin A │                             │ Plugin B │
    │ (Sonarr) │                             │ (qBit)   │
    └──────────┘                             └──────────┘
```

## Plugin Structure

A plugin consists of:

```
plugins/
└── your-plugin/
    ├── manifest.json       # Plugin metadata
    ├── your-plugin         # Compiled Go binary
    ├── web/                # Frontend assets (optional)
    │   └── main.js        # React component bundle
    └── main.go            # Plugin source code
```

### manifest.json

```json
{
  "id": "your-plugin",
  "name": "Your Plugin Name",
  "description": "Brief description",
  "version": "0.1.0",
  "executable": "your-plugin",
  "webDir": "web",
  "capabilities": ["api", "ui", "events"]
}
```

## Developing a Plugin

### 1. Project Setup

```bash
mkdir -p plugins/my-plugin
cd plugins/my-plugin
```

Create `go.mod`:

```go
module github.com/yourusername/nimbus-plugin-myplugin

go 1.21

require (
    github.com/blakestevenson/nimbus v0.0.0
    github.com/hashicorp/go-plugin v1.6.0
    google.golang.org/grpc v1.60.0
)

// Use local nimbus for development
replace github.com/blakestevenson/nimbus => ../..
```

### 2. Implement the Plugin Interface

Create `main.go`:

```go
package main

import (
    "context"
    "encoding/json"
    "net/http"
    
    "github.com/blakestevenson/nimbus/internal/plugins"
    "github.com/hashicorp/go-plugin"
)

type MyPlugin struct{}

// Metadata returns plugin information
func (p *MyPlugin) Metadata(ctx context.Context) (*plugins.PluginMetadata, error) {
    return &plugins.PluginMetadata{
        ID:          "my-plugin",
        Name:        "My Plugin",
        Version:     "0.1.0",
        Description: "Does something cool",
        Capabilities: []string{"api", "ui"},
    }, nil
}

// APIRoutes defines HTTP routes
func (p *MyPlugin) APIRoutes(ctx context.Context) ([]plugins.RouteDescriptor, error) {
    return []plugins.RouteDescriptor{
        {
            Method: "GET",
            Path:   "/api/plugins/my-plugin/endpoint",
            Auth:   "session", // "session", "apikey", or "none"
            Tag:    "",
        },
    }, nil
}

// HandleAPI handles HTTP requests
func (p *MyPlugin) HandleAPI(ctx context.Context, req *plugins.PluginHTTPRequest) (*plugins.PluginHTTPResponse, error) {
    response := map[string]interface{}{
        "message": "Hello from my plugin!",
    }
    
    body, _ := json.Marshal(response)
    
    return &plugins.PluginHTTPResponse{
        StatusCode: http.StatusOK,
        Headers: map[string][]string{
            "Content-Type": {"application/json"},
        },
        Body: body,
    }, nil
}

// UIManifest defines UI routes and nav items
func (p *MyPlugin) UIManifest(ctx context.Context) (*plugins.UIManifest, error) {
    return &plugins.UIManifest{
        NavItems: []plugins.UINavItem{
            {
                Label: "My Plugin",
                Path:  "/plugins/my-plugin",
                Icon:  "puzzle",
            },
        },
        Routes: []plugins.UIRoute{
            {
                Path:      "/plugins/my-plugin",
                BundleURL: "/plugins/my-plugin/main.js",
            },
        },
    }, nil
}

// HandleEvent handles system events
func (p *MyPlugin) HandleEvent(ctx context.Context, evt plugins.Event) error {
    // Optional: implement event handling
    return nil
}

func main() {
    plugin.Serve(&plugin.ServeConfig{
        HandshakeConfig: plugins.Handshake,
        Plugins: map[string]plugin.Plugin{
            "media-suite": &plugins.MediaSuitePluginGRPC{
                Impl: &MyPlugin{},
            },
        },
        GRPCServer: plugin.DefaultGRPCServer,
    })
}
```

### 3. Using the Core SDK

Access Nimbus core services:

```go
// The SDK is available via dependency injection when your plugin is loaded
// For now, plugins interact with core via the HTTP API or by returning
// data that the core processes.

// Future: SDK will be passed to plugin methods
```

### 4. Build the Plugin

```bash
go build -o my-plugin main.go
```

### 5. Create Frontend UI (Optional)

Create `web/main.js`:

```javascript
import React from 'react';

export default function MyPluginPage() {
  return React.createElement(
    'div',
    { className: 'p-6' },
    React.createElement('h1', { className: 'text-3xl font-bold' }, 'My Plugin'),
    React.createElement('p', null, 'Hello from my plugin UI!')
  );
}
```

**Note**: The UI bundle must be a JavaScript module that default-exports a React component.

### 6. Install the Plugin

```bash
# Create plugin directory
sudo mkdir -p /var/lib/nimbus/plugins/my-plugin

# Copy files
sudo cp my-plugin /var/lib/nimbus/plugins/my-plugin/
sudo cp manifest.json /var/lib/nimbus/plugins/my-plugin/
sudo cp -r web /var/lib/nimbus/plugins/my-plugin/
```

### 7. Enable Plugins

Set environment variables:

```bash
export ENABLE_PLUGINS=true
export PLUGINS_DIR=/var/lib/nimbus/plugins
```

Restart the Nimbus server.

## Plugin API Reference

### PluginMetadata

```go
type PluginMetadata struct {
    ID           string   // Unique identifier
    Name         string   // Display name
    Version      string   // Semantic version
    Description  string   // Brief description
    Capabilities []string // ["api", "ui", "events", "compat:sonarr"]
}
```

### RouteDescriptor

```go
type RouteDescriptor struct {
    Method string // HTTP method: GET, POST, PUT, DELETE, PATCH
    Path   string // Route path: /api/plugins/{id}/...
    Auth   string // "session", "apikey", "none"
    Tag    string // Optional tag: "compat:sonarr", "internal"
}
```

### PluginHTTPRequest

```go
type PluginHTTPRequest struct {
    Method  string
    Path    string
    Query   map[string][]string
    Headers map[string][]string
    Body    []byte
    UserID  *int64   // Set if Auth="session"
    Scopes  []string // Future: permission scopes
}
```

### PluginHTTPResponse

```go
type PluginHTTPResponse struct {
    StatusCode int
    Headers    map[string][]string
    Body       []byte
}
```

### UIManifest

```go
type UIManifest struct {
    NavItems []UINavItem
    Routes   []UIRoute
}

type UINavItem struct {
    Label string // Display text
    Path  string // Route path
    Group string // Optional grouping
    Icon  string // Optional icon name
}

type UIRoute struct {
    Path      string // Frontend route
    BundleURL string // Path to JS bundle
}
```

## Authentication

Plugins can require authentication per-route:

- **`Auth: "none"`**: Public endpoint, no authentication required
- **`Auth: "session"`**: Requires logged-in user; `UserID` will be populated
- **`Auth: "apikey"`**: Requires API key (future feature)

## Events (Future)

Plugins will be able to subscribe to system events:

```go
type Event struct {
    Type      string                 // "download.finished", "media.imported"
    Data      map[string]interface{}
    Timestamp time.Time
}

func (p *MyPlugin) HandleEvent(ctx context.Context, evt Event) error {
    switch evt.Type {
    case "media.imported":
        // Handle event
    }
    return nil
}
```

## Best Practices

1. **Error Handling**: Always return appropriate HTTP status codes
2. **Logging**: Use structured logging (future: use SDK logger)
3. **Security**: Validate all input; use appropriate auth levels
4. **Versioning**: Follow semantic versioning
5. **Dependencies**: Minimize external dependencies
6. **Testing**: Test your plugin independently before installing

## Compatibility Layers

Plugins can provide compatibility with existing tools (Sonarr, Radarr, etc.):

```go
capabilities: ["api", "ui", "compat:sonarr"]

// Routes can specify compatibility tags
{
    Method: "GET",
    Path:   "/api/v3/system/status",
    Auth:   "apikey",
    Tag:    "compat:sonarr",
}
```

These routes can optionally be served on a separate port for isolation.

## Troubleshooting

### Plugin Not Loading

- Check logs: `journalctl -u nimbus -f`
- Verify manifest.json is valid JSON
- Ensure binary has execute permissions: `chmod +x plugin-binary`
- Check `ENABLE_PLUGINS=true` is set

### UI Not Appearing

- Verify `web/main.js` exists and is accessible
- Check browser console for loading errors
- Ensure React is available globally (or use a bundler)

### API Routes Not Working

- Check route path matches exactly
- Verify authentication requirements
- Look for errors in plugin logs

## Example Plugin

See `plugins/example-plugin/` for a complete working example.

## Future Enhancements

- **SDK Access**: Direct access to Nimbus services (config, media, downloads)
- **Event System**: Subscribe to system events
- **Plugin Marketplace**: Discover and install plugins
- **Hot Reload**: Update plugins without restart
- **Plugin Dependencies**: Plugins can depend on other plugins
- **Permissions System**: Fine-grained access control

## Questions?

File an issue on GitHub or join the community Discord.
