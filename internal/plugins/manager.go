package plugins

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sync"

	"github.com/blakestevenson/nimbus/internal/configstore"
	"github.com/blakestevenson/nimbus/internal/db/generated"
	"github.com/hashicorp/go-plugin"
	"go.uber.org/zap"
)

// LoadedPlugin represents a plugin that has been loaded and is running
type LoadedPlugin struct {
	Meta      *PluginMetadata
	Client    MediaSuitePlugin // RPC client
	Routes    []RouteDescriptor
	UI        *UIManifest
	RawClient *plugin.Client // The underlying go-plugin client
}

// PluginManager manages the lifecycle of plugins
type PluginManager struct {
	queries     *generated.Queries
	configStore *configstore.Store
	logger      *zap.Logger
	pluginsDir  string
	sdk         *SDK

	mu      sync.RWMutex
	plugins map[string]*LoadedPlugin
}

// PluginManifest is the manifest.json file in each plugin directory
type PluginManifest struct {
	ID           string   `json:"id"`
	Name         string   `json:"name"`
	Description  string   `json:"description"`
	Version      string   `json:"version"`
	Executable   string   `json:"executable"` // Relative path to binary (e.g., "plugin")
	WebDir       string   `json:"webDir"`     // Relative path to web assets (e.g., "web")
	Capabilities []string `json:"capabilities"`
}

// NewPluginManager creates a new plugin manager
func NewPluginManager(
	queries *generated.Queries,
	configStore *configstore.Store,
	logger *zap.Logger,
	pluginsDir string,
) *PluginManager {
	return &PluginManager{
		queries:     queries,
		configStore: configStore,
		logger:      logger.With(zap.String("component", "plugin-manager")),
		pluginsDir:  pluginsDir,
		sdk:         NewSDK(queries, configStore, logger),
		plugins:     make(map[string]*LoadedPlugin),
	}
}

// Initialize discovers and loads all enabled plugins
func (pm *PluginManager) Initialize(ctx context.Context) error {
	pm.logger.Info("Initializing plugin manager", zap.String("plugins_dir", pm.pluginsDir))

	// Ensure plugins directory exists
	if err := os.MkdirAll(pm.pluginsDir, 0755); err != nil {
		return fmt.Errorf("failed to create plugins directory: %w", err)
	}

	// Discover plugins from filesystem
	manifests, err := pm.discoverPlugins()
	if err != nil {
		return fmt.Errorf("failed to discover plugins: %w", err)
	}

	pm.logger.Info("Discovered plugins", zap.Int("count", len(manifests)))

	// Upsert plugin metadata into database
	for _, manifest := range manifests {
		if err := pm.upsertPluginMetadata(ctx, manifest); err != nil {
			pm.logger.Error("Failed to upsert plugin metadata",
				zap.String("plugin_id", manifest.ID),
				zap.Error(err))
		}
	}

	// Load enabled plugins
	enabledPlugins, err := pm.queries.ListEnabledPlugins(ctx)
	if err != nil {
		return fmt.Errorf("failed to list enabled plugins: %w", err)
	}

	for _, dbPlugin := range enabledPlugins {
		if err := pm.loadPlugin(ctx, dbPlugin.ID); err != nil {
			pm.logger.Error("Failed to load plugin",
				zap.String("plugin_id", dbPlugin.ID),
				zap.Error(err))
			continue
		}
	}

	pm.logger.Info("Plugin manager initialized", zap.Int("loaded", len(pm.plugins)))
	return nil
}

// Shutdown stops all running plugins
func (pm *PluginManager) Shutdown() {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	pm.logger.Info("Shutting down plugin manager")

	for id, lp := range pm.plugins {
		pm.logger.Info("Stopping plugin", zap.String("plugin_id", id))
		if lp.RawClient != nil {
			lp.RawClient.Kill()
		}
	}

	pm.plugins = make(map[string]*LoadedPlugin)
}

// ListPlugins returns all loaded plugins
func (pm *PluginManager) ListPlugins() []*LoadedPlugin {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	plugins := make([]*LoadedPlugin, 0, len(pm.plugins))
	for _, lp := range pm.plugins {
		plugins = append(plugins, lp)
	}

	return plugins
}

// GetPlugin returns a loaded plugin by ID
func (pm *PluginManager) GetPlugin(id string) (*LoadedPlugin, bool) {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	lp, ok := pm.plugins[id]
	return lp, ok
}

// EnablePlugin enables a plugin and loads it
func (pm *PluginManager) EnablePlugin(ctx context.Context, id string) error {
	pm.logger.Info("Enabling plugin", zap.String("plugin_id", id))

	// Enable in database
	if err := pm.queries.EnablePlugin(ctx, id); err != nil {
		return fmt.Errorf("failed to enable plugin in database: %w", err)
	}

	// Load the plugin
	return pm.loadPlugin(ctx, id)
}

