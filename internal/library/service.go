package library

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/blakestevenson/nimbus/internal/db/generated"
	"github.com/blakestevenson/nimbus/internal/media"

	"github.com/jackc/pgx/v5/pgtype"
	"go.uber.org/zap"
)

// =============================================================================
// Service - Scanner service for upserting media items
// =============================================================================
// This service contains the business logic for inserting or updating media
// items based on parsed filename information. It handles:
//   - Movie imports
//   - TV series/season/episode hierarchy
//   - Music artist/album/track hierarchy
//   - Book imports
//   - Media file tracking
//   - Media relation management
// =============================================================================

type Service struct {
	queries     *generated.Queries
	logger      *zap.Logger
	tmdbBaseURL string
	enableTMDB  bool
}

// NewService creates a new scanner service
func NewService(queries *generated.Queries, logger *zap.Logger) *Service {
	return &Service{
		queries:     queries,
		logger:      logger,
		tmdbBaseURL: "http://localhost:8080/api/plugins/tmdb/enrich",
		enableTMDB:  true, // Can be configured later
	}
}

// =============================================================================
// UpsertMovie - Create or update a movie media item
// =============================================================================
// Strategy:
//   1. Use UpsertMediaItem to insert/update based on (title, year, kind)
//   2. Upsert the media_files entry linking file path to media item
//   3. Return the media item ID and whether it was created (true) or updated (false)
//
// Returns:
//   - itemID: The media item database ID
//   - created: true if new item was created, false if existing was updated
//   - error: Any error during upsert
// =============================================================================

func (s *Service) UpsertMovie(ctx context.Context, parsed *ParsedMedia, filePath string, fileSize int64) (itemID int64, created bool, err error) {
	// Generate sort title (removes articles like "The", "A", "An")
	sortTitle := generateSortTitle(parsed.Title)

	// Prepare metadata
	metadata := map[string]interface{}{
		"source": "scanner",
	}
	metadataJSON, _ := json.Marshal(metadata)

	// Prepare year (handle 0 as NULL)
	var year *int32
	if parsed.Year > 0 {
		y := int32(parsed.Year)
		year = &y
	}

	// Upsert media item
	item, err := s.queries.UpsertMediaItem(ctx, generated.UpsertMediaItemParams{
		Kind:        string(media.MediaKindMovie),
		Title:       parsed.Title,
		SortTitle:   sortTitle,
		Year:        year,
		ExternalIds: []byte("{}"),
		Metadata:    metadataJSON,
		ParentID:    nil,
	})
	if err != nil {
		return 0, false, fmt.Errorf("failed to upsert movie: %w", err)
	}

	// Check if this is a new item (created recently)
	created = item.CreatedAt.Time.Equal(item.UpdatedAt.Time)

	// Upsert media file
	if err := s.upsertMediaFile(ctx, item.ID, filePath, fileSize); err != nil {
		return item.ID, created, fmt.Errorf("failed to upsert media file: %w", err)
	}

	// Enrich with TMDB metadata (best effort, don't fail on errors)
	go s.enrichWithTMDB(context.Background(), item.ID, parsed)

	return item.ID, created, nil
}

// =============================================================================
// UpsertTVEpisode - Create or update a TV episode and its hierarchy
// =============================================================================
// Strategy:
//   1. Ensure TV series exists (create if needed)
//   2. Ensure TV season exists under series (create if needed)
//   3. Upsert the episode under the season
//   4. Create media_relations for series->season and season->episode
//   5. Upsert the media_files entry
//
// This creates the full hierarchy: Series -> Season -> Episode
//
// Returns:
//   - itemID: The episode's media item database ID
//   - created: true if new episode was created
//   - error: Any error during upsert
// =============================================================================

