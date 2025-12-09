# Nimbus

Nimbus is a self-hosted, all-in-one media automation platform designed to eventually replace tools like Sonarr, Radarr, Lidarr, and Readarr. This repository contains the core Go backend with a clean, extensible architecture ready for future plugin support and a React frontend.

## Architecture

Nimbus is built with extensibility and maintainability in mind:

- **Go backend** with clean architecture and domain separation
- **PostgreSQL** for reliable data storage with JSONB for flexible metadata
- **chi** HTTP router for fast, lightweight routing
- **sqlc** for type-safe SQL queries (schema-first approach)
- **Plugin-ready** architecture with extension points for future customization

## Tech Stack

- **Language**: Go 1.23+
- **Database**: PostgreSQL 16+
- **HTTP Router**: chi v5
- **Query Layer**: sqlc
- **Logging**: zap (structured logging)
- **Configuration**: Environment variables

## Features

### Core Media Management

- Generic media item system supporting:
  - Movies
  - TV series, seasons, and episodes
  - Music artists, albums, and tracks
  - Books and book series
- Extensible metadata storage using JSONB
- Hierarchical relationships (episodes under series, tracks under albums, etc.)
- External ID tracking (TMDB, TVDB, MusicBrainz, ISBN, etc.)

### Configuration System

- Database-backed configuration storage
- Type-safe configuration access (string, int, bool, map)
- Namespace support for plugin configurations
- REST API for configuration management

### REST API

Full-featured JSON REST API with:
- Generic media CRUD operations
- Type-specific convenience endpoints
- Search and filtering
- Pagination support
- Configuration management endpoints

## Project Structure

```
nimbus/
├── cmd/
│   └── server/          # Main entry point
├── internal/
│   ├── config/          # Configuration loading
│   ├── configstore/     # Database-backed config store
│   ├── db/              # Database layer
│   │   ├── migrations/  # SQL migrations
│   │   ├── queries/     # SQL query definitions
│   │   ├── generated/   # sqlc-generated code
│   │   └── sqlc.yaml    # sqlc configuration
│   ├── http/            # HTTP layer
│   │   └── handlers/    # HTTP handlers
│   ├── logging/         # Logging setup
│   └── media/           # Media business logic
├── Makefile             # Development tasks
├── go.mod               # Go module definition
└── README.md            # This file
```

## Getting Started

### Prerequisites

- Go 1.23 or later
- PostgreSQL 16+ (or Docker)
- sqlc (will be installed by `make install-tools`)

### Quick Start

1. **Install tools and set up the development environment:**

```bash
make setup-dev
```

This will:
- Install sqlc and other development tools
- Start PostgreSQL in Docker
- Run database migrations
- Generate sqlc code

2. **Start the development server:**

```bash
make dev
```

The server will start on `http://localhost:8080`

### Manual Setup

If you prefer to set up manually:

1. **Install development tools:**

```bash
make install-tools
```

2. **Start PostgreSQL:**

Using Docker:
```bash
make docker-up
```

Or use your own PostgreSQL instance and set the `DATABASE_URL` environment variable.

3. **Run migrations:**

```bash
DATABASE_URL="postgres://nimbus:nimbus@localhost:5432/nimbus?sslmode=disable" make migrate-up
```

4. **Generate sqlc code:**

```bash
make sqlc-generate
```

5. **Run the server:**

```bash
make dev
```

### Environment Variables

Configure the application using these environment variables:

- `DATABASE_URL`: PostgreSQL connection string (default: `postgres://localhost:5432/nimbus?sslmode=disable`)
- `PORT`: HTTP server port (default: `8080`)
- `HOST`: HTTP server host (default: `0.0.0.0`)
- `ENVIRONMENT`: Environment mode - `development` or `production` (default: `development`)

Create a `.env` file in the project root for local development:

```env
DATABASE_URL=postgres://nimbus:nimbus@localhost:5432/nimbus?sslmode=disable
PORT=8080
ENVIRONMENT=development
```

## API Documentation

### Health Check

```
GET /health
```

Returns server health status.

### Media Items

#### List media items

```
GET /api/media?kind={kind}&q={search}&parent_id={parent_id}&limit={limit}&offset={offset}
```

Query parameters:
- `kind` (optional): Filter by media kind (movie, tv_series, tv_episode, music_album, music_track, book, etc.)
- `q` (optional): Search query (searches title and sort_title)
- `parent_id` (optional): Filter by parent ID (e.g., episodes of a series)
- `limit` (optional): Number of items per page (default: 20, max: 100)
- `offset` (optional): Pagination offset (default: 0)

