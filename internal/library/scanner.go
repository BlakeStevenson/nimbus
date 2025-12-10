package library

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/blakestevenson/nimbus/internal/db/generated"

	"go.uber.org/zap"
)

// =============================================================================
// Scanner - Main library scanner implementation
// =============================================================================
// The Scanner is responsible for:
//   1. Walking the filesystem to discover media files
//   2. Parsing filenames to extract metadata
//   3. Upserting media items and files into the database
//   4. Tracking progress via the scanner_state table
//   5. Logging errors and activity
//
// The scanner is designed to be run as a background goroutine and supports
// graceful cancellation via context.
// =============================================================================

type Scanner struct {
	queries    *generated.Queries
	service    *Service
	logger     *zap.Logger
	rootDir    string            // Legacy single root directory
	mediaPaths map[string]string // Media type specific paths: "movie", "tv", "music", "book"
}

// NewScanner creates a new scanner instance
func NewScanner(queries *generated.Queries, logger *zap.Logger, rootDir string) *Scanner {
	return &Scanner{
		queries:    queries,
		service:    NewService(queries, logger),
		logger:     logger,
		rootDir:    rootDir,
		mediaPaths: make(map[string]string),
	}
}

// SetMediaPath sets the library path for a specific media type
func (s *Scanner) SetMediaPath(mediaType, path string) {
	s.mediaPaths[mediaType] = path
}

// GetMediaPath returns the library path for a specific media type
// Falls back to rootDir if media-specific path is not set
func (s *Scanner) GetMediaPath(mediaType string) string {
	if path, ok := s.mediaPaths[mediaType]; ok && path != "" {
		return path
	}
	return s.rootDir
}

// =============================================================================
// Run - Main scanner execution loop
// =============================================================================
// This is the primary entry point for running a library scan. It:
//   1. Checks if a scan is already running (prevents concurrent scans)
//   2. Marks the scan as started in scanner_state
//   3. Walks the filesystem and processes each media file
//   4. Updates progress counters in real-time
//   5. Logs errors and activity
//   6. Marks the scan as finished
//
// The scan can be cancelled by cancelling the provided context.
//
// Returns error only for fatal issues (DB connection, etc). Individual file
// errors are logged but don't stop the scan.
// =============================================================================

