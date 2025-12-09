package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/blakestevenson/nimbus/internal/plugins"
	"github.com/hashicorp/go-plugin"
)

const (
	tmdbAPIBaseURL   = "https://api.themoviedb.org/3"
	tmdbImageBaseURL = "https://image.tmdb.org/t/p/original"
	configKey        = "plugins.tmdb.api_key"
)

// TMDBPlugin implements the MediaSuitePlugin interface
type TMDBPlugin struct{}

// NewTMDBPlugin creates a new TMDB plugin instance
func NewTMDBPlugin() *TMDBPlugin {
	return &TMDBPlugin{}
}

// Metadata returns plugin metadata
func (p *TMDBPlugin) Metadata(ctx context.Context) (*plugins.PluginMetadata, error) {
	return &plugins.PluginMetadata{
		ID:           "tmdb-plugin",
		Name:         "The Movie Database (TMDB)",
		Version:      "0.1.0",
		Description:  "Fetches movie and TV show metadata from TMDB including descriptions, ratings, and cover images",
		Capabilities: []string{"api"},
	}, nil
}

// APIRoutes returns the HTTP routes this plugin provides
func (p *TMDBPlugin) APIRoutes(ctx context.Context) ([]plugins.RouteDescriptor, error) {
	return []plugins.RouteDescriptor{
		{
			Method: "GET",
			Path:   "/api/plugins/tmdb/search/movie",
			Auth:   "session",
			Tag:    "",
		},
		{
			Method: "GET",
			Path:   "/api/plugins/tmdb/search/tv",
			Auth:   "session",
			Tag:    "",
		},
		{
			Method: "GET",
			Path:   "/api/plugins/tmdb/movie/{id}",
			Auth:   "session",
			Tag:    "",
		},
		{
			Method: "GET",
			Path:   "/api/plugins/tmdb/tv/{id}",
			Auth:   "session",
			Tag:    "",
		},
		{
			Method: "POST",
			Path:   "/api/plugins/tmdb/enrich/{mediaId}",
			Auth:   "session",
			Tag:    "",
		},
	}, nil
}

// HandleAPI handles HTTP requests for this plugin's routes
func (p *TMDBPlugin) HandleAPI(ctx context.Context, req *plugins.PluginHTTPRequest) (*plugins.PluginHTTPResponse, error) {
	// Get API key from config via SDK or environment variable
	var apiKey string
	var err error

	if req.SDK != nil {
		// Try to get from config table via SDK
		apiKey, err = req.SDK.ConfigGetString(ctx, configKey)
		if err != nil {
			// Log error and fall back to environment variable
			fmt.Fprintf(os.Stderr, "TMDB plugin: Failed to get config from SDK: %v\n", err)
			apiKey = os.Getenv("TMDB_API_KEY")
		} else if apiKey == "" {
			fmt.Fprintf(os.Stderr, "TMDB plugin: Config key '%s' returned empty string\n", configKey)
			apiKey = os.Getenv("TMDB_API_KEY")
		} else {
			fmt.Fprintf(os.Stderr, "TMDB plugin: Successfully got API key from SDK config (length: %d)\n", len(apiKey))
		}
	} else {
		// No SDK available, use environment variable
		fmt.Fprintf(os.Stderr, "TMDB plugin: No SDK available, using environment variable\n")
		apiKey = os.Getenv("TMDB_API_KEY")
	}

	if apiKey == "" {
		return p.errorResponse(http.StatusInternalServerError, "TMDB API key not configured. Please set 'plugins.tmdb.api_key' in the config table or TMDB_API_KEY environment variable.")
	}

	switch {
	case req.Path == "/api/plugins/tmdb/search/movie":
		return p.handleSearchMovie(ctx, req, apiKey)
	case req.Path == "/api/plugins/tmdb/search/tv":
		return p.handleSearchTV(ctx, req, apiKey)
	case strings.HasPrefix(req.Path, "/api/plugins/tmdb/movie/"):
		return p.handleGetMovie(ctx, req, apiKey)
	case strings.HasPrefix(req.Path, "/api/plugins/tmdb/tv/"):
		return p.handleGetTV(ctx, req, apiKey)
	case strings.HasPrefix(req.Path, "/api/plugins/tmdb/enrich/"):
		return p.handleEnrichMedia(ctx, req, apiKey)
	default:
		return p.errorResponse(http.StatusNotFound, "Not found")
	}
}

