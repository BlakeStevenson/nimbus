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
			Method: "GET",
			Path:   "/api/plugins/tmdb/tv/{id}/season/{season}",
			Auth:   "session",
			Tag:    "",
		},
		{
			Method: "GET",
			Path:   "/api/plugins/tmdb/tv/{id}/season/{season}/episode/{episode}",
			Auth:   "session",
			Tag:    "",
		},
		{
			Method: "POST",
			Path:   "/api/plugins/tmdb/enrich/{mediaId}",
			Auth:   "session",
			Tag:    "",
		},
		{
			Method: "POST",
			Path:   "/api/plugins/tmdb/enrich",
			Auth:   "none", // Allow internal scanner calls without auth
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
		if err != nil || apiKey == "" {
			// Fall back to environment variable
			apiKey = os.Getenv("TMDB_API_KEY")
		}
	} else {
		// No SDK available, use environment variable
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
	case strings.Contains(req.Path, "/season/") && strings.Contains(req.Path, "/episode/"):
		return p.handleGetEpisode(ctx, req, apiKey)
	case strings.Contains(req.Path, "/season/") && !strings.Contains(req.Path, "/episode/"):
		return p.handleGetSeason(ctx, req, apiKey)
	case strings.HasPrefix(req.Path, "/api/plugins/tmdb/movie/"):
		return p.handleGetMovie(ctx, req, apiKey)
	case strings.HasPrefix(req.Path, "/api/plugins/tmdb/tv/"):
		return p.handleGetTV(ctx, req, apiKey)
	case req.Path == "/api/plugins/tmdb/enrich":
		return p.handleEnrichMediaBatch(ctx, req, apiKey)
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

// handleGetSeason gets detailed season information with episodes
func (p *TMDBPlugin) handleGetSeason(ctx context.Context, req *plugins.PluginHTTPRequest, apiKey string) (*plugins.PluginHTTPResponse, error) {
	// Parse path: /api/plugins/tmdb/tv/{id}/season/{season}
	parts := strings.Split(req.Path, "/")
	if len(parts) < 8 {
		return p.errorResponse(http.StatusBadRequest, "Invalid season path")
	}

	tvID := parts[5]
	season := parts[7]

	apiURL := fmt.Sprintf("%s/tv/%s/season/%s?api_key=%s&append_to_response=images",
		tmdbAPIBaseURL, tvID, season, apiKey)

	data, err := p.makeRequest(ctx, apiURL)
	if err != nil {
		return p.errorResponse(http.StatusInternalServerError, fmt.Sprintf("Failed to get season: %v", err))
	}

	return &plugins.PluginHTTPResponse{
		StatusCode: http.StatusOK,
		Headers:    map[string][]string{"Content-Type": {"application/json"}},
		Body:       data,
	}, nil
}

// handleGetEpisode gets detailed episode information
func (p *TMDBPlugin) handleGetEpisode(ctx context.Context, req *plugins.PluginHTTPRequest, apiKey string) (*plugins.PluginHTTPResponse, error) {
	// Parse path: /api/plugins/tmdb/tv/{id}/season/{season}/episode/{episode}
	parts := strings.Split(req.Path, "/")
	if len(parts) < 10 {
		return p.errorResponse(http.StatusBadRequest, "Invalid episode path")
	}

	tvID := parts[5]
	season := parts[7]
	episode := parts[9]

	apiURL := fmt.Sprintf("%s/tv/%s/season/%s/episode/%s?api_key=%s&append_to_response=images",
		tmdbAPIBaseURL, tvID, season, episode, apiKey)

	data, err := p.makeRequest(ctx, apiURL)
	if err != nil {
		return p.errorResponse(http.StatusInternalServerError, fmt.Sprintf("Failed to get episode: %v", err))
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

	// Fetch metadata from TMDB including external IDs
	var apiURL string
	if reqBody.Type == "movie" {
		apiURL = fmt.Sprintf("%s/movie/%s?api_key=%s&append_to_response=credits,images,external_ids", tmdbAPIBaseURL, reqBody.TMDBID, apiKey)
	} else if reqBody.Type == "tv" {
		apiURL = fmt.Sprintf("%s/tv/%s?api_key=%s&append_to_response=credits,images,external_ids", tmdbAPIBaseURL, reqBody.TMDBID, apiKey)
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

	// Extract external IDs
	externalIDs := make(map[string]interface{})
	if extIDs, ok := tmdbData["external_ids"].(map[string]interface{}); ok {
		if imdbID, ok := extIDs["imdb_id"].(string); ok && imdbID != "" {
			externalIDs["imdb_id"] = imdbID
		}
		if tvdbID, ok := extIDs["tvdb_id"].(float64); ok && tvdbID > 0 {
			externalIDs["tvdb_id"] = int(tvdbID)
		}
		if tvrageID, ok := extIDs["tvrage_id"].(float64); ok && tvrageID > 0 {
			externalIDs["tvrage_id"] = int(tvrageID)
		}
		if facebookID, ok := extIDs["facebook_id"].(string); ok && facebookID != "" {
			externalIDs["facebook_id"] = facebookID
		}
		if instagramID, ok := extIDs["instagram_id"].(string); ok && instagramID != "" {
			externalIDs["instagram_id"] = instagramID
		}
		if twitterID, ok := extIDs["twitter_id"].(string); ok && twitterID != "" {
			externalIDs["twitter_id"] = twitterID
		}
	}

	// Build response with instructions for updating the media item
	response := map[string]interface{}{
		"media_id":     mediaID,
		"metadata":     metadata,
		"external_ids": externalIDs,
		"message":      "Metadata fetched successfully. Update the media item's metadata and external_ids columns with this data.",
		"sql_example": fmt.Sprintf(
			"UPDATE media_items SET metadata = metadata || '%s'::jsonb, external_ids = '%s'::jsonb WHERE id = %s",
			mustMarshal(metadata),
			mustMarshal(externalIDs),
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

// handleEnrichMediaBatch enriches media items with TMDB metadata (for scanner)
func (p *TMDBPlugin) handleEnrichMediaBatch(ctx context.Context, req *plugins.PluginHTTPRequest, apiKey string) (*plugins.PluginHTTPResponse, error) {
	// Parse request body
	var reqBody struct {
		Title   string `json:"title"`
		Year    int    `json:"year,omitempty"`
		Kind    string `json:"kind"` // "movie" or "tv_episode"
		Season  int    `json:"season,omitempty"`
		Episode int    `json:"episode,omitempty"`
	}

	if err := json.Unmarshal(req.Body, &reqBody); err != nil {
		return p.errorResponse(http.StatusBadRequest, "Invalid request body")
	}

	if reqBody.Title == "" || reqBody.Kind == "" {
		return p.errorResponse(http.StatusBadRequest, "title and kind are required")
	}

	metadata := make(map[string]interface{})
	externalIDs := make(map[string]interface{})

	// Handle movies
	if reqBody.Kind == "movie" {
		// Search for the movie
		searchURL := fmt.Sprintf("%s/search/movie?api_key=%s&query=%s",
			tmdbAPIBaseURL, apiKey, url.QueryEscape(reqBody.Title))
		if reqBody.Year > 0 {
			searchURL += fmt.Sprintf("&year=%d", reqBody.Year)
		}

		searchData, err := p.makeRequest(ctx, searchURL)
		if err != nil {
			return p.errorResponse(http.StatusInternalServerError, fmt.Sprintf("Failed to search movie: %v", err))
		}

		var searchResult map[string]interface{}
		if err := json.Unmarshal(searchData, &searchResult); err != nil {
			return p.errorResponse(http.StatusInternalServerError, "Failed to parse search results")
		}

		results, ok := searchResult["results"].([]interface{})
		if !ok || len(results) == 0 {
			return p.errorResponse(http.StatusNotFound, "No results found")
		}

		// Get first result
		firstResult := results[0].(map[string]interface{})
		tmdbID := fmt.Sprintf("%.0f", firstResult["id"].(float64))

		// Fetch full movie details
		movieURL := fmt.Sprintf("%s/movie/%s?api_key=%s&append_to_response=credits,images,external_ids",
			tmdbAPIBaseURL, tmdbID, apiKey)
		movieData, err := p.makeRequest(ctx, movieURL)
		if err != nil {
			return p.errorResponse(http.StatusInternalServerError, "Failed to fetch movie details")
		}

		var movieDetails map[string]interface{}
		if err := json.Unmarshal(movieData, &movieDetails); err != nil {
			return p.errorResponse(http.StatusInternalServerError, "Failed to parse movie details")
		}

		metadata = extractMetadata(movieDetails, "movie", tmdbID)
	}

	// Handle TV series
	if reqBody.Kind == "tv_series" {
		// Search for the TV show
		searchURL := fmt.Sprintf("%s/search/tv?api_key=%s&query=%s",
			tmdbAPIBaseURL, apiKey, url.QueryEscape(reqBody.Title))

		searchData, err := p.makeRequest(ctx, searchURL)
		if err != nil {
			return p.errorResponse(http.StatusInternalServerError, fmt.Sprintf("Failed to search TV show: %v", err))
		}

		var searchResult map[string]interface{}
		if err := json.Unmarshal(searchData, &searchResult); err != nil {
			return p.errorResponse(http.StatusInternalServerError, "Failed to parse search results")
		}

		results, ok := searchResult["results"].([]interface{})
		if !ok || len(results) == 0 {
			return p.errorResponse(http.StatusNotFound, "No results found")
		}

		// Get first result
		firstResult := results[0].(map[string]interface{})
		tmdbID := fmt.Sprintf("%.0f", firstResult["id"].(float64))

		// Fetch full TV series details
		seriesURL := fmt.Sprintf("%s/tv/%s?api_key=%s&append_to_response=credits,images,external_ids",
			tmdbAPIBaseURL, tmdbID, apiKey)
		seriesData, err := p.makeRequest(ctx, seriesURL)
		if err != nil {
			return p.errorResponse(http.StatusInternalServerError, "Failed to fetch series details")
		}

		var seriesDetails map[string]interface{}
		if err := json.Unmarshal(seriesData, &seriesDetails); err != nil {
			return p.errorResponse(http.StatusInternalServerError, "Failed to parse series details")
		}

		metadata = extractMetadata(seriesDetails, "tv_series", tmdbID)
	}

	// Handle TV seasons
	if reqBody.Kind == "tv_season" {
		// For seasons, we need the series title to search
		searchURL := fmt.Sprintf("%s/search/tv?api_key=%s&query=%s",
			tmdbAPIBaseURL, apiKey, url.QueryEscape(reqBody.Title))

		searchData, err := p.makeRequest(ctx, searchURL)
		if err != nil {
			return p.errorResponse(http.StatusInternalServerError, fmt.Sprintf("Failed to search TV show: %v", err))
		}

		var searchResult map[string]interface{}
		if err := json.Unmarshal(searchData, &searchResult); err != nil {
			return p.errorResponse(http.StatusInternalServerError, "Failed to parse search results")
		}

		results, ok := searchResult["results"].([]interface{})
		if !ok || len(results) == 0 {
			return p.errorResponse(http.StatusNotFound, "No results found")
		}

		// Get first result
		firstResult := results[0].(map[string]interface{})
		tmdbID := fmt.Sprintf("%.0f", firstResult["id"].(float64))

		// Fetch season details
		seasonURL := fmt.Sprintf("%s/tv/%s/season/%d?api_key=%s&append_to_response=images",
			tmdbAPIBaseURL, tmdbID, reqBody.Season, apiKey)
		seasonData, err := p.makeRequest(ctx, seasonURL)
		if err != nil {
			return p.errorResponse(http.StatusInternalServerError, "Failed to fetch season details")
		}

		var seasonDetails map[string]interface{}
		if err := json.Unmarshal(seasonData, &seasonDetails); err != nil {
			return p.errorResponse(http.StatusInternalServerError, "Failed to parse season details")
		}

		metadata = extractMetadata(seasonDetails, "tv_season", tmdbID)

		// Fetch series-level external IDs (seasons don't have their own external IDs)
		seriesURL := fmt.Sprintf("%s/tv/%s?api_key=%s&append_to_response=external_ids",
			tmdbAPIBaseURL, tmdbID, apiKey)
		seriesData, err := p.makeRequest(ctx, seriesURL)
		if err == nil {
			var seriesDetails map[string]interface{}
			if err := json.Unmarshal(seriesData, &seriesDetails); err == nil {
				if extIDs, ok := seriesDetails["external_ids"].(map[string]interface{}); ok {
					if imdbID, ok := extIDs["imdb_id"].(string); ok && imdbID != "" {
						externalIDs["imdb_id"] = imdbID
					}
					if tvdbID, ok := extIDs["tvdb_id"].(float64); ok && tvdbID > 0 {
						externalIDs["tvdb_id"] = int(tvdbID)
					}
					if tvrageID, ok := extIDs["tvrage_id"].(float64); ok && tvrageID > 0 {
						externalIDs["tvrage_id"] = int(tvrageID)
					}
					if facebookID, ok := extIDs["facebook_id"].(string); ok && facebookID != "" {
						externalIDs["facebook_id"] = facebookID
					}
					if instagramID, ok := extIDs["instagram_id"].(string); ok && instagramID != "" {
						externalIDs["instagram_id"] = instagramID
					}
					if twitterID, ok := extIDs["twitter_id"].(string); ok && twitterID != "" {
						externalIDs["twitter_id"] = twitterID
					}
				}
			}
		}
	}

	// Handle TV episodes
	if reqBody.Kind == "tv_episode" {
		// Search for the TV show
		searchURL := fmt.Sprintf("%s/search/tv?api_key=%s&query=%s",
			tmdbAPIBaseURL, apiKey, url.QueryEscape(reqBody.Title))

		searchData, err := p.makeRequest(ctx, searchURL)
		if err != nil {
			return p.errorResponse(http.StatusInternalServerError, fmt.Sprintf("Failed to search TV show: %v", err))
		}

		var searchResult map[string]interface{}
		if err := json.Unmarshal(searchData, &searchResult); err != nil {
			return p.errorResponse(http.StatusInternalServerError, "Failed to parse search results")
		}

		results, ok := searchResult["results"].([]interface{})
		if !ok || len(results) == 0 {
			return p.errorResponse(http.StatusNotFound, "No results found")
		}

		// Get first result
		firstResult := results[0].(map[string]interface{})
		tmdbID := fmt.Sprintf("%.0f", firstResult["id"].(float64))

		// Fetch episode details
		episodeURL := fmt.Sprintf("%s/tv/%s/season/%d/episode/%d?api_key=%s&append_to_response=images",
			tmdbAPIBaseURL, tmdbID, reqBody.Season, reqBody.Episode, apiKey)
		episodeData, err := p.makeRequest(ctx, episodeURL)
		if err != nil {
			return p.errorResponse(http.StatusInternalServerError, "Failed to fetch episode details")
		}

		var episodeDetails map[string]interface{}
		if err := json.Unmarshal(episodeData, &episodeDetails); err != nil {
			return p.errorResponse(http.StatusInternalServerError, "Failed to parse episode details")
		}

		metadata = extractMetadata(episodeDetails, "tv_episode", tmdbID)

		// Fetch series-level external IDs (episodes don't have their own external IDs)
		seriesURL := fmt.Sprintf("%s/tv/%s?api_key=%s&append_to_response=external_ids",
			tmdbAPIBaseURL, tmdbID, apiKey)
		seriesData, err := p.makeRequest(ctx, seriesURL)
		if err == nil {
			var seriesDetails map[string]interface{}
			if err := json.Unmarshal(seriesData, &seriesDetails); err == nil {
				if extIDs, ok := seriesDetails["external_ids"].(map[string]interface{}); ok {
					if imdbID, ok := extIDs["imdb_id"].(string); ok && imdbID != "" {
						externalIDs["imdb_id"] = imdbID
					}
					if tvdbID, ok := extIDs["tvdb_id"].(float64); ok && tvdbID > 0 {
						externalIDs["tvdb_id"] = int(tvdbID)
					}
					if tvrageID, ok := extIDs["tvrage_id"].(float64); ok && tvrageID > 0 {
						externalIDs["tvrage_id"] = int(tvrageID)
					}
					if facebookID, ok := extIDs["facebook_id"].(string); ok && facebookID != "" {
						externalIDs["facebook_id"] = facebookID
					}
					if instagramID, ok := extIDs["instagram_id"].(string); ok && instagramID != "" {
						externalIDs["instagram_id"] = instagramID
					}
					if twitterID, ok := extIDs["twitter_id"].(string); ok && twitterID != "" {
						externalIDs["twitter_id"] = twitterID
					}
				}
			}
		}
	}

	responseData := map[string]interface{}{
		"metadata": metadata,
		"success":  true,
	}

	// Add external_ids if we have any
	if len(externalIDs) > 0 {
		responseData["external_ids"] = externalIDs
	}

	body, _ := json.Marshal(responseData)

	return &plugins.PluginHTTPResponse{
		StatusCode: http.StatusOK,
		Headers:    map[string][]string{"Content-Type": {"application/json"}},
		Body:       body,
	}, nil
}

// extractMetadata extracts relevant metadata from TMDB response
func extractMetadata(tmdbData map[string]interface{}, mediaType string, tmdbID string) map[string]interface{} {
	metadata := map[string]interface{}{
		"tmdb_id": tmdbID,
		"type":    mediaType,
	}

	if overview, ok := tmdbData["overview"].(string); ok && overview != "" {
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

	if stillPath, ok := tmdbData["still_path"].(string); ok && stillPath != "" {
		metadata["still_url"] = tmdbImageBaseURL + stillPath
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

	if airDate, ok := tmdbData["air_date"].(string); ok {
		metadata["air_date"] = airDate
	}

	if genres, ok := tmdbData["genres"].([]interface{}); ok {
		metadata["genres"] = genres
	}

	if runtime, ok := tmdbData["runtime"].(float64); ok {
		metadata["runtime"] = int(runtime)
	}

	if name, ok := tmdbData["name"].(string); ok {
		metadata["episode_name"] = name
	}

	// Extract season number (for tv_season media type)
	if seasonNumber, ok := tmdbData["season_number"].(float64); ok {
		metadata["season_number"] = int(seasonNumber)
	}

	// Extract episode number (for tv_episode media type)
	if episodeNumber, ok := tmdbData["episode_number"].(float64); ok {
		metadata["episode_number"] = int(episodeNumber)
		metadata["episode"] = int(episodeNumber) // Also store as "episode" for compatibility
	}

	// Extract season number for episodes (episodes have both season_number and episode_number)
	if mediaType == "tv_episode" {
		if seasonNumber, ok := tmdbData["season_number"].(float64); ok {
			metadata["season"] = int(seasonNumber)
			metadata["season_number"] = int(seasonNumber)
		}
	}

	// Extract external IDs into metadata (for backward compatibility)
	// These will also be saved to the external_ids column separately
	if extIDs, ok := tmdbData["external_ids"].(map[string]interface{}); ok {
		if imdbID, ok := extIDs["imdb_id"].(string); ok && imdbID != "" {
			metadata["imdb_id"] = imdbID
		}
		if tvdbID, ok := extIDs["tvdb_id"].(float64); ok && tvdbID > 0 {
			metadata["tvdb_id"] = fmt.Sprintf("%.0f", tvdbID)
		}
		if tvrageID, ok := extIDs["tvrage_id"].(float64); ok && tvrageID > 0 {
			metadata["tvrage_id"] = fmt.Sprintf("%.0f", tvrageID)
		}
	}

	return metadata
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

// IsIndexer returns false as TMDB is not an indexer plugin
func (p *TMDBPlugin) IsIndexer(ctx context.Context) (bool, error) {
	return false, nil
}

// Search is not implemented for TMDB plugin
func (p *TMDBPlugin) Search(ctx context.Context, req *plugins.IndexerSearchRequest) (*plugins.IndexerSearchResponse, error) {
	return nil, fmt.Errorf("TMDB plugin does not support search")
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
