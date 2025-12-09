# Usenet Indexer Plugin

A Nimbus plugin that provides Usenet search and indexing capabilities using Newznab-compatible indexers (similar to Sonarr/Radarr).

## Features

- **Newznab API Integration**: Connect to any Newznab-compatible indexer
- **Multi-Type Search**: Search for TV shows, movies, or general content
- **RSS Feed Support**: Enable/disable RSS feed functionality
- **Category Filtering**: Configure separate categories for TV shows and movies
- **Advanced Search Parameters**: Support for TVDB ID, IMDB ID, season/episode search
- **Web UI**: Full configuration interface with search testing
- **Secure Configuration**: API keys are masked in the UI

## API Endpoints

### Configuration

- `GET /api/plugins/usenet-indexer/config` - Get current configuration
- `POST /api/plugins/usenet-indexer/config` - Update configuration
- `POST /api/plugins/usenet-indexer/test` - Test connection to indexer

### Search

- `GET /api/plugins/usenet-indexer/search` - General search
  - Query params: `q`, `categories`, `limit`, `offset`
  
- `GET /api/plugins/usenet-indexer/search/tv` - TV show search
  - Query params: `q`, `categories`, `tvdbid`, `tvrageid`, `season`, `episode`, `limit`, `offset`
  
- `GET /api/plugins/usenet-indexer/search/movie` - Movie search
  - Query params: `q`, `categories`, `imdbid`, `limit`, `offset`

### RSS

- `GET /api/plugins/usenet-indexer/rss` - Get RSS feed
  - Query params: `categories`, `limit`
  - Requires RSS to be enabled in configuration

## Configuration

Access the configuration UI at `/plugins/usenet-indexer` after enabling the plugin.

### Required Settings

- **API URL**: The base URL of your Newznab indexer (e.g., `https://indexer.example.com`)
- **API Key**: Your Newznab API key from the indexer

### Optional Settings

- **Enable Indexer**: Enable/disable the indexer
- **Enable RSS Feed**: Enable/disable RSS feed access
- **TV Categories**: Comma-separated category IDs for TV shows (default: `5030,5040`)
- **Movie Categories**: Comma-separated category IDs for movies (default: `2000,2010,2020,2030,2040,2050,2060`)

## Common Newznab Categories

### TV Shows
- `5000` - TV (All)
- `5030` - TV HD
- `5040` - TV SD
- `5070` - TV Anime

### Movies
- `2000` - Movies (All)
- `2010` - Movies Foreign
- `2020` - Movies Other
- `2030` - Movies SD
- `2040` - Movies HD
- `2045` - Movies UHD
- `2050` - Movies BluRay
- `2060` - Movies 3D

## Installation

1. Build the plugin:
   ```bash
   cd plugins/usenet-indexer
   ./build.sh
   ```

2. Create the plugin directory:
   ```bash
   mkdir -p /var/lib/nimbus/plugins/usenet-indexer
   ```

3. Copy files:
   ```bash
   cp usenet-indexer /var/lib/nimbus/plugins/usenet-indexer/
   cp manifest.json /var/lib/nimbus/plugins/usenet-indexer/
   cp -r web /var/lib/nimbus/plugins/usenet-indexer/
   ```

4. Enable plugins in Nimbus:
   ```bash
   export ENABLE_PLUGINS=true
   export PLUGINS_DIR=/var/lib/nimbus/plugins
   ```

5. Restart the Nimbus server

6. Navigate to the Plugins page in the UI and enable "Usenet Indexer"

## Example Usage

### Search for a TV Show
```bash
curl "http://localhost:8080/api/plugins/usenet-indexer/search/tv?q=Breaking+Bad&season=1&episode=1" \
  -H "Cookie: session=..."
```

### Search for a Movie
```bash
curl "http://localhost:8080/api/plugins/usenet-indexer/search/movie?q=Inception&imdbid=tt1375666" \
  -H "Cookie: session=..."
```

### Get RSS Feed
```bash
curl "http://localhost:8080/api/plugins/usenet-indexer/rss?limit=50" \
  -H "Cookie: session=..."
```

## Response Format

All search and RSS endpoints return:

```json
{
  "releases": [
    {
      "id": "guid-string",
      "title": "Release.Title.1080p.WEB-DL",
      "guid": "guid-string",
      "link": "https://indexer.example.com/details/...",
      "comments": "https://indexer.example.com/comments/...",
      "publishDate": "2024-01-01T12:00:00Z",
      "category": "5040",
      "size": 1234567890,
      "description": "Release description",
      "downloadUrl": "https://indexer.example.com/download/...",
      "attributes": {
        "season": "1",
        "episode": "1",
        "tvdbid": "123456"
      }
    }
  ],
  "count": 1
}
```

## Development

### Building
```bash
go build -o usenet-indexer main.go newznab.go
```

### Testing
Use the built-in test functionality in the UI or use curl to test endpoints.

## Compatible Indexers

This plugin works with any Newznab-compatible indexer, including:
- NZBgeek
- NZBFinder
- DrunkenSlug
- NZB.su
- Usenet-Crawler
- Any custom Newznab indexer

## License

Part of the Nimbus media suite project.
