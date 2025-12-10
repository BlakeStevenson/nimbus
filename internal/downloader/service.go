package downloader

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/blakestevenson/nimbus/internal/plugins"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

// Service provides a unified interface for managing downloads across all downloader plugins
type Service struct {
	pluginManager *plugins.PluginManager
	db            *pgxpool.Pool
	logger        *zap.Logger
	httpClient    *http.Client
	baseURL       string // Base URL for internal API calls (e.g., "http://localhost:8080")
}

// NewService creates a new downloader service
func NewService(pluginManager *plugins.PluginManager, db *pgxpool.Pool, logger *zap.Logger) *Service {
	return &Service{
		pluginManager: pluginManager,
		db:            db,
		logger:        logger.With(zap.String("component", "downloader-service")),
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		baseURL: "http://localhost:8080", // Default, should be configurable
	}
}

// SetBaseURL sets the base URL for internal API calls
func (s *Service) SetBaseURL(baseURL string) {
	s.baseURL = baseURL
}

// Initialize synchronizes pending downloads from the database to their respective plugin queues
func (s *Service) Initialize(ctx context.Context) error {
	s.logger.Info("Initializing downloader service and syncing queued downloads")

	// Get all downloads that are queued or downloading (active states)
	query := `
		SELECT id, plugin_id, name, status, progress, total_bytes, downloaded_bytes,
		       url, file_name, destination_path, error_message, priority,
		       created_at, started_at, completed_at, metadata, media_item_id
		FROM downloads
		WHERE status IN ('queued', 'downloading', 'processing')
		ORDER BY created_at ASC
	`

	rows, err := s.db.Query(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to query pending downloads: %w", err)
	}
	defer rows.Close()

	syncCount := 0
	for rows.Next() {
		var download Download
		var metadataJSON []byte
		var progress int
		var mediaItemID *int64

		err := rows.Scan(
			&download.ID,
			&download.PluginID,
			&download.Name,
			&download.Status,
			&progress,
			&download.TotalBytes,
			&download.DownloadedBytes,
			&download.URL,
			&download.FileName,
			&download.DestinationPath,
			&download.ErrorMessage,
			&download.Priority,
			&download.CreatedAt,
			&download.StartedAt,
			&download.CompletedAt,
			&metadataJSON,
			&mediaItemID,
		)
		if err != nil {
			s.logger.Error("Failed to scan download row during initialization", zap.Error(err))
			continue
		}

		download.Progress = float64(progress)

		if len(metadataJSON) > 0 {
			if err := json.Unmarshal(metadataJSON, &download.Metadata); err != nil {
				s.logger.Warn("Failed to unmarshal metadata", zap.Error(err))
			}
		}

		// Recreate the download in the plugin's queue
		s.logger.Info("Syncing download to plugin queue",
			zap.String("download_id", download.ID),
			zap.String("plugin_id", download.PluginID),
			zap.String("name", download.Name))

		// Prepare request to recreate the download in the plugin
		reqBody := map[string]interface{}{
			"name":     download.Name,
			"priority": download.Priority,
			"metadata": download.Metadata,
		}

		if download.URL != "" {
			reqBody["url"] = download.URL
		}

		bodyJSON, err := json.Marshal(reqBody)
		if err != nil {
			s.logger.Error("Failed to marshal download request during sync",
				zap.String("download_id", download.ID),
				zap.Error(err))
			continue
		}

		// Get the plugin
		plugin, exists := s.pluginManager.GetPlugin(download.PluginID)
		if !exists {
			s.logger.Warn("Plugin not found for download during sync, will retry later",
				zap.String("download_id", download.ID),
				zap.String("plugin_id", download.PluginID))
			continue
		}

		// Call plugin to recreate the download
		pluginReq := &plugins.PluginHTTPRequest{
			Method:  "POST",
			Path:    fmt.Sprintf("/api/plugins/%s/downloads", download.PluginID),
			Headers: map[string][]string{"Content-Type": {"application/json"}},
			Body:    bodyJSON,
			Query:   map[string][]string{},
		}

		pluginResp, err := plugin.Client.HandleAPI(ctx, pluginReq)
		if err != nil {
			s.logger.Error("Failed to sync download to plugin",
				zap.String("download_id", download.ID),
				zap.String("plugin_id", download.PluginID),
				zap.Error(err))
			continue
		}

		if pluginResp.StatusCode != http.StatusOK && pluginResp.StatusCode != http.StatusCreated {
			s.logger.Warn("Failed to sync download to plugin, marking as failed",
				zap.String("download_id", download.ID),
				zap.String("plugin_id", download.PluginID),
				zap.Int("status_code", pluginResp.StatusCode),
				zap.String("response", string(pluginResp.Body)))

			// Mark the download as failed in the database since we can't sync it
			_, err = s.db.Exec(ctx, `
				UPDATE downloads
				SET status = 'failed',
				    error_message = 'Failed to restore download on server restart: NZB data not available',
				    updated_at = CURRENT_TIMESTAMP
				WHERE id = $1
			`, download.ID)
			if err != nil {
				s.logger.Error("Failed to mark download as failed", zap.String("download_id", download.ID), zap.Error(err))
			}
			continue
		}

		syncCount++
	}

	if err := rows.Err(); err != nil {
		return fmt.Errorf("error iterating downloads during sync: %w", err)
	}

	s.logger.Info("Downloader service initialization complete",
		zap.Int("synced_downloads", syncCount))

	return nil
}

