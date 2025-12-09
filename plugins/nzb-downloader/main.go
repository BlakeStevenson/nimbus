package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/blakestevenson/nimbus/internal/plugins"
	"github.com/hashicorp/go-plugin"
)

// NZBDownloaderPlugin implements the MediaSuitePlugin interface
type NZBDownloaderPlugin struct {
	downloadManager *DownloadManager
}

// Configuration keys
const (
	configPrefix      = "plugins.nzb-downloader"
	configServers     = configPrefix + ".servers"
	configDownloadDir = configPrefix + ".download_dir"
	configConnections = configPrefix + ".connections"
)

// NNTPServer represents an NNTP server configuration
type NNTPServer struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Host        string `json:"host"`
	Port        int    `json:"port"`
	Username    string `json:"username"`
	Password    string `json:"password"`
	UseSSL      bool   `json:"use_ssl"`
	Enabled     bool   `json:"enabled"`
	Connections int    `json:"connections"`
	Priority    int    `json:"priority"`
}

// Download represents a download job
type Download struct {
	ID              string       `json:"id"`
	Name            string       `json:"name"`
	Status          string       `json:"status"` // queued, downloading, paused, completed, failed
	Progress        float64      `json:"progress"`
	TotalBytes      int64        `json:"total_bytes"`
	DownloadedBytes int64        `json:"downloaded_bytes"`
	Speed           int64        `json:"speed"` // bytes per second
	ETA             int64        `json:"eta"`   // seconds
	AddedAt         time.Time    `json:"added_at"`
	StartedAt       *time.Time   `json:"started_at,omitempty"`
	CompletedAt     *time.Time   `json:"completed_at,omitempty"`
	Error           string       `json:"error,omitempty"`
	NZBData         *NZB         `json:"-"`
	Servers         []NNTPServer `json:"-"`              // Snapshot of enabled servers at time of creation
	DownloadDir     string       `json:"-"`              // Download directory
	Logs            []string     `json:"logs,omitempty"` // Recent log messages
	logMu           sync.Mutex   `json:"-"`
}

// AddLog adds a log message to the download
func (d *Download) AddLog(msg string) {
	d.logMu.Lock()
	defer d.logMu.Unlock()

	timestamp := time.Now().Format("15:04:05")
	d.Logs = append(d.Logs, fmt.Sprintf("[%s] %s", timestamp, msg))

	// Keep only last 50 log lines
	if len(d.Logs) > 50 {
		d.Logs = d.Logs[len(d.Logs)-50:]
	}

	// Also write to stderr for debugging
	fmt.Fprintf(os.Stderr, "[%s] %s\n", d.Name, msg)
}

// DownloadManager manages the download queue
type DownloadManager struct {
	mu        sync.RWMutex
	downloads map[string]*Download
	queue     []string
	active    map[string]bool
	maxActive int
	ctx       context.Context
	cancel    context.CancelFunc
}

// NewDownloadManager creates a new download manager
func NewDownloadManager(maxActive int) *DownloadManager {
	ctx, cancel := context.WithCancel(context.Background())
	return &DownloadManager{
		downloads: make(map[string]*Download),
		queue:     []string{},
		active:    make(map[string]bool),
		maxActive: maxActive,
		ctx:       ctx,
		cancel:    cancel,
	}
}

// Metadata returns plugin metadata
func (p *NZBDownloaderPlugin) Metadata(ctx context.Context) (*plugins.PluginMetadata, error) {
	return &plugins.PluginMetadata{
		ID:           "nzb-downloader",
		Name:         "NZB Downloader",
		Version:      "0.1.0",
		Description:  "Download NZB files from Usenet with queue management and monitoring",
		Capabilities: []string{"api", "ui"},
	}, nil
}

