package http

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/blakestevenson/nimbus/internal/auth"
	"github.com/blakestevenson/nimbus/internal/configstore"
	"github.com/blakestevenson/nimbus/internal/db/generated"
	"github.com/blakestevenson/nimbus/internal/downloader"
	"github.com/blakestevenson/nimbus/internal/http/handlers"
	"github.com/blakestevenson/nimbus/internal/httputil"
	"github.com/blakestevenson/nimbus/internal/indexer"
	"github.com/blakestevenson/nimbus/internal/library"
	"github.com/blakestevenson/nimbus/internal/media"
	"github.com/blakestevenson/nimbus/internal/plugins"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

// NewRouter creates and configures the HTTP router
func NewRouter(
	mediaService media.Service,
	authService auth.Service,
	configStore *configstore.Store,
	queries *generated.Queries,
	db interface{}, // *pgxpool.Pool
	libraryRootPath string,
	pluginManager interface{}, // *plugins.PluginManager or nil
	logger *zap.Logger,
) http.Handler {
	r := chi.NewRouter()

	// Middleware
	r.Use(RecoverMiddleware(logger))
	r.Use(LoggingMiddleware(logger))
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(CORSMiddleware)
	r.Use(middleware.Compress(5))

	// Handlers
	mediaHandler := handlers.NewMediaHandler(mediaService, logger)
	authHandler := handlers.NewAuthHandler(authService, logger)
	configHandler := handlers.NewConfigHandler(configStore, logger)
	libraryHandler := library.NewHandler(queries, logger, libraryRootPath)
	fileHandler := library.NewFileHandler(queries, logger)

	// Load media-specific library paths from config
	ctx := context.Background()
	mediaPathConfigs := map[string]string{
		"movie": "library.movie_path",
		"tv":    "library.tv_path",
		"music": "library.music_path",
		"book":  "library.book_path",
	}

	for mediaType, configKey := range mediaPathConfigs {
		if pathValue, err := configStore.Get(ctx, configKey); err == nil {
			var path string
			if err := json.Unmarshal(pathValue, &path); err == nil && path != "" {
				libraryHandler.SetMediaPath(mediaType, path)
				logger.Info("loaded media-specific path",
					zap.String("media_type", mediaType),
					zap.String("path", path))
			}
		}
	}

	// Initialize indexer service if plugin manager is available
	var indexerService *indexer.Service
	if pluginManager != nil {
		if pm, ok := pluginManager.(*plugins.PluginManager); ok {
			indexerService = indexer.NewService(pm, logger)
		}
	}

	// Initialize downloader service if plugin manager is available
	var downloaderService *downloader.Service
	if pluginManager != nil && db != nil {
		logger.Info("Setting up downloader service")
		if pm, ok := pluginManager.(*plugins.PluginManager); ok {
			// Cast db to pgxpool.Pool
			if dbPool, ok := db.(*pgxpool.Pool); ok {
				logger.Info("Creating downloader service")
				downloaderService = downloader.NewService(pm, dbPool, logger)
				// Sync pending downloads from database to plugin queues
				logger.Info("Initializing downloader service")
				if err := downloaderService.Initialize(context.Background()); err != nil {
					logger.Error("Failed to initialize downloader service", zap.Error(err))
				}
			} else {
				logger.Warn("db is not a pgxpool.Pool")
			}
		} else {
			logger.Warn("pluginManager is not a *plugins.PluginManager")
		}
	} else {
		logger.Warn("pluginManager or db is nil", zap.Bool("pm_nil", pluginManager == nil), zap.Bool("db_nil", db == nil))
	}

	// Health check
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		httputil.RespondJSON(w, http.StatusOK, map[string]string{
			"status": "ok",
		})
	})

	// API routes
	r.Route("/api", func(r chi.Router) {
		// Public auth routes (no authentication required)
		r.Route("/auth", func(r chi.Router) {
			r.Post("/register", authHandler.Register)
			r.Post("/login", authHandler.Login)
			r.Post("/refresh", authHandler.RefreshToken)
			r.Post("/logout", authHandler.Logout)
		})

		// Protected auth routes (require authentication)
		r.Group(func(r chi.Router) {
			r.Use(AuthMiddleware(authService, logger))

			r.Get("/auth/me", authHandler.Me)
			r.Put("/auth/me", authHandler.UpdateProfile)
		})

		// Protected media routes (require authentication)
		r.Group(func(r chi.Router) {
			r.Use(AuthMiddleware(authService, logger))

			// Media routes
			r.Route("/media", func(r chi.Router) {
				r.Get("/", mediaHandler.ListMediaItems)
				r.Post("/", mediaHandler.CreateMediaItem)
				r.Get("/{id}", mediaHandler.GetMediaItem)
				r.Put("/{id}", mediaHandler.UpdateMediaItem)
				r.Delete("/{id}", mediaHandler.DeleteMediaItem)

				// Media file routes
				r.Get("/{id}/files", fileHandler.GetMediaFiles)
				r.Delete("/{id}/with-files", fileHandler.DeleteMediaItemWithFiles)

				// Individual file deletion
				r.Delete("/files/{fileId}", fileHandler.DeleteMediaFile)

				// Interactive search route (if indexer service is available)
				if indexerService != nil {
					setupSearchRoutes(r, indexerService, queries, logger)
				}
			})

			// Type-specific convenience routes
			r.Get("/movies", mediaHandler.ListMovies)
			r.Route("/tv", func(r chi.Router) {
				r.Get("/series", mediaHandler.ListTVSeries)
				r.Get("/series/{id}/episodes", mediaHandler.ListTVEpisodes)
			})
			r.Get("/books", mediaHandler.ListBooks)
		})

		// Protected config routes (require authentication and admin)
		r.Group(func(r chi.Router) {
			r.Use(AuthMiddleware(authService, logger))
			r.Use(RequireAdminMiddleware(logger))

			r.Route("/config", func(r chi.Router) {
				r.Get("/", configHandler.ListConfig)
				r.Get("/{key}", configHandler.GetConfig)
				r.Put("/{key}", configHandler.SetConfig)
				r.Delete("/{key}", configHandler.DeleteConfig)
			})
		})

		// Protected library routes (require authentication)
		r.Group(func(r chi.Router) {
			r.Use(AuthMiddleware(authService, logger))

			r.Route("/library", func(r chi.Router) {
				// Status endpoint - available to all authenticated users
				r.Get("/scan/status", libraryHandler.GetScanStatus)

				// Admin-only endpoints
				r.Group(func(r chi.Router) {
					r.Use(RequireAdminMiddleware(logger))

					r.Post("/scan", libraryHandler.StartScan)
					r.Post("/scan/stop", libraryHandler.StopScan)
					r.Post("/scan/reset", libraryHandler.ResetScanner)
				})
			})
		})

		// Unified indexer routes (require authentication)
		if indexerService != nil {
			r.Group(func(r chi.Router) {
				r.Use(AuthMiddleware(authService, logger))

				setupIndexerRoutes(r, indexerService, logger)
			})
		}

		// Internal API routes (no authentication required - for plugin-to-host communication)
		if downloaderService != nil {
			if dbPool, ok := db.(*pgxpool.Pool); ok {
				// Create download handler for internal routes
				downloadHandler := downloader.NewHandler(downloaderService, queries, configStore, dbPool, logger)

				// Import endpoint - internal use by plugins only
				r.Post("/downloads/import", downloadHandler.ImportCompletedDownload)
			}
		}

		// Internal media query endpoint - for plugins to look up media items
		r.Get("/internal/media", mediaHandler.ListMediaItems)

		// Internal download sync endpoint - for plugins to sync download state to database
		if downloaderService != nil {
			r.Put("/internal/downloads/{id}", func(w http.ResponseWriter, r *http.Request) {
				downloadID := chi.URLParam(r, "id")

				var payload map[string]interface{}
				if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
					http.Error(w, "Invalid request body", http.StatusBadRequest)
					return
				}

				// Upsert download to database
				if err := downloaderService.UpsertDownload(r.Context(), downloadID, payload); err != nil {
					logger.Error("Failed to upsert download", zap.Error(err), zap.String("id", downloadID))
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}

				w.WriteHeader(http.StatusOK)
			})
		}

		// Unified downloader routes (require authentication)
		if downloaderService != nil {
			r.Group(func(r chi.Router) {
				r.Use(AuthMiddleware(authService, logger))

				// Cast db to pgxpool.Pool for downloader routes
				if dbPool, ok := db.(*pgxpool.Pool); ok {
					setupDownloaderRoutes(r, downloaderService, queries, configStore, dbPool, logger)
				}
			})
		}

		// Plugin management routes (require authentication and admin)
		if pluginManager != nil {
			r.Group(func(r chi.Router) {
				r.Use(AuthMiddleware(authService, logger))
				r.Use(RequireAdminMiddleware(logger))

				setupPluginRoutes(r, pluginManager, logger)
			})
		}
	})

	// Serve plugin static files (no auth required for bundles)
	if pluginManager != nil {
		setupPluginStaticRoutes(r, pluginManager, logger)
	}

	// Register plugin API routes (auth handled per-route by plugin descriptor)
	if pluginManager != nil {
		registerPluginAPIRoutes(r, pluginManager, authService, logger)
	}

	return r
}