// DownloadRequest represents a unified download request
type DownloadRequest struct {
	PluginID    string                 `json:"plugin_id"`    // Which downloader plugin to use (e.g., "nzb-downloader")
	Name        string                 `json:"name"`         // Display name for the download
	URL         string                 `json:"url"`          // Optional: URL to download from
	FileContent []byte                 `json:"file_content"` // Optional: File content (e.g., NZB or torrent file)
	FileName    string                 `json:"file_name"`    // Original filename
	Priority    int                    `json:"priority"`     // Download priority (higher = more important)
	Metadata    map[string]interface{} `json:"metadata"`     // Plugin-specific metadata
}

// Download represents a download in the system
type Download struct {
	ID              string                 `json:"id"`
	PluginID        string                 `json:"plugin_id"`
	Name            string                 `json:"name"`
	Status          string                 `json:"status"`
	Progress        float64                `json:"progress"` // Changed to float64 to match plugin output
	TotalBytes      *int64                 `json:"total_bytes,omitempty"`
	DownloadedBytes int64                  `json:"downloaded_bytes"`
	Speed           int64                  `json:"speed,omitempty"` // Download speed in bytes per second
	URL             string                 `json:"url,omitempty"`
	FileName        string                 `json:"file_name,omitempty"`
	DestinationPath string                 `json:"destination_path,omitempty"`
	ErrorMessage    string                 `json:"error_message,omitempty"`
	QueuePosition   *int                   `json:"queue_position,omitempty"`
	Priority        int                    `json:"priority"`
	CreatedAt       time.Time              `json:"created_at,omitempty"`
	AddedAt         time.Time              `json:"added_at,omitempty"` // Alternative field name from some plugins
	StartedAt       *time.Time             `json:"started_at,omitempty"`
	CompletedAt     *time.Time             `json:"completed_at,omitempty"`
	Metadata        map[string]interface{} `json:"metadata,omitempty"`
}

// DownloadResponse represents aggregated download information
type DownloadResponse struct {
	Downloads []Download
	Total     int
}

// DownloaderInfo contains information about an available downloader plugin
type DownloaderInfo struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Version     string `json:"version"`
	Description string `json:"description"`
}

