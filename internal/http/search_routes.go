package http

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"

	"github.com/blakestevenson/nimbus/internal/db/generated"
	"github.com/blakestevenson/nimbus/internal/indexer"
	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"
)

// setupSearchRoutes registers the interactive search API endpoints
func setupSearchRoutes(r interface {
	Get(pattern string, handlerFn http.HandlerFunc)
}, indexerService *indexer.Service, queries *generated.Queries, logger *zap.Logger) {
	// Interactive search for specific media items
	// Note: This is called within r.Route("/media", ...) so the pattern is relative
	r.Get("/{id}/search", func(w http.ResponseWriter, r *http.Request) {
		handleInteractiveSearch(w, r, indexerService, queries, logger)
	})
}

// handleInteractiveSearch performs an interactive search for a specific media item
func handleInteractiveSearch(w http.ResponseWriter, r *http.Request, indexerService *indexer.Service, queries *generated.Queries, logger *zap.Logger) {
	// Extract media ID from URL parameter
	mediaIDStr := chi.URLParam(r, "id")
	if mediaIDStr == "" {
		http.Error(w, "Media ID not found", http.StatusBadRequest)
		return
	}

	mediaID, err := strconv.ParseInt(mediaIDStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid media ID", http.StatusBadRequest)
		return
	}

	// Fetch media item from database
	media, err := queries.GetMediaItem(r.Context(), mediaID)
	if err != nil {
		logger.Error("Failed to get media item", zap.Error(err), zap.Int64("media_id", mediaID))
		http.Error(w, "Media item not found", http.StatusNotFound)
		return
	}

	// For seasons and episodes, we need the parent series title
	var seriesTitle string
	if media.Kind == "tv_season" || media.Kind == "tv_episode" {
		seriesTitle, err = getSeriesTitle(r.Context(), queries, media)
		if err != nil {
			logger.Warn("Failed to get series title, using media title", zap.Error(err))
			seriesTitle = media.Title
		}
	}

	// Build search request based on media kind and metadata
	searchReq := buildSearchRequestFromMediaWithQueries(media, seriesTitle, queries, r.Context())

	// Debug output to stderr
	fmt.Fprintf(os.Stderr, "SEARCH DEBUG: mediaID=%d kind=%s tvdbid=%s season=%d episode=%d query=%s\n",
		mediaID, media.Kind, searchReq.TVDBID, searchReq.Season, searchReq.Episode, searchReq.Query)

	// Log the search request
	logger.Info("Interactive search initiated",
		zap.Int64("media_id", mediaID),
		zap.String("kind", media.Kind),
		zap.String("title", media.Title),
		zap.String("search_type", searchReq.Type),
		zap.String("tvdb_id", searchReq.TVDBID),
		zap.Int("season", searchReq.Season),
		zap.Int("episode", searchReq.Episode),
		zap.String("query", searchReq.Query))

	// Pass authentication cookie to indexer service
	cookies := r.Cookies()

	// Perform search
	resp, err := indexerService.SearchWithAuth(r.Context(), searchReq, cookies)
	if err != nil {
		logger.Error("Interactive search failed", zap.Error(err))
		http.Error(w, fmt.Sprintf("Search failed: %v", err), http.StatusInternalServerError)
		return
	}

	// Return results
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]interface{}{
		"media_id": mediaID,
		"releases": resp.Releases,
		"total":    resp.Total,
		"sources":  resp.Sources,
		"metadata": map[string]interface{}{
			"kind":  media.Kind,
			"title": media.Title,
			"year":  media.Year,
		},
	}); err != nil {
		logger.Error("Failed to encode search response", zap.Error(err))
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

// getSeriesTitle retrieves the series title for a season or episode
func getSeriesTitle(ctx context.Context, queries *generated.Queries, media generated.MediaItem) (string, error) {
	// For episodes, go up two levels (episode -> season -> series)
	// For seasons, go up one level (season -> series)

	if media.Kind == "tv_episode" {
		// Get parent season
		if media.ParentID != nil {
			season, err := queries.GetMediaItem(ctx, *media.ParentID)
			if err != nil {
				return "", fmt.Errorf("failed to get season: %w", err)
			}
			// Get parent series
			if season.ParentID != nil {
				series, err := queries.GetMediaItem(ctx, *season.ParentID)
				if err != nil {
					return "", fmt.Errorf("failed to get series: %w", err)
				}
				return series.Title, nil
			}
		}
	} else if media.Kind == "tv_season" {
		// Get parent series
		if media.ParentID != nil {
			series, err := queries.GetMediaItem(ctx, *media.ParentID)
			if err != nil {
				return "", fmt.Errorf("failed to get series: %w", err)
			}
			return series.Title, nil
		}
	}

	return media.Title, nil
}

// buildSearchRequestFromMedia constructs an indexer search request from a media item
func buildSearchRequestFromMedia(media generated.MediaItem, seriesTitle string) indexer.SearchRequest {
	return buildSearchRequestFromMediaWithQueries(media, seriesTitle, nil, context.Background())
}

// buildSearchRequestFromMediaWithQueries constructs an indexer search request with database access
func buildSearchRequestFromMediaWithQueries(media generated.MediaItem, seriesTitle string, queries *generated.Queries, ctx context.Context) indexer.SearchRequest {
	// Use series title for TV content if provided, otherwise use media title
	queryTitle := media.Title
	if seriesTitle != "" {
		queryTitle = seriesTitle
	}

	req := indexer.SearchRequest{
		Query: queryTitle,
		Limit: 100,
	}

	// Parse metadata
	var metadata map[string]interface{}
	if len(media.Metadata) > 0 {
		_ = json.Unmarshal(media.Metadata, &metadata)
	}

	// Parse external_ids (prioritized over metadata)
	var externalIDs map[string]interface{}
	if len(media.ExternalIds) > 0 {
		_ = json.Unmarshal(media.ExternalIds, &externalIDs)
	}

	// If this is a season or episode and we don't have tvdb_id, try to get it from the parent
	// Check if we have tvdb_id in current external IDs
	hasTVDBID := false
	if externalIDs != nil {
		if _, ok := externalIDs["tvdb_id"]; ok {
			hasTVDBID = true
		}
	}

	if (media.Kind == "tv_season" || media.Kind == "tv_episode") && !hasTVDBID && queries != nil && media.ParentID != nil {
		// Try to get parent's external IDs
		if parent, err := queries.GetMediaItem(ctx, *media.ParentID); err == nil {
			if len(parent.ExternalIds) > 0 {
				var parentExternalIDs map[string]interface{}
				json.Unmarshal(parent.ExternalIds, &parentExternalIDs)

				// Merge parent external IDs into our map
				if parentExternalIDs != nil {
					if externalIDs == nil {
						externalIDs = make(map[string]interface{})
					}
					for k, v := range parentExternalIDs {
						if _, exists := externalIDs[k]; !exists {
							externalIDs[k] = v
						}
					}
				}
			}
			// For episodes, if we still don't have tvdb_id, try the grandparent (series)
			if media.Kind == "tv_episode" && parent.ParentID != nil {
				hasTVDBID := false
				if externalIDs != nil {
					if _, ok := externalIDs["tvdb_id"]; ok {
						hasTVDBID = true
					}
				}
				if !hasTVDBID {
					if grandparent, err := queries.GetMediaItem(ctx, *parent.ParentID); err == nil {
						if len(grandparent.ExternalIds) > 0 {
							var grandparentExternalIDs map[string]interface{}
							json.Unmarshal(grandparent.ExternalIds, &grandparentExternalIDs)

							// Merge grandparent external IDs
							if grandparentExternalIDs != nil {
								if externalIDs == nil {
									externalIDs = make(map[string]interface{})
								}
								for k, v := range grandparentExternalIDs {
									if _, exists := externalIDs[k]; !exists {
										externalIDs[k] = v
									}
								}
							}
						}
					}
				}
			}
		}
	}

	// Helper function to get ID from external_ids or fallback to metadata
	getID := func(extKey, metaKey string) string {
		// Try external_ids first
		if externalIDs != nil {
			if val, ok := externalIDs[extKey].(string); ok && val != "" {
				return val
			}
			// Handle numeric IDs
			if val, ok := externalIDs[extKey].(float64); ok && val > 0 {
				return fmt.Sprintf("%.0f", val)
			}
		}
		// Fallback to metadata
		if metadata != nil {
			if val, ok := metadata[metaKey].(string); ok && val != "" {
				return val
			}
			if val, ok := metadata[metaKey].(float64); ok && val > 0 {
				return fmt.Sprintf("%.0f", val)
			}
		}
		return ""
	}

	// Configure search based on media kind
	switch media.Kind {
	case "movie":
		req.Type = "movie"
		req.IMDBID = getID("imdb_id", "imdb_id")
		req.TMDBID = getID("tmdb_id", "tmdb_id")

	case "tv_episode":
		req.Type = "tv"
		// Always include series title as query for fallback searches
		if seriesTitle != "" {
			req.Query = seriesTitle
		}
		if season, ok := metadata["season"].(float64); ok {
			req.Season = int(season)
		}
		if episode, ok := metadata["episode"].(float64); ok {
			req.Episode = int(episode)
		}
		req.TVDBID = getID("tvdb_id", "tvdb_id")

	case "tv_season":
		req.Type = "tv"
		if season, ok := metadata["season_number"].(float64); ok {
			req.Season = int(season)
			// Add season query (e.g., "S01") to prioritize season packs
			req.Query = fmt.Sprintf("S%02d", int(season))
		}
		req.TVDBID = getID("tvdb_id", "tvdb_id")

	case "tv_series":
		req.Type = "tv"
		req.TVDBID = getID("tvdb_id", "tvdb_id")

	default:
		req.Type = "general"
	}

	return req
}
