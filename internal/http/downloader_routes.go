package http

import (
	"encoding/json"
	"net/http"

	"github.com/blakestevenson/nimbus/internal/configstore"
	"github.com/blakestevenson/nimbus/internal/db/generated"
	"github.com/blakestevenson/nimbus/internal/downloader"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

// setupDownloaderRoutes registers the unified downloader API endpoints
func setupDownloaderRoutes(r chi.Router, downloaderService *downloader.Service, queries *generated.Queries, configStore *configstore.Store, db *pgxpool.Pool, logger *zap.Logger) {
	// List available downloaders
	r.Get("/downloaders", func(w http.ResponseWriter, r *http.Request) {
		downloaders := downloaderService.ListDownloaders()

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(map[string]interface{}{
			"downloaders": downloaders,
			"count":       len(downloaders),
		}); err != nil {
			logger.Error("Failed to encode downloaders response", zap.Error(err))
			http.Error(w, "Internal server error", http.StatusInternalServerError)
		}
	})

	// List all downloads (optionally filtered)
	r.Get("/downloads", func(w http.ResponseWriter, r *http.Request) {
		pluginID := r.URL.Query().Get("plugin_id")
		status := r.URL.Query().Get("status")

		resp, err := downloaderService.ListDownloads(r.Context(), pluginID, status)
		if err != nil {
			logger.Error("Failed to list downloads", zap.Error(err))
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(map[string]interface{}{
			"downloads": resp.Downloads,
			"total":     resp.Total,
		}); err != nil {
			logger.Error("Failed to encode downloads response", zap.Error(err))
			http.Error(w, "Internal server error", http.StatusInternalServerError)
		}
	})

	// Create a new download
	r.Post("/downloads", func(w http.ResponseWriter, r *http.Request) {
		var req downloader.DownloadRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			logger.Error("Failed to decode request body", zap.Error(err))
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		logger.Info("Received download request",
			zap.String("plugin_id", req.PluginID),
			zap.String("name", req.Name),
			zap.String("url", req.URL))

		download, err := downloaderService.CreateDownload(r.Context(), req)
		if err != nil {
			logger.Error("Failed to create download", zap.Error(err))
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		if err := json.NewEncoder(w).Encode(download); err != nil {
			logger.Error("Failed to encode download response", zap.Error(err))
		}
	})

	// Get a specific download
	r.Get("/downloads/{plugin_id}/{download_id}", func(w http.ResponseWriter, r *http.Request) {
		pluginID := chi.URLParam(r, "plugin_id")
		downloadID := chi.URLParam(r, "download_id")

		download, err := downloaderService.GetDownload(r.Context(), downloadID, pluginID)
		if err != nil {
			logger.Error("Failed to get download", zap.Error(err))
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(download); err != nil {
			logger.Error("Failed to encode download response", zap.Error(err))
		}
	})

	// Pause a download
	r.Post("/downloads/{plugin_id}/{download_id}/pause", func(w http.ResponseWriter, r *http.Request) {
		pluginID := chi.URLParam(r, "plugin_id")
		downloadID := chi.URLParam(r, "download_id")

		if err := downloaderService.PauseDownload(r.Context(), downloadID, pluginID); err != nil {
			logger.Error("Failed to pause download", zap.Error(err))
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusNoContent)
	})

	// Resume a download
	r.Post("/downloads/{plugin_id}/{download_id}/resume", func(w http.ResponseWriter, r *http.Request) {
		pluginID := chi.URLParam(r, "plugin_id")
		downloadID := chi.URLParam(r, "download_id")

		if err := downloaderService.ResumeDownload(r.Context(), downloadID, pluginID); err != nil {
			logger.Error("Failed to resume download", zap.Error(err))
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusNoContent)
	})

	// Retry a download
	r.Post("/downloads/{plugin_id}/{download_id}/retry", func(w http.ResponseWriter, r *http.Request) {
		pluginID := chi.URLParam(r, "plugin_id")
		downloadID := chi.URLParam(r, "download_id")

		if err := downloaderService.RetryDownload(r.Context(), downloadID, pluginID); err != nil {
			logger.Error("Failed to retry download", zap.Error(err))
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusNoContent)
	})

	// Cancel/delete a download
	r.Delete("/downloads/{plugin_id}/{download_id}", func(w http.ResponseWriter, r *http.Request) {
		pluginID := chi.URLParam(r, "plugin_id")
		downloadID := chi.URLParam(r, "download_id")

		if err := downloaderService.CancelDownload(r.Context(), downloadID, pluginID); err != nil {
			logger.Error("Failed to cancel download", zap.Error(err))
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusNoContent)
	})
}