// saveDownloadToDB persists a download to the database
func (s *Service) saveDownloadToDB(ctx context.Context, download *Download, userID *int) error {
	query := `
		INSERT INTO downloads (
			id, plugin_id, name, status, progress, total_bytes, downloaded_bytes,
			url, file_name, destination_path, error_message, priority,
			created_at, started_at, completed_at, metadata, created_by_user_id, media_item_id
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18)
		ON CONFLICT (id) DO UPDATE SET
			status = EXCLUDED.status,
			progress = EXCLUDED.progress,
			downloaded_bytes = EXCLUDED.downloaded_bytes,
			error_message = EXCLUDED.error_message,
			started_at = EXCLUDED.started_at,
			completed_at = EXCLUDED.completed_at,
			media_item_id = EXCLUDED.media_item_id,
			updated_at = CURRENT_TIMESTAMP
	`

	metadataJSON, err := json.Marshal(download.Metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	createdAt := download.CreatedAt
	if createdAt.IsZero() && !download.AddedAt.IsZero() {
		createdAt = download.AddedAt
	}

	// Extract media_item_id from metadata if present
	var mediaItemID *int64
	if download.Metadata != nil {
		s.logger.Debug("Download metadata", zap.Any("metadata", download.Metadata))
		if mediaIDRaw, ok := download.Metadata["media_id"]; ok {
			s.logger.Debug("Found media_id in metadata", zap.Any("media_id_raw", mediaIDRaw), zap.String("type", fmt.Sprintf("%T", mediaIDRaw)))
			// Handle both string and numeric types
			switch v := mediaIDRaw.(type) {
			case string:
				if v != "" {
					var id int64
					if _, err := fmt.Sscanf(v, "%d", &id); err == nil {
						mediaItemID = &id
						s.logger.Debug("Parsed media_id from string", zap.Int64("media_item_id", id))
					}
				}
			case float64:
				id := int64(v)
				mediaItemID = &id
				s.logger.Debug("Converted media_id from float64", zap.Int64("media_item_id", id))
			case int:
				id := int64(v)
				mediaItemID = &id
				s.logger.Debug("Converted media_id from int", zap.Int64("media_item_id", id))
			case int64:
				mediaItemID = &v
				s.logger.Debug("Using media_id as int64", zap.Int64("media_item_id", v))
			}
		} else {
			s.logger.Debug("No media_id found in metadata")
		}
	} else {
		s.logger.Debug("No metadata provided")
	}

	if mediaItemID != nil {
		s.logger.Info("Saving download with media_item_id", zap.String("download_id", download.ID), zap.Int64("media_item_id", *mediaItemID))
	}

	_, err = s.db.Exec(ctx, query,
		download.ID,
		download.PluginID,
		download.Name,
		download.Status,
		int(download.Progress), // Convert float to int for DB
		download.TotalBytes,
		download.DownloadedBytes,
		download.URL,
		download.FileName,
		download.DestinationPath,
		download.ErrorMessage,
		download.Priority,
		createdAt,
		download.StartedAt,
		download.CompletedAt,
		metadataJSON,
		userID,
		mediaItemID,
	)

	return err
}

// CreateDownload creates a new download via the appropriate plugin
func (s *Service) CreateDownload(ctx context.Context, req DownloadRequest) (*Download, error) {
	s.logger.Info("CreateDownload called",
		zap.String("plugin_id", req.PluginID),
		zap.String("name", req.Name))

	// Verify the plugin exists and is a downloader
	plugin, exists := s.pluginManager.GetPlugin(req.PluginID)
	if !exists {
		// Log available plugins for debugging
		allPlugins := s.pluginManager.ListPlugins()
		pluginIDs := make([]string, len(allPlugins))
		for i, p := range allPlugins {
			pluginIDs[i] = p.Meta.ID
		}
		s.logger.Error("Plugin not found",
			zap.String("requested_plugin_id", req.PluginID),
			zap.Int("requested_plugin_id_len", len(req.PluginID)),
			zap.Strings("available_plugins", pluginIDs))
		return nil, fmt.Errorf("plugin %s not found", req.PluginID)
	}

	// Check if plugin is a downloader (use cached field for performance)
	if !plugin.IsDownloader {
		return nil, fmt.Errorf("plugin %s is not a downloader", req.PluginID)
	}

	// Prepare request body
	reqBody := map[string]interface{}{
		"name":     req.Name,
		"priority": req.Priority,
		"metadata": req.Metadata,
	}

	if req.URL != "" {
		reqBody["url"] = req.URL
	}

	if len(req.FileContent) > 0 {
		reqBody["file_content"] = req.FileContent
		reqBody["file_name"] = req.FileName
	}

	bodyJSON, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Call plugin directly via RPC instead of HTTP to avoid auth issues
	pluginReq := &plugins.PluginHTTPRequest{
		Method:  "POST",
		Path:    fmt.Sprintf("/api/plugins/%s/downloads", req.PluginID),
		Headers: map[string][]string{"Content-Type": {"application/json"}},
		Body:    bodyJSON,
		Query:   map[string][]string{},
	}

	pluginResp, err := plugin.Client.HandleAPI(ctx, pluginReq)
	if err != nil {
		return nil, fmt.Errorf("failed to call plugin: %w", err)
	}

	if pluginResp.StatusCode != http.StatusOK && pluginResp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("plugin returned HTTP %d: %s", pluginResp.StatusCode, string(pluginResp.Body))
	}

	var download Download
	if err := json.Unmarshal(pluginResp.Body, &download); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// Ensure plugin_id is set (plugin might not return it)
	download.PluginID = req.PluginID

	// Persist to database
	if err := s.saveDownloadToDB(ctx, &download, nil); err != nil {
		s.logger.Error("Failed to persist download to database",
			zap.Error(err),
			zap.String("download_id", download.ID))
		// Don't fail the request, download is still created in plugin
	}

	s.logger.Info("Download created and persisted",
		zap.String("download_id", download.ID),
		zap.String("plugin_id", req.PluginID),
		zap.String("name", req.Name))

	return &download, nil
}