func (s *Service) UpsertTVEpisode(ctx context.Context, parsed *ParsedMedia, filePath string, fileSize int64) (itemID int64, created bool, err error) {
	// Step 1: Ensure TV series exists
	seriesID, err := s.ensureTVSeries(ctx, parsed.Title, parsed.Year)
	if err != nil {
		return 0, false, fmt.Errorf("failed to ensure TV series: %w", err)
	}

	// Step 2: Ensure TV season exists
	seasonID, err := s.ensureTVSeason(ctx, seriesID, parsed.Season, parsed.Title)
	if err != nil {
		return 0, false, fmt.Errorf("failed to ensure TV season: %w", err)
	}

	// Step 3: Upsert the episode
	// Use episode title if available, otherwise fall back to S01E02 format
	episodeTitle := parsed.EpisodeTitle
	if episodeTitle == "" {
		episodeTitle = fmt.Sprintf("S%02dE%02d", parsed.Season, parsed.Episode)
	}
	sortTitle := episodeTitle

	metadata := map[string]interface{}{
		"source":  "scanner",
		"season":  parsed.Season,
		"episode": parsed.Episode,
	}
	if parsed.EpisodeTitle != "" {
		metadata["episode_title"] = parsed.EpisodeTitle
	}
	metadataJSON, _ := json.Marshal(metadata)

	item, err := s.queries.UpsertMediaItem(ctx, generated.UpsertMediaItemParams{
		Kind:        string(media.MediaKindTVEpisode),
		Title:       episodeTitle,
		SortTitle:   sortTitle,
		Year:        nil,
		ExternalIds: []byte("{}"),
		Metadata:    metadataJSON,
		ParentID:    &seasonID,
	})
	if err != nil {
		return 0, false, fmt.Errorf("failed to upsert episode: %w", err)
	}

	created = item.CreatedAt.Time.Equal(item.UpdatedAt.Time)

	// Step 4: Create media relation (season -> episode)
	if err := s.upsertMediaRelation(ctx, seasonID, item.ID, "season-episode", float64(parsed.Episode)); err != nil {
		s.logger.Warn("failed to upsert media relation", zap.Error(err))
	}

	// Step 5: Upsert media file
	if err := s.upsertMediaFile(ctx, item.ID, filePath, fileSize); err != nil {
		return item.ID, created, fmt.Errorf("failed to upsert media file: %w", err)
	}

	// Enrich with TMDB metadata (best effort, don't fail on errors)
	go s.enrichWithTMDB(context.Background(), item.ID, parsed)

	return item.ID, created, nil
}

// =============================================================================
// UpsertMusicTrack - Create or update a music track and its hierarchy
// =============================================================================
// Strategy:
//   1. Ensure music artist exists (create if needed)
//   2. Ensure music album exists under artist (create if needed)
//   3. Upsert the track under the album
//   4. Create media_relations for artist->album and album->track
//   5. Upsert the media_files entry
//
// This creates the full hierarchy: Artist -> Album -> Track
// =============================================================================

func (s *Service) UpsertMusicTrack(ctx context.Context, parsed *ParsedMedia, filePath string, fileSize int64) (itemID int64, created bool, err error) {
	// Step 1: Ensure artist exists
	artistID, err := s.ensureMusicArtist(ctx, parsed.Artist)
	if err != nil {
		return 0, false, fmt.Errorf("failed to ensure artist: %w", err)
	}

	// Step 2: Ensure album exists
	albumID, err := s.ensureMusicAlbum(ctx, artistID, parsed.Album)
	if err != nil {
		return 0, false, fmt.Errorf("failed to ensure album: %w", err)
	}

	// Step 3: Upsert the track
	sortTitle := generateSortTitle(parsed.Title)

	metadata := map[string]interface{}{
		"source":       "scanner",
		"track_number": parsed.Track,
		"artist":       parsed.Artist,
		"album":        parsed.Album,
	}
	metadataJSON, _ := json.Marshal(metadata)

	item, err := s.queries.UpsertMediaItem(ctx, generated.UpsertMediaItemParams{
		Kind:        string(media.MediaKindMusicTrack),
		Title:       parsed.Title,
		SortTitle:   sortTitle,
		Year:        nil,
		ExternalIds: []byte("{}"),
		Metadata:    metadataJSON,
		ParentID:    &albumID,
	})
	if err != nil {
		return 0, false, fmt.Errorf("failed to upsert track: %w", err)
	}

	created = item.CreatedAt.Time.Equal(item.UpdatedAt.Time)

	// Step 4: Create media relation (album -> track)
	if err := s.upsertMediaRelation(ctx, albumID, item.ID, "album-track", float64(parsed.Track)); err != nil {
		s.logger.Warn("failed to upsert media relation", zap.Error(err))
	}

	// Step 5: Upsert media file
	if err := s.upsertMediaFile(ctx, item.ID, filePath, fileSize); err != nil {
		return item.ID, created, fmt.Errorf("failed to upsert media file: %w", err)
	}

	return item.ID, created, nil
}

