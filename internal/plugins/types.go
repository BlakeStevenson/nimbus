package plugins

import (
	"context"
	"fmt"
	"time"

	"github.com/blakestevenson/nimbus/internal/plugins/proto"
	"github.com/hashicorp/go-plugin"
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

	// SDK access (for plugin-side SDK calls)
	SDKServerID uint32       `json:"sdk_server_id,omitempty"`
	SDK         SDKInterface `json:"-"` // SDK client for plugins to use
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

	// Indexer facet - OPTIONAL
	// Returns whether the plugin provides indexer capabilities
	IsIndexer(ctx context.Context) (bool, error)
	// Searches for content using the indexer
	Search(ctx context.Context, req *IndexerSearchRequest) (*IndexerSearchResponse, error)
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

// IndexerSearchRequest represents a search request to an indexer plugin
type IndexerSearchRequest struct {
	// Search query string
	Query string `json:"query"`

	// Type of content: "general", "tv", "movie"
	Type string `json:"type"`

	// Categories to search in (indexer-specific)
	Categories []string `json:"categories,omitempty"`

	// TV-specific parameters
	TVDBID   string `json:"tvdbid,omitempty"`
	TVRageID string `json:"tvrageid,omitempty"`
	Season   int    `json:"season,omitempty"`
	Episode  int    `json:"episode,omitempty"`

	// Movie-specific parameters
	IMDBID string `json:"imdbid,omitempty"`
	TMDBID string `json:"tmdbid,omitempty"`

	// Pagination
	Limit  int `json:"limit,omitempty"`
	Offset int `json:"offset,omitempty"`
}

// IndexerSearchResponse represents the response from an indexer search
type IndexerSearchResponse struct {
	// List of releases found
	Releases []IndexerRelease `json:"releases"`

	// Total count of results (may be > len(Releases) if paginated)
	Total int `json:"total"`

	// Indexer that provided these results
	IndexerID   string `json:"indexer_id"`
	IndexerName string `json:"indexer_name"`
}

// IndexerRelease represents a single release from an indexer
type IndexerRelease struct {
	// Unique identifier for this release
	GUID string `json:"guid"`

	// Release title
	Title string `json:"title"`

	// Link to details page
	Link string `json:"link,omitempty"`

	// Link to comments
	Comments string `json:"comments,omitempty"`

	// Publication date
	PublishDate time.Time `json:"publish_date"`

	// Category ID
	Category string `json:"category,omitempty"`

	// Size in bytes
	Size int64 `json:"size"`

	// Download URL (NZB file URL)
	DownloadURL string `json:"download_url"`

	// Description (may contain HTML)
	Description string `json:"description,omitempty"`

	// Additional attributes (season, episode, tvdbid, imdbid, etc.)
	Attributes map[string]string `json:"attributes,omitempty"`

	// Indexer that provided this release
	IndexerID   string `json:"indexer_id"`
	IndexerName string `json:"indexer_name"`
}

// GetSDKClient creates an SDK client from a PluginHTTPRequest
// This should be called from within a plugin's HandleAPI method
func GetSDKClient(req *PluginHTTPRequest, broker interface{}) (*GRPCSDKClient, error) {
	if req.SDKServerID == 0 {
		return nil, fmt.Errorf("no SDK server available")
	}

	grpcBroker, ok := broker.(*plugin.GRPCBroker)
	if !ok {
		return nil, fmt.Errorf("invalid broker type")
	}

	conn, err := grpcBroker.Dial(req.SDKServerID)
	if err != nil {
		return nil, fmt.Errorf("failed to dial SDK server: %w", err)
	}

	return &GRPCSDKClient{
		client: proto.NewSDKServiceClient(conn),
	}, nil
}

// SDKInterface defines the methods plugins can call on the SDK
type SDKInterface interface {
	ConfigGet(ctx context.Context, key string) (interface{}, error)
	ConfigGetString(ctx context.Context, key string) (string, error)
	ConfigSet(ctx context.Context, key string, value interface{}) error
	ConfigDelete(ctx context.Context, key string) error
}
