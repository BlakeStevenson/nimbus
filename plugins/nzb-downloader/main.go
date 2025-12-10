package main

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
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
	sdk             plugins.SDKInterface
	sdkMu           sync.RWMutex
}

// Configuration keys
const (
	configPrefix      = "plugins.nzb-downloader"
	configServers     = configPrefix + ".servers"
	configDownloadDir = configPrefix + ".download_dir"
	configConnections = configPrefix + ".connections"
	configDownloads   = configPrefix + ".downloads" // Persisted download state
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
	ID              string                 `json:"id"`
	Name            string                 `json:"name"`
	Status          string                 `json:"status"` // queued, downloading, processing, paused, completed, failed
	Progress        float64                `json:"progress"`
	TotalBytes      int64                  `json:"total_bytes"`
	DownloadedBytes int64                  `json:"downloaded_bytes"`
	Speed           int64                  `json:"speed"`               // bytes per second
	ETA             int64                  `json:"eta"`                 // seconds
	URL             string                 `json:"url,omitempty"`       // Original download URL
	FileName        string                 `json:"file_name,omitempty"` // Original filename if uploaded
	Priority        int                    `json:"priority"`
	Metadata        map[string]interface{} `json:"metadata,omitempty"`
	AddedAt         time.Time              `json:"added_at"`
	StartedAt       *time.Time             `json:"started_at,omitempty"`
	CompletedAt     *time.Time             `json:"completed_at,omitempty"`
	Error           string                 `json:"error,omitempty"`
	NZBData         *NZB                   `json:"-"`
	Servers         []NNTPServer           `json:"-"`              // Snapshot of enabled servers at time of creation
	DownloadDir     string                 `json:"-"`              // Download directory
	Logs            []string               `json:"logs,omitempty"` // Recent log messages
	logMu           sync.Mutex             `json:"-"`
	cancelDownload  context.CancelFunc     `json:"-"` // Cancel function for this download
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
	// Store SDK for later use if not already set
	if req.SDK != nil {
		p.sdkMu.Lock()
		if p.sdk == nil {
			p.sdk = req.SDK
			// Load persisted downloads on first API call
			go p.loadDownloads(context.Background(), req.SDK)
		}
		p.sdkMu.Unlock()
	}

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

			// Check for action routes (pause, resume, retry)
			if len(parts) == 7 && req.Method == "POST" {
				action := parts[6]
				switch action {
				case "pause":
					return p.handlePauseDownload(ctx, req, downloadID)
				case "resume":
					return p.handleResumeDownload(ctx, req, downloadID)
				case "retry":
					return p.handleRetryDownload(ctx, req, downloadID)
				}
			}

			// Direct operations
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
		server.ID = generateID()
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

	// Return downloads in queue order to maintain consistent ordering in UI
	downloads := make([]*Download, 0, len(p.downloadManager.queue))
	for _, id := range p.downloadManager.queue {
		if dl, exists := p.downloadManager.downloads[id]; exists {
			downloads = append(downloads, dl)
		}
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

	// After moving, check if the first queued item changed
	// If so, pause any active downloads and restart the queue
	var firstQueuedID string
	for _, id := range p.downloadManager.queue {
		dl := p.downloadManager.downloads[id]
		if dl.Status == "queued" || dl.Status == "downloading" {
			firstQueuedID = id
			break
		}
	}

	// If there's an active download and it's not the first in queue, cancel it
	for activeID := range p.downloadManager.active {
		if activeID != firstQueuedID {
			dl := p.downloadManager.downloads[activeID]

			// Cancel the download
			if dl.cancelDownload != nil {
				dl.AddLog("Download paused due to queue reordering")
				dl.cancelDownload()
			}

			// Reset status to queued
			dl.Status = "queued"
			dl.StartedAt = nil
			delete(p.downloadManager.active, activeID)
		}
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

	// Persist download state
	if req.SDK != nil {
		go p.saveDownloads(context.Background(), req.SDK)
	}

	return jsonResponse(http.StatusOK, map[string]string{"message": "Download deleted successfully"})
}

func (p *NZBDownloaderPlugin) handlePauseDownload(ctx context.Context, req *plugins.PluginHTTPRequest, downloadID string) (*plugins.PluginHTTPResponse, error) {
	p.downloadManager.mu.Lock()
	defer p.downloadManager.mu.Unlock()

	// Check if download exists
	dl, exists := p.downloadManager.downloads[downloadID]
	if !exists {
		return jsonResponse(http.StatusNotFound, map[string]string{"error": "Download not found"})
	}

	// Can only pause downloading items
	if dl.Status != "downloading" && dl.Status != "queued" {
		return jsonResponse(http.StatusBadRequest, map[string]string{"error": "Download is not active"})
	}

	// Cancel the download
	if dl.cancelDownload != nil {
		dl.AddLog("Download paused by user")
		dl.cancelDownload()
	}

	// Update status
	dl.Status = "paused"
	dl.StartedAt = nil
	delete(p.downloadManager.active, downloadID)

	return jsonResponse(http.StatusOK, map[string]string{"message": "Download paused successfully"})
}

func (p *NZBDownloaderPlugin) handleResumeDownload(ctx context.Context, req *plugins.PluginHTTPRequest, downloadID string) (*plugins.PluginHTTPResponse, error) {
	p.downloadManager.mu.Lock()
	defer p.downloadManager.mu.Unlock()

	// Check if download exists
	dl, exists := p.downloadManager.downloads[downloadID]
	if !exists {
		return jsonResponse(http.StatusNotFound, map[string]string{"error": "Download not found"})
	}

	// Can only resume paused or queued items (idempotent - allow resuming already queued downloads)
	if dl.Status != "paused" && dl.Status != "queued" {
		return jsonResponse(http.StatusBadRequest, map[string]string{"error": fmt.Sprintf("Download cannot be resumed (status: %s)", dl.Status)})
	}

	// If already queued, this is a no-op (idempotent)
	if dl.Status == "queued" {
		dl.AddLog("Download already queued")
		return jsonResponse(http.StatusOK, map[string]string{"message": "Download already queued"})
	}

	// Reset status to queued so it gets picked up by the queue processor
	dl.Status = "queued"
	dl.Error = ""
	dl.AddLog("Download resumed by user")

	return jsonResponse(http.StatusOK, map[string]string{"message": "Download resumed successfully"})
}

func (p *NZBDownloaderPlugin) handleRetryDownload(ctx context.Context, req *plugins.PluginHTTPRequest, downloadID string) (*plugins.PluginHTTPResponse, error) {
	p.downloadManager.mu.Lock()
	defer p.downloadManager.mu.Unlock()

	// Check if download exists
	dl, exists := p.downloadManager.downloads[downloadID]
	if !exists {
		return jsonResponse(http.StatusNotFound, map[string]string{"error": "Download not found"})
	}

	// Can only retry failed or cancelled items
	if dl.Status != "failed" && dl.Status != "cancelled" {
		return jsonResponse(http.StatusBadRequest, map[string]string{"error": "Download is not failed or cancelled"})
	}

	// Reset download state
	dl.Status = "queued"
	dl.Progress = 0
	dl.DownloadedBytes = 0
	dl.Error = ""
	dl.StartedAt = nil
	dl.CompletedAt = nil
	dl.AddLog("Download retry requested by user")

	return jsonResponse(http.StatusOK, map[string]string{"message": "Download retry initiated"})
}

func (p *NZBDownloaderPlugin) handleAddDownload(ctx context.Context, req *plugins.PluginHTTPRequest) (*plugins.PluginHTTPResponse, error) {
	fmt.Fprintf(os.Stderr, "[NZB-DOWNLOADER] handleAddDownload called\n")
	fmt.Fprintf(os.Stderr, "[NZB-DOWNLOADER] Request body: %s\n", string(req.Body))

	if req.SDK == nil {
		return jsonResponse(http.StatusInternalServerError, map[string]string{"error": "SDK not available"})
	}

	// Parse multipart form for NZB file upload or URL
	var nzbData *NZB
	var downloadName string

	// Check if it's a URL or file upload
	var input struct {
		URL      string                 `json:"url"`
		NZB      string                 `json:"nzb"`
		Name     string                 `json:"name"`
		Priority int                    `json:"priority"`
		Metadata map[string]interface{} `json:"metadata"`
	}

	var err error
	if jsonErr := json.Unmarshal(req.Body, &input); jsonErr == nil && (input.URL != "" || input.NZB != "") {
		fmt.Fprintf(os.Stderr, "[NZB-DOWNLOADER] Parsed input - URL: %s, Name: %s\n", input.URL, input.Name)
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

	// Generate download ID
	downloadID := generateID()

	// Create a unique subdirectory for this download to avoid file conflicts
	downloadDirStr = filepath.Join(downloadDirStr, downloadID)

	// Create download with snapshot of servers and config
	download := &Download{
		ID:              downloadID,
		Name:            downloadName,
		Status:          "queued",
		Progress:        0,
		TotalBytes:      totalBytes,
		DownloadedBytes: 0,
		URL:             input.URL,      // Preserve original URL
		FileName:        input.Name,     // Preserve original filename
		Priority:        input.Priority, // Preserve priority
		Metadata:        input.Metadata, // Preserve metadata (includes media_id)
		AddedAt:         time.Now(),
		NZBData:         nzbData,
		Servers:         enabledServers,
		DownloadDir:     downloadDirStr,
	}

	p.downloadManager.mu.Lock()
	p.downloadManager.downloads[download.ID] = download
	p.downloadManager.queue = append(p.downloadManager.queue, download.ID)
	queueLen := len(p.downloadManager.queue)
	p.downloadManager.mu.Unlock()

	fmt.Fprintf(os.Stderr, "[NZB-DOWNLOADER] Download added to queue - ID: %s, Name: %s, Queue length: %d\n", download.ID, download.Name, queueLen)

	// Persist download state
	if req.SDK != nil {
		go p.saveDownloads(context.Background(), req.SDK)
	}

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
	fmt.Fprintf(os.Stderr, "[NZB-DOWNLOADER] Queue processor started\n")
	for {
		select {
		case <-p.downloadManager.ctx.Done():
			fmt.Fprintf(os.Stderr, "[NZB-DOWNLOADER] Queue processor stopping\n")
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

			fmt.Fprintf(os.Stderr, "[NZB-DOWNLOADER] Starting download: %s\n", nextID)

			// Start download
			p.downloadManager.active[nextID] = true
			download := p.downloadManager.downloads[nextID]
			download.Status = "downloading"
			now := time.Now()
			download.StartedAt = &now

			// Create a cancellable context for this download
			downloadCtx, downloadCancel := context.WithCancel(context.Background())
			download.cancelDownload = downloadCancel

			p.downloadManager.mu.Unlock()

			// Download in background (servers and config are in Download struct)
			go p.downloadNZB(downloadCtx, download)
		}
	}
}

func (p *NZBDownloaderPlugin) downloadNZB(ctx context.Context, download *Download) {
	defer func() {
		p.downloadManager.mu.Lock()
		delete(p.downloadManager.active, download.ID)
		p.downloadManager.mu.Unlock()
	}()

	// Use the provided context which can be cancelled for pause functionality
	downloadCtx := ctx

	// Use servers and download directory from Download struct (captured at creation time)
	if len(download.Servers) == 0 {
		download.Status = "failed"
		download.Error = "No servers configured for this download"
		p.persistDownloadState()
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
		p.persistDownloadState()
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
		p.persistDownloadState()
		return
	}
	defer downloader.Close()

	// Start the download
	if err := downloader.Download(download, downloadDirStr); err != nil {
		// Check if it was cancelled (paused) vs actual error
		if ctx.Err() == context.Canceled {
			// Download was paused, status should already be set to "paused"
			download.AddLog("Download cancelled")
			return
		}

		// Actual error occurred
		download.Status = "failed"
		download.Error = fmt.Sprintf("Download failed: %v", err)
		// Clean up failed download files
		p.cleanupFailedDownload(downloadDirStr, download)
		p.persistDownloadState()
		return
	}

	// Mark as processing - this allows next download to start
	download.Status = "processing"
	download.Progress = 100
	download.AddLog("Download complete, processing files...")

	// Run post-processing in background (doesn't block queue)
	// Note: We don't defer anything here since processing happens after download completes
	go func() {
		// Post-process files (extraction, cleanup, etc.)
		if err := downloader.PostProcess(downloadDirStr); err != nil {
			download.AddLog(fmt.Sprintf("Post-processing failed: %v", err))
			download.Status = "failed"
			download.Error = fmt.Sprintf("Post-processing failed: %v", err)
			p.persistDownloadState()
			return
		}

		// Check if this is a season pack download
		mediaKind, _ := download.Metadata["media_kind"].(string)
		if mediaKind == "tv_season" {
			// Find all episode files
			episodeFiles, err := findAllMediaFiles(downloadDirStr)
			if err != nil || len(episodeFiles) == 0 {
				download.AddLog(fmt.Sprintf("ERROR: Could not find episode files: %v", err))
				download.Status = "failed"
				download.Error = fmt.Sprintf("Could not find episode files: %v", err)
				return
			} else if len(episodeFiles) == 1 {
				// Single file marked as season pack - treat as single episode
				download.AddLog("Detected single episode (misidentified as season pack)")
				download.AddLog(fmt.Sprintf("Found episode file: %s", filepath.Base(episodeFiles[0])))

				// Import using the media_id directly (it's actually an episode ID)
				if _, ok := download.Metadata["media_id"]; ok {
					if err := importToLibrary(download, episodeFiles[0]); err != nil {
						download.AddLog(fmt.Sprintf("Import failed: %v", err))
						download.Status = "failed"
						download.Error = fmt.Sprintf("Import failed: %v", err)
						return
					} else {
						download.AddLog("Import completed successfully")
					}
				} else {
					download.AddLog("ERROR: No media_id found - cannot import")
					download.Status = "failed"
					download.Error = "No media_id found - cannot import"
					return
				}
			} else {
				// Multiple files - actual season pack
				download.AddLog(fmt.Sprintf("Detected season pack, processing %d episodes...", len(episodeFiles)))

				// Get the season media_id to query for episodes
				seasonMediaID, _ := download.Metadata["media_id"]
				if seasonMediaID == nil {
					download.AddLog("ERROR: No media_id found for season - cannot import episodes")
					download.Status = "failed"
					download.Error = "No media_id found for season - cannot import episodes"
					return
				} else {
					// Import each episode file
					successCount := 0
					failCount := 0

					for _, file := range episodeFiles {
						fileName := filepath.Base(file)
						download.AddLog(fmt.Sprintf("Processing: %s", fileName))

						// Parse season and episode from filename
						season, episode, found := parseEpisodeFromFilename(fileName)
						if !found {
							download.AddLog(fmt.Sprintf("  Could not parse season/episode from filename, skipping"))
							failCount++
							continue
						}

						download.AddLog(fmt.Sprintf("  Detected S%02dE%02d", season, episode))

						// Find the episode in the database
						episodeMediaID, err := findEpisodeMediaID(seasonMediaID, season, episode)
						if err != nil {
							download.AddLog(fmt.Sprintf("  Could not find episode in database: %v", err))
							failCount++
							continue
						}

						download.AddLog(fmt.Sprintf("  Found episode media_id: %d", episodeMediaID))

						// Import this episode
						if err := importEpisodeFile(file, episodeMediaID); err != nil {
							download.AddLog(fmt.Sprintf("  Import failed: %v", err))
							failCount++
						} else {
							download.AddLog(fmt.Sprintf("  Import successful"))
							successCount++
						}
					}

					download.AddLog(fmt.Sprintf("Season pack import complete: %d succeeded, %d failed", successCount, failCount))

					// If all imports failed, mark download as failed
					if failCount > 0 && successCount == 0 {
						download.AddLog("ERROR: All episode imports failed")
						download.Status = "failed"
						download.Error = fmt.Sprintf("All %d episode imports failed", failCount)
						return
					} else if failCount > 0 {
						download.AddLog(fmt.Sprintf("WARNING: %d episode imports failed, but %d succeeded", failCount, successCount))
					}
				}
			}
		} else {
			// Single episode download or movie
			mainFile, err := findMainMediaFile(downloadDirStr)
			if err != nil {
				download.AddLog(fmt.Sprintf("ERROR: Could not find main media file: %v", err))
				download.Status = "failed"
				download.Error = fmt.Sprintf("Could not find main media file: %v", err)
				return
			} else {
				download.AddLog(fmt.Sprintf("Found main media file: %s", filepath.Base(mainFile)))
			}

			// Trigger import if we have media metadata
			if download.Metadata != nil {
				if shouldImport(download.Metadata) {
					download.AddLog("Importing to library...")
					if err := importToLibrary(download, mainFile); err != nil {
						download.AddLog(fmt.Sprintf("Import failed: %v", err))
						download.Status = "failed"
						download.Error = fmt.Sprintf("Import failed: %v", err)
						return
					} else {
						download.AddLog("Import completed successfully")
					}
				}
			}
		}

		// Mark as completed
		download.Status = "completed"
		now := time.Now()
		download.CompletedAt = &now
		download.AddLog("Processing completed successfully")
		p.persistDownloadState()
	}()
}

// UIManifest returns the UI configuration for this plugin
func (p *NZBDownloaderPlugin) UIManifest(ctx context.Context) (*plugins.UIManifest, error) {
	return &plugins.UIManifest{
		NavItems: []plugins.UINavItem{},
		Routes:   []plugins.UIRoute{},
		ConfigSection: &plugins.ConfigSection{
			Title:       "NZB Downloader Settings",
			Description: "Configure NNTP servers and download settings for Usenet downloads",
			Fields: []plugins.ConfigField{
				{
					Key:          configDownloadDir,
					Label:        "Download Directory",
					Description:  "Directory where downloaded files will be saved",
					Type:         "text",
					DefaultValue: "/tmp/nzb-downloads",
					Required:     true,
					Placeholder:  "/path/to/downloads",
				},
				{
					Key:          configConnections,
					Label:        "Max Connections",
					Description:  "Maximum number of concurrent connections per server",
					Type:         "number",
					DefaultValue: "10",
					Required:     false,
					Placeholder:  "10",
					Validation: &plugins.ConfigFieldValidation{
						Min:          intPtr(1),
						Max:          intPtr(50),
						ErrorMessage: "Must be between 1 and 50",
					},
				},
				{
					Key:          configServers,
					Label:        "NNTP Servers",
					Description:  "Configure your Usenet server connections",
					Type:         "custom",
					DefaultValue: "[]",
					Required:     false,
				},
			},
		},
	}, nil
}

// Helper function to create int32 pointers
func intPtr(i int32) *int32 {
	return &i
}

// HandleEvent handles system events
func (p *NZBDownloaderPlugin) HandleEvent(ctx context.Context, evt plugins.Event) error {
	return nil
}

// IsIndexer returns false as this is not an indexer plugin
func (p *NZBDownloaderPlugin) IsIndexer(ctx context.Context) (bool, error) {
	return false, nil
}

// Search is not implemented for downloader plugins
func (p *NZBDownloaderPlugin) Search(ctx context.Context, req *plugins.IndexerSearchRequest) (*plugins.IndexerSearchResponse, error) {
	return nil, fmt.Errorf("NZB downloader plugin does not support search")
}

// IsDownloader returns true as this plugin provides downloader functionality
func (p *NZBDownloaderPlugin) IsDownloader(ctx context.Context) (bool, error) {
	return true, nil
}

// cleanupFailedDownload removes all files from a failed download
func (p *NZBDownloaderPlugin) cleanupFailedDownload(downloadDir string, download *Download) {
	download.AddLog("Cleaning up files from failed download...")

	entries, err := os.ReadDir(downloadDir)
	if err != nil {
		download.AddLog(fmt.Sprintf("Failed to read download directory for cleanup: %v", err))
		return
	}

	removedCount := 0
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		filePath := filepath.Join(downloadDir, entry.Name())
		if err := os.Remove(filePath); err != nil {
			download.AddLog(fmt.Sprintf("Failed to remove %s: %v", entry.Name(), err))
		} else {
			removedCount++
		}
	}

	if removedCount > 0 {
		download.AddLog(fmt.Sprintf("Removed %d files from failed download", removedCount))
	}
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

// PersistedDownload is a simplified version of Download for storage (excludes runtime fields)
type PersistedDownload struct {
	ID              string                 `json:"id"`
	Name            string                 `json:"name"`
	Status          string                 `json:"status"`
	Progress        float64                `json:"progress"`
	TotalBytes      int64                  `json:"total_bytes"`
	DownloadedBytes int64                  `json:"downloaded_bytes"`
	URL             string                 `json:"url,omitempty"`
	FileName        string                 `json:"file_name,omitempty"`
	Priority        int                    `json:"priority"`
	Metadata        map[string]interface{} `json:"metadata,omitempty"`
	AddedAt         time.Time              `json:"added_at"`
	StartedAt       *time.Time             `json:"started_at,omitempty"`
	CompletedAt     *time.Time             `json:"completed_at,omitempty"`
	Error           string                 `json:"error,omitempty"`
}

func (p *NZBDownloaderPlugin) saveDownloads(ctx context.Context, sdk plugins.SDKInterface) error {
	p.downloadManager.mu.RLock()
	defer p.downloadManager.mu.RUnlock()

	// Convert downloads to persistable format
	persistedDownloads := make([]PersistedDownload, 0, len(p.downloadManager.queue))
	for _, id := range p.downloadManager.queue {
		if dl, exists := p.downloadManager.downloads[id]; exists {
			persistedDownloads = append(persistedDownloads, PersistedDownload{
				ID:              dl.ID,
				Name:            dl.Name,
				Status:          dl.Status,
				Progress:        dl.Progress,
				TotalBytes:      dl.TotalBytes,
				DownloadedBytes: dl.DownloadedBytes,
				URL:             dl.URL,
				FileName:        dl.FileName,
				Priority:        dl.Priority,
				Metadata:        dl.Metadata,
				AddedAt:         dl.AddedAt,
				StartedAt:       dl.StartedAt,
				CompletedAt:     dl.CompletedAt,
				Error:           dl.Error,
			})
		}
	}

	return sdk.ConfigSet(ctx, configDownloads, persistedDownloads)
}

func (p *NZBDownloaderPlugin) loadDownloads(ctx context.Context, sdk plugins.SDKInterface) error {
	val, err := sdk.ConfigGet(ctx, configDownloads)
	if err != nil {
		return nil // No saved downloads
	}

	if val == nil {
		return nil
	}

	var persistedDownloads []PersistedDownload
	switch v := val.(type) {
	case []interface{}:
		jsonData, _ := json.Marshal(v)
		if err := json.Unmarshal(jsonData, &persistedDownloads); err != nil {
			return err
		}
	case string:
		if err := json.Unmarshal([]byte(v), &persistedDownloads); err != nil {
			return err
		}
	default:
		jsonData, _ := json.Marshal(v)
		if err := json.Unmarshal(jsonData, &persistedDownloads); err != nil {
			return err
		}
	}

	// Restore downloads to manager (skip downloads that are actively downloading)
	p.downloadManager.mu.Lock()
	defer p.downloadManager.mu.Unlock()

	for _, pd := range persistedDownloads {
		// Reset downloading status to queued on restart
		if pd.Status == "downloading" || pd.Status == "processing" {
			pd.Status = "queued"
			pd.Progress = 0
			pd.DownloadedBytes = 0
			pd.StartedAt = nil
		}

		download := &Download{
			ID:              pd.ID,
			Name:            pd.Name,
			Status:          pd.Status,
			Progress:        pd.Progress,
			TotalBytes:      pd.TotalBytes,
			DownloadedBytes: pd.DownloadedBytes,
			URL:             pd.URL,
			FileName:        pd.FileName,
			Priority:        pd.Priority,
			Metadata:        pd.Metadata,
			AddedAt:         pd.AddedAt,
			StartedAt:       pd.StartedAt,
			CompletedAt:     pd.CompletedAt,
			Error:           pd.Error,
		}

		p.downloadManager.downloads[download.ID] = download
		p.downloadManager.queue = append(p.downloadManager.queue, download.ID)
	}

	return nil
}

// persistDownloadState saves download state to config store and PostgreSQL database (non-blocking)
func (p *NZBDownloaderPlugin) persistDownloadState() {
	p.sdkMu.RLock()
	sdk := p.sdk
	p.sdkMu.RUnlock()

	if sdk != nil {
		// Save to config store (for plugin internal state)
		p.saveDownloads(context.Background(), sdk)

		// Also sync to PostgreSQL database (for unified /api/downloads endpoint)
		go p.syncDownloadsToDatabase()
	}
}

// syncDownloadsToDatabase syncs all downloads to the PostgreSQL database via internal API
func (p *NZBDownloaderPlugin) syncDownloadsToDatabase() {
	p.downloadManager.mu.RLock()
	downloads := make([]*Download, 0, len(p.downloadManager.queue))
	for _, id := range p.downloadManager.queue {
		if dl, exists := p.downloadManager.downloads[id]; exists {
			downloads = append(downloads, dl)
		}
	}
	p.downloadManager.mu.RUnlock()

	// Sync each download to database via internal HTTP endpoint
	for _, dl := range downloads {
		p.syncDownloadToDatabase(dl)
	}
}

// syncDownloadToDatabase syncs a single download to the PostgreSQL database
func (p *NZBDownloaderPlugin) syncDownloadToDatabase(dl *Download) {
	// Create request payload for unified downloads API
	payload := map[string]interface{}{
		"id":               dl.ID,
		"plugin_id":        "nzb-downloader",
		"name":             dl.Name,
		"status":           dl.Status,
		"progress":         dl.Progress,
		"total_bytes":      dl.TotalBytes,
		"downloaded_bytes": dl.DownloadedBytes,
		"url":              dl.URL,
		"file_name":        dl.FileName,
		"error_message":    dl.Error,
		"priority":         dl.Priority,
		"metadata":         dl.Metadata,
		"created_at":       dl.AddedAt,
		"started_at":       dl.StartedAt,
		"completed_at":     dl.CompletedAt,
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return
	}

	// Call internal sync endpoint (no auth required for internal calls)
	req, err := http.NewRequest("PUT", fmt.Sprintf("http://localhost:8080/api/internal/downloads/%s", dl.ID), strings.NewReader(string(payloadBytes)))
	if err != nil {
		return
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()
}

func generateID() string {
	// Generate a random 16-character alphanumeric ID using crypto/rand
	const charset = "abcdefghijklmnopqrstuvwxyz0123456789"
	const idLength = 16

	b := make([]byte, idLength)
	randomBytes := make([]byte, idLength)

	if _, err := rand.Read(randomBytes); err != nil {
		// Fallback to timestamp-based ID if random generation fails
		return fmt.Sprintf("dl-%d", time.Now().UnixNano())
	}

	for i := range b {
		b[i] = charset[int(randomBytes[i])%len(charset)]
	}

	return string(b)
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

// findAllMediaFiles finds all media files in a directory
func findAllMediaFiles(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory: %v", err)
	}

	var mediaFiles []string
	mediaExtensions := []string{".mkv", ".mp4", ".avi", ".m4v", ".ts", ".m2ts", ".wmv", ".mov"}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		ext := strings.ToLower(filepath.Ext(name))

		// Check if it's a media file
		isMedia := false
		for _, mediaExt := range mediaExtensions {
			if ext == mediaExt {
				isMedia = true
				break
			}
		}

		if isMedia {
			mediaFiles = append(mediaFiles, filepath.Join(dir, name))
		}
	}

	if len(mediaFiles) == 0 {
		return nil, fmt.Errorf("no media files found in directory")
	}

	return mediaFiles, nil
}

// findMainMediaFile finds the largest media file in a directory
func findMainMediaFile(dir string) (string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", fmt.Errorf("failed to read directory: %v", err)
	}

	var largestFile string
	var largestSize int64

	mediaExtensions := []string{".mkv", ".mp4", ".avi", ".m4v", ".ts", ".m2ts", ".wmv", ".mov"}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		ext := strings.ToLower(filepath.Ext(name))

		// Check if it's a media file
		isMedia := false
		for _, mediaExt := range mediaExtensions {
			if ext == mediaExt {
				isMedia = true
				break
			}
		}

		if !isMedia {
			continue
		}

		info, err := entry.Info()
		if err != nil {
			continue
		}

		if info.Size() > largestSize {
			largestSize = info.Size()
			largestFile = filepath.Join(dir, name)
		}
	}

	if largestFile == "" {
		return "", fmt.Errorf("no media files found in directory")
	}

	return largestFile, nil
}

// shouldImport checks if download has enough metadata to trigger import
func shouldImport(metadata map[string]interface{}) bool {
	// Check if media_id is present (manual download for specific media)
	if mediaID, ok := metadata["media_id"]; ok && mediaID != nil {
		return true
	}

	// Check if we have basic media info
	if _, hasTitle := metadata["title"]; hasTitle {
		// For movies, title is enough
		if mediaType, ok := metadata["media_type"].(string); ok {
			if mediaType == "movie" {
				return true
			}
			// For TV, we need season and episode
			if mediaType == "tv" || mediaType == "tv_episode" {
				_, hasSeason := metadata["season"]
				_, hasEpisode := metadata["episode"]
				return hasSeason && hasEpisode
			}
		}
	}

	return false
}

// parseEpisodeFromFilename extracts season and episode numbers from a filename
// Supports patterns like: S01E01, s01e01, 1x01, etc.
func parseEpisodeFromFilename(filename string) (season int, episode int, found bool) {
	lowerName := strings.ToLower(filename)

	// Pattern 1: S01E01 or s01e01
	re := regexp.MustCompile(`s(\d+)e(\d+)`)
	if matches := re.FindStringSubmatch(lowerName); len(matches) == 3 {
		fmt.Sscanf(matches[1], "%d", &season)
		fmt.Sscanf(matches[2], "%d", &episode)
		return season, episode, true
	}

	// Pattern 2: 1x01
	re = regexp.MustCompile(`(\d+)x(\d+)`)
	if matches := re.FindStringSubmatch(lowerName); len(matches) == 3 {
		fmt.Sscanf(matches[1], "%d", &season)
		fmt.Sscanf(matches[2], "%d", &episode)
		return season, episode, true
	}

	return 0, 0, false
}

// findEpisodeMediaID queries the Nimbus API to find the episode media_item_id
func findEpisodeMediaID(seasonMediaID interface{}, season int, episode int) (int64, error) {
	// Convert seasonMediaID to int64
	var seasonID int64
	switch v := seasonMediaID.(type) {
	case int:
		seasonID = int64(v)
	case int64:
		seasonID = v
	case float64:
		seasonID = int64(v)
	case string:
		parsed, err := fmt.Sscanf(v, "%d", &seasonID)
		if err != nil || parsed != 1 {
			return 0, fmt.Errorf("invalid season media_id format: %v", v)
		}
	default:
		return 0, fmt.Errorf("unsupported season media_id type: %T", v)
	}

	// Query the internal API for episodes of this season (no auth required)
	url := fmt.Sprintf("http://localhost:8080/api/internal/media?parent_id=%d&kind=tv_episode", seasonID)
	resp, err := http.Get(url)
	if err != nil {
		return 0, fmt.Errorf("failed to query episodes: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return 0, fmt.Errorf("API returned error: %s", string(body))
	}

	var result struct {
		Items []struct {
			ID       int64                  `json:"id"`
			Metadata map[string]interface{} `json:"metadata"`
		} `json:"items"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return 0, fmt.Errorf("failed to decode response: %v", err)
	}

	// Find the episode with matching season/episode numbers
	for _, item := range result.Items {
		if meta := item.Metadata; meta != nil {
			itemSeason, _ := meta["season"].(float64)
			itemEpisode, _ := meta["episode"].(float64)
			if int(itemSeason) == season && int(itemEpisode) == episode {
				return item.ID, nil
			}
		}
	}

	return 0, fmt.Errorf("episode S%02dE%02d not found in database", season, episode)
}

// importEpisodeFile imports a single episode file using the import API
func importEpisodeFile(sourcePath string, mediaItemID int64) error {
	importReq := map[string]interface{}{
		"source_path":   sourcePath,
		"media_item_id": mediaItemID,
	}

	reqBody, err := json.Marshal(importReq)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %v", err)
	}

	req, err := http.NewRequest("POST", "http://localhost:8080/api/downloads/import", strings.NewReader(string(reqBody)))
	if err != nil {
		return fmt.Errorf("failed to create request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to call import API: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("import API error: %s", string(body))
	}

	return nil
}

// importToLibrary calls the Nimbus import API to import completed download
func importToLibrary(download *Download, sourcePath string) error {
	// Build basic import request
	importReq := map[string]interface{}{
		"source_path": sourcePath,
	}

	// If we have a media_item_id, that's all the backend needs to look up the details
	if mediaID, ok := download.Metadata["media_id"]; ok {
		// Convert media_id to int64 (it might be a string or float64 from JSON)
		var mediaItemID int64
		switch v := mediaID.(type) {
		case int:
			mediaItemID = int64(v)
		case int64:
			mediaItemID = v
		case float64:
			mediaItemID = int64(v)
		case string:
			// Try to parse string as int
			parsed, err := fmt.Sscanf(v, "%d", &mediaItemID)
			if err != nil || parsed != 1 {
				return fmt.Errorf("invalid media_id format: %v", v)
			}
		default:
			return fmt.Errorf("unsupported media_id type: %T", v)
		}

		importReq["media_item_id"] = mediaItemID
		fmt.Fprintf(os.Stderr, "[NZB-DOWNLOADER] Importing with media_item_id: %d\n", mediaItemID)
	} else {
		// No media_item_id, skip import - we can't import without knowing what media item this is for
		return fmt.Errorf("no media_item_id found in download metadata - cannot import")
	}

	// Marshal request
	reqBody, err := json.Marshal(importReq)
	if err != nil {
		return fmt.Errorf("failed to marshal import request: %v", err)
	}

	// Call import endpoint
	req, err := http.NewRequest("POST", "http://localhost:8080/api/downloads/import", strings.NewReader(string(reqBody)))
	if err != nil {
		return fmt.Errorf("failed to create request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to call import API: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("import API returned error: %s", string(body))
	}

	// Parse response to get final path
	var importResult map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&importResult); err != nil {
		return fmt.Errorf("failed to decode import response: %v", err)
	}

	if finalPath, ok := importResult["final_path"].(string); ok {
		download.AddLog(fmt.Sprintf("Imported to: %s", finalPath))
	}

	return nil
}

func main() {
	nzbPlugin := &NZBDownloaderPlugin{
		downloadManager: NewDownloadManager(1), // Max 1 concurrent download (each uses many connections)
	}

	// Start the download queue processor
	go nzbPlugin.processDownloadQueue(nzbPlugin.downloadManager.ctx)

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
