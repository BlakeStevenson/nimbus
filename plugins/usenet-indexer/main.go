package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/blakestevenson/nimbus/internal/plugins"
	"github.com/hashicorp/go-plugin"
)

// UsenetIndexerPlugin implements the MediaSuitePlugin interface
type UsenetIndexerPlugin struct{}

// Configuration keys
const (
	configPrefix   = "plugins.usenet-indexer"
	configIndexers = configPrefix + ".indexers"
)

// IndexerConfig represents a single indexer configuration
type IndexerConfig struct {
	ID              string   `json:"id"`
	Name            string   `json:"name"`
	URL             string   `json:"url"`
	APIKey          string   `json:"api_key"`
	Enabled         bool     `json:"enabled"`
	Priority        int      `json:"priority"`
	TVCategories    []string `json:"tv_categories"`
	MovieCategories []string `json:"movie_categories"`
}

// Metadata returns plugin metadata
func (p *UsenetIndexerPlugin) Metadata(ctx context.Context) (*plugins.PluginMetadata, error) {
	return &plugins.PluginMetadata{
		ID:           "usenet-indexer",
		Name:         "Usenet Indexer",
		Version:      "0.1.0",
		Description:  "Search and index Usenet content using Newznab-compatible indexers",
		Capabilities: []string{"api", "ui"},
	}, nil
}

// APIRoutes returns the HTTP routes this plugin provides
func (p *UsenetIndexerPlugin) APIRoutes(ctx context.Context) ([]plugins.RouteDescriptor, error) {
	return []plugins.RouteDescriptor{
		// Indexer management
		{
			Method: "GET",
			Path:   "/api/plugins/usenet-indexer/indexers",
			Auth:   "session",
			Tag:    "",
		},
		{
			Method: "POST",
			Path:   "/api/plugins/usenet-indexer/indexers",
			Auth:   "session",
			Tag:    "",
		},
		{
			Method: "PUT",
			Path:   "/api/plugins/usenet-indexer/indexers/{id}",
			Auth:   "session",
			Tag:    "",
		},
		{
			Method: "DELETE",
			Path:   "/api/plugins/usenet-indexer/indexers/{id}",
			Auth:   "session",
			Tag:    "",
		},
		{
			Method: "POST",
			Path:   "/api/plugins/usenet-indexer/indexers/{id}/test",
			Auth:   "session",
			Tag:    "",
		},
		// Search endpoints
		{
			Method: "GET",
			Path:   "/api/plugins/usenet-indexer/search",
			Auth:   "session",
			Tag:    "",
		},
		{
			Method: "GET",
			Path:   "/api/plugins/usenet-indexer/search/tv",
			Auth:   "session",
			Tag:    "",
		},
		{
			Method: "GET",
			Path:   "/api/plugins/usenet-indexer/search/movie",
			Auth:   "session",
			Tag:    "",
		},
		{
			Method: "GET",
			Path:   "/api/plugins/usenet-indexer/rss",
			Auth:   "session",
			Tag:    "",
		},
	}, nil
}

// HandleAPI handles HTTP requests for this plugin's routes
func (p *UsenetIndexerPlugin) HandleAPI(ctx context.Context, req *plugins.PluginHTTPRequest) (*plugins.PluginHTTPResponse, error) {
	// Handle indexer management endpoints
	if strings.HasPrefix(req.Path, "/api/plugins/usenet-indexer/indexers") {
		if req.Path == "/api/plugins/usenet-indexer/indexers" {
			if req.Method == "GET" {
				return p.handleListIndexers(ctx, req)
			}
			return p.handleCreateIndexer(ctx, req)
		}

		// Extract indexer ID from path
		parts := strings.Split(req.Path, "/")
		if len(parts) >= 6 {
			indexerID := parts[5]

			if len(parts) == 7 && parts[6] == "test" {
				return p.handleTestIndexer(ctx, req, indexerID)
			}

			if req.Method == "PUT" {
				return p.handleUpdateIndexer(ctx, req, indexerID)
			}
			if req.Method == "DELETE" {
				return p.handleDeleteIndexer(ctx, req, indexerID)
			}
		}
	}

	// Handle search endpoints
	switch req.Path {
	case "/api/plugins/usenet-indexer/search":
		return p.handleSearch(ctx, req)
	case "/api/plugins/usenet-indexer/search/tv":
		return p.handleSearchTV(ctx, req)
	case "/api/plugins/usenet-indexer/search/movie":
		return p.handleSearchMovie(ctx, req)
	case "/api/plugins/usenet-indexer/rss":
		return p.handleRSS(ctx, req)
	default:
		return &plugins.PluginHTTPResponse{
			StatusCode: http.StatusNotFound,
			Headers:    map[string][]string{"Content-Type": {"application/json"}},
			Body:       []byte(`{"error":"Not found"}`),
		}, nil
	}
}

