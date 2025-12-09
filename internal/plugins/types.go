package plugins

import (
	"context"
	"time"
)

// PluginMetadata contains basic information about a plugin
type PluginMetadata struct {
	ID           string   `json:"id"`
	Name         string   `json:"name"`
	Version      string   `json:"version"`
	Description  string   `json:"description"`
	Capabilities []string `json:"capabilities"` // ["api", "ui", "events", "compat:sonarr"]
}

// RouteDescriptor describes an HTTP route that a plugin wants to register
type RouteDescriptor struct {
	Method string `json:"method"` // "GET", "POST", "PUT", "DELETE", "PATCH"
	Path   string `json:"path"`   // e.g., "/api/plugins/sonarr/series"
	Auth   string `json:"auth"`   // "session", "apikey", "none"
	Tag    string `json:"tag"`    // Optional: "compat:sonarr", "internal", etc.
}

// PluginHTTPRequest represents an HTTP request forwarded to a plugin
type PluginHTTPRequest struct {
	Method  string              `json:"method"`
	Path    string              `json:"path"`
	Query   map[string][]string `json:"query"`
	Headers map[string][]string `json:"headers"`
	Body    []byte              `json:"body"`

	// Auth context (populated by middleware)
	UserID *int64   `json:"user_id,omitempty"`
	Scopes []string `json:"scopes,omitempty"`
}

// PluginHTTPResponse represents an HTTP response from a plugin
type PluginHTTPResponse struct {
	StatusCode int                 `json:"status_code"`
	Headers    map[string][]string `json:"headers"`
	Body       []byte              `json:"body"`
}

// UINavItem describes a navigation item that appears in the sidebar
type UINavItem struct {
	Label string `json:"label"`           // Display text
	Path  string `json:"path"`            // Route path
	Group string `json:"group,omitempty"` // Optional grouping
	Icon  string `json:"icon,omitempty"`  // Optional icon name
}

// UIRoute describes a frontend route provided by the plugin
type UIRoute struct {
	Path      string `json:"path"`      // Route path (e.g., "/plugins/sonarr/series")
	BundleURL string `json:"bundleUrl"` // URL to the JS bundle (e.g., "/plugins/sonarr/main.js")
}

// UIManifest describes the UI extensions provided by a plugin
type UIManifest struct {
	NavItems []UINavItem `json:"navItems"`
	Routes   []UIRoute   `json:"routes"`
}

// Event represents a system event that can be sent to plugins
type Event struct {
	Type      string                 `json:"type"` // e.g., "download.finished", "media.imported"
	Data      map[string]interface{} `json:"data"`
	Timestamp time.Time              `json:"timestamp"`
}

// MediaSuitePlugin is the main interface that all plugins must implement
// Plugins can choose which facets to implement by returning empty results
// or "not implemented" errors for facets they don't support.
type MediaSuitePlugin interface {
	// Metadata facet - REQUIRED
	// Returns basic information about the plugin
	Metadata(ctx context.Context) (*PluginMetadata, error)

	// API facet - OPTIONAL
	// Returns the list of HTTP routes the plugin wants to register
	APIRoutes(ctx context.Context) ([]RouteDescriptor, error)
	// Handles an HTTP request for a registered route
	HandleAPI(ctx context.Context, req *PluginHTTPRequest) (*PluginHTTPResponse, error)

	// UI facet - OPTIONAL
	// Returns the UI manifest (nav items and routes)
	UIManifest(ctx context.Context) (*UIManifest, error)

	// Events facet - OPTIONAL (stub for future implementation)
	// Handles a system event
	HandleEvent(ctx context.Context, evt Event) error
}

// MediaItem represents a media item in the core system
// This is used by the SDK to allow plugins to query/modify media
type MediaItem struct {
	ID        int64                  `json:"id"`
	Kind      string                 `json:"kind"` // "movie", "tv-series", "book", etc.
	Title     string                 `json:"title"`
	Year      *int32                 `json:"year,omitempty"`
	Metadata  map[string]interface{} `json:"metadata"`
	ParentID  *int64                 `json:"parent_id,omitempty"`
	CreatedAt time.Time              `json:"created_at"`
	UpdatedAt time.Time              `json:"updated_at"`
}

// ConfigValue represents a configuration key-value pair
type ConfigValue struct {
	Key   string      `json:"key"`
	Value interface{} `json:"value"`
}