// DisablePlugin disables a plugin and stops it
func (pm *PluginManager) DisablePlugin(ctx context.Context, id string) error {
	pm.logger.Info("Disabling plugin", zap.String("plugin_id", id))

	// Disable in database
	if err := pm.queries.DisablePlugin(ctx, id); err != nil {
		return fmt.Errorf("failed to disable plugin in database: %w", err)
	}

	// Unload the plugin
	pm.mu.Lock()
	defer pm.mu.Unlock()

	if lp, ok := pm.plugins[id]; ok {
		if lp.RawClient != nil {
			lp.RawClient.Kill()
		}
		delete(pm.plugins, id)
	}

	return nil
}

// ============================================================================
// Internal Methods
// ============================================================================

// discoverPlugins scans the plugins directory for manifest.json files
func (pm *PluginManager) discoverPlugins() ([]PluginManifest, error) {
	entries, err := os.ReadDir(pm.pluginsDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read plugins directory: %w", err)
	}

	var manifests []PluginManifest

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		manifestPath := filepath.Join(pm.pluginsDir, entry.Name(), "manifest.json")
		data, err := os.ReadFile(manifestPath)
		if err != nil {
			pm.logger.Warn("Failed to read manifest",
				zap.String("path", manifestPath),
				zap.Error(err))
			continue
		}

		var manifest PluginManifest
		if err := json.Unmarshal(data, &manifest); err != nil {
			pm.logger.Warn("Failed to parse manifest",
				zap.String("path", manifestPath),
				zap.Error(err))
			continue
		}

		manifests = append(manifests, manifest)
	}

	return manifests, nil
}

// upsertPluginMetadata updates the database with plugin metadata
func (pm *PluginManager) upsertPluginMetadata(ctx context.Context, manifest PluginManifest) error {
	capabilities, err := json.Marshal(manifest.Capabilities)
	if err != nil {
		return fmt.Errorf("failed to marshal capabilities: %w", err)
	}

	_, err = pm.queries.UpsertPlugin(ctx, generated.UpsertPluginParams{
		ID:           manifest.ID,
		Name:         manifest.Name,
		Description:  manifest.Description,
		Version:      manifest.Version,
		Enabled:      true, // Default to enabled
		Capabilities: capabilities,
	})

	return err
}

// loadPlugin starts a plugin process and loads its metadata
func (pm *PluginManager) loadPlugin(ctx context.Context, id string) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	// Check if already loaded
	if _, ok := pm.plugins[id]; ok {
		return nil // Already loaded
	}

	// Find manifest
	manifestPath := filepath.Join(pm.pluginsDir, id, "manifest.json")
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		return fmt.Errorf("failed to read manifest: %w", err)
	}

	var manifest PluginManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return fmt.Errorf("failed to parse manifest: %w", err)
	}

	// Build path to executable
	execPath := filepath.Join(pm.pluginsDir, id, manifest.Executable)
	if _, err := os.Stat(execPath); err != nil {
		return fmt.Errorf("plugin executable not found: %w", err)
	}

	pm.logger.Info("Starting plugin process",
		zap.String("plugin_id", id),
		zap.String("executable", execPath))

	// Start plugin process
	client := plugin.NewClient(&plugin.ClientConfig{
		HandshakeConfig: Handshake,
		Plugins:         PluginMap,
		Cmd:             exec.Command(execPath),
		AllowedProtocols: []plugin.Protocol{
			plugin.ProtocolGRPC,
		},
		// Skip logger for now - go-plugin expects hclog.Logger
		// Logger: pm.logger.Named(id).Sugar(),
	})

	// Connect via RPC
	rpcClient, err := client.Client()
	if err != nil {
		client.Kill()
		return fmt.Errorf("failed to get RPC client: %w", err)
	}

	// Request the plugin
	raw, err := rpcClient.Dispense("media-suite")
	if err != nil {
		client.Kill()
		return fmt.Errorf("failed to dispense plugin: %w", err)
	}

	pluginClient := raw.(MediaSuitePlugin)

	// Fetch metadata
	meta, err := pluginClient.Metadata(ctx)
	if err != nil {
		client.Kill()
		return fmt.Errorf("failed to get plugin metadata: %w", err)
	}

	// Fetch API routes
	routes, err := pluginClient.APIRoutes(ctx)
	if err != nil {
		pm.logger.Warn("Failed to get API routes", zap.Error(err))
		routes = []RouteDescriptor{}
	}

	// Fetch UI manifest
	uiManifest, err := pluginClient.UIManifest(ctx)
	if err != nil {
		pm.logger.Warn("Failed to get UI manifest", zap.Error(err))
		uiManifest = &UIManifest{
			NavItems: []UINavItem{},
			Routes:   []UIRoute{},
		}
	}

	// Store loaded plugin
	pm.plugins[id] = &LoadedPlugin{
		Meta:      meta,
		Client:    pluginClient,
		Routes:    routes,
		UI:        uiManifest,
		RawClient: client,
	}

	pm.logger.Info("Plugin loaded successfully",
		zap.String("plugin_id", id),
		zap.String("plugin_name", meta.Name),
		zap.String("version", meta.Version),
		zap.Int("routes", len(routes)))

	return nil
}

