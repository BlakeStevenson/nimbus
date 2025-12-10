package downloader

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/blakestevenson/nimbus/internal/configstore"
	"github.com/blakestevenson/nimbus/internal/db/generated"
	"github.com/blakestevenson/nimbus/internal/httputil"
	"github.com/blakestevenson/nimbus/internal/importer"
	"github.com/blakestevenson/nimbus/internal/library"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

// Handler provides HTTP handlers for download operations
type Handler struct {
	service     *Service
	queries     *generated.Queries
	configStore *configstore.Store
	logger      *zap.Logger
	db          *pgxpool.Pool
}

// NewHandler creates a new download handler
func NewHandler(service *Service, queries *generated.Queries, configStore *configstore.Store, db *pgxpool.Pool, logger *zap.Logger) *Handler {
	return &Handler{
		service:     service,
		queries:     queries,
		configStore: configStore,
		db:          db,
		logger:      logger.With(zap.String("component", "download-handler")),
	}
}

// ImportCompletedDownload handles importing a completed download into the library
// POST /api/downloads/{id}/import
func (h *Handler) ImportCompletedDownload(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Parse request body
	var req struct {
		DownloadID   string  `json:"download_id"`
		SourcePath   string  `json:"source_path"`
		MediaType    string  `json:"media_type"` // "movie" or "tv"
		Title        string  `json:"title"`
		Year         *int    `json:"year,omitempty"`
		Season       *int    `json:"season,omitempty"`
		Episode      *int    `json:"episode,omitempty"`
		EpisodeTitle *string `json:"episode_title,omitempty"`
		Quality      *string `json:"quality,omitempty"`
		MediaItemID  *int64  `json:"media_item_id,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.RespondError(w, http.StatusBadRequest, err, "Invalid request body")
		return
	}

	// Validate required fields
	if req.SourcePath == "" {
		httputil.RespondErrorMessage(w, http.StatusBadRequest, "source_path is required")
		return
	}

	// If media_item_id is provided, look up the media item and populate required fields
	if req.MediaItemID != nil && *req.MediaItemID > 0 {
		mediaItem, err := h.queries.GetMediaItem(ctx, *req.MediaItemID)
		if err != nil {
			h.logger.Error("failed to look up media item", zap.Int64("media_item_id", *req.MediaItemID), zap.Error(err))
			httputil.RespondError(w, http.StatusNotFound, err, "Media item not found")
			return
		}

		// Map Kind to MediaType and populate fields
		switch mediaItem.Kind {
		case "movie":
			req.MediaType = "movie"
			req.Title = mediaItem.Title
			if mediaItem.Year != nil {
				year := int(*mediaItem.Year)
				req.Year = &year
			}

		case "series":
			req.MediaType = "tv"
			req.Title = mediaItem.Title
			if mediaItem.Year != nil {
				year := int(*mediaItem.Year)
				req.Year = &year
			}

		case "season":
			req.MediaType = "tv"
			req.Title = mediaItem.Title
			if mediaItem.Year != nil {
				year := int(*mediaItem.Year)
				req.Year = &year
			}

		case "episode", "tv_episode":
			req.MediaType = "tv_episode"
			// For episodes, store the episode title separately
			episodeTitle := mediaItem.Title
			req.EpisodeTitle = &episodeTitle

			// Walk up the parent chain to find the series title
			// Episode -> Season -> Series
			if mediaItem.ParentID != nil {
				// Get the season
				season, err := h.queries.GetMediaItem(ctx, *mediaItem.ParentID)
				if err == nil && season.ParentID != nil {
					// Get the series
					series, err := h.queries.GetMediaItem(ctx, *season.ParentID)
					if err == nil {
						req.Title = series.Title
						if series.Year != nil {
							year := int(*series.Year)
							req.Year = &year
						}
					} else {
						h.logger.Warn("failed to get series from season parent", zap.Error(err))
					}
				} else {
					h.logger.Warn("failed to get season parent", zap.Error(err))
				}
			}

			// Parse metadata to get season/episode numbers
			if len(mediaItem.Metadata) > 0 {
				var metadata map[string]interface{}
				if err := json.Unmarshal(mediaItem.Metadata, &metadata); err == nil {
					if seasonNum, ok := metadata["season"].(float64); ok {
						season := int(seasonNum)
						req.Season = &season
					}
					if episodeNum, ok := metadata["episode"].(float64); ok {
						episode := int(episodeNum)
						req.Episode = &episode
					}
				}
			}

		default:
			req.MediaType = mediaItem.Kind
			req.Title = mediaItem.Title
		}

		h.logger.Info("importing completed download with media_item_id",
			zap.String("download_id", req.DownloadID),
			zap.Int64("media_item_id", *req.MediaItemID),
			zap.String("source", req.SourcePath),
			zap.String("title", req.Title),
			zap.String("type", req.MediaType))
	} else {
		// No media_item_id, so validate required fields
		if req.Title == "" {
			httputil.RespondErrorMessage(w, http.StatusBadRequest, "title is required")
			return
		}
		if req.MediaType == "" {
			httputil.RespondErrorMessage(w, http.StatusBadRequest, "media_type is required")
			return
		}

		h.logger.Info("importing completed download",
			zap.String("download_id", req.DownloadID),
			zap.String("source", req.SourcePath),
			zap.String("type", req.MediaType),
			zap.String("title", req.Title))
	}

	// Create importer service
	importerService := importer.NewService(h.queries, h.configStore, h.logger)

	// Build import request
	importReq := &importer.ImportRequest{
		SourcePath:   req.SourcePath,
		MediaType:    req.MediaType,
		MediaItemID:  req.MediaItemID,
		Title:        req.Title,
		Year:         req.Year,
		Season:       req.Season,
		Episode:      req.Episode,
		EpisodeTitle: req.EpisodeTitle,
		Quality:      req.Quality,
		Metadata:     make(map[string]interface{}),
	}

	// Perform import
	result, err := importerService.Import(ctx, importReq)
	if err != nil {
		h.logger.Error("import failed",
			zap.String("download_id", req.DownloadID),
			zap.Error(err))
		httputil.RespondError(w, http.StatusInternalServerError, err, "Import failed")
		return
	}

	// Update download record in database if download_id provided
	if req.DownloadID != "" {
		updateQuery := `
			UPDATE downloads
			SET status = 'completed',
			    destination_path = $1,
			    completed_at = NOW()
			WHERE id = $2
		`
		if _, err := h.db.Exec(ctx, updateQuery, result.FinalPath, req.DownloadID); err != nil {
			h.logger.Warn("failed to update download record", zap.Error(err))
		}
	}

	h.logger.Info("import completed successfully",
		zap.String("download_id", req.DownloadID),
		zap.String("final_path", result.FinalPath))

	httputil.RespondJSON(w, http.StatusOK, result)
}

// AutoImportHandler monitors completed downloads and automatically imports them
// This is called periodically by a background worker
func (h *Handler) AutoImportCompletedDownloads(ctx context.Context) error {
	h.logger.Debug("checking for completed downloads to import")

	// Query for completed downloads that haven't been imported yet
	query := `
		SELECT d.id, d.name, d.destination_path, d.metadata, d.media_item_id
		FROM downloads d
		WHERE d.status = 'completed'
		  AND d.destination_path IS NOT NULL
		  AND d.destination_path != ''
		  AND NOT EXISTS (
		      SELECT 1 FROM media_files mf
		      WHERE mf.file_path = d.destination_path
		  )
		LIMIT 10
	`

	rows, err := h.db.Query(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to query completed downloads: %w", err)
	}
	defer rows.Close()

	count := 0
	for rows.Next() {
		var downloadID string
		var name string
		var destinationPath *string
		var metadataJSON []byte
		var mediaItemID *int64

		if err := rows.Scan(&downloadID, &name, &destinationPath, &metadataJSON, &mediaItemID); err != nil {
			h.logger.Error("failed to scan download", zap.Error(err))
			continue
		}

		if destinationPath == nil || *destinationPath == "" {
			continue
		}

		// Parse metadata to extract media information
		var metadata map[string]interface{}
		if len(metadataJSON) > 0 {
			if err := json.Unmarshal(metadataJSON, &metadata); err != nil {
				h.logger.Warn("failed to unmarshal metadata", zap.Error(err))
				metadata = make(map[string]interface{})
			}
		} else {
			metadata = make(map[string]interface{})
		}

		// Try to determine media info from metadata or filename
		importReq := h.buildImportRequest(*destinationPath, name, metadata, mediaItemID)
		if importReq == nil {
			h.logger.Warn("could not determine media info for download",
				zap.String("download_id", downloadID),
				zap.String("name", name))
			continue
		}

		// Create importer and perform import
		importerService := importer.NewService(h.queries, h.configStore, h.logger)
		result, err := importerService.Import(ctx, importReq)
		if err != nil {
			h.logger.Error("auto-import failed",
				zap.String("download_id", downloadID),
				zap.Error(err))
			continue
		}

		// Update download record
		updateQuery := `
			UPDATE downloads
			SET destination_path = $1
			WHERE id = $2
		`
		if _, err := h.db.Exec(ctx, updateQuery, result.FinalPath, downloadID); err != nil {
			h.logger.Warn("failed to update download record", zap.Error(err))
		}

		count++
		h.logger.Info("auto-imported download",
			zap.String("download_id", downloadID),
			zap.String("final_path", result.FinalPath))
	}

	if count > 0 {
		h.logger.Info("auto-import batch completed", zap.Int("imported", count))
	}

	return nil
}

// buildImportRequest attempts to build an import request from download metadata
func (h *Handler) buildImportRequest(sourcePath, name string, metadata map[string]interface{}, mediaItemID *int64) *importer.ImportRequest {
	req := &importer.ImportRequest{
		SourcePath:  sourcePath,
		MediaItemID: mediaItemID,
		Metadata:    metadata,
	}

	// Try to get media info from metadata first
	if mediaType, ok := metadata["media_type"].(string); ok {
		req.MediaType = mediaType
	}

	if title, ok := metadata["title"].(string); ok {
		req.Title = title
	}

	if year, ok := metadata["year"].(float64); ok {
		y := int(year)
		req.Year = &y
	} else if year, ok := metadata["year"].(int); ok {
		req.Year = &year
	}

	if season, ok := metadata["season"].(float64); ok {
		s := int(season)
		req.Season = &s
	} else if season, ok := metadata["season"].(int); ok {
		req.Season = &season
	}

	if episode, ok := metadata["episode"].(float64); ok {
		e := int(episode)
		req.Episode = &e
	} else if episode, ok := metadata["episode"].(int); ok {
		req.Episode = &episode
	}

	if episodeTitle, ok := metadata["episode_title"].(string); ok {
		req.EpisodeTitle = &episodeTitle
	}

	if quality, ok := metadata["quality"].(string); ok {
		req.Quality = &quality
	}

	// If we don't have media type, try to guess from filename or metadata
	if req.MediaType == "" {
		// Check if it looks like a TV show
		parsed := library.ParseFilename(name)
		if parsed != nil {
			req.MediaType = parsed.Kind
			if req.Title == "" {
				req.Title = parsed.Title
			}
			if parsed.Year != 0 {
				req.Year = &parsed.Year
			}
			if parsed.Season != 0 {
				req.Season = &parsed.Season
			}
			if parsed.Episode != 0 {
				req.Episode = &parsed.Episode
			}
		}
	}

	// Validate we have minimum required info
	if req.MediaType == "" || req.Title == "" {
		return nil
	}

	// For TV shows, require season and episode
	if (req.MediaType == "tv" || req.MediaType == "tv_episode") && (req.Season == nil || req.Episode == nil) {
		return nil
	}

	return req
}

// FindMainMediaFile finds the largest media file in a directory (for multi-file downloads)
func FindMainMediaFile(dir string) (string, error) {
	entries, err := filepath.Glob(filepath.Join(dir, "*"))
	if err != nil {
		return "", err
	}

	var largestFile string
	var largestSize int64

	mediaExtensions := []string{".mkv", ".mp4", ".avi", ".m4v", ".ts", ".m2ts"}

	for _, entry := range entries {
		info, err := os.Stat(entry)
		if err != nil || info.IsDir() {
			continue
		}

		// Check if it's a media file
		ext := strings.ToLower(filepath.Ext(entry))
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

		if info.Size() > largestSize {
			largestSize = info.Size()
			largestFile = entry
		}
	}

	if largestFile == "" {
		return "", fmt.Errorf("no media files found in directory")
	}

	return largestFile, nil
}
