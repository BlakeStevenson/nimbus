package media

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/blakestevenson/nimbus/internal/db/generated"
	"github.com/jackc/pgx/v5"
	"go.uber.org/zap"
)

// Service defines the interface for media operations
type Service interface {
	CreateMediaItem(ctx context.Context, params CreateMediaParams) (*MediaItem, error)
	GetMediaItem(ctx context.Context, id int64) (*MediaItem, error)
	ListMediaItems(ctx context.Context, filter MediaFilter) (*MediaList, error)
	UpdateMediaItem(ctx context.Context, id int64, params UpdateMediaParams) (*MediaItem, error)
	DeleteMediaItem(ctx context.Context, id int64) error
	ListChildItems(ctx context.Context, parentID int64) ([]*MediaItem, error)
}

// service implements the Service interface
type service struct {
	queries *generated.Queries
	logger  *zap.Logger
}

// NewService creates a new media service
func NewService(queries *generated.Queries, logger *zap.Logger) Service {
	return &service{
		queries: queries,
		logger:  logger,
	}
}

// CreateMediaItem creates a new media item
func (s *service) CreateMediaItem(ctx context.Context, params CreateMediaParams) (*MediaItem, error) {
	if err := params.Validate(); err != nil {
		return nil, err
	}

	// Generate sort_title if not provided
	sortTitle := params.SortTitle
	if sortTitle == "" {
		sortTitle = generateSortTitle(params.Title)
	}

	// Marshal external IDs and metadata
	externalIDs, err := MarshalMap(params.ExternalIDs)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal external IDs: %w", err)
	}

	metadata, err := MarshalMap(params.Metadata)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal metadata: %w", err)
	}

	// Create the media item
	dbItem, err := s.queries.CreateMediaItem(ctx, generated.CreateMediaItemParams{
		Kind:        string(params.Kind),
		Title:       params.Title,
		SortTitle:   sortTitle,
		Year:        params.Year,
		ExternalIds: externalIDs,
		Metadata:    metadata,
		ParentID:    params.ParentID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create media item: %w", err)
	}

	return dbItemToMediaItem(dbItem)
}

// GetMediaItem retrieves a media item by ID
func (s *service) GetMediaItem(ctx context.Context, id int64) (*MediaItem, error) {
	dbItem, err := s.queries.GetMediaItem(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("failed to get media item: %w", err)
	}

	return dbItemToMediaItem(dbItem)
}

// ListMediaItems lists media items with filtering and pagination
func (s *service) ListMediaItems(ctx context.Context, filter MediaFilter) (*MediaList, error) {
	if err := filter.Validate(); err != nil {
		return nil, err
	}

	// Build query parameters
	var kindStr *string
	if filter.Kind != nil {
		k := string(*filter.Kind)
		kindStr = &k
	}

	// Get total count
	count, err := s.queries.CountMediaItems(ctx, generated.CountMediaItemsParams{
		Kind:     kindStr,
		ParentID: filter.ParentID,
		Search:   filter.Search,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to count media items: %w", err)
	}

	// Get items
	dbItems, err := s.queries.ListMediaItems(ctx, generated.ListMediaItemsParams{
		Kind:     kindStr,
		ParentID: filter.ParentID,
		Search:   filter.Search,
		Offset:   &filter.Offset,
		Limit:    &filter.Limit,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list media items: %w", err)
	}

	// Convert to domain models
	items := make([]*MediaItem, 0, len(dbItems))
	for _, dbItem := range dbItems {
		item, err := dbItemToMediaItem(dbItem)
		if err != nil {
			s.logger.Error("failed to convert media item", zap.Error(err), zap.Int64("id", dbItem.ID))
			continue
		}
		items = append(items, item)
	}

	return &MediaList{
		Items:   items,
		Total:   count,
		Limit:   filter.Limit,
		Offset:  filter.Offset,
		HasMore: filter.Offset+filter.Limit < int32(count),
	}, nil
}

// UpdateMediaItem updates a media item
func (s *service) UpdateMediaItem(ctx context.Context, id int64, params UpdateMediaParams) (*MediaItem, error) {
	if err := params.Validate(); err != nil {
		return nil, err
	}

	// Check if item exists
	_, err := s.queries.GetMediaItem(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("failed to get media item: %w", err)
	}

	// Marshal external IDs and metadata if provided
	var externalIDs []byte
	if params.ExternalIDs != nil {
		externalIDs, err = MarshalMap(params.ExternalIDs)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal external IDs: %w", err)
		}
	}

	var metadata []byte
	if params.Metadata != nil {
		metadata, err = MarshalMap(params.Metadata)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal metadata: %w", err)
		}
	}

	// Update the media item
	dbItem, err := s.queries.UpdateMediaItem(ctx, generated.UpdateMediaItemParams{
		ID:          id,
		Title:       params.Title,
		SortTitle:   params.SortTitle,
		Year:        params.Year,
		ExternalIds: externalIDs,
		Metadata:    metadata,
		ParentID:    params.ParentID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to update media item: %w", err)
	}

	return dbItemToMediaItem(dbItem)
}

// DeleteMediaItem deletes a media item
func (s *service) DeleteMediaItem(ctx context.Context, id int64) error {
	// Check if item exists
	_, err := s.queries.GetMediaItem(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotFound
		}
		return fmt.Errorf("failed to get media item: %w", err)
	}

	if err := s.queries.DeleteMediaItem(ctx, id); err != nil {
		return fmt.Errorf("failed to delete media item: %w", err)
	}

	return nil
}

// ListChildItems lists all child items of a parent
func (s *service) ListChildItems(ctx context.Context, parentID int64) ([]*MediaItem, error) {
	dbItems, err := s.queries.ListChildMediaItems(ctx, &parentID)
	if err != nil {
		return nil, fmt.Errorf("failed to list child media items: %w", err)
	}

	items := make([]*MediaItem, 0, len(dbItems))
	for _, dbItem := range dbItems {
		item, err := dbItemToMediaItem(dbItem)
		if err != nil {
			s.logger.Error("failed to convert media item", zap.Error(err), zap.Int64("id", dbItem.ID))
			continue
		}
		items = append(items, item)
	}

	return items, nil
}

// Helper functions

func dbItemToMediaItem(dbItem generated.MediaItem) (*MediaItem, error) {
	externalIDs, err := UnmarshalMap(dbItem.ExternalIds)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal external IDs: %w", err)
	}

	metadata, err := UnmarshalMap(dbItem.Metadata)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
	}

	return &MediaItem{
		ID:          dbItem.ID,
		Kind:        MediaKind(dbItem.Kind),
		Title:       dbItem.Title,
		SortTitle:   dbItem.SortTitle,
		Year:        dbItem.Year,
		ExternalIDs: externalIDs,
		Metadata:    metadata,
		ParentID:    dbItem.ParentID,
		CreatedAt:   dbItem.CreatedAt.Time,
		UpdatedAt:   dbItem.UpdatedAt.Time,
	}, nil
}

func generateSortTitle(title string) string {
	lower := strings.ToLower(title)
	// Remove common articles
	for _, article := range []string{"the ", "a ", "an "} {
		if strings.HasPrefix(lower, article) {
			return title[len(article):]
		}
	}
	return title
}
