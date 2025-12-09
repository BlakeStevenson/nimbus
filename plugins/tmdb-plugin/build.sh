#!/bin/bash
set -e

echo "Building TMDB plugin..."
go build -o tmdb-plugin main.go
echo "✓ Build successful!"

echo ""
echo "Plugin ready at: $(pwd)/tmdb-plugin"
echo ""
echo "To use this plugin:"
echo "1. Set the TMDB API key in the config table:"
echo "   curl -X PUT 'http://localhost:8080/api/config/plugins.tmdb.api_key' \\"
echo "     -H 'Authorization: Bearer YOUR_JWT_TOKEN' \\"
echo "     -H 'Content-Type: application/json' \\"
echo "     -d '{\"value\": \"your_tmdb_api_key_here\"}'"
echo "2. Ensure ENABLE_PLUGINS=true"
echo "3. Set PLUGINS_DIR to the plugins directory"
echo "4. Restart the Nimbus server"