// =============================================================================
// UpsertBook - Create or update a book media item
// =============================================================================

func (s *Service) UpsertBook(ctx context.Context, parsed *ParsedMedia, filePath string, fileSize int64) (itemID int64, created bool, err error) {
	sortTitle := generateSortTitle(parsed.Title)

	metadata := map[string]interface{}{
		"source": "scanner",
	}
	if parsed.Author != "" {
		metadata["author"] = parsed.Author
	}
	metadataJSON, _ := json.Marshal(metadata)

	var year *int32
	if parsed.Year > 0 {
		y := int32(parsed.Year)
		year = &y
	}

	item, err := s.queries.UpsertMediaItem(ctx, generated.UpsertMediaItemParams{
		Kind:        string(media.MediaKindBook),
		Title:       parsed.Title,
		SortTitle:   sortTitle,
		Year:        year,
		ExternalIds: []byte("{}"),
		Metadata:    metadataJSON,
		ParentID:    nil,
	})
	if err != nil {
		return 0, false, fmt.Errorf("failed to upsert book: %w", err)
	}

	created = item.CreatedAt.Time.Equal(item.UpdatedAt.Time)

	if err := s.upsertMediaFile(ctx, item.ID, filePath, fileSize); err != nil {
		return item.ID, created, fmt.Errorf("failed to upsert media file: %w", err)
	}

	return item.ID, created, nil
}

// =============================================================================
// Helper Functions - Ensure hierarchy items exist
// =============================================================================

func (s *Service) ensureTVSeries(ctx context.Context, title string, year int) (int64, error) {
	sortTitle := generateSortTitle(title)

	metadata := map[string]interface{}{
		"source": "scanner",
	}
	metadataJSON, _ := json.Marshal(metadata)

	var yearPtr *int32
	if year > 0 {
		y := int32(year)
		yearPtr = &y
	}

	item, err := s.queries.UpsertMediaItem(ctx, generated.UpsertMediaItemParams{
		Kind:        string(media.MediaKindTVSeries),
		Title:       title,
		SortTitle:   sortTitle,
		Year:        yearPtr,
		ExternalIds: []byte("{}"),
		Metadata:    metadataJSON,
		ParentID:    nil,
	})
	if err != nil {
		return 0, err
	}

	// Enrich with TMDB metadata (best effort, don't fail on errors)
	parsed := &ParsedMedia{
		Kind:  "tv_series",
		Title: title,
		Year:  year,
	}
	go s.enrichWithTMDB(context.Background(), item.ID, parsed)

	return item.ID, nil
}

func (s *Service) ensureTVSeason(ctx context.Context, seriesID int64, seasonNumber int, seriesTitle string) (int64, error) {
	seasonTitle := fmt.Sprintf("Season %d", seasonNumber)

	metadata := map[string]interface{}{
		"source":        "scanner",
		"season_number": seasonNumber,
	}
	metadataJSON, _ := json.Marshal(metadata)

	item, err := s.queries.UpsertMediaItem(ctx, generated.UpsertMediaItemParams{
		Kind:        string(media.MediaKindTVSeason),
		Title:       seasonTitle,
		SortTitle:   seasonTitle,
		Year:        nil,
		ExternalIds: []byte("{}"),
		Metadata:    metadataJSON,
		ParentID:    &seriesID,
	})
	if err != nil {
		return 0, err
	}

	// Enrich with TMDB metadata (best effort, don't fail on errors)
	parsed := &ParsedMedia{
		Kind:   "tv_season",
		Title:  seriesTitle, // Use series title for TMDB search
		Season: seasonNumber,
	}
	go s.enrichWithTMDB(context.Background(), item.ID, parsed)

	// Create series -> season relation
	if err := s.upsertMediaRelation(ctx, seriesID, item.ID, "series-season", float64(seasonNumber)); err != nil {
		s.logger.Warn("failed to create series-season relation", zap.Error(err))
	}

	return item.ID, nil
}

