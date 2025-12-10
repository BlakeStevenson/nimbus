# Nimbus

A modern, extensible media management and automation server built with Go and React.

## Overview

Nimbus is a self-hosted media automation platform that helps you organize, download, and manage your media library. It features a plugin-based architecture for maximum extensibility and a modern web interface built with React.

## Features

### Core Features

- **Media Library Management**: Organize movies, TV shows, music, and more
- **Automated Downloads**: Integration with NZB downloaders via plugins
- **Metadata Integration**: Automatic metadata fetching from TMDB
- **Interactive Search**: Browse and download releases from multiple indexers
- **Plugin System**: Extend functionality with custom plugins
- **Modern Web UI**: Responsive React-based interface with real-time updates
- **User Authentication**: Secure login with multiple authentication providers

### Media Features

- Hierarchical media organization (Series → Seasons → Episodes)
- Automatic file detection and association
- Episode tracking with download status
- Quality and codec filtering
- Interactive search with sortable results
- Real-time download progress tracking

### Plugin Capabilities

- Custom REST API endpoints
- UI extensions and navigation items
- Event handling (downloads, imports, etc.)
- External service integrations

## Tech Stack

### Backend

- **Language**: Go 1.23+
- **Database**: PostgreSQL (via pgx)
- **Router**: Chi
- **Plugin System**: HashiCorp go-plugin
- **Logging**: Uber Zap

### Frontend

- **Framework**: React 18 with TypeScript
- **Build Tool**: Vite
- **UI Library**: Radix UI
- **Styling**: Tailwind CSS
- **State Management**: TanStack Query
- **Routing**: React Router

## Getting Started

### Prerequisites

- Go 1.23 or higher
- Node.js 18+ and npm
- PostgreSQL 14+
- Make (optional, for using Makefile)

### Installation

1. **Clone the repository**

```bash
git clone https://github.com/blakestevenson/nimbus.git
cd nimbus
```

2. **Set up environment variables**

```bash
cp .env.example .env
# Edit .env with your configuration
```

Required environment variables:

```env
DATABASE_URL=postgresql://user:password@localhost:5432/nimbus
JWT_SECRET=your-secret-key-here
HOST=0.0.0.0
PORT=8080
ENVIRONMENT=development
ENABLE_PLUGINS=true
PLUGINS_DIR=./plugins
```

3. **Initialize the database**

```bash
# Create the database
createdb nimbus

# Run migrations
make migrate
# Or: go run cmd/server/main.go migrate
```

4. **Build the frontend**

```bash
cd frontend
npm install
npm run build
cd ..
```

5. **Build plugins (optional)**

```bash
cd plugins/tmdb-plugin && ./build.sh && cd ../..
cd plugins/usenet-indexer && ./build.sh && cd ../..
cd plugins/nzb-downloader && ./build.sh && cd ../..
```

6. **Run the server**

```bash
# Using Make
make run

# Or directly
go run cmd/server/main.go

# Or build and run binary
go build -o nimbus cmd/server/main.go
./nimbus
```

The server will start on `http://localhost:8080`

### Development Mode

For development with hot-reload:

**Backend:**
```bash
# Install air for hot-reload
go install github.com/cosmtrek/air@latest

# Run with air
air
```

**Frontend:**
```bash
cd frontend
npm run dev
```

The frontend dev server runs on `http://localhost:5173` and proxies API requests to the backend.

## Usage

### First Time Setup

1. Navigate to `http://localhost:8080`
2. Create an admin account
3. Configure plugins in the Plugins page
4. Add media to your library

### Adding Media

- **Manual**: Add media items through the web interface
- **Import**: Scan directories for media files
- **Automatic**: Use plugins to automatically track and download media

### Searching and Downloading

1. Navigate to a TV season or episode
2. Click "Search Releases"
3. Use filters to narrow down results (quality, video codec, audio codec)
4. Click column headers to sort
5. Click "Download" on your preferred release
6. Monitor progress in the Downloads page

## Plugin Development

Nimbus uses a plugin system to extend functionality.

### Available Plugins

- **tmdb-plugin**: TMDB metadata integration
- **usenet-indexer**: NZB indexer support (Newznab API)
- **nzb-downloader**: NZB download client
- **example-plugin**: Reference implementation

### Creating a Plugin

1. Create a new directory in `plugins/`
2. Implement the plugin interface in Go
3. Add a `manifest.json`
4. Build with `./build.sh`


## Project Structure

```
nimbus/
├── cmd/
│   └── server/          # Main application entry point
├── internal/
│   ├── auth/            # Authentication system
│   ├── config/          # Configuration management
│   ├── db/              # Database layer
│   ├── downloader/      # Download orchestration
│   ├── http/            # HTTP server and routes
│   ├── importer/        # Media file importing
│   ├── library/         # Media library management
│   ├── media/           # Media CRUD operations
│   └── plugins/         # Plugin system
├── plugins/
│   ├── tmdb-plugin/     # TMDB integration
│   ├── usenet-indexer/  # Usenet indexer support
│   ├── nzb-downloader/  # NZB download client
│   └── example-plugin/  # Example plugin
├── frontend/
│   └── src/
│       ├── components/  # React components
│       ├── lib/         # API clients and utilities
│       └── pages/       # Page components
```

## API

The server exposes a REST API for all operations:

- `/api/auth/*` - Authentication endpoints
- `/api/media/*` - Media library operations
- `/api/downloads/*` - Download management
- `/api/plugins/*` - Plugin management
- `/api/config/*` - Configuration

Plugins can extend the API with custom endpoints under `/api/plugins/{plugin-id}/*`

## Configuration

### Server Configuration

Configure via environment variables or `.env` file:

- `DATABASE_URL` - PostgreSQL connection string
- `JWT_SECRET` - Secret for JWT token generation
- `HOST` - Server bind address (default: 0.0.0.0)
- `PORT` - Server port (default: 8080)
- `ENVIRONMENT` - `development` or `production`
- `ENABLE_PLUGINS` - Enable plugin system (default: false)
- `PLUGINS_DIR` - Directory containing plugins (default: ./plugins)

### Plugin Configuration

Each plugin can store configuration in the database via the config store API.

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add some amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## License

This project is licensed under the MIT License - see the LICENSE file for details.

## Acknowledgments

- [Radix UI](https://www.radix-ui.com/) for accessible UI components
- [TanStack Query](https://tanstack.com/query) for powerful data synchronization
- [HashiCorp go-plugin](https://github.com/hashicorp/go-plugin) for the plugin system
- [Chi](https://github.com/go-chi/chi) for the HTTP router

## Support

For issues, questions, or contributions, please open an issue on GitHub.
