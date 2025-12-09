# TMDB Plugin Setup Guide

## Quick Setup

### 1. Get a TMDB API Key

1. Go to [https://www.themoviedb.org/settings/api](https://www.themoviedb.org/settings/api)
2. Sign up for a free account if you don't have one
3. Request an API key (it's instant and free)
4. Copy your API key

### 2. Configure the API Key in Nimbus

The TMDB plugin reads the API key from the Nimbus configuration table. There are three ways to set it:

#### Option A: Using the Nimbus API (Recommended)

```bash
curl -X PUT "http://localhost:8080/api/config/plugins.tmdb.api_key" \
  -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"value": "your_tmdb_api_key_here"}'
```

Replace `YOUR_JWT_TOKEN` with your actual session token and `your_tmdb_api_key_here` with your TMDB API key.

#### Option B: Direct Database Insert

```sql
INSERT INTO config (key, value) 
VALUES ('plugins.tmdb.api_key', '"your_tmdb_api_key_here"'::jsonb)
ON CONFLICT (key) 
DO UPDATE SET value = EXCLUDED.value, updated_at = NOW();
```

**Important**: The value must be a JSON string (wrapped in quotes).

#### Option C: Using psql

```bash
psql postgres://postgres:postgres@localhost:5432/nimbus -c \
  "INSERT INTO config (key, value) VALUES ('plugins.tmdb.api_key', '\"your_tmdb_api_key_here\"'::jsonb) ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value, updated_at = NOW();"
```

### 3. Verify Configuration

Check that the API key was set correctly:

```bash
curl -X GET "http://localhost:8080/api/config/plugins.tmdb.api_key" \
  -H "Authorization: Bearer YOUR_JWT_TOKEN"
```

You should see:

```json
{
  "key": "plugins.tmdb.api_key",
  "value": "your_tmdb_api_key_here"
}
```

### 4. Test the Plugin

Search for a movie:

```bash
curl -X GET "http://localhost:8080/api/plugins/tmdb/search/movie?query=Inception" \
  -H "Authorization: Bearer YOUR_JWT_TOKEN"
```

You should get results from TMDB.

## Configuration Notes

- **Storage Location**: The API key is stored in the `config` table in the database
- **Security**: The config API requires authentication via JWT token
- **Format**: The value must be stored as a JSON string (with quotes)
- **Updates**: You can update the API key at any time using the same commands

## How It Works

1. When a request comes to the TMDB plugin, it fetches the API key from the Nimbus config table
2. The plugin makes an internal HTTP request to `http://localhost:8080/api/config/plugins.tmdb.api_key`
3. It forwards the user's authentication token to ensure proper access control
4. Once retrieved, the API key is used to make requests to the TMDB API

## Troubleshooting

### "TMDB API key not configured"

This means the plugin couldn't find the API key in the config table. Make sure:

1. The key is set in the config table (check with the verify command above)
2. The key name is exactly `plugins.tmdb.api_key`
3. The value is a JSON string (wrapped in quotes)

### "Failed to get TMDB API key from config"

This could mean:

1. The Nimbus server is not running
2. The authentication token is invalid
3. Network connectivity issues between the plugin and Nimbus API

### TMDB API Errors

If you get errors from TMDB:

1. Verify your API key is valid at [https://www.themoviedb.org/settings/api](https://www.themoviedb.org/settings/api)
2. Check you haven't exceeded the rate limits (TMDB allows 40 requests per 10 seconds)
3. Ensure you have internet connectivity

## Environment Variables

The plugin uses these optional environment variables:

- `NIMBUS_API_URL`: The URL of the Nimbus API (default: `http://localhost:8080`)

If your Nimbus server is running on a different host or port, set this:

```bash
export NIMBUS_API_URL=http://your-host:your-port
```

## Security Considerations

- The API key is stored in the database, not in environment variables or files
- All requests to get the config require authentication
- The API key is transmitted between the plugin and TMDB over HTTPS
- Consider using database encryption at rest for sensitive configuration data