// syncDownloadFromPlugin fetches the latest status from a plugin and updates the database
func (s *Service) syncDownloadFromPlugin(ctx context.Context, pluginID string, downloadID string) error {
	plugin, exists := s.pluginManager.GetPlugin(pluginID)
	if !exists {
		return fmt.Errorf("plugin %s not found", pluginID)
	}

	pluginReq := &plugins.PluginHTTPRequest{
		Method:  "GET",
		Path:    fmt.Sprintf("/api/plugins/%s/downloads/%s", pluginID, downloadID),
		Headers: map[string][]string{},
		Body:    nil,
		Query:   map[string][]string{},
	}

	pluginResp, err := plugin.Client.HandleAPI(ctx, pluginReq)
	if err != nil {
		return fmt.Errorf("failed to get download from plugin: %w", err)
	}

	if pluginResp.StatusCode != http.StatusOK {
		return fmt.Errorf("plugin returned HTTP %d", pluginResp.StatusCode)
	}

	var download Download
	if err := json.Unmarshal(pluginResp.Body, &download); err != nil {
		return fmt.Errorf("failed to decode download: %w", err)
	}

	download.PluginID = pluginID
	return s.saveDownloadToDB(ctx, &download, nil)
}

// ListDownloads retrieves all downloads from the database, syncing with plugins for active downloads
func (s *Service) ListDownloads(ctx context.Context, pluginID string, status string) (*DownloadResponse, error) {
	// Build query with optional filters
	query := `
		SELECT id, plugin_id, name, status, progress, total_bytes, downloaded_bytes,
		       url, file_name, destination_path, error_message, queue_position, priority,
		       created_at, started_at, completed_at, metadata, media_item_id
		FROM downloads
		WHERE 1=1
	`
	args := []interface{}{}
	argNum := 1

	if pluginID != "" {
		query += fmt.Sprintf(" AND plugin_id = $%d", argNum)
		args = append(args, pluginID)
		argNum++
	}

	if status != "" {
		query += fmt.Sprintf(" AND status = $%d", argNum)
		args = append(args, status)
		argNum++
	}

	query += " ORDER BY created_at DESC"

	rows, err := s.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query downloads: %w", err)
	}
	defer rows.Close()

	allDownloads := []Download{}

	for rows.Next() {
		var download Download
		var metadataJSON []byte
		var progress int
		var mediaItemID *int64

		err := rows.Scan(
			&download.ID,
			&download.PluginID,
			&download.Name,
			&download.Status,
			&progress,
			&download.TotalBytes,
			&download.DownloadedBytes,
			&download.URL,
			&download.FileName,
			&download.DestinationPath,
			&download.ErrorMessage,
			&download.QueuePosition,
			&download.Priority,
			&download.CreatedAt,
			&download.StartedAt,
			&download.CompletedAt,
			&metadataJSON,
			&mediaItemID,
		)
		if err != nil {
			s.logger.Error("Failed to scan download row", zap.Error(err))
			continue
		}

		download.Progress = float64(progress)

		if len(metadataJSON) > 0 {
			if err := json.Unmarshal(metadataJSON, &download.Metadata); err != nil {
				s.logger.Warn("Failed to unmarshal metadata", zap.Error(err))
			}
		}

		// Mark for live data fetch
		allDownloads = append(allDownloads, download)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating downloads: %w", err)
	}

	// Fetch live data from plugins for active downloads
	// Group downloads by plugin for efficiency
	pluginDownloads := make(map[string][]int) // plugin_id -> indices in allDownloads
	for i, download := range allDownloads {
		if download.Status == "downloading" || download.Status == "queued" || download.Status == "processing" {
			pluginDownloads[download.PluginID] = append(pluginDownloads[download.PluginID], i)
		}
	}

	// Fetch all downloads from each plugin once
	for pluginID, indices := range pluginDownloads {
		plugin, exists := s.pluginManager.GetPlugin(pluginID)
		if !exists {
			continue
		}

		pluginReq := &plugins.PluginHTTPRequest{
			Method:  "GET",
			Path:    fmt.Sprintf("/api/plugins/%s/downloads", pluginID),
			Headers: map[string][]string{},
			Body:    nil,
			Query:   map[string][]string{},
		}

		pluginResp, err := plugin.Client.HandleAPI(ctx, pluginReq)
		if err != nil || pluginResp.StatusCode != http.StatusOK {
			continue
		}

		var pluginDownloadsList struct {
			Downloads []Download `json:"downloads"`
		}
		if err := json.Unmarshal(pluginResp.Body, &pluginDownloadsList); err != nil {
			continue
		}

		// Create a map of plugin downloads by ID for quick lookup
		liveDownloadMap := make(map[string]*Download)
		for i := range pluginDownloadsList.Downloads {
			liveDownloadMap[pluginDownloadsList.Downloads[i].ID] = &pluginDownloadsList.Downloads[i]
		}

		// Update downloads with live data
		for _, idx := range indices {
			if liveDownload, found := liveDownloadMap[allDownloads[idx].ID]; found {
				allDownloads[idx].Status = liveDownload.Status
				allDownloads[idx].Progress = liveDownload.Progress
				allDownloads[idx].DownloadedBytes = liveDownload.DownloadedBytes
				allDownloads[idx].Speed = liveDownload.Speed
				allDownloads[idx].ErrorMessage = liveDownload.ErrorMessage
				allDownloads[idx].StartedAt = liveDownload.StartedAt
				allDownloads[idx].CompletedAt = liveDownload.CompletedAt

				// Persist updated status to database
				if err := s.saveDownloadToDB(ctx, &allDownloads[idx], nil); err != nil {
					s.logger.Debug("Failed to persist updated download",
						zap.String("download_id", allDownloads[idx].ID),
						zap.Error(err))
				}
			}
		}
	}

	return &DownloadResponse{
		Downloads: allDownloads,
		Total:     len(allDownloads),
	}, nil
}

