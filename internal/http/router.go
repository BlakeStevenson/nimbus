package http

import (
	"net/http"

	"github.com/blakestevenson/nimbus/internal/auth"
	"github.com/blakestevenson/nimbus/internal/configstore"
	"github.com/blakestevenson/nimbus/internal/http/handlers"
	"github.com/blakestevenson/nimbus/internal/httputil"
	"github.com/blakestevenson/nimbus/internal/media"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"go.uber.org/zap"
)

// NewRouter creates and configures the HTTP router
func NewRouter(
	mediaService media.Service,
	authService auth.Service,
	configStore *configstore.Store,
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
	})

	return r
}
