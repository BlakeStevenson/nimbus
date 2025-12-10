package plugins

import (
	"io"
	"net/http"

	"github.com/blakestevenson/nimbus/internal/httputil"
	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"
)

// APIHandlers provides HTTP handlers for plugin management
type APIHandlers struct {
	manager *PluginManager
	logger  *zap.Logger
}

// NewAPIHandlers creates new API handlers for plugins
func NewAPIHandlers(manager *PluginManager, logger *zap.Logger) *APIHandlers {
	return &APIHandlers{
		manager: manager,
		logger:  logger.With(zap.String("component", "plugin-api")),
	}
}

// ListPlugins returns all plugins from the database
// GET /api/plugins
func (h *APIHandlers) ListPlugins(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	dbPlugins, err := h.manager.GetDBPlugins(ctx)
	if err != nil {
		h.logger.Error("Failed to list plugins", zap.Error(err))
		httputil.RespondError(w, http.StatusInternalServerError, err, "Failed to list plugins")
		return
	}

	// Convert to JSON-friendly format
	plugins := make([]map[string]interface{}, len(dbPlugins))
	for i, dbPlugin := range dbPlugins {
		plugins[i] = ConvertDBPluginToJSON(dbPlugin)
	}

	httputil.RespondJSON(w, http.StatusOK, plugins)
}

// GetPluginUIManifest returns the UI manifest for a plugin
// GET /api/plugins/{id}/ui-manifest
func (h *APIHandlers) GetPluginUIManifest(w http.ResponseWriter, r *http.Request) {
	pluginID := chi.URLParam(r, "id")

	lp, ok := h.manager.GetPlugin(pluginID)
	if !ok {
		httputil.RespondErrorMessage(w, http.StatusNotFound, "Plugin not found or not loaded")
		return
	}

	// Return UI manifest with plugin metadata
	response := map[string]interface{}{
		"id":            lp.Meta.ID,
		"displayName":   lp.Meta.Name,
		"navItems":      lp.UI.NavItems,
		"routes":        lp.UI.Routes,
		"configSection": lp.UI.ConfigSection,
	}

	httputil.RespondJSON(w, http.StatusOK, response)
}

// EnablePlugin enables a plugin
// POST /api/plugins/{id}/enable
func (h *APIHandlers) EnablePlugin(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	pluginID := chi.URLParam(r, "id")

	h.logger.Info("Enabling plugin via API", zap.String("plugin_id", pluginID))

	if err := h.manager.EnablePlugin(ctx, pluginID); err != nil {
		h.logger.Error("Failed to enable plugin",
			zap.String("plugin_id", pluginID),
			zap.Error(err))
		httputil.RespondError(w, http.StatusInternalServerError, err, "Failed to enable plugin")
		return
	}

	httputil.RespondJSON(w, http.StatusOK, map[string]interface{}{
		"message": "Plugin enabled successfully",
		"id":      pluginID,
	})
}

// DisablePlugin disables a plugin
// POST /api/plugins/{id}/disable
func (h *APIHandlers) DisablePlugin(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	pluginID := chi.URLParam(r, "id")

	h.logger.Info("Disabling plugin via API", zap.String("plugin_id", pluginID))

	if err := h.manager.DisablePlugin(ctx, pluginID); err != nil {
		h.logger.Error("Failed to disable plugin",
			zap.String("plugin_id", pluginID),
			zap.Error(err))
		httputil.RespondError(w, http.StatusInternalServerError, err, "Failed to disable plugin")
		return
	}

	httputil.RespondJSON(w, http.StatusOK, map[string]interface{}{
		"message": "Plugin disabled successfully",
		"id":      pluginID,
	})
}

// HandlePluginAPI forwards an HTTP request to a plugin (public method for router)
func (h *APIHandlers) HandlePluginAPI(w http.ResponseWriter, r *http.Request, lp *LoadedPlugin, route RouteDescriptor) {
	h.makePluginAPIHandler(lp, route)(w, r)
}

// makePluginAPIHandler creates an HTTP handler that forwards requests to a plugin
func (h *APIHandlers) makePluginAPIHandler(lp *LoadedPlugin, route RouteDescriptor) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		// Read request body
		body, err := io.ReadAll(r.Body)
		if err != nil {
			h.logger.Error("Failed to read request body", zap.Error(err))
			httputil.RespondError(w, http.StatusBadRequest, err, "Failed to read request body")
			return
		}
		defer r.Body.Close()

		// Build plugin request
		pluginReq := &PluginHTTPRequest{
			Method:  r.Method,
			Path:    r.URL.Path,
			Query:   r.URL.Query(),
			Headers: r.Header,
			Body:    body,
		}

		// Extract user context if authenticated (set by auth middleware using key "user")
		if userID := getUserIDFromRequest(r); userID != nil {
			pluginReq.UserID = userID
		}

		// TODO: Extract scopes from context if applicable
		// This could be based on user roles or API key permissions
		// pluginReq.Scopes = extractScopes(ctx)

		// Log before forwarding to plugin
		h.logger.Info("Forwarding request to plugin",
			zap.String("plugin_id", lp.Meta.ID),
			zap.String("path", r.URL.Path),
			zap.String("method", r.Method))

		// Forward request to plugin
		pluginResp, err := lp.Client.HandleAPI(ctx, pluginReq)
		if err != nil {
			h.logger.Error("Plugin API handler failed",
				zap.String("plugin_id", lp.Meta.ID),
				zap.String("path", r.URL.Path),
				zap.Error(err))
			httputil.RespondError(w, http.StatusInternalServerError, err, "Plugin error")
			return
		}

		h.logger.Info("Plugin returned response",
			zap.String("plugin_id", lp.Meta.ID),
			zap.Int("status_code", pluginResp.StatusCode))

		// Write response headers
		for k, values := range pluginResp.Headers {
			for _, v := range values {
				w.Header().Add(k, v)
			}
		}

		// Write status code and body
		w.WriteHeader(pluginResp.StatusCode)
		if len(pluginResp.Body) > 0 {
			if _, err := w.Write(pluginResp.Body); err != nil {
				h.logger.Error("Failed to write response body", zap.Error(err))
			}
		}
	}
}

// ServePluginStatic serves static files from a plugin's web directory
// GET /plugins/{id}/*
func (h *APIHandlers) ServePluginStatic(w http.ResponseWriter, r *http.Request) {
	pluginID := chi.URLParam(r, "id")
	filePath := chi.URLParam(r, "*")

	realPath, err := h.manager.ServePluginFile(pluginID, filePath)
	if err != nil {
		h.logger.Warn("Failed to serve plugin file",
			zap.String("plugin_id", pluginID),
			zap.String("file", filePath),
			zap.Error(err))
		http.NotFound(w, r)
		return
	}

	http.ServeFile(w, r, realPath)
}

// getUserClaims extracts user claims from the request context
// Note: Must use the same context key string as auth middleware ("user")
func getUserClaims(r *http.Request) (map[string]interface{}, bool) {
	claims, ok := r.Context().Value("user").(map[string]interface{})
	return claims, ok
}

// getUserIDFromRequest extracts the user ID from the request context
func getUserIDFromRequest(r *http.Request) *int64 {
	// Try to get from claims context (set by auth middleware)
	if claims, ok := r.Context().Value("user").(map[string]interface{}); ok {
		if userID, ok := claims["user_id"].(int64); ok {
			return &userID
		}
		// Try float64 (JSON numbers)
		if userIDFloat, ok := claims["user_id"].(float64); ok {
			userID := int64(userIDFloat)
			return &userID
		}
	}
	return nil
}