#### Get a media item

```
GET /api/media/{id}
```

#### Create a media item

```
POST /api/media
Content-Type: application/json

{
  "kind": "movie",
  "title": "The Matrix",
  "sort_title": "Matrix, The",
  "year": 1999,
  "external_ids": {
    "tmdb": "603",
    "imdb": "tt0133093"
  },
  "metadata": {
    "runtime": 136,
    "language": "en",
    "genres": ["Action", "Sci-Fi"]
  }
}
```

#### Update a media item

```
PUT /api/media/{id}
Content-Type: application/json

{
  "title": "The Matrix",
  "year": 1999,
  "metadata": {
    "runtime": 136
  }
}
```

#### Delete a media item

```
DELETE /api/media/{id}
```

### Type-Specific Endpoints

#### Movies

```
GET /api/movies?q={search}&limit={limit}&offset={offset}
```

#### TV Series

```
GET /api/tv/series?q={search}&limit={limit}&offset={offset}
```

#### TV Episodes

```
GET /api/tv/series/{id}/episodes
```

Lists all episodes for a TV series.

#### Books

```
GET /api/books?q={search}&limit={limit}&offset={offset}
```

### Configuration

#### Get configuration value

```
GET /api/config/{key}
```

Example:
```
GET /api/config/library.root_path
```

#### Set configuration value

```
PUT /api/config/{key}
Content-Type: application/json

{
  "value": "/media"
}
```

#### List all configuration

```
GET /api/config
```

Optional query parameter:
- `prefix`: Filter by key prefix (e.g., `prefix=library.`)

#### Delete configuration

```
DELETE /api/config/{key}
```

## Development

### Running Tests

```bash
make test
```

This runs all tests with race detection and generates a coverage report.

### Formatting Code

```bash
make fmt
```

### Generating sqlc Code

After modifying SQL queries or schema:

```bash
make sqlc-generate
```

### Database Migrations

To add a new migration, create a new SQL file in `internal/db/migrations/`:

```sql
-- internal/db/migrations/0002_add_something.sql
ALTER TABLE media_items ADD COLUMN new_field TEXT;
```

Then run:

```bash
make migrate-up
```

Note: For production use, consider using a proper migration tool like [golang-migrate](https://github.com/golang-migrate/migrate).

### Cleaning Build Artifacts

```bash
make clean
```

## Media Types

Nimbus supports the following media kinds out of the box:

- `movie` - Movies
- `tv_series` - TV series
- `tv_season` - TV seasons (child of tv_series)
- `tv_episode` - TV episodes (child of tv_season or tv_series)
- `music_artist` - Music artists
- `music_album` - Music albums (child of music_artist)
- `music_track` - Music tracks (child of music_album)
- `book` - Books
- `book_series` - Book series

The architecture is designed to support additional media types through plugins in the future.

## Database Schema

### media_items

The core table for all media:

- `id` - Unique identifier
- `kind` - Media type (movie, tv_series, etc.)
- `title` - Display title
- `sort_title` - Title for sorting
- `year` - Release year (optional)
- `external_ids` - JSONB field for external IDs (TMDB, IMDB, etc.)
- `metadata` - JSONB field for type-specific metadata
- `parent_id` - Reference to parent media item (for hierarchies)
- `created_at` - Creation timestamp
- `updated_at` - Last update timestamp

### config

Configuration storage:

- `key` - Configuration key (e.g., "library.root_path")
- `value` - JSONB value
- `updated_at` - Last update timestamp

## Extensibility

The architecture is designed with future plugin support in mind:

1. **Generic media items** - The `kind` field and JSONB metadata allow new media types without schema changes
2. **Configuration system** - Plugin settings can use the `plugin.<plugin_id>.<setting>` namespace
3. **Service layer** - Business logic is isolated and can be extended
4. **Clean architecture** - Clear separation of concerns makes it easy to add new features

## Roadmap

- [ ] Implement migration tool integration
- [ ] Add authentication and authorization
- [ ] Plugin system architecture
- [ ] React frontend
- [ ] Media file scanning and monitoring
- [ ] Download client integrations
- [ ] Indexer integrations
- [ ] Automated media management workflows
- [ ] Notification system
- [ ] User management and permissions

## Contributing

Contributions are welcome! Please ensure:

- Code is formatted with `go fmt`
- Tests pass with `make test`
- New features include tests
- Documentation is updated

## License

MIT License - see LICENSE file for details
