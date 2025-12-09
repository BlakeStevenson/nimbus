# TMDB Plugin for Nimbus

This plugin integrates with The Movie Database (TMDB) API to fetch rich metadata for movies and TV shows, including descriptions, ratings, cover images, and more.

## Features

- Search for movies and TV shows by title and year
- Fetch detailed metadata including:
  - Descriptions/overviews
  - User ratings and vote counts
  - Poster and backdrop images
  - Release dates
  - Genres
  - Runtime
  - Cast and crew information
- Enrich existing media items with TMDB metadata

## Configuration

### API Key

You need a TMDB API key to use this plugin. Get one for free at [https://www.themoviedb.org/settings/api](https://www.themoviedb.org/settings/api)

The plugin supports two configuration methods (in order of priority):

1. **Config Table** (Recommended): Stored in the Nimbus database - accessed via SDK
2. **Environment Variable**: `TMDB_API_KEY` in `.env` file (fallback)

The plugin automatically uses the Nimbus SDK to fetch the API key from the config table. If the SDK is unavailable or the key is not set, it falls back to the environment variable.

#### Method 1: Config Table (Recommended)

You can set the API key using the config API:

```bash
curl -X PUT "http://localhost:8080/api/config/plugins.tmdb.api_key" \
  -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"value": "your_tmdb_api_key_here"}'
```

Or directly in the database:

```sql
INSERT INTO config (key, value) 
VALUES ('plugins.tmdb.api_key', '"your_tmdb_api_key_here"'::jsonb)
ON CONFLICT (key) 
DO UPDATE SET value = EXCLUDED.value, updated_at = NOW();
```

Note: The value in the database must be a JSON string (wrapped in quotes).

#### Method 2: Environment Variable (Fallback)

Add the API key to your `.env` file:

```bash
TMDB_API_KEY=your_tmdb_api_key_here
```

This method is used as a fallback if the config table lookup fails or the key is not set in the database. Restart the Nimbus server after adding the environment variable.

## Installation

1. Build the plugin:
   ```bash
   cd plugins/tmdb-plugin
   go build -o tmdb-plugin main.go
   ```

2. The plugin will be automatically discovered by Nimbus if `ENABLE_PLUGINS=true` is set.

## API Endpoints

### Search for Movies

```
GET /api/plugins/tmdb/search/movie?query=<title>&year=<year>
```

**Parameters:**
- `query` (required): Movie title to search for
- `year` (optional): Release year to filter results

**Example:**
```bash
curl -X GET "http://localhost:8080/api/plugins/tmdb/search/movie?query=Inception&year=2010" \
  -H "Authorization: Bearer YOUR_JWT_TOKEN"
```

### Search for TV Shows

```
GET /api/plugins/tmdb/search/tv?query=<title>&year=<year>
```

**Parameters:**
- `query` (required): TV show title to search for
- `year` (optional): First air date year to filter results

**Example:**
```bash
curl -X GET "http://localhost:8080/api/plugins/tmdb/search/tv?query=Breaking%20Bad" \
  -H "Authorization: Bearer YOUR_JWT_TOKEN"
```

### Get Movie Details

```
GET /api/plugins/tmdb/movie/{tmdb_id}
```

**Parameters:**
- `tmdb_id` (required): TMDB movie ID

**Example:**
```bash
curl -X GET "http://localhost:8080/api/plugins/tmdb/movie/27205" \
  -H "Authorization: Bearer YOUR_JWT_TOKEN"
```

### Get TV Show Details

```
GET /api/plugins/tmdb/tv/{tmdb_id}
```

**Parameters:**
- `tmdb_id` (required): TMDB TV show ID

**Example:**
```bash
curl -X GET "http://localhost:8080/api/plugins/tmdb/tv/1396" \
  -H "Authorization: Bearer YOUR_JWT_TOKEN"
```

### Enrich Media Item

```
POST /api/plugins/tmdb/enrich/{media_id}
```

**Parameters:**
- `media_id` (required): Nimbus media item ID

**Request Body:**
```json
{
  "tmdb_id": "27205",
  "type": "movie"
}
```

**Fields:**
- `tmdb_id` (required): TMDB ID for the movie or TV show
- `type` (required): Either "movie" or "tv"

**Response:**
```json
{
  "media_id": "123",
  "metadata": {
    "tmdb_id": "27205",
    "type": "movie",
    "description": "Cobb, a skilled thief...",
    "rating": 8.8,
    "vote_count": 35000,
    "poster_url": "https://image.tmdb.org/t/p/original/...",
    "backdrop_url": "https://image.tmdb.org/t/p/original/...",
    "release_date": "2010-07-16",
    "genres": [...],
    "runtime": 148
  },
  "message": "Metadata fetched successfully. Update the media item's metadata column with this data.",
  "sql_example": "UPDATE media_items SET metadata = metadata || '{...}'::jsonb WHERE id = 123"
}
```

**Example:**
```bash
curl -X POST "http://localhost:8080/api/plugins/tmdb/enrich/123" \
  -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"tmdb_id": "27205", "type": "movie"}'
```

## Usage Workflow

1. **Search for content**: Use the search endpoints to find the TMDB ID for your media
2. **Get details** (optional): Fetch full details to verify it's the correct match
3. **Enrich media item**: Use the enrich endpoint to fetch and store metadata

### Example Workflow

```bash
# 1. Search for a movie
curl -X GET "http://localhost:8080/api/plugins/tmdb/search/movie?query=Inception&year=2010" \
  -H "Authorization: Bearer YOUR_JWT_TOKEN"

# Response includes: { "results": [{ "id": 27205, "title": "Inception", ... }] }

# 2. Enrich your media item (ID 123) with TMDB data
curl -X POST "http://localhost:8080/api/plugins/tmdb/enrich/123" \
  -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"tmdb_id": "27205", "type": "movie"}'

# 3. The metadata is now stored in the media_items table
```

## Metadata Schema

The plugin stores the following metadata in the `metadata` JSONB column:

```json
{
  "tmdb_id": "string",
  "type": "movie|tv",
  "description": "string",
  "rating": "number",
  "vote_count": "number",
  "poster_url": "string",
  "backdrop_url": "string",
  "release_date": "string (YYYY-MM-DD)",
  "first_air_date": "string (YYYY-MM-DD, TV only)",
  "genres": [{ "id": number, "name": "string" }],
  "runtime": "number (minutes, movies only)"
}
```

## Authentication

All endpoints require authentication using a session token (JWT). Include the token in the `Authorization` header:

```
Authorization: Bearer YOUR_JWT_TOKEN
```

## Error Responses

The plugin returns standard HTTP status codes:

- `200 OK`: Success
- `400 Bad Request`: Missing or invalid parameters
- `404 Not Found`: Route not found
- `500 Internal Server Error`: Server error or TMDB API error

Error responses include a JSON body:
```json
{
  "error": "Error message description"
}
```

## Development

### Building

```bash
cd plugins/tmdb-plugin
go build -o tmdb-plugin main.go
```

### Testing

You can test the plugin independently by running it with the Nimbus server:

```bash
# Make sure TMDB_API_KEY is set in .env
export ENABLE_PLUGINS=true
export PLUGINS_DIR=/path/to/nimbus/plugins

# Start Nimbus server
./server
```

## Troubleshooting

### "TMDB API key not configured"

Make sure `plugins.tmdb.api_key` is set in the config table. You can check by running:

```bash
curl -X GET "http://localhost:8080/api/config/plugins.tmdb.api_key" \
  -H "Authorization: Bearer YOUR_JWT_TOKEN"
```

If not set, use the configuration instructions above to add it.

### "Failed to search TMDB" or "Failed to get movie/TV show"

- Check that your API key is valid
- Verify you have internet connectivity
- Check TMDB API status at [https://www.themoviedb.org/](https://www.themoviedb.org/)

### Plugin not loading

- Ensure `ENABLE_PLUGINS=true` in `.env`
- Check that `PLUGINS_DIR` points to the correct directory
- Verify `manifest.json` exists and is valid JSON
- Check server logs for plugin loading errors

## License

Same as Nimbus core project.