func (s *Service) ensureMusicArtist(ctx context.Context, artistName string) (int64, error) {
	if artistName == "" {
		artistName = "Unknown Artist"
	}

	sortTitle := generateSortTitle(artistName)

	metadata := map[string]interface{}{
		"source": "scanner",
	}
	metadataJSON, _ := json.Marshal(metadata)

	item, err := s.queries.UpsertMediaItem(ctx, generated.UpsertMediaItemParams{
		Kind:        string(media.MediaKindMusicArtist),
		Title:       artistName,
		SortTitle:   sortTitle,
		Year:        nil,
		ExternalIds: []byte("{}"),
		Metadata:    metadataJSON,
		ParentID:    nil,
	})
	if err != nil {
		return 0, err
	}

	return item.ID, nil
}

func (s *Service) ensureMusicAlbum(ctx context.Context, artistID int64, albumName string) (int64, error) {
	if albumName == "" {
		albumName = "Unknown Album"
	}

	sortTitle := generateSortTitle(albumName)

	metadata := map[string]interface{}{
		"source": "scanner",
	}
	metadataJSON, _ := json.Marshal(metadata)

	item, err := s.queries.UpsertMediaItem(ctx, generated.UpsertMediaItemParams{
		Kind:        string(media.MediaKindMusicAlbum),
		Title:       albumName,
		SortTitle:   sortTitle,
		Year:        nil,
		ExternalIds: []byte("{}"),
		Metadata:    metadataJSON,
		ParentID:    &artistID,
	})
	if err != nil {
		return 0, err
	}

	// Create artist -> album relation
	if err := s.upsertMediaRelation(ctx, artistID, item.ID, "artist-album", 0); err != nil {
		s.logger.Warn("failed to create artist-album relation", zap.Error(err))
	}

	return item.ID, nil
}

// =============================================================================
// Upsert Media File - Link a file path to a media item
// =============================================================================

func (s *Service) upsertMediaFile(ctx context.Context, mediaItemID int64, filePath string, fileSize int64) error {
	_, err := s.queries.UpsertMediaFile(ctx, generated.UpsertMediaFileParams{
		MediaItemID: &mediaItemID,
		Path:        filePath,
		Size:        &fileSize,
		Hash:        nil, // TODO: Implement file hashing if needed
	})
	return err
}

// =============================================================================
// Upsert Media Relation - Create parent-child relationships
// =============================================================================

func (s *Service) upsertMediaRelation(ctx context.Context, parentID, childID int64, relationType string, sortIndex float64) error {
	// Convert sortIndex to pgtype.Numeric
	var sortIndexNumeric pgtype.Numeric
	if sortIndex > 0 {
		// Convert float to numeric representation
		// For simple integer sort indices, we can use a basic conversion
		if err := sortIndexNumeric.Scan(sortIndex); err != nil {
			// If scan fails, set as invalid (NULL)
			sortIndexNumeric.Valid = false
		}
	}

	_, err := s.queries.UpsertMediaRelation(ctx, generated.UpsertMediaRelationParams{
		ParentID:  parentID,
		ChildID:   childID,
		Relation:  relationType,
		SortIndex: sortIndexNumeric,
		Metadata:  []byte("{}"),
	})
	return err
}

// =============================================================================
// generateSortTitle - Remove articles and normalize for sorting
// =============================================================================
// Removes leading articles ("The", "A", "An") and normalizes spacing.
//
// Examples:
//   "The Dark Knight" -> "Dark Knight"
//   "A Beautiful Mind" -> "Beautiful Mind"
//   "An American Tail" -> "American Tail"
// =============================================================================