// handleSearchMovie searches for movies on TMDB
func (p *TMDBPlugin) handleSearchMovie(ctx context.Context, req *plugins.PluginHTTPRequest, apiKey string) (*plugins.PluginHTTPResponse, error) {
	query := p.getQueryParam(req, "query")
	if query == "" {
		return p.errorResponse(http.StatusBadRequest, "query parameter is required")
	}

	year := p.getQueryParam(req, "year")

	apiURL := fmt.Sprintf("%s/search/movie?api_key=%s&query=%s", tmdbAPIBaseURL, apiKey, url.QueryEscape(query))
	if year != "" {
		apiURL += "&year=" + year
	}

	data, err := p.makeRequest(ctx, apiURL)
	if err != nil {
		return p.errorResponse(http.StatusInternalServerError, fmt.Sprintf("Failed to search TMDB: %v", err))
	}

	return &plugins.PluginHTTPResponse{
		StatusCode: http.StatusOK,
		Headers:    map[string][]string{"Content-Type": {"application/json"}},
		Body:       data,
	}, nil
}

// handleSearchTV searches for TV shows on TMDB
func (p *TMDBPlugin) handleSearchTV(ctx context.Context, req *plugins.PluginHTTPRequest, apiKey string) (*plugins.PluginHTTPResponse, error) {
	query := p.getQueryParam(req, "query")
	if query == "" {
		return p.errorResponse(http.StatusBadRequest, "query parameter is required")
	}

	year := p.getQueryParam(req, "year")

	apiURL := fmt.Sprintf("%s/search/tv?api_key=%s&query=%s", tmdbAPIBaseURL, apiKey, url.QueryEscape(query))
	if year != "" {
		apiURL += "&first_air_date_year=" + year
	}

	data, err := p.makeRequest(ctx, apiURL)
	if err != nil {
		return p.errorResponse(http.StatusInternalServerError, fmt.Sprintf("Failed to search TMDB: %v", err))
	}

	return &plugins.PluginHTTPResponse{
		StatusCode: http.StatusOK,
		Headers:    map[string][]string{"Content-Type": {"application/json"}},
		Body:       data,
	}, nil
}

// handleGetMovie gets detailed movie information
func (p *TMDBPlugin) handleGetMovie(ctx context.Context, req *plugins.PluginHTTPRequest, apiKey string) (*plugins.PluginHTTPResponse, error) {
	parts := strings.Split(req.Path, "/")
	if len(parts) < 5 {
		return p.errorResponse(http.StatusBadRequest, "Invalid movie ID")
	}
	movieID := parts[len(parts)-1]

	apiURL := fmt.Sprintf("%s/movie/%s?api_key=%s&append_to_response=credits,images", tmdbAPIBaseURL, movieID, apiKey)

	data, err := p.makeRequest(ctx, apiURL)
	if err != nil {
		return p.errorResponse(http.StatusInternalServerError, fmt.Sprintf("Failed to get movie: %v", err))
	}

	return &plugins.PluginHTTPResponse{
		StatusCode: http.StatusOK,
		Headers:    map[string][]string{"Content-Type": {"application/json"}},
		Body:       data,
	}, nil
}

// handleGetTV gets detailed TV show information
func (p *TMDBPlugin) handleGetTV(ctx context.Context, req *plugins.PluginHTTPRequest, apiKey string) (*plugins.PluginHTTPResponse, error) {
	parts := strings.Split(req.Path, "/")
	if len(parts) < 5 {
		return p.errorResponse(http.StatusBadRequest, "Invalid TV show ID")
	}
	tvID := parts[len(parts)-1]

	apiURL := fmt.Sprintf("%s/tv/%s?api_key=%s&append_to_response=credits,images", tmdbAPIBaseURL, tvID, apiKey)

	data, err := p.makeRequest(ctx, apiURL)
	if err != nil {
		return p.errorResponse(http.StatusInternalServerError, fmt.Sprintf("Failed to get TV show: %v", err))
	}

	return &plugins.PluginHTTPResponse{
		StatusCode: http.StatusOK,
		Headers:    map[string][]string{"Content-Type": {"application/json"}},
		Body:       data,
	}, nil
}