func (s *Scanner) Run(ctx context.Context) error {
	s.logger.Info("starting library scan")

	// Check if a scan is already running
	state, err := s.queries.GetScannerState(ctx)
	if err != nil {
		return fmt.Errorf("failed to get scanner state: %w", err)
	}

	if state.Running {
		s.logger.Warn("scan already in progress, aborting")
		return fmt.Errorf("scan already in progress")
	}

	// Mark scan as started
	if _, err := s.queries.StartScan(ctx); err != nil {
		return fmt.Errorf("failed to start scan: %w", err)
	}

	// Ensure we mark the scan as finished on exit
	defer func() {
		if _, err := s.queries.FinishScan(ctx); err != nil {
			s.logger.Error("failed to finish scan", zap.Error(err))
		}
	}()

	// Log scan start
	if err := s.appendLog(ctx, "info", "Scan started"); err != nil {
		s.logger.Warn("failed to append log", zap.Error(err))
	}

	// Collect all paths to scan
	pathsToScan := []string{}

	// Add media-specific paths if configured
	mediaTypes := []string{"movie", "tv", "music", "book"}
	for _, mediaType := range mediaTypes {
		if path := s.GetMediaPath(mediaType); path != "" && path != s.rootDir {
			pathsToScan = append(pathsToScan, path)
			s.logger.Info("scanning media-specific path",
				zap.String("media_type", mediaType),
				zap.String("path", path))
		}
	}

	// Add root directory if no media-specific paths are configured, or as fallback
	if len(pathsToScan) == 0 {
		pathsToScan = append(pathsToScan, s.rootDir)
		s.logger.Info("scanning root directory", zap.String("root", s.rootDir))
	}

	// Walk all configured paths and collect files
	var allFiles []string
	for _, path := range pathsToScan {
		s.logger.Info("walking filesystem", zap.String("path", path))
		files, err := WalkMediaFiles(path)
		if err != nil {
			errMsg := fmt.Sprintf("failed to walk filesystem at %s: %v", path, err)
			s.appendError(ctx, errMsg)
			s.logger.Warn("failed to walk path", zap.String("path", path), zap.Error(err))
			continue // Continue with other paths even if one fails
		}
		allFiles = append(allFiles, files...)
	}

	totalFiles := len(allFiles)
	s.logger.Info("found media files", zap.Int("count", totalFiles))
	s.appendLog(ctx, "info", fmt.Sprintf("Found %d media files across all library paths", totalFiles))

	// Process each file
	var filesScanned, itemsCreated, itemsUpdated int32

	for i, filePath := range allFiles {
		// Check for context cancellation
		select {
		case <-ctx.Done():
			s.logger.Info("scan cancelled by context")
			s.appendLog(ctx, "warn", "Scan cancelled")
			return ctx.Err()
		default:
		}

		// Process the file
		created, err := s.processFile(ctx, filePath)
		if err != nil {
			errMsg := fmt.Sprintf("Error processing %s: %v", filePath, err)
			s.logger.Warn("file processing error",
				zap.String("path", filePath),
				zap.Error(err))
			s.appendError(ctx, errMsg)
			continue
		}

		// Update counters
		filesScanned++
		if created {
			itemsCreated++
		} else {
			itemsUpdated++
		}

		// Batch update progress every 10 files
		if i%10 == 0 || i == totalFiles-1 {
			if _, err := s.queries.UpdateScanProgress(ctx, generated.UpdateScanProgressParams{
				FilesScanned: filesScanned,
				ItemsCreated: itemsCreated,
				ItemsUpdated: itemsUpdated,
			}); err != nil {
				s.logger.Warn("failed to update scan progress", zap.Error(err))
			}

			// Reset counters after batch update
			filesScanned = 0
			itemsCreated = 0
			itemsUpdated = 0

			// Log progress
			progress := float64(i+1) / float64(totalFiles) * 100
			s.logger.Info("scan progress",
				zap.Int("processed", i+1),
				zap.Int("total", totalFiles),
				zap.Float64("percent", progress))
		}
	}

	// Final log
	finalState, _ := s.queries.GetScannerState(ctx)
	logMsg := fmt.Sprintf("Scan completed: %d files scanned, %d items created, %d items updated",
		finalState.FilesScanned, finalState.ItemsCreated, finalState.ItemsUpdated)
	s.appendLog(ctx, "info", logMsg)
	s.logger.Info("scan completed",
		zap.Int32("files_scanned", finalState.FilesScanned),
		zap.Int32("items_created", finalState.ItemsCreated),
		zap.Int32("items_updated", finalState.ItemsUpdated))

	return nil
}

// =============================================================================
// processFile - Process a single media file
// =============================================================================
// Strategy:
//   1. Get file info (size)
//   2. Parse filename to extract metadata
//   3. Call appropriate upsert function based on media type
//   4. Return whether the item was created or updated
//
// Returns:
//   - created: true if new item, false if updated
//   - error: any error during processing
// =============================================================================

