package http

import (
	"net/http"

	"github.com/blakestevenson/nimbus/internal/auth"
	"github.com/blakestevenson/nimbus/internal/configstore"
	"github.com/blakestevenson/nimbus/internal/db/generated"
	"github.com/blakestevenson/nimbus/internal/http/handlers"
	"github.com/blakestevenson/nimbus/internal/httputil"
	"github.com/blakestevenson/nimbus/internal/indexer"
	"github.com/blakestevenson/nimbus/internal/library"
	"github.com/blakestevenson/nimbus/internal/media"
	"github.com/blakestevenson/nimbus/internal/plugins"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"go.uber.org/zap"
)

// NewRouter creates and configures the HTTP router
func NewRouter(
	mediaService media.Service,
	authService auth.Service,
	configStore *configstore.Store,
	queries *generated.Queries,
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

	// Initialize indexer service if plugin manager is available
	var indexerService *indexer.Service
	if pluginManager != nil {
		logger.Debug("Plugin manager provided to router")
		if pm, ok := pluginManager.(*plugins.PluginManager); ok {
			indexerService = indexer.NewService(pm, logger)
			logger.Info("Indexer service initialized")
		}
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