// APIRoutes returns the HTTP routes this plugin provides
func (p *NZBDownloaderPlugin) APIRoutes(ctx context.Context) ([]plugins.RouteDescriptor, error) {
	return []plugins.RouteDescriptor{
		// Server management
		{Method: "GET", Path: "/api/plugins/nzb-downloader/servers", Auth: "session"},
		{Method: "POST", Path: "/api/plugins/nzb-downloader/servers", Auth: "session"},
		{Method: "PUT", Path: "/api/plugins/nzb-downloader/servers/{id}", Auth: "session"},
		{Method: "DELETE", Path: "/api/plugins/nzb-downloader/servers/{id}", Auth: "session"},
		{Method: "POST", Path: "/api/plugins/nzb-downloader/servers/{id}/test", Auth: "session"},
		// Download management
		{Method: "GET", Path: "/api/plugins/nzb-downloader/downloads", Auth: "session"},
		{Method: "GET", Path: "/api/plugins/nzb-downloader/downloads/stream", Auth: "session"},
		{Method: "POST", Path: "/api/plugins/nzb-downloader/downloads", Auth: "session"},
		{Method: "POST", Path: "/api/plugins/nzb-downloader/downloads/move", Auth: "session"},
		{Method: "DELETE", Path: "/api/plugins/nzb-downloader/downloads/{id}", Auth: "session"},
		{Method: "POST", Path: "/api/plugins/nzb-downloader/downloads/{id}/pause", Auth: "session"},
		{Method: "POST", Path: "/api/plugins/nzb-downloader/downloads/{id}/resume", Auth: "session"},
		{Method: "POST", Path: "/api/plugins/nzb-downloader/downloads/{id}/retry", Auth: "session"},
		// Configuration
		{Method: "GET", Path: "/api/plugins/nzb-downloader/config", Auth: "session"},
		{Method: "POST", Path: "/api/plugins/nzb-downloader/config", Auth: "session"},
	}, nil
}

// HandleAPI handles HTTP requests for this plugin's routes
func (p *NZBDownloaderPlugin) HandleAPI(ctx context.Context, req *plugins.PluginHTTPRequest) (*plugins.PluginHTTPResponse, error) {
	// Server management
	if req.Path == "/api/plugins/nzb-downloader/servers" {
		if req.Method == "GET" {
			return p.handleListServers(ctx, req)
		}
		return p.handleCreateServer(ctx, req)
	}

	// Server operations with ID
	if strings.HasPrefix(req.Path, "/api/plugins/nzb-downloader/servers/") {
		parts := strings.Split(req.Path, "/")
		if len(parts) >= 6 {
			serverID := parts[5]

			switch req.Method {
			case "DELETE":
				return p.handleDeleteServer(ctx, req, serverID)
			case "PUT":
				return p.handleUpdateServer(ctx, req, serverID)
			case "POST":
				// Check if it's a test request
				if len(parts) == 7 && parts[6] == "test" {
					return p.handleTestServer(ctx, req, serverID)
				}
			}
		}
	}

	// Download management
	if req.Path == "/api/plugins/nzb-downloader/downloads" {
		if req.Method == "GET" {
			return p.handleListDownloads(ctx, req)
		}
		return p.handleAddDownload(ctx, req)
	}

	if req.Path == "/api/plugins/nzb-downloader/downloads/move" {
		return p.handleMoveDownloads(ctx, req)
	}

	// Download operations with ID
	if strings.HasPrefix(req.Path, "/api/plugins/nzb-downloader/downloads/") && req.Path != "/api/plugins/nzb-downloader/downloads/move" {
		parts := strings.Split(req.Path, "/")
		if len(parts) >= 6 {
			downloadID := parts[5]

			switch req.Method {
			case "DELETE":
				return p.handleDeleteDownload(ctx, req, downloadID)
			}
		}
	}

	// Configuration
	if req.Path == "/api/plugins/nzb-downloader/config" {
		if req.Method == "GET" {
			return p.handleGetConfig(ctx, req)
		}
		return p.handleSetConfig(ctx, req)
	}

	return jsonResponse(http.StatusNotFound, map[string]string{"error": "Not found"})
}

// Server Management Handlers