// GetDownload retrieves a specific download by ID from the database and syncs with plugin
func (s *Service) GetDownload(ctx context.Context, downloadID string, pluginID string) (*Download, error) {
	// First, try to get from database
	var download Download
	var metadataJSON []byte
	var progress int
	var mediaItemID *int64

	err := s.db.QueryRow(ctx, `
		SELECT id, plugin_id, name, status, progress, total_bytes, downloaded_bytes,
		       url, file_name, destination_path, error_message, queue_position, priority,
		       created_at, started_at, completed_at, metadata, media_item_id
		FROM downloads
		WHERE id = $1 AND plugin_id = $2
	`, downloadID, pluginID).Scan(
		&download.ID,
		&download.PluginID,
		&download.Name,
		&download.Status,
		&progress,
		&download.TotalBytes,
		&download.DownloadedBytes,
		&download.URL,
		&download.FileName,
		&download.DestinationPath,
		&download.ErrorMessage,
		&download.QueuePosition,
		&download.Priority,
		&download.CreatedAt,
		&download.StartedAt,
		&download.CompletedAt,
		&metadataJSON,
		&mediaItemID,
	)

	if err != nil {
		return nil, fmt.Errorf("download not found: %w", err)
	}

	download.Progress = float64(progress)

	if len(metadataJSON) > 0 {
		if err := json.Unmarshal(metadataJSON, &download.Metadata); err != nil {
			s.logger.Warn("Failed to unmarshal metadata", zap.Error(err))
		}
	}

	// For active downloads, get live data from plugin
	if download.Status == "downloading" || download.Status == "queued" || download.Status == "processing" {
		plugin, exists := s.pluginManager.GetPlugin(pluginID)
		if exists {
			pluginReq := &plugins.PluginHTTPRequest{
				Method:  "GET",
				Path:    fmt.Sprintf("/api/plugins/%s/downloads/%s", pluginID, downloadID),
				Headers: map[string][]string{},
				Body:    nil,
				Query:   map[string][]string{},
			}

			pluginResp, err := plugin.Client.HandleAPI(ctx, pluginReq)
			if err == nil && pluginResp.StatusCode == http.StatusOK {
				var liveDownload Download
				if err := json.Unmarshal(pluginResp.Body, &liveDownload); err == nil {
					download.Status = liveDownload.Status
					download.Progress = liveDownload.Progress
					download.DownloadedBytes = liveDownload.DownloadedBytes
					download.Speed = liveDownload.Speed
					download.ErrorMessage = liveDownload.ErrorMessage
				}
			}
		}
	}

	return &download, nil
}

