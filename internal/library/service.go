package library

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

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
	queries *generated.Queries
	logger  *zap.Logger
}

// NewService creates a new scanner service
func NewService(queries *generated.Queries, logger *zap.Logger) *Service {
	return &Service{
		queries: queries,
		logger:  logger,
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
	seasonID, err := s.ensureTVSeason(ctx, seriesID, parsed.Season)
	if err != nil {
		return 0, false, fmt.Errorf("failed to ensure TV season: %w", err)
	}

	// Step 3: Upsert the episode
	episodeTitle := fmt.Sprintf("S%02dE%02d", parsed.Season, parsed.Episode)
	sortTitle := episodeTitle

	metadata := map[string]interface{}{
		"source":  "scanner",
		"season":  parsed.Season,
		"episode": parsed.Episode,
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

	return item.ID, nil
}

func (s *Service) ensureTVSeason(ctx context.Context, seriesID int64, seasonNumber int) (int64, error) {
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