func generateSortTitle(title string) string {
	lower := strings.ToLower(strings.TrimSpace(title))

	// Remove leading articles
	articles := []string{"the ", "a ", "an "}
	for _, article := range articles {
		if strings.HasPrefix(lower, article) {
			title = strings.TrimSpace(title[len(article):])
			break
		}
	}

	return title
}

// =============================================================================
// enrichWithTMDB - Fetch and store TMDB metadata for a media item
// =============================================================================
// This method calls the TMDB plugin to fetch metadata and updates the media
// item's metadata column with poster URLs, descriptions, ratings, etc.
//
// Note: This is a best-effort operation. Failures are logged but don't fail
// the overall scan.
// =============================================================================

func (s *Service) enrichWithTMDB(ctx context.Context, itemID int64, parsed *ParsedMedia) {
	if !s.enableTMDB {
		return
	}

	// Build request payload
	payload := map[string]interface{}{
		"title": parsed.Title,
		"kind":  parsed.Kind,
	}

	if parsed.Year > 0 {
		payload["year"] = parsed.Year
	}

	if parsed.Kind == "tv_season" {
		payload["season"] = parsed.Season
	}

	if parsed.Kind == "tv_episode" {
		payload["season"] = parsed.Season
		payload["episode"] = parsed.Episode
	}

	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		s.logger.Warn("Failed to marshal TMDB request", zap.Error(err))
		return
	}

	// Make HTTP request to TMDB plugin
	req, err := http.NewRequestWithContext(ctx, "POST", s.tmdbBaseURL, bytes.NewReader(payloadJSON))
	if err != nil {
		s.logger.Warn("Failed to create TMDB request", zap.Error(err))
		return
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		s.logger.Warn("Failed to call TMDB plugin", zap.Error(err))
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		s.logger.Warn("TMDB plugin returned error",
			zap.Int("status", resp.StatusCode),
			zap.String("body", string(body)))
		return
	}

	// Parse response
	var tmdbResp struct {
		Metadata map[string]interface{} `json:"metadata"`
		Success  bool                   `json:"success"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&tmdbResp); err != nil {
		s.logger.Warn("Failed to decode TMDB response", zap.Error(err))
		return
	}

	if !tmdbResp.Success || len(tmdbResp.Metadata) == 0 {
		s.logger.Debug("No TMDB metadata returned", zap.Int64("item_id", itemID))
		return
	}

	// Update media item with TMDB metadata
	metadataJSON, err := json.Marshal(tmdbResp.Metadata)
	if err != nil {
		s.logger.Warn("Failed to marshal TMDB metadata", zap.Error(err))
		return
	}

	// Merge with existing metadata using JSONB concat operator
	_, err = s.queries.UpdateMediaMetadata(ctx, generated.UpdateMediaMetadataParams{
		ID:       itemID,
		Metadata: metadataJSON,
	})

	if err != nil {
		s.logger.Warn("Failed to update media metadata",
			zap.Int64("item_id", itemID),
			zap.Error(err))
		return
	}

	// Update external_ids with TMDB ID if available
	if tmdbID, ok := tmdbResp.Metadata["tmdb_id"].(string); ok && tmdbID != "" {
		externalIDs := map[string]interface{}{
			"tmdb": tmdbID,
		}
		externalIDsJSON, err := json.Marshal(externalIDs)
		if err == nil {
			_, err = s.queries.UpdateMediaExternalIDs(ctx, generated.UpdateMediaExternalIDsParams{
				ID:          itemID,
				ExternalIds: externalIDsJSON,
			})
			if err != nil {
				s.logger.Warn("Failed to update external IDs",
					zap.Int64("item_id", itemID),
					zap.Error(err))
			}
		}
	}

	s.logger.Debug("Successfully enriched media with TMDB data",
		zap.Int64("item_id", itemID),
		zap.String("title", parsed.Title))
}
