#!/bin/bash
# Build script for the Usenet Indexer plugin

set -e

echo "Building Usenet Indexer plugin..."

# Build the Go binary
go build -o usenet-indexer main.go newznab.go

echo "✓ Plugin binary built: usenet-indexer"
echo ""
echo "To install this plugin:"
echo "  1. Create the plugin directory: mkdir -p /var/lib/nimbus/plugins/usenet-indexer"
echo "  2. Copy files:"
echo "     - cp usenet-indexer /var/lib/nimbus/plugins/usenet-indexer/"
echo "     - cp manifest.json /var/lib/nimbus/plugins/usenet-indexer/"
echo "     - cp -r web /var/lib/nimbus/plugins/usenet-indexer/"
echo "  3. Enable plugins: export ENABLE_PLUGINS=true"
echo "  4. Set plugins directory: export PLUGINS_DIR=/var/lib/nimbus/plugins"
echo "  5. Restart Nimbus server"
