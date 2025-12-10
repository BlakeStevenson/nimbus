package library

import (
	"encoding/json"
	"net/http"
	"os"
	"strconv"

	"github.com/blakestevenson/nimbus/internal/db/generated"
	"github.com/blakestevenson/nimbus/internal/httputil"
	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"
)

// =============================================================================
// FileHandler - HTTP handlers for media file management
// =============================================================================

type FileHandler struct {
	queries *generated.Queries
	logger  *zap.Logger
}

// NewFileHandler creates a new file handler
func NewFileHandler(queries *generated.Queries, logger *zap.Logger) *FileHandler {
	return &FileHandler{
		queries: queries,
		logger:  logger,
	}
}

// =============================================================================
// GetMediaFiles - GET /api/media/{id}/files
// =============================================================================
// Retrieves all files associated with a media item.
//
// Response:
//   - 200 OK: List of media files
//   - 404 Not Found: Media item not found
//   - 500 Internal Server Error: Database error
// =============================================================================

func (h *FileHandler) GetMediaFiles(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	mediaIDStr := chi.URLParam(r, "id")
	mediaID, err := strconv.ParseInt(mediaIDStr, 10, 64)
	if err != nil {
		httputil.RespondErrorMessage(w, http.StatusBadRequest, "Invalid media ID")
		return
	}

	// Get files for this media item
	files, err := h.queries.ListMediaFilesByItem(ctx, &mediaID)
	if err != nil {
		h.logger.Error("failed to get media files", zap.Error(err))
		httputil.RespondErrorMessage(w, http.StatusInternalServerError, "Failed to get media files")
		return
	}

	// Also get files for child items (e.g., episodes in a season)
	// First get the media item to check its kind
	mediaItem, err := h.queries.GetMediaItem(ctx, mediaID)
	if err == nil && mediaItem.Kind == "tv_season" {
		// Get all children (episodes)
		children, err := h.queries.ListMediaItems(ctx, generated.ListMediaItemsParams{
			ParentID: &mediaID,
		})
		if err == nil {
			// Get files for each child episode
			for _, child := range children {
				childID := child.ID
				childFiles, err := h.queries.ListMediaFilesByItem(ctx, &childID)
				if err == nil {
					files = append(files, childFiles...)
				}
			}
		}
	}

	// Convert to response format
	type FileResponse struct {
		ID          int64   `json:"id"`
		MediaItemID *int64  `json:"media_item_id"`
		Path        string  `json:"path"`
		Size        *int64  `json:"size"`
		Hash        *string `json:"hash"`
		CreatedAt   string  `json:"created_at"`
		UpdatedAt   string  `json:"updated_at"`
	}

	response := make([]FileResponse, len(files))
	for i, file := range files {
		response[i] = FileResponse{
			ID:          file.ID,
			MediaItemID: file.MediaItemID,
			Path:        file.Path,
			Size:        file.Size,
			Hash:        file.Hash,
			CreatedAt:   file.CreatedAt.Time.Format("2006-01-02T15:04:05Z07:00"),
			UpdatedAt:   file.UpdatedAt.Time.Format("2006-01-02T15:04:05Z07:00"),
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// =============================================================================
// DeleteMediaFile - DELETE /api/media/files/{fileId}
// =============================================================================
// Deletes a specific media file entry from the database and optionally
// deletes the physical file from disk.
//
// Query Parameters:
//   - delete_physical: "true" to also delete the file from disk (default: false)
//
// Response:
//   - 204 No Content: File deleted successfully
//   - 400 Bad Request: Invalid file ID
//   - 404 Not Found: File not found
//   - 500 Internal Server Error: Database or filesystem error
// =============================================================================

func (h *FileHandler) DeleteMediaFile(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	fileIDStr := chi.URLParam(r, "fileId")
	fileID, err := strconv.ParseInt(fileIDStr, 10, 64)
	if err != nil {
		httputil.RespondErrorMessage(w, http.StatusBadRequest, "Invalid file ID")
		return
	}

	// Get file info before deleting
	file, err := h.queries.GetMediaFile(ctx, fileID)
	if err != nil {
		httputil.RespondErrorMessage(w, http.StatusNotFound, "File not found")
		return
	}

	// Check if we should delete the physical file
	deletePhysical := r.URL.Query().Get("delete_physical") == "true"

	// Delete from database first
	if err := h.queries.DeleteMediaFile(ctx, fileID); err != nil {
		h.logger.Error("failed to delete media file from database", zap.Error(err))
		httputil.RespondErrorMessage(w, http.StatusInternalServerError, "Failed to delete file")
		return
	}

	// Optionally delete physical file
	if deletePhysical {
		if err := os.Remove(file.Path); err != nil {
			h.logger.Warn("failed to delete physical file",
				zap.String("path", file.Path),
				zap.Error(err))
			// Don't return error - database entry is already deleted
		}
	}

	w.WriteHeader(http.StatusNoContent)
}

// =============================================================================
// DeleteMediaItemWithFiles - DELETE /api/media/{id}/with-files
// =============================================================================
// Deletes a media item and optionally deletes all associated physical files.
//
// Query Parameters:
//   - delete_files: "true" to also delete physical files (default: false)
//
// Response:
//   - 204 No Content: Media item deleted successfully
//   - 400 Bad Request: Invalid media ID
//   - 404 Not Found: Media item not found
//   - 500 Internal Server Error: Database or filesystem error
// =============================================================================

func (h *FileHandler) DeleteMediaItemWithFiles(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	mediaIDStr := chi.URLParam(r, "id")
	mediaID, err := strconv.ParseInt(mediaIDStr, 10, 64)
	if err != nil {
		httputil.RespondErrorMessage(w, http.StatusBadRequest, "Invalid media ID")
		return
	}

	// Check if we should delete physical files
	deleteFiles := r.URL.Query().Get("delete_files") == "true"

	// Get all files before deleting if we need to delete physical files
	var filePaths []string
	if deleteFiles {
		files, err := h.queries.ListMediaFilesByItem(ctx, &mediaID)
		if err != nil {
			h.logger.Error("failed to get media files", zap.Error(err))
			httputil.RespondErrorMessage(w, http.StatusInternalServerError, "Failed to get media files")
			return
		}

		for _, file := range files {
			filePaths = append(filePaths, file.Path)
		}
	}

	// Delete media item (cascade will delete media_files entries)
	if err := h.queries.DeleteMediaItem(ctx, mediaID); err != nil {
		h.logger.Error("failed to delete media item", zap.Error(err))
		httputil.RespondErrorMessage(w, http.StatusInternalServerError, "Failed to delete media item")
		return
	}

	// Optionally delete physical files
	if deleteFiles {
		for _, path := range filePaths {
			if err := os.Remove(path); err != nil {
				h.logger.Warn("failed to delete physical file",
					zap.String("path", path),
					zap.Error(err))
				// Continue deleting other files even if one fails
			}
		}
	}

	w.WriteHeader(http.StatusNoContent)
}