// Indexer Management Handlers

func (p *UsenetIndexerPlugin) handleListIndexers(ctx context.Context, req *plugins.PluginHTTPRequest) (*plugins.PluginHTTPResponse, error) {
	if req.SDK == nil {
		return jsonResponse(http.StatusInternalServerError, map[string]string{"error": "SDK not available"})
	}

	indexers, err := p.getIndexers(ctx, req.SDK)
	if err != nil {
		return jsonResponse(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	// Ensure we never return nil for indexers array
	if indexers == nil {
		indexers = []IndexerConfig{}
	}

	// Mask API keys
	for i := range indexers {
		indexers[i].APIKey = maskAPIKey(indexers[i].APIKey)
	}

	return jsonResponse(http.StatusOK, map[string]interface{}{"indexers": indexers})
}

func (p *UsenetIndexerPlugin) handleCreateIndexer(ctx context.Context, req *plugins.PluginHTTPRequest) (*plugins.PluginHTTPResponse, error) {
	if req.SDK == nil {
		return jsonResponse(http.StatusInternalServerError, map[string]string{"error": "SDK not available"})
	}

	var indexer IndexerConfig
	if err := json.Unmarshal(req.Body, &indexer); err != nil {
		return jsonResponse(http.StatusBadRequest, map[string]string{"error": "Invalid JSON"})
	}

	// Validate required fields
	if indexer.Name == "" {
		return jsonResponse(http.StatusBadRequest, map[string]string{"error": "Name is required"})
	}
	if indexer.URL == "" {
		return jsonResponse(http.StatusBadRequest, map[string]string{"error": "URL is required"})
	}
	if indexer.APIKey == "" {
		return jsonResponse(http.StatusBadRequest, map[string]string{"error": "API key is required"})
	}

	// Generate ID if not provided
	if indexer.ID == "" {
		indexer.ID = generateID(indexer.Name)
	}

	indexers, err := p.getIndexers(ctx, req.SDK)
	if err != nil {
		return jsonResponse(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	// Ensure indexers is initialized
	if indexers == nil {
		indexers = []IndexerConfig{}
	}

	// Check for duplicate ID
	for _, existing := range indexers {
		if existing.ID == indexer.ID {
			return jsonResponse(http.StatusConflict, map[string]string{"error": "Indexer ID already exists"})
		}
	}

	indexers = append(indexers, indexer)

	if err := p.saveIndexers(ctx, req.SDK, indexers); err != nil {
		return jsonResponse(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	// Mask API key in response
	indexer.APIKey = maskAPIKey(indexer.APIKey)
	return jsonResponse(http.StatusCreated, indexer)
}

func (p *UsenetIndexerPlugin) handleUpdateIndexer(ctx context.Context, req *plugins.PluginHTTPRequest, indexerID string) (*plugins.PluginHTTPResponse, error) {
	if req.SDK == nil {
		return jsonResponse(http.StatusInternalServerError, map[string]string{"error": "SDK not available"})
	}

	var updatedIndexer IndexerConfig
	if err := json.Unmarshal(req.Body, &updatedIndexer); err != nil {
		return jsonResponse(http.StatusBadRequest, map[string]string{"error": "Invalid JSON"})
	}

	indexers, err := p.getIndexers(ctx, req.SDK)
	if err != nil {
		return jsonResponse(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	found := false
	for i, indexer := range indexers {
		if indexer.ID == indexerID {
			// Preserve existing API key if the update contains a masked value
			if strings.Contains(updatedIndexer.APIKey, "*") {
				updatedIndexer.APIKey = indexer.APIKey
			}

			updatedIndexer.ID = indexerID // Ensure ID doesn't change
			indexers[i] = updatedIndexer
			found = true
			break
		}
	}

	if !found {
		return jsonResponse(http.StatusNotFound, map[string]string{"error": "Indexer not found"})
	}

	if err := p.saveIndexers(ctx, req.SDK, indexers); err != nil {
		return jsonResponse(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	// Mask API key in response
	updatedIndexer.APIKey = maskAPIKey(updatedIndexer.APIKey)
	return jsonResponse(http.StatusOK, updatedIndexer)
}

func (p *UsenetIndexerPlugin) handleDeleteIndexer(ctx context.Context, req *plugins.PluginHTTPRequest, indexerID string) (*plugins.PluginHTTPResponse, error) {
	if req.SDK == nil {
		return jsonResponse(http.StatusInternalServerError, map[string]string{"error": "SDK not available"})
	}

	indexers, err := p.getIndexers(ctx, req.SDK)
	if err != nil {
		return jsonResponse(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	found := false
	newIndexers := []IndexerConfig{}
	for _, indexer := range indexers {
		if indexer.ID != indexerID {
			newIndexers = append(newIndexers, indexer)
		} else {
			found = true
		}
	}

	if !found {
		return jsonResponse(http.StatusNotFound, map[string]string{"error": "Indexer not found"})
	}

	if err := p.saveIndexers(ctx, req.SDK, newIndexers); err != nil {
		return jsonResponse(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return jsonResponse(http.StatusOK, map[string]string{"message": "Indexer deleted"})
}

func (p *UsenetIndexerPlugin) handleTestIndexer(ctx context.Context, req *plugins.PluginHTTPRequest, indexerID string) (*plugins.PluginHTTPResponse, error) {
	if req.SDK == nil {
		return jsonResponse(http.StatusInternalServerError, map[string]string{"error": "SDK not available"})
	}

	indexers, err := p.getIndexers(ctx, req.SDK)
	if err != nil {
		return jsonResponse(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	var indexer *IndexerConfig
	for _, idx := range indexers {
		if idx.ID == indexerID {
			indexer = &idx
			break
		}
	}

	if indexer == nil {
		return jsonResponse(http.StatusNotFound, map[string]string{"error": "Indexer not found"})
	}

	// Test connection using the Newznab client
	client := NewNewznabClient(indexer.URL, indexer.APIKey)
	if err := client.TestConnection(); err != nil {
		return jsonResponse(http.StatusOK, map[string]interface{}{
			"success": false,
			"error":   fmt.Sprintf("Connection failed: %v", err),
		})
	}

	return jsonResponse(http.StatusOK, map[string]interface{}{
		"success": true,
		"message": "Connection successful",
	})
}

// Search Handlers

func (p *UsenetIndexerPlugin) handleSearch(ctx context.Context, req *plugins.PluginHTTPRequest) (*plugins.PluginHTTPResponse, error) {
	if req.SDK == nil {
		return jsonResponse(http.StatusInternalServerError, map[string]string{"error": "SDK not available"})
	}

	indexers, err := p.getEnabledIndexers(ctx, req.SDK)
	if err != nil {
		return jsonResponse(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	if len(indexers) == 0 {
		return jsonResponse(http.StatusBadRequest, map[string]string{"error": "No enabled indexers configured"})
	}

	params := p.parseSearchParams(req.Query)

	results, err := p.searchMultipleIndexers(ctx, indexers, params, func(client *NewznabClient, params SearchParams) ([]Release, error) {
		return client.Search(params)
	})

	if err != nil {
		return jsonResponse(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return jsonResponse(http.StatusOK, map[string]interface{}{
		"releases": results,
		"count":    len(results),
	})
}

func (p *UsenetIndexerPlugin) handleSearchTV(ctx context.Context, req *plugins.PluginHTTPRequest) (*plugins.PluginHTTPResponse, error) {
	if req.SDK == nil {
		return jsonResponse(http.StatusInternalServerError, map[string]string{"error": "SDK not available"})
	}

	indexers, err := p.getEnabledIndexers(ctx, req.SDK)
	if err != nil {
		return jsonResponse(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	if len(indexers) == 0 {
		return jsonResponse(http.StatusBadRequest, map[string]string{"error": "No enabled indexers configured"})
	}

	params := p.parseSearchParams(req.Query)

	results, err := p.searchMultipleIndexers(ctx, indexers, params, func(client *NewznabClient, params SearchParams) ([]Release, error) {
		return client.SearchTV(params)
	})

	if err != nil {
		return jsonResponse(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return jsonResponse(http.StatusOK, map[string]interface{}{
		"releases": results,
		"count":    len(results),
	})
}

func (p *UsenetIndexerPlugin) handleSearchMovie(ctx context.Context, req *plugins.PluginHTTPRequest) (*plugins.PluginHTTPResponse, error) {
	if req.SDK == nil {
		return jsonResponse(http.StatusInternalServerError, map[string]string{"error": "SDK not available"})
	}

	indexers, err := p.getEnabledIndexers(ctx, req.SDK)
	if err != nil {
		return jsonResponse(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	if len(indexers) == 0 {
		return jsonResponse(http.StatusBadRequest, map[string]string{"error": "No enabled indexers configured"})
	}

	params := p.parseSearchParams(req.Query)

	results, err := p.searchMultipleIndexers(ctx, indexers, params, func(client *NewznabClient, params SearchParams) ([]Release, error) {
		return client.SearchMovie(params)
	})

	if err != nil {
		return jsonResponse(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return jsonResponse(http.StatusOK, map[string]interface{}{
		"releases": results,
		"count":    len(results),
	})
}

func (p *UsenetIndexerPlugin) handleRSS(ctx context.Context, req *plugins.PluginHTTPRequest) (*plugins.PluginHTTPResponse, error) {
	if req.SDK == nil {
		return jsonResponse(http.StatusInternalServerError, map[string]string{"error": "SDK not available"})
	}

	indexers, err := p.getEnabledIndexers(ctx, req.SDK)
	if err != nil {
		return jsonResponse(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	if len(indexers) == 0 {
		return jsonResponse(http.StatusBadRequest, map[string]string{"error": "No enabled indexers configured"})
	}

	// Get categories and limit from query params
	var categories []string
	if cats := req.Query["categories"]; len(cats) > 0 {
		categories = strings.Split(cats[0], ",")
	}

	limit := 100
	if limitStr := req.Query["limit"]; len(limitStr) > 0 {
		if l, err := strconv.Atoi(limitStr[0]); err == nil {
			limit = l
		}
	}

	// Aggregate RSS feeds from all enabled indexers
	type indexerResult struct {
		releases []Release
		err      error
	}

	resultChan := make(chan indexerResult, len(indexers))
	var wg sync.WaitGroup

	for _, indexer := range indexers {
		wg.Add(1)
		go func(idx IndexerConfig) {
			defer wg.Done()

			client := NewNewznabClient(idx.URL, idx.APIKey)
			releases, err := client.GetRSSFeed(categories, limit)

			resultChan <- indexerResult{releases: releases, err: err}
		}(indexer)
	}

	// Wait for all requests to complete
	go func() {
		wg.Wait()
		close(resultChan)
	}()

	// Collect results
	allReleases := []Release{}
	for result := range resultChan {
		if result.err != nil {
			fmt.Fprintf(os.Stderr, "RSS feed error from indexer: %v\n", result.err)
			continue
		}
		allReleases = append(allReleases, result.releases...)
	}

	// Sort by publish date (newest first)
	sort.Slice(allReleases, func(i, j int) bool {
		return allReleases[i].PublishDate.After(allReleases[j].PublishDate)
	})

	// Apply limit to aggregated results
	if len(allReleases) > limit {
		allReleases = allReleases[:limit]
	}

	return jsonResponse(http.StatusOK, map[string]interface{}{
		"releases": allReleases,
		"count":    len(allReleases),
	})
}

// UIManifest returns the UI configuration for this plugin
func (p *UsenetIndexerPlugin) UIManifest(ctx context.Context) (*plugins.UIManifest, error) {
	return &plugins.UIManifest{
		NavItems: []plugins.UINavItem{},
		Routes:   []plugins.UIRoute{},
		ConfigSection: &plugins.ConfigSection{
			Title:       "Usenet Indexer Settings",
			Description: "Configure Newznab-compatible Usenet indexers for searching releases",
			Fields: []plugins.ConfigField{
				{
					Key:          configIndexers,
					Label:        "Indexers",
					Description:  "Configure your Newznab-compatible indexers",
					Type:         "custom",
					DefaultValue: "[]",
					Required:     false,
				},
			},
		},
	}, nil
}

// HandleEvent handles system events (not implemented)
func (p *UsenetIndexerPlugin) HandleEvent(ctx context.Context, evt plugins.Event) error {
	return nil
}

// IsIndexer returns true to indicate this plugin provides indexer functionality
func (p *UsenetIndexerPlugin) IsIndexer(ctx context.Context) (bool, error) {
	return true, nil
}

// Search implements the unified indexer search interface
func (p *UsenetIndexerPlugin) Search(ctx context.Context, req *plugins.IndexerSearchRequest) (*plugins.IndexerSearchResponse, error) {
	// Note: This method doesn't have direct SDK access like HandleAPI does
	// For now, we return an error indicating this should be called via the HTTP API
	// In the future, we could pass SDK through the RPC call
	return &plugins.IndexerSearchResponse{
		Releases:    []plugins.IndexerRelease{},
		Total:       0,
		IndexerID:   "usenet-indexer",
		IndexerName: "Usenet Indexer",
	}, fmt.Errorf("direct RPC search requires SDK access - please use HTTP API endpoints instead")
}

// IsDownloader returns false as this is not a downloader plugin
func (p *UsenetIndexerPlugin) IsDownloader(ctx context.Context) (bool, error) {
	return false, nil
}

// Helper functions

func (p *UsenetIndexerPlugin) getIndexers(ctx context.Context, sdk plugins.SDKInterface) ([]IndexerConfig, error) {
	val, err := sdk.ConfigGet(ctx, configIndexers)
	if err != nil {
		return []IndexerConfig{}, nil
	}

	if val == nil {
		return []IndexerConfig{}, nil
	}

	var indexers []IndexerConfig
	switch v := val.(type) {
	case []interface{}:
		jsonData, _ := json.Marshal(v)
		json.Unmarshal(jsonData, &indexers)
	case string:
		json.Unmarshal([]byte(v), &indexers)
	default:
		jsonData, _ := json.Marshal(v)
		json.Unmarshal(jsonData, &indexers)
	}

	return indexers, nil
}

func (p *UsenetIndexerPlugin) saveIndexers(ctx context.Context, sdk plugins.SDKInterface, indexers []IndexerConfig) error {
	return sdk.ConfigSet(ctx, configIndexers, indexers)
}

func (p *UsenetIndexerPlugin) getEnabledIndexers(ctx context.Context, sdk plugins.SDKInterface) ([]IndexerConfig, error) {
	allIndexers, err := p.getIndexers(ctx, sdk)
	if err != nil {
		return nil, err
	}

	enabledIndexers := []IndexerConfig{}
	for _, indexer := range allIndexers {
		if indexer.Enabled {
			enabledIndexers = append(enabledIndexers, indexer)
		}
	}

	// Sort by priority (lower priority value = higher priority)
	sort.Slice(enabledIndexers, func(i, j int) bool {
		return enabledIndexers[i].Priority < enabledIndexers[j].Priority
	})

	return enabledIndexers, nil
}

// searchMultipleIndexers searches across multiple indexers in parallel and aggregates results
func (p *UsenetIndexerPlugin) searchMultipleIndexers(
	ctx context.Context,
	indexers []IndexerConfig,
	params SearchParams,
	searchFunc func(*NewznabClient, SearchParams) ([]Release, error),
) ([]Release, error) {
	type indexerResult struct {
		indexerName string
		releases    []Release
		err         error
	}

	resultChan := make(chan indexerResult, len(indexers))
	var wg sync.WaitGroup

	for _, indexer := range indexers {
		wg.Add(1)
		go func(idx IndexerConfig) {
			defer wg.Done()

			// Create a copy of params for this indexer
			indexerParams := params

			// Use indexer-specific categories if none specified in request
			if len(indexerParams.Categories) == 0 {
				// Determine which categories to use based on search type
				// For TV searches, use TVCategories; for movie searches, use MovieCategories
				// For general searches, don't specify categories
				if strings.Contains(fmt.Sprintf("%v", searchFunc), "SearchTV") {
					indexerParams.Categories = idx.TVCategories
				} else if strings.Contains(fmt.Sprintf("%v", searchFunc), "SearchMovie") {
					indexerParams.Categories = idx.MovieCategories
				}
			}

			client := NewNewznabClient(idx.URL, idx.APIKey)
			releases, err := searchFunc(client, indexerParams)

			// Tag releases with indexer name
			for i := range releases {
				releases[i].Attributes["indexer"] = idx.Name
				releases[i].Attributes["indexer_id"] = idx.ID
				releases[i].IndexerID = idx.ID
				releases[i].IndexerName = idx.Name
			}

			resultChan <- indexerResult{
				indexerName: idx.Name,
				releases:    releases,
				err:         err,
			}
		}(indexer)
	}

	// Wait for all requests to complete
	go func() {
		wg.Wait()
		close(resultChan)
	}()

	// Collect results and track indexers that returned nothing
	allReleases := []Release{}
	indexersNeedingFallback := []IndexerConfig{}
	var lastError error

	for result := range resultChan {
		if result.err != nil {
			fmt.Fprintf(os.Stderr, "Search error from indexer %s: %v\n", result.indexerName, result.err)
			lastError = result.err
			continue
		}

		if len(result.releases) > 0 {
			allReleases = append(allReleases, result.releases...)
		} else {
			// This indexer returned 0 results - may need fallback
			for _, idx := range indexers {
				if idx.Name == result.indexerName {
					indexersNeedingFallback = append(indexersNeedingFallback, idx)
					break
				}
			}
		}
	}

	// If some indexers returned no results with tvdbid, retry with query-based search
	if len(indexersNeedingFallback) > 0 && params.TVDBID != "" && params.Query != "" && (params.Season > 0 || params.Episode > 0) {
		// Retry those indexers without tvdbid (will use query parameter)
		fallbackParams := params
		fallbackParams.TVDBID = ""

		fallbackResultChan := make(chan indexerResult, len(indexersNeedingFallback))
		var fallbackWg sync.WaitGroup

		for _, indexer := range indexersNeedingFallback {
			fallbackWg.Add(1)
			go func(idx IndexerConfig) {
				defer fallbackWg.Done()

				client := NewNewznabClient(idx.URL, idx.APIKey)
				releases, err := searchFunc(client, fallbackParams)

				// Tag releases with indexer name
				for i := range releases {
					releases[i].Attributes["indexer"] = idx.Name
					releases[i].Attributes["indexer_id"] = idx.ID
					releases[i].IndexerID = idx.ID
					releases[i].IndexerName = idx.Name
				}

				fallbackResultChan <- indexerResult{
					indexerName: idx.Name,
					releases:    releases,
					err:         err,
				}
			}(indexer)
		}

		// Wait for fallback searches
		go func() {
			fallbackWg.Wait()
			close(fallbackResultChan)
		}()

		// Collect fallback results and filter by series name
		for result := range fallbackResultChan {
			if result.err != nil {
				continue
			}
			if len(result.releases) > 0 {
				// Filter releases to match series name exactly
				filtered := filterBySeriesName(result.releases, params.Query)
				allReleases = append(allReleases, filtered...)
			}
		}
	}

	// If all indexers failed, return the last error
	if len(allReleases) == 0 && lastError != nil {
		return nil, fmt.Errorf("all indexers failed, last error: %w", lastError)
	}

	// Sort results by publish date (newest first)
	sort.Slice(allReleases, func(i, j int) bool {
		return allReleases[i].PublishDate.After(allReleases[j].PublishDate)
	})

	// Remove duplicates based on GUID
	seen := make(map[string]bool)
	uniqueReleases := []Release{}
	for _, release := range allReleases {
		if !seen[release.GUID] {
			seen[release.GUID] = true
			uniqueReleases = append(uniqueReleases, release)
		}
	}

	return uniqueReleases, nil
}

func (p *UsenetIndexerPlugin) parseSearchParams(query map[string][]string) SearchParams {
	params := SearchParams{}

	if q := query["q"]; len(q) > 0 {
		params.Query = q[0]
	}
	if cats := query["categories"]; len(cats) > 0 {
		params.Categories = strings.Split(cats[0], ",")
	}
	if tvdbid := query["tvdbid"]; len(tvdbid) > 0 {
		params.TVDBID = tvdbid[0]
	}
	if tvrageid := query["tvrageid"]; len(tvrageid) > 0 {
		params.TVRageID = tvrageid[0]
	}
	if imdbid := query["imdbid"]; len(imdbid) > 0 {
		params.IMDBID = imdbid[0]
	}
	if season := query["season"]; len(season) > 0 {
		if s, err := strconv.Atoi(season[0]); err == nil {
			params.Season = s
		}
	}
	if episode := query["episode"]; len(episode) > 0 {
		if e, err := strconv.Atoi(episode[0]); err == nil {
			params.Episode = e
		}
	}
	if limit := query["limit"]; len(limit) > 0 {
		if l, err := strconv.Atoi(limit[0]); err == nil {
			params.Limit = l
		}
	}
	if offset := query["offset"]; len(offset) > 0 {
		if o, err := strconv.Atoi(offset[0]); err == nil {
			params.Offset = o
		}
	}

	return params
}

func generateID(name string) string {
	id := strings.ToLower(name)
	id = strings.ReplaceAll(id, " ", "-")
	id = strings.ReplaceAll(id, "_", "-")
	allowed := "abcdefghijklmnopqrstuvwxyz0123456789-"
	result := ""
	for _, c := range id {
		if strings.ContainsRune(allowed, c) {
			result += string(c)
		}
	}
	if result == "" {
		result = fmt.Sprintf("indexer-%d", time.Now().Unix())
	}
	return result
}

func maskAPIKey(key string) string {
	if key == "" {
		return ""
	}
	if len(key) <= 8 {
		return strings.Repeat("*", len(key))
	}
	return key[:4] + strings.Repeat("*", len(key)-8) + key[len(key)-4:]
}

func jsonResponse(statusCode int, data interface{}) (*plugins.PluginHTTPResponse, error) {
	body, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	return &plugins.PluginHTTPResponse{
		StatusCode: statusCode,
		Headers: map[string][]string{
			"Content-Type": {"application/json"},
		},
		Body: body,
	}, nil
}

// filterBySeriesName filters releases to only include those that match the series name
// This helps avoid false matches like "The Rookie Feds" when searching for "The Rookie"
func filterBySeriesName(releases []Release, seriesName string) []Release {
	filtered := []Release{}

	// Normalize series name for comparison (lowercase, replace spaces with dots)
	normalizedSeries := strings.ToLower(strings.ReplaceAll(seriesName, " ", "."))

	for _, release := range releases {
		// Normalize release title
		normalizedTitle := strings.ToLower(release.Title)

		// Check if the release title starts with the series name
		if strings.HasPrefix(normalizedTitle, normalizedSeries) {
			// Check what comes after the series name
			afterSeries := normalizedTitle[len(normalizedSeries):]

			// Must be followed by:
			// - Nothing (exact match)
			// - Year in parentheses like (2018)
			// - Season marker like .S01 or .s01
			// - Direct season marker like S01 (no dot)
			if len(afterSeries) == 0 ||
				afterSeries[0] == '(' || // Year
				(len(afterSeries) >= 2 && afterSeries[0] == '.' && (afterSeries[1] == 's' || afterSeries[1] == 'S')) || // .S01
				(afterSeries[0] == 's' || afterSeries[0] == 'S') { // S01
				filtered = append(filtered, release)
			}
		}
	}

	return filtered
}

func main() {
	// Create plugin instance
	usenetPlugin := &UsenetIndexerPlugin{}

	// Serve the plugin using go-plugin
	plugin.Serve(&plugin.ServeConfig{
		HandshakeConfig: plugins.Handshake,
		Plugins: map[string]plugin.Plugin{
			"media-suite": &plugins.MediaSuitePluginGRPC{
				Impl: usenetPlugin,
			},
		},
		GRPCServer: plugin.DefaultGRPCServer,
	})
}
