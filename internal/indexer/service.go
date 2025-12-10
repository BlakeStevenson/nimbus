package indexer

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/blakestevenson/nimbus/internal/plugins"
	"go.uber.org/zap"
)

// Service provides a unified interface for searching across all indexer plugins
type Service struct {
	pluginManager *plugins.PluginManager
	logger        *zap.Logger
	httpClient    *http.Client
	baseURL       string // Base URL for internal API calls (e.g., "http://localhost:8080")
}

// NewService creates a new indexer service
func NewService(pluginManager *plugins.PluginManager, logger *zap.Logger) *Service {
	return &Service{
		pluginManager: pluginManager,
		logger:        logger.With(zap.String("component", "indexer-service")),
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		baseURL: "http://localhost:8080", // Default, should be configurable
	}
}

// SetBaseURL sets the base URL for internal API calls
func (s *Service) SetBaseURL(baseURL string) {
	s.baseURL = baseURL
}

// SearchRequest represents a unified search request
type SearchRequest struct {
	Query      string
	Type       string // "general", "tv", "movie"
	Categories []string
	TVDBID     string
	TVRageID   string
	Season     int
	Episode    int
	IMDBID     string
	TMDBID     string
	Limit      int
	Offset     int
}

// SearchResponse represents aggregated search results from all indexers
type SearchResponse struct {
	Releases []plugins.IndexerRelease
	Total    int
	Sources  []string // List of indexer IDs that were searched
}

// Search performs a search across all available indexer plugins
func (s *Service) Search(ctx context.Context, req SearchRequest) (*SearchResponse, error) {
	return s.SearchWithAuth(ctx, req, nil)
}

// SearchWithAuth performs a search with authentication cookies
func (s *Service) SearchWithAuth(ctx context.Context, req SearchRequest, cookies []*http.Cookie) (*SearchResponse, error) {
	// Get all indexer plugins
	indexerPlugins := s.pluginManager.ListIndexerPlugins()

	if len(indexerPlugins) == 0 {
		return &SearchResponse{
			Releases: []plugins.IndexerRelease{},
			Total:    0,
			Sources:  []string{},
		}, nil
	}

	s.logger.Info("Searching across indexer plugins",
		zap.Int("indexer_count", len(indexerPlugins)),
		zap.String("query", req.Query),
		zap.String("type", req.Type))

	// Search all indexers in parallel using HTTP API
	type result struct {
		releases []plugins.IndexerRelease
		err      error
		pluginID string
	}

	resultChan := make(chan result, len(indexerPlugins))
	var wg sync.WaitGroup

	for _, plugin := range indexerPlugins {
		wg.Add(1)
		go func(p *plugins.LoadedPlugin) {
			defer wg.Done()

			releases, err := s.searchPluginViaHTTP(ctx, p.Meta.ID, req, cookies)
			resultChan <- result{
				releases: releases,
				err:      err,
				pluginID: p.Meta.ID,
			}
		}(plugin)
	}

	// Wait for all searches to complete
	go func() {
		wg.Wait()
		close(resultChan)
	}()

	// Collect results and track which indexers need fallback
	allReleases := []plugins.IndexerRelease{}
	sources := []string{}
	pluginsNeedingFallback := []string{}
	var lastError error

	for res := range resultChan {
		if res.err != nil {
			s.logger.Warn("Search failed for indexer",
				zap.String("plugin_id", res.pluginID),
				zap.Error(res.err))
			lastError = res.err
			continue
		}

		if res.releases != nil && len(res.releases) > 0 {
			allReleases = append(allReleases, res.releases...)
			sources = append(sources, res.pluginID)
		} else {
			// This indexer returned 0 results, may need fallback
			pluginsNeedingFallback = append(pluginsNeedingFallback, res.pluginID)
		}
	}

	// If all indexers failed, return error
	if len(allReleases) == 0 && lastError != nil {
		return nil, fmt.Errorf("all indexers failed: %w", lastError)
	}

	// If some indexers returned no results with tvdbid for TV episodes/seasons, retry those with title-based search
	if len(pluginsNeedingFallback) > 0 && req.Type == "tv" && req.TVDBID != "" && req.Query != "" && (req.Season > 0 || req.Episode > 0) {
		s.logger.Info("Some indexers returned no results with tvdbid, retrying with title-based search",
			zap.String("series_title", req.Query),
			zap.Int("season", req.Season),
			zap.Int("episode", req.Episode),
			zap.Int("indexers_needing_fallback", len(pluginsNeedingFallback)))

		// Retry with title-based search (tvdbid will be empty in the retry)
		fallbackReq := req
		fallbackReq.TVDBID = ""

		// Search only the indexers that returned no results
		fallbackResultChan := make(chan result, len(pluginsNeedingFallback))
		var fallbackWg sync.WaitGroup

		for _, pluginID := range pluginsNeedingFallback {
			fallbackWg.Add(1)
			go func(pid string) {
				defer fallbackWg.Done()

				releases, err := s.searchPluginViaHTTP(ctx, pid, fallbackReq, cookies)
				fallbackResultChan <- result{
					releases: releases,
					err:      err,
					pluginID: pid,
				}
			}(pluginID)
		}

		// Wait for fallback searches
		go func() {
			fallbackWg.Wait()
			close(fallbackResultChan)
		}()

		// Collect fallback results
		for res := range fallbackResultChan {
			if res.err != nil {
				s.logger.Warn("Fallback search failed for indexer",
					zap.String("plugin_id", res.pluginID),
					zap.Error(res.err))
				continue
			}

			if res.releases != nil && len(res.releases) > 0 {
				allReleases = append(allReleases, res.releases...)
				sources = append(sources, res.pluginID)
			}
		}
	}

	// Remove duplicates based on GUID
	uniqueReleases := s.deduplicateReleases(allReleases)

	// For TV searches with query (title-based), filter by exact series name match
	if req.Type == "tv" && req.Query != "" && req.TVDBID == "" {
		uniqueReleases = s.filterBySeriesName(uniqueReleases, req.Query)
	}

	// For season searches (season specified but no episode), filter out individual episodes
	if req.Type == "tv" && req.Season > 0 && req.Episode == 0 {
		uniqueReleases = s.filterSeasonPacks(uniqueReleases)
	}

	// Sort by publish date (newest first)
	sort.Slice(uniqueReleases, func(i, j int) bool {
		return uniqueReleases[i].PublishDate.After(uniqueReleases[j].PublishDate)
	})

	// Apply limit if specified
	if req.Limit > 0 && len(uniqueReleases) > req.Limit {
		uniqueReleases = uniqueReleases[:req.Limit]
	}

	resp := &SearchResponse{
		Releases: uniqueReleases,
		Total:    len(uniqueReleases),
		Sources:  sources,
	}

	// Debug: Add metadata about fallback
	if len(pluginsNeedingFallback) > 0 && req.Type == "tv" && req.TVDBID != "" {
		s.logger.Error("FALLBACK DEBUG",
			zap.Int("plugins_needing_fallback", len(pluginsNeedingFallback)),
			zap.String("had_tvdbid", "yes"))
	}

	return resp, nil
}