// PauseDownload pauses a download
func (s *Service) PauseDownload(ctx context.Context, downloadID string, pluginID string) error {
	return s.makeControlRequest(ctx, downloadID, pluginID, "pause", "POST")
}

// ResumeDownload resumes a paused download
func (s *Service) ResumeDownload(ctx context.Context, downloadID string, pluginID string) error {
	// First try to resume the download in the plugin
	err := s.makeControlRequest(ctx, downloadID, pluginID, "resume", "POST")

	// If the plugin returns an error (like download not found or not paused),
	// it might be because the server restarted and the download wasn't synced.
	// In that case, try to recreate it from the database
	if err != nil {
		s.logger.Warn("Failed to resume download in plugin, attempting to recreate",
			zap.String("download_id", downloadID),
			zap.String("plugin_id", pluginID),
			zap.Error(err))

		// Get the download from database
		var download Download
		var metadataJSON []byte
		var progress int
		var mediaItemID *int64

		queryErr := s.db.QueryRow(ctx, `
			SELECT id, plugin_id, name, status, progress, total_bytes, downloaded_bytes,
			       url, file_name, destination_path, error_message, priority,
			       created_at, started_at, completed_at, metadata, media_item_id
			FROM downloads
			WHERE id = $1 AND plugin_id = $2
		`, downloadID, pluginID).Scan(
			&download.ID,
			&download.PluginID,
			&download.Name,
			&download.Status,
			&progress,
			&download.TotalBytes,
			&download.DownloadedBytes,
			&download.URL,
			&download.FileName,
			&download.DestinationPath,
			&download.ErrorMessage,
			&download.Priority,
			&download.CreatedAt,
			&download.StartedAt,
			&download.CompletedAt,
			&metadataJSON,
			&mediaItemID,
		)

		if queryErr != nil {
			return fmt.Errorf("download not found in database: %w", queryErr)
		}

		if len(metadataJSON) > 0 {
			json.Unmarshal(metadataJSON, &download.Metadata)
		}

		// Only recreate if it's paused and has a URL
		if download.Status != "paused" || download.URL == "" {
			return err // Return original error
		}

		// Recreate the download in the plugin
		plugin, exists := s.pluginManager.GetPlugin(pluginID)
		if !exists {
			return fmt.Errorf("plugin %s not found", pluginID)
		}

		reqBody := map[string]interface{}{
			"name":     download.Name,
			"url":      download.URL,
			"priority": download.Priority,
			"metadata": download.Metadata,
		}

		bodyJSON, marshalErr := json.Marshal(reqBody)
		if marshalErr != nil {
			return fmt.Errorf("failed to marshal request: %w", marshalErr)
		}

		pluginReq := &plugins.PluginHTTPRequest{
			Method:  "POST",
			Path:    fmt.Sprintf("/api/plugins/%s/downloads", pluginID),
			Headers: map[string][]string{"Content-Type": {"application/json"}},
			Body:    bodyJSON,
			Query:   map[string][]string{},
		}

		pluginResp, createErr := plugin.Client.HandleAPI(ctx, pluginReq)
		if createErr != nil {
			return fmt.Errorf("failed to recreate download: %w", createErr)
		}

		if pluginResp.StatusCode != http.StatusOK && pluginResp.StatusCode != http.StatusCreated {
			return fmt.Errorf("plugin returned HTTP %d when recreating: %s", pluginResp.StatusCode, string(pluginResp.Body))
		}

		// Update database to mark as queued
		_, updateErr := s.db.Exec(ctx, `
			UPDATE downloads
			SET status = 'queued',
			    updated_at = CURRENT_TIMESTAMP
			WHERE id = $1
		`, downloadID)

		if updateErr != nil {
			s.logger.Error("Failed to update download status after recreation", zap.Error(updateErr))
		}

		s.logger.Info("Successfully recreated and resumed download",
			zap.String("download_id", downloadID),
			zap.String("plugin_id", pluginID))

		return nil
	}

	return nil
}

