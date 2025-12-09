package handlers

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/blakestevenson/nimbus/internal/httputil"
	"github.com/blakestevenson/nimbus/internal/media"
	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"
)

// MediaHandler handles media-related HTTP requests
type MediaHandler struct {
	service media.Service
	logger  *zap.Logger
}

// NewMediaHandler creates a new media handler
func NewMediaHandler(service media.Service, logger *zap.Logger) *MediaHandler {
	return &MediaHandler{
		service: service,
		logger:  logger,
	}
}

// CreateMediaItem handles POST /api/media
func (h *MediaHandler) CreateMediaItem(w http.ResponseWriter, r *http.Request) {
	var params media.CreateMediaParams
	if err := httputil.DecodeJSON(r, &params); err != nil {
		httputil.RespondError(w, http.StatusBadRequest, err, "invalid request body")
		return
	}

	item, err := h.service.CreateMediaItem(r.Context(), params)
	if err != nil {
		if errors.Is(err, media.ErrInvalidKind) || errors.Is(err, media.ErrTitleRequired) {
			httputil.RespondError(w, http.StatusBadRequest, err, "validation error")
			return
		}
		httputil.LogError(h.logger, err, "failed to create media item")
		httputil.RespondErrorMessage(w, http.StatusInternalServerError, "failed to create media item")
		return
	}

	httputil.RespondJSON(w, http.StatusCreated, item)
}

// GetMediaItem handles GET /api/media/{id}
func (h *MediaHandler) GetMediaItem(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(chi.URLParam(r, "id"))
	if err != nil {
		httputil.RespondError(w, http.StatusBadRequest, err, "invalid ID")
		return
	}

	item, err := h.service.GetMediaItem(r.Context(), id)
	if err != nil {
		if errors.Is(err, media.ErrNotFound) {
			httputil.RespondErrorMessage(w, http.StatusNotFound, "media item not found")
			return
		}
		httputil.LogError(h.logger, err, "failed to get media item", zap.Int64("id", id))
		httputil.RespondErrorMessage(w, http.StatusInternalServerError, "failed to get media item")
		return
	}

	httputil.RespondJSON(w, http.StatusOK, item)
}

// ListMediaItems handles GET /api/media
func (h *MediaHandler) ListMediaItems(w http.ResponseWriter, r *http.Request) {
	filter := media.MediaFilter{
		Limit:  20,
		Offset: 0,
	}

	// Parse query parameters
	if kindStr := r.URL.Query().Get("kind"); kindStr != "" {
		kind := media.MediaKind(kindStr)
		filter.Kind = &kind
	}

	if search := r.URL.Query().Get("q"); search != "" {
		filter.Search = &search
	}

	if parentIDStr := r.URL.Query().Get("parent_id"); parentIDStr != "" {
		parentID, err := strconv.ParseInt(parentIDStr, 10, 64)
		if err == nil {
			filter.ParentID = &parentID
		}
	}

	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if limit, err := strconv.ParseInt(limitStr, 10, 32); err == nil {
			filter.Limit = int32(limit)
		}
	}

	if offsetStr := r.URL.Query().Get("offset"); offsetStr != "" {
		if offset, err := strconv.ParseInt(offsetStr, 10, 32); err == nil {
			filter.Offset = int32(offset)
		}
	}

	list, err := h.service.ListMediaItems(r.Context(), filter)
	if err != nil {
		httputil.LogError(h.logger, err, "failed to list media items")
		httputil.RespondErrorMessage(w, http.StatusInternalServerError, "failed to list media items")
		return
	}

	httputil.RespondJSON(w, http.StatusOK, list)
}