// ListIndexers returns information about all available indexer plugins
func (s *Service) ListIndexers() []IndexerInfo {
	indexerPlugins := s.pluginManager.ListIndexerPlugins()

	indexers := make([]IndexerInfo, len(indexerPlugins))
	for i, plugin := range indexerPlugins {
		indexers[i] = IndexerInfo{
			ID:          plugin.Meta.ID,
			Name:        plugin.Meta.Name,
			Version:     plugin.Meta.Version,
			Description: plugin.Meta.Description,
		}
	}

	return indexers
}

// IndexerInfo contains information about an available indexer
type IndexerInfo struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Version     string `json:"version"`
	Description string `json:"description"`
}

// searchPluginViaHTTP searches a plugin using its HTTP API endpoints
func (s *Service) searchPluginViaHTTP(ctx context.Context, pluginID string, req SearchRequest, cookies []*http.Cookie) ([]plugins.IndexerRelease, error) {
	// Build the appropriate endpoint based on search type
	var endpoint string
	switch req.Type {
	case "tv":
		endpoint = fmt.Sprintf("%s/api/plugins/%s/search/tv", s.baseURL, pluginID)
	case "movie":
		endpoint = fmt.Sprintf("%s/api/plugins/%s/search/movie", s.baseURL, pluginID)
	default:
		endpoint = fmt.Sprintf("%s/api/plugins/%s/search", s.baseURL, pluginID)
	}

	// Build query parameters
	params := url.Values{}
	if req.Query != "" {
		params.Add("q", req.Query)
	}
	if len(req.Categories) > 0 {
		for _, cat := range req.Categories {
			params.Add("categories", cat)
		}
	}
	if req.TVDBID != "" {
		params.Add("tvdbid", req.TVDBID)
	}
	if req.TVRageID != "" {
		params.Add("tvrageid", req.TVRageID)
	}
	if req.Season > 0 {
		params.Add("season", strconv.Itoa(req.Season))
	}
	if req.Episode > 0 {
		params.Add("episode", strconv.Itoa(req.Episode))
	}
	if req.IMDBID != "" {
		params.Add("imdbid", req.IMDBID)
	}
	if req.TMDBID != "" {
		params.Add("tmdbid", req.TMDBID)
	}
	if req.Limit > 0 {
		params.Add("limit", strconv.Itoa(req.Limit))
	}
	if req.Offset > 0 {
		params.Add("offset", strconv.Itoa(req.Offset))
	}

	fullURL := endpoint
	if len(params) > 0 {
		fullURL = fmt.Sprintf("%s?%s", endpoint, params.Encode())
	}

	// Log the search URL for debugging
	s.logger.Info("Searching plugin via HTTP",
		zap.String("plugin_id", pluginID),
		zap.String("url", fullURL),
		zap.String("tvdbid", req.TVDBID),
		zap.Int("season", req.Season),
		zap.Int("episode", req.Episode))

	// Create HTTP request
	httpReq, err := http.NewRequestWithContext(ctx, "GET", fullURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Add cookies for authentication
	if cookies != nil {
		for _, cookie := range cookies {
			httpReq.AddCookie(cookie)
		}
	}

	// Make the request
	resp, err := s.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
	}

	// Parse response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var searchResp struct {
		Releases []plugins.IndexerRelease `json:"releases"`
		Count    int                      `json:"count"`
	}

	if err := json.Unmarshal(body, &searchResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return searchResp.Releases, nil
}

