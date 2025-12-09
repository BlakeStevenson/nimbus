#!/bin/bash
# Build script for the NZB Downloader plugin

set -e

echo "Building NZB Downloader plugin..."

# Build the Go binary
go build -o nzb-downloader .

echo "✓ Plugin binary built: nzb-downloader"
echo ""
echo "To install this plugin:"
echo "  1. Create the plugin directory: mkdir -p /var/lib/nimbus/plugins/nzb-downloader"
echo "  2. Copy files:"
echo "     - cp nzb-downloader /var/lib/nimbus/plugins/nzb-downloader/"
echo "     - cp manifest.json /var/lib/nimbus/plugins/nzb-downloader/"
echo "     - cp -r web /var/lib/nimbus/plugins/nzb-downloader/"
echo "  3. Enable plugins: export ENABLE_PLUGINS=true"
echo "  4. Set plugins directory: export PLUGINS_DIR=/var/lib/nimbus/plugins"
echo "  5. Restart Nimbus server"
