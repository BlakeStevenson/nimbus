package plugins

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/blakestevenson/nimbus/internal/configstore"
	"github.com/blakestevenson/nimbus/internal/db/generated"
	"go.uber.org/zap"
)

// SDK provides plugins with access to core Nimbus functionality
// This allows plugins to interact with the database, configuration, and other services
type SDK struct {
	queries     *generated.Queries
	configStore *configstore.Store
	logger      *zap.Logger
}

// NewSDK creates a new SDK instance for plugin use
func NewSDK(queries *generated.Queries, configStore *configstore.Store, logger *zap.Logger) *SDK {
	return &SDK{
		queries:     queries,
		configStore: configStore,
		logger:      logger.With(zap.String("component", "plugin-sdk")),
	}
}

// ============================================================================
// Configuration Methods
// ============================================================================

// ConfigGet retrieves a configuration value by key
// The value is returned as a JSON-decoded interface{}
func (sdk *SDK) ConfigGet(ctx context.Context, key string) (interface{}, error) {
	value, err := sdk.configStore.Get(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("failed to get config key %s: %w", key, err)
	}

	var result interface{}
	if err := json.Unmarshal(value, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config value: %w", err)
	}

	return result, nil
}

// ConfigGetString is a convenience method to get a string config value
func (sdk *SDK) ConfigGetString(ctx context.Context, key string) (string, error) {
	val, err := sdk.ConfigGet(ctx, key)
	if err != nil {
		return "", err
	}

	str, ok := val.(string)
	if !ok {
		return "", fmt.Errorf("config value for key %s is not a string", key)
	}

	return str, nil
}

// ConfigSet stores a configuration value
// The value is JSON-encoded before storage by configStore
func (sdk *SDK) ConfigSet(ctx context.Context, key string, value interface{}) error {
	// Pass the raw value to configStore.Set, which will handle JSON marshaling
	// (Previously this was double-encoding by marshaling here AND in configStore.Set)
	if err := sdk.configStore.Set(ctx, key, value); err != nil {
		return fmt.Errorf("failed to set config key %s: %w", key, err)
	}

	return nil
}

// ConfigDelete removes a configuration key
func (sdk *SDK) ConfigDelete(ctx context.Context, key string) error {
	if err := sdk.configStore.Delete(ctx, key); err != nil {
		return fmt.Errorf("failed to delete config key %s: %w", key, err)
	}

	return nil
}

// ============================================================================
// Media Methods
// ============================================================================

// FindMediaByID retrieves a media item by its ID
func (sdk *SDK) FindMediaByID(ctx context.Context, id int64) (*MediaItem, error) {
	dbMedia, err := sdk.queries.GetMediaItem(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get media item %d: %w", id, err)
	}

	return sdk.convertDBMediaToMediaItem(dbMedia), nil
}

// ListMediaByKind retrieves all media items of a specific kind (e.g., "movie", "tv-series")
func (sdk *SDK) ListMediaByKind(ctx context.Context, kind string) ([]*MediaItem, error) {
	dbMediaList, err := sdk.queries.ListMediaItemsByKind(ctx, generated.ListMediaItemsByKindParams{
		Kind:   kind,
		Limit:  1000, // Default limit
		Offset: 0,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list media items by kind %s: %w", kind, err)
	}

	items := make([]*MediaItem, len(dbMediaList))
	for i, dbMedia := range dbMediaList {
		items[i] = sdk.convertDBMediaToMediaItem(dbMedia)
	}

	return items, nil
}

// CreateMediaItem creates a new media item
func (sdk *SDK) CreateMediaItem(ctx context.Context, item *MediaItem) (*MediaItem, error) {
	metadata, err := json.Marshal(item.Metadata)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal metadata: %w", err)
	}

	params := generated.CreateMediaItemParams{
		Kind:      item.Kind,
		Title:     item.Title,
		SortTitle: item.Title, // Use title as sort_title by default
		Metadata:  metadata,
	}

	if item.Year != nil {
		params.Year = item.Year
	}

	if item.ParentID != nil {
		params.ParentID = item.ParentID
	}

	dbMedia, err := sdk.queries.CreateMediaItem(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("failed to create media item: %w", err)
	}

	return sdk.convertDBMediaToMediaItem(dbMedia), nil
}

// UpdateMediaItem updates an existing media item
func (sdk *SDK) UpdateMediaItem(ctx context.Context, item *MediaItem) (*MediaItem, error) {
	metadata, err := json.Marshal(item.Metadata)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal metadata: %w", err)
	}

	params := generated.UpdateMediaItemParams{
		ID:        item.ID,
		Title:     &item.Title,
		SortTitle: &item.Title,
		Metadata:  metadata,
	}

	if item.Year != nil {
		params.Year = item.Year
	}

	if item.ParentID != nil {
		params.ParentID = item.ParentID
	}

	dbMedia, err := sdk.queries.UpdateMediaItem(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("failed to update media item: %w", err)
	}

	return sdk.convertDBMediaToMediaItem(dbMedia), nil
}

// DeleteMediaItem deletes a media item by ID
func (sdk *SDK) DeleteMediaItem(ctx context.Context, id int64) error {
	if err := sdk.queries.DeleteMediaItem(ctx, id); err != nil {
		return fmt.Errorf("failed to delete media item %d: %w", id, err)
	}

	return nil
}

// ============================================================================
// Logging Methods
// ============================================================================

// Log provides access to a logger for the plugin
func (sdk *SDK) Log() *zap.Logger {
	return sdk.logger
}

// LogInfo logs an info message
func (sdk *SDK) LogInfo(msg string, fields ...zap.Field) {
	sdk.logger.Info(msg, fields...)
}

// LogError logs an error message
func (sdk *SDK) LogError(msg string, fields ...zap.Field) {
	sdk.logger.Error(msg, fields...)
}

// LogWarn logs a warning message
func (sdk *SDK) LogWarn(msg string, fields ...zap.Field) {
	sdk.logger.Warn(msg, fields...)
}

// LogDebug logs a debug message
func (sdk *SDK) LogDebug(msg string, fields ...zap.Field) {
	sdk.logger.Debug(msg, fields...)
}

// ============================================================================
// Helper Methods
// ============================================================================

func (sdk *SDK) convertDBMediaToMediaItem(dbMedia generated.MediaItem) *MediaItem {
	var metadata map[string]interface{}
	if len(dbMedia.Metadata) > 0 {
		_ = json.Unmarshal(dbMedia.Metadata, &metadata)
	}

	return &MediaItem{
		ID:        dbMedia.ID,
		Kind:      dbMedia.Kind,
		Title:     dbMedia.Title,
		Year:      dbMedia.Year,
		Metadata:  metadata,
		ParentID:  dbMedia.ParentID,
		CreatedAt: dbMedia.CreatedAt.Time,
		UpdatedAt: dbMedia.UpdatedAt.Time,
	}
}