func (p *NZBDownloaderPlugin) handleListServers(ctx context.Context, req *plugins.PluginHTTPRequest) (*plugins.PluginHTTPResponse, error) {
	if req.SDK == nil {
		return jsonResponse(http.StatusInternalServerError, map[string]string{"error": "SDK not available"})
	}

	servers, err := p.getServers(ctx, req.SDK)
	if err != nil {
		return jsonResponse(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	// Ensure we never return nil for servers array
	if servers == nil {
		servers = []NNTPServer{}
	}

	// DEBUG: Log raw server data
	fmt.Fprintf(os.Stderr, "handleListServers: Retrieved %d servers\n", len(servers))
	for i, srv := range servers {
		fmt.Fprintf(os.Stderr, "  Server %d: ID=%s, Name=%s, Enabled=%v\n", i, srv.ID, srv.Name, srv.Enabled)
	}

	// Mask passwords
	for i := range servers {
		servers[i].Password = maskPassword(servers[i].Password)
	}

	return jsonResponse(http.StatusOK, map[string]interface{}{"servers": servers})
}

func (p *NZBDownloaderPlugin) handleCreateServer(ctx context.Context, req *plugins.PluginHTTPRequest) (*plugins.PluginHTTPResponse, error) {
	if req.SDK == nil {
		return jsonResponse(http.StatusInternalServerError, map[string]string{"error": "SDK not available"})
	}

	var server NNTPServer
	if err := json.Unmarshal(req.Body, &server); err != nil {
		return jsonResponse(http.StatusBadRequest, map[string]string{"error": "Invalid JSON"})
	}

	if server.ID == "" {
		server.ID = generateID(server.Name)
	}

	servers, err := p.getServers(ctx, req.SDK)
	if err != nil {
		return jsonResponse(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	// Ensure servers is initialized
	if servers == nil {
		servers = []NNTPServer{}
	}

	for _, existing := range servers {
		if existing.ID == server.ID {
			return jsonResponse(http.StatusConflict, map[string]string{"error": "Server ID already exists"})
		}
	}

	servers = append(servers, server)

	if err := p.saveServers(ctx, req.SDK, servers); err != nil {
		return jsonResponse(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return jsonResponse(http.StatusCreated, server)
}

func (p *NZBDownloaderPlugin) handleUpdateServer(ctx context.Context, req *plugins.PluginHTTPRequest, serverID string) (*plugins.PluginHTTPResponse, error) {
	if req.SDK == nil {
		return jsonResponse(http.StatusInternalServerError, map[string]string{"error": "SDK not available"})
	}

	var updatedServer NNTPServer
	if err := json.Unmarshal(req.Body, &updatedServer); err != nil {
		return jsonResponse(http.StatusBadRequest, map[string]string{"error": "Invalid JSON"})
	}

	servers, err := p.getServers(ctx, req.SDK)
	if err != nil {
		return jsonResponse(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	found := false
	for i, server := range servers {
		if server.ID == serverID {
			updatedServer.ID = serverID // Ensure ID doesn't change
			servers[i] = updatedServer
			found = true
			break
		}
	}

	if !found {
		return jsonResponse(http.StatusNotFound, map[string]string{"error": "Server not found"})
	}

	if err := p.saveServers(ctx, req.SDK, servers); err != nil {
		return jsonResponse(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return jsonResponse(http.StatusOK, updatedServer)
}

func (p *NZBDownloaderPlugin) handleDeleteServer(ctx context.Context, req *plugins.PluginHTTPRequest, serverID string) (*plugins.PluginHTTPResponse, error) {
	if req.SDK == nil {
		return jsonResponse(http.StatusInternalServerError, map[string]string{"error": "SDK not available"})
	}

	servers, err := p.getServers(ctx, req.SDK)
	if err != nil {
		return jsonResponse(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	found := false
	newServers := []NNTPServer{}
	for _, server := range servers {
		if server.ID != serverID {
			newServers = append(newServers, server)
		} else {
			found = true
		}
	}

	if !found {
		return jsonResponse(http.StatusNotFound, map[string]string{"error": "Server not found"})
	}

	if err := p.saveServers(ctx, req.SDK, newServers); err != nil {
		return jsonResponse(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return jsonResponse(http.StatusOK, map[string]string{"message": "Server deleted"})
}

func (p *NZBDownloaderPlugin) handleTestServer(ctx context.Context, req *plugins.PluginHTTPRequest, serverID string) (*plugins.PluginHTTPResponse, error) {
	if req.SDK == nil {
		return jsonResponse(http.StatusInternalServerError, map[string]string{"error": "SDK not available"})
	}

	servers, err := p.getServers(ctx, req.SDK)
	if err != nil {
		return jsonResponse(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	var server *NNTPServer
	for _, s := range servers {
		if s.ID == serverID {
			server = &s
			break
		}
	}

	if server == nil {
		return jsonResponse(http.StatusNotFound, map[string]string{"error": "Server not found"})
	}

	// Try to connect and authenticate
	conn, err := DialNNTP(server.Host, server.Port, server.UseSSL)
	if err != nil {
		return jsonResponse(http.StatusOK, map[string]interface{}{
			"success": false,
			"error":   fmt.Sprintf("Connection failed: %v", err),
		})
	}
	defer conn.Close()

	if err := conn.Authenticate(server.Username, server.Password); err != nil {
		return jsonResponse(http.StatusOK, map[string]interface{}{
			"success": false,
			"error":   fmt.Sprintf("Authentication failed: %v", err),
		})
	}

	return jsonResponse(http.StatusOK, map[string]interface{}{
		"success": true,
		"message": "Connection successful",
	})
}

// Download Management Handlers

func (p *NZBDownloaderPlugin) handleListDownloads(ctx context.Context, req *plugins.PluginHTTPRequest) (*plugins.PluginHTTPResponse, error) {
	p.downloadManager.mu.RLock()
	defer p.downloadManager.mu.RUnlock()

	downloads := make([]*Download, 0, len(p.downloadManager.downloads))
	for _, dl := range p.downloadManager.downloads {
		downloads = append(downloads, dl)
	}

	return jsonResponse(http.StatusOK, map[string]interface{}{"downloads": downloads})
}

func (p *NZBDownloaderPlugin) handleMoveDownloads(ctx context.Context, req *plugins.PluginHTTPRequest) (*plugins.PluginHTTPResponse, error) {
	var input struct {
		DownloadIDs []string `json:"download_ids"`
		Direction   string   `json:"direction"`
	}

	if err := json.Unmarshal(req.Body, &input); err != nil {
		return jsonResponse(http.StatusBadRequest, map[string]string{"error": "Invalid JSON"})
	}

	if len(input.DownloadIDs) == 0 {
		return jsonResponse(http.StatusBadRequest, map[string]string{"error": "No download IDs provided"})
	}

	p.downloadManager.mu.Lock()
	defer p.downloadManager.mu.Unlock()

	// Find indices of downloads to move
	indices := make([]int, 0)
	for _, id := range input.DownloadIDs {
		for i, queueID := range p.downloadManager.queue {
			if queueID == id {
				indices = append(indices, i)
				break
			}
		}
	}

	if len(indices) == 0 {
		return jsonResponse(http.StatusNotFound, map[string]string{"error": "Downloads not found in queue"})
	}

	// Sort indices for consistent movement
	sort.Ints(indices)

	switch input.Direction {
	case "top":
		// Move to top in order
		moved := make([]string, 0, len(indices))
		remaining := make([]string, 0, len(p.downloadManager.queue)-len(indices))

		for _, idx := range indices {
			moved = append(moved, p.downloadManager.queue[idx])
		}
		for i, id := range p.downloadManager.queue {
			isSelected := false
			for _, idx := range indices {
				if i == idx {
					isSelected = true
					break
				}
			}
			if !isSelected {
				remaining = append(remaining, id)
			}
		}
		p.downloadManager.queue = append(moved, remaining...)

	case "bottom":
		// Move to bottom in order
		remaining := make([]string, 0, len(p.downloadManager.queue)-len(indices))
		moved := make([]string, 0, len(indices))

		for i, id := range p.downloadManager.queue {
			isSelected := false
			for _, idx := range indices {
				if i == idx {
					isSelected = true
					moved = append(moved, id)
					break
				}
			}
			if !isSelected {
				remaining = append(remaining, id)
			}
		}
		p.downloadManager.queue = append(remaining, moved...)

	case "up":
		// Move up by one position
		for i := 0; i < len(indices); i++ {
			idx := indices[i]
			if idx > 0 {
				// Swap with previous item
				p.downloadManager.queue[idx], p.downloadManager.queue[idx-1] = p.downloadManager.queue[idx-1], p.downloadManager.queue[idx]
				// Update index for next iteration
				indices[i]--
			}
		}

	case "down":
		// Move down by one position (reverse order to avoid conflicts)
		for i := len(indices) - 1; i >= 0; i-- {
			idx := indices[i]
			if idx < len(p.downloadManager.queue)-1 {
				// Swap with next item
				p.downloadManager.queue[idx], p.downloadManager.queue[idx+1] = p.downloadManager.queue[idx+1], p.downloadManager.queue[idx]
			}
		}

	default:
		return jsonResponse(http.StatusBadRequest, map[string]string{"error": "Invalid direction"})
	}

	return jsonResponse(http.StatusOK, map[string]string{"message": "Downloads moved successfully"})
}

func (p *NZBDownloaderPlugin) handleDeleteDownload(ctx context.Context, req *plugins.PluginHTTPRequest, downloadID string) (*plugins.PluginHTTPResponse, error) {
	p.downloadManager.mu.Lock()
	defer p.downloadManager.mu.Unlock()

	// Check if download exists
	if _, exists := p.downloadManager.downloads[downloadID]; !exists {
		return jsonResponse(http.StatusNotFound, map[string]string{"error": "Download not found"})
	}

	// Remove from downloads map
	delete(p.downloadManager.downloads, downloadID)

	// Remove from active downloads
	delete(p.downloadManager.active, downloadID)

	// Remove from queue
	newQueue := make([]string, 0, len(p.downloadManager.queue))
	for _, id := range p.downloadManager.queue {
		if id != downloadID {
			newQueue = append(newQueue, id)
		}
	}
	p.downloadManager.queue = newQueue

	return jsonResponse(http.StatusOK, map[string]string{"message": "Download deleted successfully"})
}

func (p *NZBDownloaderPlugin) handleAddDownload(ctx context.Context, req *plugins.PluginHTTPRequest) (*plugins.PluginHTTPResponse, error) {
	if req.SDK == nil {
		return jsonResponse(http.StatusInternalServerError, map[string]string{"error": "SDK not available"})
	}

	// Parse multipart form for NZB file upload or URL
	var nzbData *NZB
	var downloadName string

	// Check if it's a URL or file upload
	var input struct {
		URL  string `json:"url"`
		NZB  string `json:"nzb"`
		Name string `json:"name"`
	}

	var err error
	if jsonErr := json.Unmarshal(req.Body, &input); jsonErr == nil && (input.URL != "" || input.NZB != "") {
		if input.URL != "" {
			// Download NZB from URL
			resp, err := http.Get(input.URL)
			if err != nil {
				return jsonResponse(http.StatusBadRequest, map[string]string{"error": "Failed to download NZB"})
			}
			defer resp.Body.Close()

			nzbData, err = ParseNZB(resp.Body)
			if err != nil {
				return jsonResponse(http.StatusBadRequest, map[string]string{"error": "Failed to parse NZB"})
			}

			// Use provided name or extract from URL
			if input.Name != "" {
				downloadName = input.Name
			} else {
				// Try to extract filename from URL
				urlParts := strings.Split(input.URL, "/")
				if len(urlParts) > 0 {
					downloadName = urlParts[len(urlParts)-1]
					// Remove .nzb extension if present
					downloadName = strings.TrimSuffix(downloadName, ".nzb")
				}
			}
		} else if input.NZB != "" {
			// Parse NZB content from JSON
			nzbData, err = ParseNZB(io.NopCloser(strings.NewReader(input.NZB)))
			if err != nil {
				return jsonResponse(http.StatusBadRequest, map[string]string{"error": "Failed to parse NZB"})
			}

			// Use provided name from the file upload
			downloadName = input.Name
		}
	} else {
		// Parse uploaded NZB file
		nzbData, err = ParseNZB(io.NopCloser(strings.NewReader(string(req.Body))))
		if err != nil {
			return jsonResponse(http.StatusBadRequest, map[string]string{"error": "Failed to parse NZB"})
		}

		// Try to extract a sensible name from the NZB metadata
		// Use the first file's name attribute if available
		if len(nzbData.Files) > 0 && nzbData.Files[0].FileName != "" {
			downloadName = nzbData.Files[0].FileName
		}
	}

	// Clean up the name and provide fallback
	if downloadName == "" {
		downloadName = fmt.Sprintf("download-%d", time.Now().Unix())
	} else {
		// Remove any path separators and clean up
		downloadName = filepath.Base(downloadName)
		// Remove .nzb extension if present
		downloadName = strings.TrimSuffix(downloadName, ".nzb")
		downloadName = strings.TrimSpace(downloadName)
	}

	// Get enabled servers and download directory now (while SDK is valid)
	allServers, err := p.getServers(ctx, req.SDK)
	if err != nil {
		return jsonResponse(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	enabledServers := []NNTPServer{}
	for _, srv := range allServers {
		if srv.Enabled {
			enabledServers = append(enabledServers, srv)
		}
	}

	if len(enabledServers) == 0 {
		return jsonResponse(http.StatusBadRequest, map[string]string{"error": "No enabled NNTP servers configured"})
	}

	// Get download directory
	downloadDir, err := req.SDK.ConfigGet(ctx, configDownloadDir)
	if err != nil || downloadDir == "" {
		downloadDir = "/tmp/nzb-downloads"
	}

	downloadDirStr, ok := downloadDir.(string)
	if !ok {
		downloadDirStr = "/tmp/nzb-downloads"
	}

	// Calculate total size
	var totalBytes int64
	for _, file := range nzbData.Files {
		for _, seg := range file.Segments {
			totalBytes += seg.Bytes
		}
	}

	// Create download with snapshot of servers and config
	download := &Download{
		ID:              generateID(downloadName),
		Name:            downloadName,
		Status:          "queued",
		Progress:        0,
		TotalBytes:      totalBytes,
		DownloadedBytes: 0,
		AddedAt:         time.Now(),
		NZBData:         nzbData,
		Servers:         enabledServers,
		DownloadDir:     downloadDirStr,
	}

	p.downloadManager.mu.Lock()
	p.downloadManager.downloads[download.ID] = download
	p.downloadManager.queue = append(p.downloadManager.queue, download.ID)
	p.downloadManager.mu.Unlock()

	// Start download processor if not running (no SDK needed, config is in Download struct)
	go p.processDownloadQueue(ctx)

	return jsonResponse(http.StatusCreated, download)
}

// Configuration Handlers

func (p *NZBDownloaderPlugin) handleGetConfig(ctx context.Context, req *plugins.PluginHTTPRequest) (*plugins.PluginHTTPResponse, error) {
	if req.SDK == nil {
		return jsonResponse(http.StatusInternalServerError, map[string]string{"error": "SDK not available"})
	}

	downloadDir, _ := req.SDK.ConfigGet(ctx, configDownloadDir)
	connections, _ := req.SDK.ConfigGet(ctx, configConnections)

	config := map[string]interface{}{
		"download_dir": downloadDir,
		"connections":  connections,
	}

	return jsonResponse(http.StatusOK, config)
}

func (p *NZBDownloaderPlugin) handleSetConfig(ctx context.Context, req *plugins.PluginHTTPRequest) (*plugins.PluginHTTPResponse, error) {
	if req.SDK == nil {
		return jsonResponse(http.StatusInternalServerError, map[string]string{"error": "SDK not available"})
	}

	var config map[string]interface{}
	if err := json.Unmarshal(req.Body, &config); err != nil {
		return jsonResponse(http.StatusBadRequest, map[string]string{"error": "Invalid JSON"})
	}

	if downloadDir, ok := config["download_dir"].(string); ok {
		req.SDK.ConfigSet(ctx, configDownloadDir, downloadDir)
	}
	if connections, ok := config["connections"].(float64); ok {
		req.SDK.ConfigSet(ctx, configConnections, int(connections))
	}

	return jsonResponse(http.StatusOK, map[string]string{"message": "Configuration saved"})
}

// Download Processing

func (p *NZBDownloaderPlugin) processDownloadQueue(ctx context.Context) {
	for {
		select {
		case <-p.downloadManager.ctx.Done():
			return
		default:
			p.downloadManager.mu.Lock()

			// Check if we can start more downloads
			if len(p.downloadManager.active) >= p.downloadManager.maxActive {
				p.downloadManager.mu.Unlock()
				time.Sleep(time.Second)
				continue
			}

			// Find next queued download
			var nextID string
			for _, id := range p.downloadManager.queue {
				dl := p.downloadManager.downloads[id]
				if dl.Status == "queued" && !p.downloadManager.active[id] {
					nextID = id
					break
				}
			}

			if nextID == "" {
				p.downloadManager.mu.Unlock()
				time.Sleep(time.Second)
				continue
			}

			// Start download
			p.downloadManager.active[nextID] = true
			download := p.downloadManager.downloads[nextID]
			download.Status = "downloading"
			now := time.Now()
			download.StartedAt = &now

			p.downloadManager.mu.Unlock()

			// Download in background (servers and config are in Download struct)
			go p.downloadNZB(ctx, download)
		}
	}
}

func (p *NZBDownloaderPlugin) downloadNZB(ctx context.Context, download *Download) {
	defer func() {
		p.downloadManager.mu.Lock()
		delete(p.downloadManager.active, download.ID)
		p.downloadManager.mu.Unlock()
	}()

	// Create a dedicated context for this download that won't be cancelled
	downloadCtx := context.Background()

	// Use servers and download directory from Download struct (captured at creation time)
	if len(download.Servers) == 0 {
		download.Status = "failed"
		download.Error = "No servers configured for this download"
		return
	}

	downloadDirStr := download.DownloadDir
	if downloadDirStr == "" {
		downloadDirStr = "/tmp/nzb-downloads"
	}

	// Create download directory
	if err := os.MkdirAll(downloadDirStr, 0755); err != nil {
		download.Status = "failed"
		download.Error = fmt.Sprintf("Failed to create download directory: %v", err)
		return
	}

	// Use the first enabled server
	server := download.Servers[0]

	download.AddLog(fmt.Sprintf("Starting download using server %s:%d", server.Host, server.Port))

	// Create fast downloader with connection pool
	downloader, err := NewFastDownloader(downloadCtx, server, download)
	if err != nil {
		download.Status = "failed"
		download.Error = fmt.Sprintf("Failed to create downloader: %v", err)
		return
	}
	defer downloader.Close()

	// Start the download
	if err := downloader.Download(download, downloadDirStr); err != nil {
		download.Status = "failed"
		download.Error = fmt.Sprintf("Download failed: %v", err)
		return
	}

	download.Status = "completed"
	now := time.Now()
	download.CompletedAt = &now
	download.Progress = 100

	download.AddLog("Download completed successfully")
}

// UIManifest returns the UI configuration for this plugin
func (p *NZBDownloaderPlugin) UIManifest(ctx context.Context) (*plugins.UIManifest, error) {
	return &plugins.UIManifest{
		NavItems: []plugins.UINavItem{
			{
				Label: "NZB Downloader",
				Path:  "/plugins/nzb-downloader",
				Icon:  "download",
			},
		},
		Routes: []plugins.UIRoute{
			{
				Path:      "/plugins/nzb-downloader",
				BundleURL: "/src/plugins-nzb-downloader.tsx",
			},
		},
	}, nil
}

// HandleEvent handles system events
func (p *NZBDownloaderPlugin) HandleEvent(ctx context.Context, evt plugins.Event) error {
	return nil
}

// Helper functions

func (p *NZBDownloaderPlugin) getServers(ctx context.Context, sdk plugins.SDKInterface) ([]NNTPServer, error) {
	val, err := sdk.ConfigGet(ctx, configServers)
	if err != nil {
		return []NNTPServer{}, nil
	}

	if val == nil {
		return []NNTPServer{}, nil
	}

	var servers []NNTPServer
	switch v := val.(type) {
	case []interface{}:
		jsonData, _ := json.Marshal(v)
		if err := json.Unmarshal(jsonData, &servers); err != nil {
			fmt.Fprintf(os.Stderr, "ERROR unmarshaling []interface{}: %v, data: %s\n", err, string(jsonData))
		} else {
			fmt.Fprintf(os.Stderr, "SUCCESS: Parsed %d servers from []interface{}\n", len(servers))
			for i, s := range servers {
				fmt.Fprintf(os.Stderr, "  Server %d: %s, enabled=%v\n", i, s.Name, s.Enabled)
			}
		}
	case string:
		if err := json.Unmarshal([]byte(v), &servers); err != nil {
			fmt.Fprintf(os.Stderr, "ERROR unmarshaling string: %v, data: %s\n", err, v)
		} else {
			fmt.Fprintf(os.Stderr, "SUCCESS: Parsed %d servers from string\n", len(servers))
			for i, s := range servers {
				fmt.Fprintf(os.Stderr, "  Server %d: %s, enabled=%v\n", i, s.Name, s.Enabled)
			}
		}
	default:
		fmt.Fprintf(os.Stderr, "WARNING: Unexpected type %T, value: %+v\n", v, v)
		jsonData, _ := json.Marshal(v)
		if err := json.Unmarshal(jsonData, &servers); err != nil {
			fmt.Fprintf(os.Stderr, "ERROR unmarshaling default: %v, data: %s\n", err, string(jsonData))
		} else {
			fmt.Fprintf(os.Stderr, "SUCCESS: Parsed %d servers from default marshal\n", len(servers))
			for i, s := range servers {
				fmt.Fprintf(os.Stderr, "  Server %d: %s, enabled=%v\n", i, s.Name, s.Enabled)
			}
		}
	}

	return servers, nil
}

func (p *NZBDownloaderPlugin) saveServers(ctx context.Context, sdk plugins.SDKInterface, servers []NNTPServer) error {
	return sdk.ConfigSet(ctx, configServers, servers)
}

func generateID(name string) string {
	id := strings.ToLower(name)
	id = strings.ReplaceAll(id, " ", "-")
	id = strings.ReplaceAll(id, "_", "-")
	allowed := "abcdefghijklmnopqrstuvwxyz0123456789-"
	result := ""
	for _, c := range id {
		if strings.ContainsRune(allowed, c) {
			result += string(c)
		}
	}
	if result == "" {
		result = fmt.Sprintf("item-%d", time.Now().Unix())
	}
	return result
}

func maskPassword(password string) string {
	if password == "" {
		return ""
	}
	if len(password) <= 4 {
		return strings.Repeat("*", len(password))
	}
	return password[:2] + strings.Repeat("*", len(password)-4) + password[len(password)-2:]
}

func jsonResponse(statusCode int, data interface{}) (*plugins.PluginHTTPResponse, error) {
	body, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	return &plugins.PluginHTTPResponse{
		StatusCode: statusCode,
		Headers: map[string][]string{
			"Content-Type": {"application/json"},
		},
		Body: body,
	}, nil
}

func main() {
	nzbPlugin := &NZBDownloaderPlugin{
		downloadManager: NewDownloadManager(3), // Max 3 concurrent downloads
	}

	plugin.Serve(&plugin.ServeConfig{
		HandshakeConfig: plugins.Handshake,
		Plugins: map[string]plugin.Plugin{
			"media-suite": &plugins.MediaSuitePluginGRPC{
				Impl: nzbPlugin,
			},
		},
		GRPCServer: plugin.DefaultGRPCServer,
	})
}
