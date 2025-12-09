package http

import (
	"context"
	"net/http"

	"github.com/blakestevenson/nimbus/internal/auth"
	"github.com/blakestevenson/nimbus/internal/plugins"
	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"
)

// setupPluginRoutes sets up the plugin management API routes
func setupPluginRoutes(r chi.Router, pluginManager interface{}, logger *zap.Logger) {
	pm, ok := pluginManager.(*plugins.PluginManager)
	if !ok {
		logger.Error("Invalid plugin manager type")
		return
	}

	handlers := plugins.NewAPIHandlers(pm, logger)

	r.Route("/plugins", func(r chi.Router) {
		r.Get("/", handlers.ListPlugins)
		r.Get("/{id}/ui-manifest", handlers.GetPluginUIManifest)
		r.Post("/{id}/enable", handlers.EnablePlugin)
		r.Post("/{id}/disable", handlers.DisablePlugin)
	})
}

// setupPluginStaticRoutes sets up routes for serving plugin static files
func setupPluginStaticRoutes(r chi.Router, pluginManager interface{}, logger *zap.Logger) {
	pm, ok := pluginManager.(*plugins.PluginManager)
	if !ok {
		logger.Error("Invalid plugin manager type")
		return
	}

	handlers := plugins.NewAPIHandlers(pm, logger)

	// Serve plugin static files (JS bundles, assets, etc.)
	r.Get("/plugins/{id}/*", handlers.ServePluginStatic)
}

// registerPluginAPIRoutes registers plugin-provided API routes
func registerPluginAPIRoutes(r chi.Router, pluginManager interface{}, authService auth.Service, logger *zap.Logger) {
	pm, ok := pluginManager.(*plugins.PluginManager)
	if !ok {
		logger.Error("Invalid plugin manager type")
		return
	}

	handlers := plugins.NewAPIHandlers(pm, logger)

	// Get all loaded plugins and register their routes
	loadedPlugins := pm.ListPlugins()

	for _, lp := range loadedPlugins {
		logger.Info("Registering plugin API routes",
			zap.String("plugin_id", lp.Meta.ID),
			zap.Int("route_count", len(lp.Routes)))

		for _, route := range lp.Routes {
			handler := makePluginRouteHandler(lp, route, handlers, authService, logger)

			r.Method(route.Method, route.Path, handler)

			logger.Debug("Registered plugin route",
				zap.String("plugin_id", lp.Meta.ID),
				zap.String("method", route.Method),
				zap.String("path", route.Path),
				zap.String("auth", route.Auth))
		}
	}
}

// makePluginRouteHandler creates an HTTP handler for a plugin route with appropriate auth
func makePluginRouteHandler(
	lp *plugins.LoadedPlugin,
	route plugins.RouteDescriptor,
	handlers *plugins.APIHandlers,
	authService auth.Service,
	logger *zap.Logger,
) http.HandlerFunc {
	baseHandler := makePluginAPIHandlerWrapper(lp, route, handlers)

	// Apply authentication based on route.Auth
	switch route.Auth {
	case "session":
		// Require authenticated session
		return func(w http.ResponseWriter, r *http.Request) {
			// Apply auth middleware inline
			if err := checkAuth(r, authService); err != nil {
				logger.Warn("Plugin route auth failed",
					zap.String("plugin_id", lp.Meta.ID),
					zap.String("path", route.Path),
					zap.Error(err))
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}
			baseHandler(w, r)
		}

	case "apikey":
		// TODO: Implement API key authentication
		// For now, treat as "none"
		logger.Warn("API key auth not yet implemented for plugin route",
			zap.String("plugin_id", lp.Meta.ID),
			zap.String("path", route.Path))
		return baseHandler

	case "none":
		fallthrough
	default:
		// No authentication required
		return baseHandler
	}
}

// makePluginAPIHandlerWrapper wraps the plugin API handler
func makePluginAPIHandlerWrapper(
	lp *plugins.LoadedPlugin,
	route plugins.RouteDescriptor,
	handlers *plugins.APIHandlers,
) http.HandlerFunc {
	// This is a bit of a hack to get access to the private method
	// In production, you might want to make this method public or refactor
	return func(w http.ResponseWriter, r *http.Request) {
		// Forward to plugin via the handlers
		handlers.HandlePluginAPI(w, r, lp, route)
	}
}

// checkAuth is a helper to check authentication inline
func checkAuth(r *http.Request, authService auth.Service) error {
	token, err := extractToken(r)
	if err != nil {
		return err
	}

	claims, err := authService.ValidateToken(r.Context(), token)
	if err != nil {
		return err
	}

	// Add claims to context
	ctx := r.Context()
	ctx = contextWithUser(ctx, claims)
	*r = *r.WithContext(ctx)

	return nil
}

// Helper to extract token from Authorization header or cookie
func extractToken(r *http.Request) (string, error) {
	// Check Authorization header first
	authHeader := r.Header.Get("Authorization")
	if authHeader != "" {
		// Bearer token
		if len(authHeader) > 7 && authHeader[:7] == "Bearer " {
			return authHeader[7:], nil
		}
	}

	// Check cookie
	cookie, err := r.Cookie("access_token")
	if err == nil {
		return cookie.Value, nil
	}

	return "", http.ErrNoCookie
}

// contextWithUser adds user claims to context
func contextWithUser(ctx context.Context, claims *auth.Claims) context.Context {
	return context.WithValue(ctx, "user", claims)
}
