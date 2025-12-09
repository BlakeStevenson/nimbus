# Example Plugin

A simple demonstration plugin for the Nimbus media management system.

## Features

This example plugin demonstrates:

- **API Routes**: Both public and authenticated endpoints
- **UI Integration**: Custom React page with navigation
- **Plugin Lifecycle**: Enable/disable without server restart
- **Best Practices**: Clean code structure and error handling

## API Endpoints

### GET /api/plugins/example/hello

Public endpoint that returns a greeting.

**Response:**
```json
{
  "message": "Hello from the Example Plugin!",
  "version": "0.1.0",
  "plugin": "example-plugin"
}
```

### GET /api/plugins/example/status

Authenticated endpoint that returns plugin status.

**Requires**: Active session (logged-in user)

**Response:**
```json
{
  "status": "running",
  "plugin": "example-plugin",
  "version": "0.1.0",
  "user_id": 123,
  "authenticated": true
}
```

## UI Features

The plugin provides a web page at `/plugins/example-plugin` that:

- Displays plugin information
- Allows testing API endpoints via buttons
- Shows responses in formatted JSON
- Demonstrates Nimbus UI integration

## Building

```bash
./build.sh
```

Or manually:

```bash
go build -o example-plugin main.go
```

## Installation

### Local Development

For development alongside the Nimbus codebase:

```bash
# Set environment variables
export ENABLE_PLUGINS=true
export PLUGINS_DIR=$(pwd)/plugins

# Build and run Nimbus
cd ../..
go run cmd/server/main.go
```

### Production Installation

```bash
# Create plugin directory
sudo mkdir -p /var/lib/nimbus/plugins/example-plugin

# Copy files
sudo cp example-plugin /var/lib/nimbus/plugins/example-plugin/
sudo cp manifest.json /var/lib/nimbus/plugins/example-plugin/
sudo cp -r web /var/lib/nimbus/plugins/example-plugin/

# Ensure binary is executable
sudo chmod +x /var/lib/nimbus/plugins/example-plugin/example-plugin

# Configure Nimbus
echo "ENABLE_PLUGINS=true" | sudo tee -a /etc/nimbus/nimbus.env
echo "PLUGINS_DIR=/var/lib/nimbus/plugins" | sudo tee -a /etc/nimbus/nimbus.env

# Restart Nimbus
sudo systemctl restart nimbus
```

## Verification

1. **Check Logs**: `sudo journalctl -u nimbus -f`
   - Look for "Plugin loaded successfully" messages

2. **Access UI**: Navigate to `/plugins` in Nimbus
   - Verify the example plugin appears in the list
   - Check that it's enabled

3. **Test Navigation**: Click on "Example Plugin" in the sidebar
   - Should navigate to `/plugins/example-plugin`

4. **Test API**: Use the buttons on the plugin page
   - "Call hello endpoint" should work without authentication
   - "Call status endpoint" requires being logged in

## Code Structure

```
example-plugin/
├── main.go           # Plugin implementation
├── manifest.json     # Plugin metadata
├── go.mod           # Go dependencies
├── build.sh         # Build script
├── web/
│   └── main.js      # React UI component
└── README.md        # This file
```

## Extending This Example

### Add More Routes

```go
func (p *ExamplePlugin) APIRoutes(ctx context.Context) ([]plugins.RouteDescriptor, error) {
    return []plugins.RouteDescriptor{
        // ... existing routes ...
        {
            Method: "POST",
            Path:   "/api/plugins/example/create",
            Auth:   "session",
            Tag:    "",
        },
    }, nil
}
```

### Add Database Access (Future)

```go
// When SDK is available
func (p *ExamplePlugin) handleCreate(ctx context.Context, req *plugins.PluginHTTPRequest) (*plugins.PluginHTTPResponse, error) {
    // Use SDK to access database
    item, err := p.sdk.CreateMediaItem(ctx, &plugins.MediaItem{
        Kind:  "custom",
        Title: "Example Item",
    })
    // ...
}
```

### Enhance the UI

Modify `web/main.js` to add more features:
- Forms for data input
- Tables for displaying data
- Integration with Nimbus UI components

## License

Same as Nimbus core (MIT)

## Support

See the main [PLUGINS.md](../../PLUGINS.md) documentation for:
- Complete API reference
- Best practices
- Troubleshooting guide
