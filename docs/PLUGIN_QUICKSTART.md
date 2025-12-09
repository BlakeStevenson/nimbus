# Plugin System Quick Start

This guide will help you get started with the Nimbus plugin system.

## What are Plugins?

Plugins extend Nimbus with custom functionality:

- **API Extensions**: Add custom REST endpoints (e.g., Sonarr/Radarr compatibility)
- **UI Extensions**: Add new pages and navigation items
- **Event Handlers**: React to system events (downloads, imports, etc.)
- **Integrations**: Connect to external services

## Prerequisites

- Nimbus server installed and running
- Go 1.21+ (for building plugins)
- Basic understanding of Go and React

## Quick Start

### 1. Try the Example Plugin

```bash
# Navigate to the example plugin
cd plugins/example-plugin

# Build it
./build.sh

# Set environment variables for development
export ENABLE_PLUGINS=true
export PLUGINS_DIR=$(pwd)/..

# Run Nimbus (from repo root)
cd ../..
go run cmd/server/main.go
```

### 2. Access the Plugin

1. Open Nimbus in your browser
2. Log in as an admin user
3. Navigate to **Plugins** page
4. Verify "Example Plugin" is listed and enabled
5. Check the sidebar - you should see "Example Plugin"
6. Click it to view the plugin's UI

### 3. Test the Plugin

The example plugin provides a UI at `/plugins/example-plugin`:

- Click "Call /api/plugins/example/hello" (public endpoint)
- Click "Call /api/plugins/example/status" (authenticated endpoint)
- View the JSON responses

## Creating Your First Plugin

### Step 1: Create Plugin Directory

```bash
mkdir -p plugins/my-first-plugin
cd plugins/my-first-plugin
```

### Step 2: Create manifest.json

```json
{
  "id": "my-first-plugin",
  "name": "My First Plugin",
  "description": "Learning the plugin system",
  "version": "0.1.0",
  "executable": "my-first-plugin",
  "webDir": "web",
  "capabilities": ["api"]
}
```

### Step 3: Create main.go

```go
package main

import (
    "context"
    "encoding/json"
    "net/http"
    
    "github.com/blakestevenson/nimbus/internal/plugins"
    "github.com/hashicorp/go-plugin"
)

type MyFirstPlugin struct{}

func (p *MyFirstPlugin) Metadata(ctx context.Context) (*plugins.PluginMetadata, error) {
    return &plugins.PluginMetadata{
        ID:          "my-first-plugin",
        Name:        "My First Plugin",
        Version:     "0.1.0",
        Description: "My first Nimbus plugin",
        Capabilities: []string{"api"},
    }, nil
}

func (p *MyFirstPlugin) APIRoutes(ctx context.Context) ([]plugins.RouteDescriptor, error) {
    return []plugins.RouteDescriptor{
        {
            Method: "GET",
            Path:   "/api/plugins/my-first-plugin/hello",
            Auth:   "none",
        },
    }, nil
}

func (p *MyFirstPlugin) HandleAPI(ctx context.Context, req *plugins.PluginHTTPRequest) (*plugins.PluginHTTPResponse, error) {
    data := map[string]string{"message": "Hello from my first plugin!"}
    body, _ := json.Marshal(data)
    
    return &plugins.PluginHTTPResponse{
        StatusCode: http.StatusOK,
        Headers:    map[string][]string{"Content-Type": {"application/json"}},
        Body:       body,
    }, nil
}

func (p *MyFirstPlugin) UIManifest(ctx context.Context) (*plugins.UIManifest, error) {
    return &plugins.UIManifest{}, nil // No UI for now
}

func (p *MyFirstPlugin) HandleEvent(ctx context.Context, evt plugins.Event) error {
    return nil // No event handling
}

func main() {
    plugin.Serve(&plugin.ServeConfig{
        HandshakeConfig: plugins.Handshake,
        Plugins: map[string]plugin.Plugin{
            "media-suite": &plugins.MediaSuitePluginGRPC{
                Impl: &MyFirstPlugin{},
            },
        },
        GRPCServer: plugin.DefaultGRPCServer,
    })
}
```

### Step 4: Create go.mod

```bash
go mod init github.com/yourusername/my-first-plugin
go mod edit -replace github.com/blakestevenson/nimbus=../..
go mod tidy
```

### Step 5: Build and Test

```bash
go build -o my-first-plugin main.go

# Restart Nimbus (it will discover the new plugin)
# Then test:
curl http://localhost:8080/api/plugins/my-first-plugin/hello
```

## Common Plugin Patterns

### Pattern 1: Public API Endpoint

```go
{
    Method: "GET",
    Path:   "/api/plugins/myplugin/public",
    Auth:   "none",
}
```

### Pattern 2: Authenticated Endpoint

```go
{
    Method: "POST",
    Path:   "/api/plugins/myplugin/secure",
    Auth:   "session",
}

// In handler, access user:
if req.UserID != nil {
    userID := *req.UserID
    // ...
}
```

### Pattern 3: UI Page

```go
// In UIManifest:
NavItems: []plugins.UINavItem{
    {
        Label: "My Plugin",
        Path:  "/plugins/myplugin",
    },
},
Routes: []plugins.UIRoute{
    {
        Path:      "/plugins/myplugin",
        BundleURL: "/plugins/myplugin/main.js",
    },
},
```

## Plugin Development Tips

### 1. Use Logs

Check Nimbus logs to debug:

```bash
# View real-time logs
journalctl -u nimbus -f

# Or if running in development
# Logs will appear in your terminal
```

### 2. Enable/Disable for Testing

Use the Plugins page in Nimbus UI to quickly enable/disable your plugin without restarting.

### 3. Hot Reload

After making changes:
1. Rebuild: `go build -o my-plugin main.go`
2. Disable plugin in UI
3. Enable plugin in UI

### 4. Test API Endpoints

Use curl or tools like Postman:

```bash
# Test public endpoint
curl http://localhost:8080/api/plugins/myplugin/endpoint

# Test authenticated endpoint
curl -H "Authorization: Bearer YOUR_TOKEN" \
     http://localhost:8080/api/plugins/myplugin/secure
```

## Next Steps

- Read the full [Plugin Documentation](../PLUGINS.md)
- Explore the [Example Plugin](../plugins/example-plugin)
- Check out community plugins (coming soon)
- Join the Discord for help

## Common Issues

**Plugin not appearing in UI**
- Check `ENABLE_PLUGINS=true` is set
- Verify `manifest.json` is valid
- Check logs for errors

**API routes not working**
- Ensure plugin is enabled
- Check route path matches exactly
- Verify authentication settings

**UI not loading**
- Check `web/main.js` exists
- Verify bundle exports a default React component
- Check browser console for errors

## Resources

- [Full Plugin Documentation](../PLUGINS.md)
- [Example Plugin Source](../plugins/example-plugin)
- [Go Plugin Library](https://github.com/hashicorp/go-plugin)
- [Nimbus API Reference](./API.md) (coming soon)

Happy plugin building! 🚀
