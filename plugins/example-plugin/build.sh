#!/bin/bash
# Build script for the example plugin

set -e

echo "Building example plugin..."

# Build the Go binary
go build -o example-plugin main.go

echo "✓ Plugin binary built: example-plugin"
echo ""
echo "To install this plugin:"
echo "  1. Create the plugin directory: mkdir -p /var/lib/nimbus/plugins/example-plugin"
echo "  2. Copy files:"
echo "     - cp example-plugin /var/lib/nimbus/plugins/example-plugin/"
echo "     - cp manifest.json /var/lib/nimbus/plugins/example-plugin/"
echo "     - cp -r web /var/lib/nimbus/plugins/example-plugin/"
echo "  3. Enable plugins: export ENABLE_PLUGINS=true"
echo "  4. Set plugins directory: export PLUGINS_DIR=/var/lib/nimbus/plugins"
echo "  5. Restart Nimbus server"