// CancelDownload cancels a download
func (s *Service) CancelDownload(ctx context.Context, downloadID string, pluginID string) error {
	err := s.makeControlRequest(ctx, downloadID, pluginID, "", "DELETE")
	if err == nil {
		// Also delete from database
		_, err = s.db.Exec(ctx, "DELETE FROM downloads WHERE id = $1 AND plugin_id = $2", downloadID, pluginID)
	}
	return err
}

// RetryDownload retries a failed download
func (s *Service) RetryDownload(ctx context.Context, downloadID string, pluginID string) error {
	return s.makeControlRequest(ctx, downloadID, pluginID, "retry", "POST")
}

// makeControlRequest is a helper for making control requests via RPC (pause/resume/cancel/retry)
func (s *Service) makeControlRequest(ctx context.Context, downloadID string, pluginID string, action string, method string) error {
	plugin, exists := s.pluginManager.GetPlugin(pluginID)
	if !exists {
		return fmt.Errorf("plugin %s not found", pluginID)
	}

	path := fmt.Sprintf("/api/plugins/%s/downloads/%s", pluginID, downloadID)
	if action != "" {
		path = fmt.Sprintf("%s/%s", path, action)
	}

	pluginReq := &plugins.PluginHTTPRequest{
		Method:  method,
		Path:    path,
		Headers: map[string][]string{},
		Body:    nil,
		Query:   map[string][]string{},
	}

	pluginResp, err := plugin.Client.HandleAPI(ctx, pluginReq)
	if err != nil {
		return fmt.Errorf("failed to call plugin: %w", err)
	}

	if pluginResp.StatusCode != http.StatusOK && pluginResp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("plugin returned HTTP %d: %s", pluginResp.StatusCode, string(pluginResp.Body))
	}

	// Sync the updated download back to database
	if err := s.syncDownloadFromPlugin(ctx, pluginID, downloadID); err != nil {
		s.logger.Debug("Failed to sync download after control action",
			zap.String("download_id", downloadID),
			zap.String("action", action),
			zap.Error(err))
	}

	return nil
}

// ListDownloaders returns information about all available downloader plugins
func (s *Service) ListDownloaders() []DownloaderInfo {
	downloaderPlugins := s.pluginManager.ListDownloaderPlugins()

	downloaders := make([]DownloaderInfo, len(downloaderPlugins))
	for i, plugin := range downloaderPlugins {
		downloaders[i] = DownloaderInfo{
			ID:          plugin.Meta.ID,
			Name:        plugin.Meta.Name,
			Version:     plugin.Meta.Version,
			Description: plugin.Meta.Description,
		}
	}

	return downloaders
}
