package http

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/blakestevenson/nimbus/internal/indexer"
	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"
)

// setupIndexerRoutes registers the unified indexer API endpoints
func setupIndexerRoutes(r chi.Router, indexerService *indexer.Service, logger *zap.Logger) {
	// List available indexers
	r.Get("/indexers", func(w http.ResponseWriter, r *http.Request) {
		indexers := indexerService.ListIndexers()

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(map[string]interface{}{
			"indexers": indexers,
			"count":    len(indexers),
		}); err != nil {
			logger.Error("Failed to encode indexers response", zap.Error(err))
			http.Error(w, "Internal server error", http.StatusInternalServerError)
		}
	})

	// Unified search endpoints
	r.Get("/indexers/search", func(w http.ResponseWriter, r *http.Request) {
		req := parseIndexerSearchRequest(r)
		req.Type = "general"

		resp, err := indexerService.Search(r.Context(), req)
		if err != nil {
			logger.Error("Indexer search failed", zap.Error(err))
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(map[string]interface{}{
			"releases": resp.Releases,
			"total":    resp.Total,
			"sources":  resp.Sources,
		}); err != nil {
			logger.Error("Failed to encode search response", zap.Error(err))
			http.Error(w, "Internal server error", http.StatusInternalServerError)
		}
	})

	r.Get("/indexers/search/tv", func(w http.ResponseWriter, r *http.Request) {
		req := parseIndexerSearchRequest(r)
		req.Type = "tv"

		resp, err := indexerService.Search(r.Context(), req)
		if err != nil {
			logger.Error("Indexer TV search failed", zap.Error(err))
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(map[string]interface{}{
			"releases": resp.Releases,
			"total":    resp.Total,
			"sources":  resp.Sources,
		}); err != nil {
			logger.Error("Failed to encode TV search response", zap.Error(err))
			http.Error(w, "Internal server error", http.StatusInternalServerError)
		}
	})

	r.Get("/indexers/search/movie", func(w http.ResponseWriter, r *http.Request) {
		req := parseIndexerSearchRequest(r)
		req.Type = "movie"

		resp, err := indexerService.Search(r.Context(), req)
		if err != nil {
			logger.Error("Indexer movie search failed", zap.Error(err))
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(map[string]interface{}{
			"releases": resp.Releases,
			"total":    resp.Total,
			"sources":  resp.Sources,
		}); err != nil {
			logger.Error("Failed to encode movie search response", zap.Error(err))
			http.Error(w, "Internal server error", http.StatusInternalServerError)
		}
	})
}

// parseIndexerSearchRequest parses query parameters into an IndexerSearchRequest
func parseIndexerSearchRequest(r *http.Request) indexer.SearchRequest {
	query := r.URL.Query()

	req := indexer.SearchRequest{
		Query:      query.Get("q"),
		Categories: []string{},
		Limit:      100, // Default limit
	}

	// Parse categories
	if cats := query.Get("categories"); cats != "" {
		req.Categories = []string{cats}
	}

	// Parse TV-specific parameters
	req.TVDBID = query.Get("tvdbid")
	req.TVRageID = query.Get("tvrageid")

	if season := query.Get("season"); season != "" {
		if s, err := strconv.Atoi(season); err == nil {
			req.Season = s
		}
	}

	if episode := query.Get("episode"); episode != "" {
		if e, err := strconv.Atoi(episode); err == nil {
			req.Episode = e
		}
	}

	// Parse movie-specific parameters
	req.IMDBID = query.Get("imdbid")
	req.TMDBID = query.Get("tmdbid")

	// Parse pagination
	if limit := query.Get("limit"); limit != "" {
		if l, err := strconv.Atoi(limit); err == nil {
			req.Limit = l
		}
	}

	if offset := query.Get("offset"); offset != "" {
		if o, err := strconv.Atoi(offset); err == nil {
			req.Offset = o
		}
	}

	return req
}