// GetPluginsDir returns the plugins directory path
func (pm *PluginManager) GetPluginsDir() string {
	return pm.pluginsDir
}

// GetSDK returns the SDK instance for use by internal systems
func (pm *PluginManager) GetSDK() *SDK {
	return pm.sdk
}

// GetDBPlugins retrieves all plugins from the database (for API responses)
func (pm *PluginManager) GetDBPlugins(ctx context.Context) ([]generated.Plugin, error) {
	return pm.queries.ListPlugins(ctx)
}

// ConvertDBPluginToJSON converts a database plugin to a JSON-serializable format
func ConvertDBPluginToJSON(dbPlugin generated.Plugin) map[string]interface{} {
	var capabilities []string
	if len(dbPlugin.Capabilities) > 0 {
		_ = json.Unmarshal(dbPlugin.Capabilities, &capabilities)
	}

	return map[string]interface{}{
		"id":           dbPlugin.ID,
		"name":         dbPlugin.Name,
		"description":  dbPlugin.Description,
		"version":      dbPlugin.Version,
		"enabled":      dbPlugin.Enabled,
		"capabilities": capabilities,
		"created_at":   dbPlugin.CreatedAt.Time,
		"updated_at":   dbPlugin.UpdatedAt.Time,
	}
}

// ServePluginFile serves a static file from a plugin's web directory
func (pm *PluginManager) ServePluginFile(pluginID, filePath string) (string, error) {
	// Security: prevent directory traversal
	cleanPath := filepath.Clean(filePath)
	if filepath.IsAbs(cleanPath) || len(cleanPath) > 0 && cleanPath[0] == '.' {
		return "", fmt.Errorf("invalid file path")
	}

	// Build full path
	fullPath := filepath.Join(pm.pluginsDir, pluginID, "web", cleanPath)

	// Verify file exists and is within plugin directory
	realPath, err := filepath.EvalSymlinks(fullPath)
	if err != nil {
		return "", fmt.Errorf("file not found: %w", err)
	}

	expectedPrefix := filepath.Join(pm.pluginsDir, pluginID, "web")
	if !filepath.HasPrefix(realPath, expectedPrefix) {
		return "", fmt.Errorf("path traversal detected")
	}

	return realPath, nil
}

// RegisterRoutes registers plugin API routes with the router
// This should be called after plugins are loaded
func (pm *PluginManager) RegisterRoutes(router interface{}, handlers *APIHandlers) {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	// Type assert to chi.Router
	chiRouter, ok := router.(interface {
		Method(method, pattern string, h http.HandlerFunc)
		MethodFunc(method, pattern string, h http.HandlerFunc)
	})
	if !ok {
		pm.logger.Error("Invalid router type provided to RegisterRoutes")
		return
	}

	for id, lp := range pm.plugins {
		pm.logger.Info("Registering plugin routes",
			zap.String("plugin_id", id),
			zap.Int("route_count", len(lp.Routes)))

		for _, route := range lp.Routes {
			handler := handlers.makePluginAPIHandler(lp, route)
			chiRouter.Method(route.Method, route.Path, handler)

			pm.logger.Debug("Registered plugin route",
				zap.String("plugin_id", id),
				zap.String("method", route.Method),
				zap.String("path", route.Path))
		}
	}
}

// manifestToUpsertParams helper to convert PluginManifest to upsert params
func manifestToUpsertParams(manifest PluginManifest, enabled bool) (generated.UpsertPluginParams, error) {
	capabilities, err := json.Marshal(manifest.Capabilities)
	if err != nil {
		return generated.UpsertPluginParams{}, err
	}

	return generated.UpsertPluginParams{
		ID:           manifest.ID,
		Name:         manifest.Name,
		Description:  manifest.Description,
		Version:      manifest.Version,
		Enabled:      enabled,
		Capabilities: capabilities,
	}, nil
}
