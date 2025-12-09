# NZB Downloader Plugin

A Nimbus plugin that downloads NZB files from Usenet with queue management, monitoring, and multi-server support.

## Features

- **NZB File Support**: Parse and download NZB files from URLs or uploads
- **Multi-Server Support**: Configure multiple NNTP servers with priority
- **Download Queue**: Automatic queue management with concurrent downloads
- **Real-time Monitoring**: Track download progress, speed, and ETA
- **SSL/TLS Support**: Secure connections to NNTP servers
- **Web UI**: Full download management interface

## Configuration

### NNTP Servers

Configure one or more NNTP servers with:
- **Name**: Friendly name for the server
- **Host**: NNTP server hostname
- **Port**: Port number (typically 119 for standard, 563 for SSL)
- **Username/Password**: Authentication credentials
- **Use SSL**: Enable SSL/TLS connection
- **Connections**: Number of concurrent connections (default: 10)
- **Priority**: Server priority (lower = higher priority)
- **Enabled**: Enable/disable the server

### Download Settings

- **Download Directory**: Where to save downloaded files (default: `/tmp/nzb-downloads`)
- **Max Concurrent Downloads**: Maximum simultaneous downloads (default: 3)

## API Endpoints

### Server Management

- `GET /api/plugins/nzb-downloader/servers` - List all NNTP servers
- `POST /api/plugins/nzb-downloader/servers` - Add new server
- `PUT /api/plugins/nzb-downloader/servers/{id}` - Update server
- `DELETE /api/plugins/nzb-downloader/servers/{id}` - Delete server
- `POST /api/plugins/nzb-downloader/servers/{id}/test` - Test server connection

### Download Management

- `GET /api/plugins/nzb-downloader/downloads` - List all downloads
- `POST /api/plugins/nzb-downloader/downloads` - Add new download (NZB URL or file)
- `DELETE /api/plugins/nzb-downloader/downloads/{id}` - Remove download
- `POST /api/plugins/nzb-downloader/downloads/{id}/pause` - Pause download
- `POST /api/plugins/nzb-downloader/downloads/{id}/resume` - Resume download
- `POST /api/plugins/nzb-downloader/downloads/{id}/retry` - Retry failed download

### Configuration

- `GET /api/plugins/nzb-downloader/config` - Get configuration
- `POST /api/plugins/nzb-downloader/config` - Update configuration

## Usage

### Adding Downloads

**Via URL:**
```bash
curl -X POST http://localhost:8080/api/plugins/nzb-downloader/downloads \
  -H "Content-Type: application/json" \
  -H "Cookie: session=..." \
  -d '{"url": "https://example.com/file.nzb", "name": "My Download"}'
```

**Via File Upload:**
Upload an NZB file through the UI or send the NZB file contents as the request body.

### Download Status

Downloads go through these states:
- **queued**: Waiting to start
- **downloading**: Currently downloading
- **paused**: Manually paused
- **completed**: Successfully completed
- **failed**: Failed with error

## Implementation Details

### NZB Parser

Custom XML parser that extracts:
- File metadata (poster, date, subject)
- Newsgroup information
- Segment details (message IDs, sizes, numbers)
- Total file sizes

### NNTP Client

Custom NNTP client supporting:
- Standard and SSL/TLS connections
- AUTHINFO authentication
- Article retrieval by message ID
- Group selection
- Connection pooling (multiple connections per server)

### Download Queue

- Concurrent download management
- Automatic queue processing
- Progress tracking
- Error handling and retry logic
- Real-time statistics (speed, ETA)

## Installation

1. Build the plugin:
   ```bash
   cd plugins/nzb-downloader
   ./build.sh
   ```

2. Install to Nimbus:
   ```bash
   mkdir -p /var/lib/nimbus/plugins/nzb-downloader
   cp nzb-downloader /var/lib/nimbus/plugins/nzb-downloader/
   cp manifest.json /var/lib/nimbus/plugins/nzb-downloader/
   cp -r web /var/lib/nimbus/plugins/nzb-downloader/
   ```

3. Enable plugins and restart Nimbus server

4. Navigate to the Plugins page and enable "NZB Downloader"

## Requirements

- At least one NNTP server subscription
- Valid NNTP credentials
- Sufficient disk space for downloads

## Limitations

- Currently implements basic yEnc decoding (downloads raw articles)
- No PAR2 verification/repair yet
- No automatic extraction of archives
- Single-threaded segment downloads per file

## Future Enhancements

- Full yEnc decoding
- PAR2 verification and repair
- Automatic archive extraction
- Multi-threaded segment downloading
- Server failover and retry logic
- Bandwidth limiting
- Schedule downloads
- Integration with Usenet Indexer plugin

## License

Part of the Nimbus media suite project.
