# Quick Start Guide

Get Nimbus up and running in 5 minutes.

## Prerequisites

- Go 1.23 or later
- Docker (for PostgreSQL)

## Steps

### 1. One-Command Setup

From the project root, run:

```bash
make setup-dev
```

This will:
- Install sqlc
- Start PostgreSQL in Docker
- Run database migrations
- Generate sqlc code

### 2. Start the Server

```bash
make dev
```

The server will start on `http://localhost:8080`

### 3. Test the API

Check health:
```bash
curl http://localhost:8080/health
```

Create a movie:
```bash
curl -X POST http://localhost:8080/api/media \
  -H "Content-Type: application/json" \
  -d '{
    "kind": "movie",
    "title": "The Matrix",
    "year": 1999,
    "external_ids": {"tmdb": "603"},
    "metadata": {"runtime": 136, "language": "en"}
  }'
```

List movies:
```bash
curl http://localhost:8080/api/movies
```

Get configuration:
```bash
curl http://localhost:8080/api/config/library.root_path
```

## What's Next?

- Read the [full README](README.md) for detailed documentation
- Explore the API endpoints
- Check out the database schema in `internal/db/migrations/`
- Review the code structure in `internal/`

## Troubleshooting

### PostgreSQL connection failed

Make sure Docker is running and PostgreSQL is started:
```bash
make docker-up
```

### sqlc not found

Install development tools:
```bash
make install-tools
```

### Port 8080 already in use

Change the port in your `.env` file or set the `PORT` environment variable:
```bash
PORT=3000 make dev
```

### Database migration errors

Drop and recreate the database:
```bash
make docker-down
make docker-up
make migrate-up
```