// UpdateMediaItem handles PUT /api/media/{id}
func (h *MediaHandler) UpdateMediaItem(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(chi.URLParam(r, "id"))
	if err != nil {
		httputil.RespondError(w, http.StatusBadRequest, err, "invalid ID")
		return
	}

	var params media.UpdateMediaParams
	if err := httputil.DecodeJSON(r, &params); err != nil {
		httputil.RespondError(w, http.StatusBadRequest, err, "invalid request body")
		return
	}

	item, err := h.service.UpdateMediaItem(r.Context(), id, params)
	if err != nil {
		if errors.Is(err, media.ErrNotFound) {
			httputil.RespondErrorMessage(w, http.StatusNotFound, "media item not found")
			return
		}
		if errors.Is(err, media.ErrTitleRequired) {
			httputil.RespondError(w, http.StatusBadRequest, err, "validation error")
			return
		}
		httputil.LogError(h.logger, err, "failed to update media item", zap.Int64("id", id))
		httputil.RespondErrorMessage(w, http.StatusInternalServerError, "failed to update media item")
		return
	}

	httputil.RespondJSON(w, http.StatusOK, item)
}

// DeleteMediaItem handles DELETE /api/media/{id}
func (h *MediaHandler) DeleteMediaItem(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(chi.URLParam(r, "id"))
	if err != nil {
		httputil.RespondError(w, http.StatusBadRequest, err, "invalid ID")
		return
	}

	if err := h.service.DeleteMediaItem(r.Context(), id); err != nil {
		if errors.Is(err, media.ErrNotFound) {
			httputil.RespondErrorMessage(w, http.StatusNotFound, "media item not found")
			return
		}
		httputil.LogError(h.logger, err, "failed to delete media item", zap.Int64("id", id))
		httputil.RespondErrorMessage(w, http.StatusInternalServerError, "failed to delete media item")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// ListMovies handles GET /api/movies
func (h *MediaHandler) ListMovies(w http.ResponseWriter, r *http.Request) {
	kind := media.MediaKindMovie
	h.listByKind(w, r, kind)
}

// ListTVSeries handles GET /api/tv/series
func (h *MediaHandler) ListTVSeries(w http.ResponseWriter, r *http.Request) {
	kind := media.MediaKindTVSeries
	h.listByKind(w, r, kind)
}

// ListTVEpisodes handles GET /api/tv/series/{id}/episodes
func (h *MediaHandler) ListTVEpisodes(w http.ResponseWriter, r *http.Request) {
	parentID, err := parseID(chi.URLParam(r, "id"))
	if err != nil {
		httputil.RespondError(w, http.StatusBadRequest, err, "invalid ID")
		return
	}

	items, err := h.service.ListChildItems(r.Context(), parentID)
	if err != nil {
		httputil.LogError(h.logger, err, "failed to list episodes", zap.Int64("series_id", parentID))
		httputil.RespondErrorMessage(w, http.StatusInternalServerError, "failed to list episodes")
		return
	}

	httputil.RespondJSON(w, http.StatusOK, map[string]interface{}{
		"items": items,
		"total": len(items),
	})
}

// ListBooks handles GET /api/books
func (h *MediaHandler) ListBooks(w http.ResponseWriter, r *http.Request) {
	kind := media.MediaKindBook
	h.listByKind(w, r, kind)
}

// Helper methods

func (h *MediaHandler) listByKind(w http.ResponseWriter, r *http.Request, kind media.MediaKind) {
	filter := media.MediaFilter{
		Kind:   &kind,
		Limit:  20,
		Offset: 0,
	}

	if search := r.URL.Query().Get("q"); search != "" {
		filter.Search = &search
	}

	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if limit, err := strconv.ParseInt(limitStr, 10, 32); err == nil {
			filter.Limit = int32(limit)
		}
	}

	if offsetStr := r.URL.Query().Get("offset"); offsetStr != "" {
		if offset, err := strconv.ParseInt(offsetStr, 10, 32); err == nil {
			filter.Offset = int32(offset)
		}
	}

	list, err := h.service.ListMediaItems(r.Context(), filter)
	if err != nil {
		httputil.LogError(h.logger, err, "failed to list media items", zap.String("kind", string(kind)))
		httputil.RespondErrorMessage(w, http.StatusInternalServerError, "failed to list media items")
		return
	}

	httputil.RespondJSON(w, http.StatusOK, list)
}

func parseID(idStr string) (int64, error) {
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		return 0, errors.New("invalid ID format")
	}
	return id, nil
}