// handleEnrichMedia enriches a media item with TMDB metadata
func (p *TMDBPlugin) handleEnrichMedia(ctx context.Context, req *plugins.PluginHTTPRequest, apiKey string) (*plugins.PluginHTTPResponse, error) {
	parts := strings.Split(req.Path, "/")
	if len(parts) < 5 {
		return p.errorResponse(http.StatusBadRequest, "Invalid media ID")
	}
	mediaID := parts[len(parts)-1]

	// Parse request body to get TMDB ID and type
	var reqBody struct {
		TMDBID string `json:"tmdb_id"`
		Type   string `json:"type"` // "movie" or "tv"
	}

	if err := json.Unmarshal(req.Body, &reqBody); err != nil {
		return p.errorResponse(http.StatusBadRequest, "Invalid request body")
	}

	if reqBody.TMDBID == "" || reqBody.Type == "" {
		return p.errorResponse(http.StatusBadRequest, "tmdb_id and type are required")
	}

	// Fetch metadata from TMDB
	var apiURL string
	if reqBody.Type == "movie" {
		apiURL = fmt.Sprintf("%s/movie/%s?api_key=%s&append_to_response=credits,images", tmdbAPIBaseURL, reqBody.TMDBID, apiKey)
	} else if reqBody.Type == "tv" {
		apiURL = fmt.Sprintf("%s/tv/%s?api_key=%s&append_to_response=credits,images", tmdbAPIBaseURL, reqBody.TMDBID, apiKey)
	} else {
		return p.errorResponse(http.StatusBadRequest, "type must be 'movie' or 'tv'")
	}

	data, err := p.makeRequest(ctx, apiURL)
	if err != nil {
		return p.errorResponse(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch TMDB data: %v", err))
	}

	// Parse TMDB response
	var tmdbData map[string]interface{}
	if err := json.Unmarshal(data, &tmdbData); err != nil {
		return p.errorResponse(http.StatusInternalServerError, "Failed to parse TMDB response")
	}

	// Extract relevant metadata
	metadata := map[string]interface{}{
		"tmdb_id": reqBody.TMDBID,
		"type":    reqBody.Type,
	}

	if overview, ok := tmdbData["overview"].(string); ok {
		metadata["description"] = overview
	}

	if voteAverage, ok := tmdbData["vote_average"].(float64); ok {
		metadata["rating"] = voteAverage
	}

	if voteCount, ok := tmdbData["vote_count"].(float64); ok {
		metadata["vote_count"] = voteCount
	}

	if posterPath, ok := tmdbData["poster_path"].(string); ok && posterPath != "" {
		metadata["poster_url"] = tmdbImageBaseURL + posterPath
	}

	if backdropPath, ok := tmdbData["backdrop_path"].(string); ok && backdropPath != "" {
		metadata["backdrop_url"] = tmdbImageBaseURL + backdropPath
	}

	if releaseDate, ok := tmdbData["release_date"].(string); ok {
		metadata["release_date"] = releaseDate
	}

	if firstAirDate, ok := tmdbData["first_air_date"].(string); ok {
		metadata["first_air_date"] = firstAirDate
	}

	if genres, ok := tmdbData["genres"].([]interface{}); ok {
		metadata["genres"] = genres
	}

	if runtime, ok := tmdbData["runtime"].(float64); ok {
		metadata["runtime"] = int(runtime)
	}

	// Build response with instructions for updating the media item
	response := map[string]interface{}{
		"media_id": mediaID,
		"metadata": metadata,
		"message":  "Metadata fetched successfully. Update the media item's metadata column with this data.",
		"sql_example": fmt.Sprintf(
			"UPDATE media_items SET metadata = metadata || '%s'::jsonb WHERE id = %s",
			mustMarshal(metadata),
			mediaID,
		),
	}

	body, _ := json.Marshal(response)

	return &plugins.PluginHTTPResponse{
		StatusCode: http.StatusOK,
		Headers:    map[string][]string{"Content-Type": {"application/json"}},
		Body:       body,
	}, nil
}

// UIManifest returns the UI configuration for this plugin
func (p *TMDBPlugin) UIManifest(ctx context.Context) (*plugins.UIManifest, error) {
	return &plugins.UIManifest{
		NavItems: []plugins.UINavItem{},
		Routes:   []plugins.UIRoute{},
	}, nil
}

// HandleEvent handles system events
func (p *TMDBPlugin) HandleEvent(ctx context.Context, evt plugins.Event) error {
	return nil
}

// Helper functions

// getAPIKey fetches the TMDB API key from the Nimbus config table

func (p *TMDBPlugin) makeRequest(ctx context.Context, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("TMDB API returned status %d", resp.StatusCode)
	}

	// Read the response body
	buf := make([]byte, 0, 4096)
	tmp := make([]byte, 1024)
	for {
		n, err := resp.Body.Read(tmp)
		if n > 0 {
			buf = append(buf, tmp[:n]...)
		}
		if err != nil {
			break
		}
	}

	return buf, nil
}

func (p *TMDBPlugin) getQueryParam(req *plugins.PluginHTTPRequest, key string) string {
	if values, ok := req.Query[key]; ok && len(values) > 0 {
		return values[0]
	}
	return ""
}

func (p *TMDBPlugin) errorResponse(statusCode int, message string) (*plugins.PluginHTTPResponse, error) {
	body, _ := json.Marshal(map[string]string{"error": message})
	return &plugins.PluginHTTPResponse{
		StatusCode: statusCode,
		Headers:    map[string][]string{"Content-Type": {"application/json"}},
		Body:       body,
	}, nil
}

func mustMarshal(v interface{}) string {
	data, _ := json.Marshal(v)
	return string(data)
}

func main() {
	tmdbPlugin := NewTMDBPlugin()

	plugin.Serve(&plugin.ServeConfig{
		HandshakeConfig: plugins.Handshake,
		Plugins: map[string]plugin.Plugin{
			"media-suite": &plugins.MediaSuitePluginGRPC{
				Impl: tmdbPlugin,
			},
		},
		GRPCServer: plugin.DefaultGRPCServer,
	})
}