func (s *Scanner) processFile(ctx context.Context, filePath string) (created bool, err error) {
	// Get file info
	fileSize, err := GetFileInfo(filePath)
	if err != nil {
		return false, fmt.Errorf("failed to get file info: %w", err)
	}

	// Parse filename
	parsed := ParseFilename(filePath)
	if parsed == nil {
		return false, fmt.Errorf("failed to parse filename")
	}

	// Log what we're processing
	s.logger.Debug("processing file",
		zap.String("path", filePath),
		zap.String("kind", parsed.Kind),
		zap.String("title", parsed.Title))

	// Dispatch to appropriate handler based on media kind
	var itemID int64
	switch parsed.Kind {
	case "movie":
		itemID, created, err = s.service.UpsertMovie(ctx, parsed, filePath, fileSize)

	case "tv_episode":
		itemID, created, err = s.service.UpsertTVEpisode(ctx, parsed, filePath, fileSize)

	case "music_track":
		itemID, created, err = s.service.UpsertMusicTrack(ctx, parsed, filePath, fileSize)

	case "book":
		itemID, created, err = s.service.UpsertBook(ctx, parsed, filePath, fileSize)

	default:
		return false, fmt.Errorf("unsupported media kind: %s", parsed.Kind)
	}

	if err != nil {
		return false, fmt.Errorf("failed to upsert %s: %w", parsed.Kind, err)
	}

	s.logger.Debug("processed file",
		zap.String("path", filePath),
		zap.Int64("item_id", itemID),
		zap.Bool("created", created))

	return created, nil
}

// =============================================================================
// appendLog - Add a log entry to the scanner_state log array
// =============================================================================

func (s *Scanner) appendLog(ctx context.Context, level, message string) error {
	logEntry := map[string]interface{}{
		"timestamp": time.Now().Format(time.RFC3339),
		"level":     level,
		"message":   message,
	}

	logJSON, err := json.Marshal([]interface{}{logEntry})
	if err != nil {
		return err
	}

	_, err = s.queries.AppendScanLog(ctx, logJSON)
	return err
}

// =============================================================================
// appendError - Add an error to the scanner_state errors array
// =============================================================================

func (s *Scanner) appendError(ctx context.Context, message string) error {
	errorEntry := map[string]interface{}{
		"timestamp": time.Now().Format(time.RFC3339),
		"message":   message,
	}

	errorJSON, err := json.Marshal([]interface{}{errorEntry})
	if err != nil {
		return err
	}

	_, err = s.queries.AppendScanError(ctx, errorJSON)
	return err
}

// =============================================================================
// GetScanStatus - Retrieve current scanner state
// =============================================================================
// Returns a friendly representation of the scanner state suitable for API responses.
// =============================================================================

type ScanStatus struct {
	Running      bool       `json:"running"`
	StartedAt    *string    `json:"started_at"`
	FinishedAt   *string    `json:"finished_at"`
	FilesScanned int32      `json:"files_scanned"`
	ItemsCreated int32      `json:"items_created"`
	ItemsUpdated int32      `json:"items_updated"`
	Errors       []LogEntry `json:"errors"`
	Log          []LogEntry `json:"log"`
}

type LogEntry struct {
	Timestamp string `json:"timestamp"`
	Message   string `json:"message"`
	Level     string `json:"level,omitempty"`
}

func (s *Scanner) GetScanStatus(ctx context.Context) (*ScanStatus, error) {
	state, err := s.queries.GetScannerState(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get scanner state: %w", err)
	}

	status := &ScanStatus{
		Running:      state.Running,
		FilesScanned: state.FilesScanned,
		ItemsCreated: state.ItemsCreated,
		ItemsUpdated: state.ItemsUpdated,
		Errors:       []LogEntry{},
		Log:          []LogEntry{},
	}

	// Convert timestamps
	if state.StartedAt.Valid {
		ts := state.StartedAt.Time.Format(time.RFC3339)
		status.StartedAt = &ts
	}
	if state.FinishedAt.Valid {
		ts := state.FinishedAt.Time.Format(time.RFC3339)
		status.FinishedAt = &ts
	}

	// Parse errors JSON
	if len(state.Errors) > 2 { // More than just "[]"
		var errors []LogEntry
		if err := json.Unmarshal(state.Errors, &errors); err == nil {
			status.Errors = errors
		}
	}

	// Parse log JSON
	if len(state.Log) > 2 { // More than just "[]"
		var logs []LogEntry
		if err := json.Unmarshal(state.Log, &logs); err == nil {
			status.Log = logs
		}
	}

	return status, nil
}