// deduplicateReleases removes duplicate releases based on GUID
func (s *Service) deduplicateReleases(releases []plugins.IndexerRelease) []plugins.IndexerRelease {
	seen := make(map[string]bool)
	unique := []plugins.IndexerRelease{}

	for _, release := range releases {
		if !seen[release.GUID] {
			seen[release.GUID] = true
			unique = append(unique, release)
		}
	}

	return unique
}

// filterSeasonPacks removes individual episodes from season search results
// Season packs should not have episode identifiers like E01, E02, etc.
func (s *Service) filterSeasonPacks(releases []plugins.IndexerRelease) []plugins.IndexerRelease {
	filtered := []plugins.IndexerRelease{}

	for _, release := range releases {
		// Check if title contains episode identifier or multi-season identifier
		matched := false
		if len(release.Title) > 0 {
			title := release.Title

			// Look for episode identifiers: E followed by digits (E01, E02, etc.) case insensitive
			for i := 0; i < len(title)-2; i++ {
				if (title[i] == 'E' || title[i] == 'e') &&
					i+3 < len(title) &&
					title[i+1] >= '0' && title[i+1] <= '9' &&
					title[i+2] >= '0' && title[i+2] <= '9' {
					matched = true
					break
				}
			}

			// Look for multi-season patterns: S01-S08, S01-08, etc.
			if !matched {
				for i := 0; i < len(title)-5; i++ {
					if (title[i] == 'S' || title[i] == 's') &&
						i+6 < len(title) &&
						title[i+1] >= '0' && title[i+1] <= '9' &&
						title[i+2] >= '0' && title[i+2] <= '9' &&
						title[i+3] == '-' {
						matched = true
						break
					}
				}
			}
		}

		// Only include releases that don't have episode or multi-season identifiers
		if !matched {
			filtered = append(filtered, release)
		}
	}

	return filtered
}

// filterBySeriesName filters releases to only include those that match the series name
// This helps avoid false matches like "The Rookie Feds" when searching for "The Rookie"
func (s *Service) filterBySeriesName(releases []plugins.IndexerRelease, seriesName string) []plugins.IndexerRelease {
	filtered := []plugins.IndexerRelease{}

	// Normalize series name for comparison (lowercase, replace spaces with dots)
	normalizedSeries := strings.ToLower(strings.ReplaceAll(seriesName, " ", "."))

	for _, release := range releases {
		// Normalize release title
		normalizedTitle := strings.ToLower(release.Title)

		// Check if the release title starts with the series name
		// e.g., "The.Rookie.S01E01" matches "The.Rookie" but "The.Rookie.Feds.S01E01" does not
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
