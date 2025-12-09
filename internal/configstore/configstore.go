package configstore

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/blakestevenson/nimbus/internal/db/generated"
)

// Store provides type-safe access to the config table
type Store struct {
	queries *generated.Queries
}

// New creates a new config store
func New(queries *generated.Queries) *Store {
	return &Store{
		queries: queries,
	}
}

// Get retrieves a configuration value as raw JSON
func (s *Store) Get(ctx context.Context, key string) (json.RawMessage, error) {
	cfg, err := s.queries.GetConfig(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("failed to get config %s: %w", key, err)
	}
	return cfg.Value, nil
}

// GetWithMetadata retrieves a configuration entry including its metadata
func (s *Store) GetWithMetadata(ctx context.Context, key string) (generated.Config, error) {
	cfg, err := s.queries.GetConfig(ctx, key)
	if err != nil {
		return generated.Config{}, fmt.Errorf("failed to get config %s: %w", key, err)
	}
	return cfg, nil
}

// Set stores a configuration value
func (s *Store) Set(ctx context.Context, key string, value any) error {
	jsonValue, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("failed to marshal config value: %w", err)
	}

	_, err = s.queries.SetConfig(ctx, generated.SetConfigParams{
		Key:     key,
		Value:   jsonValue,
		Column3: nil, // Don't update metadata when setting value
	})
	if err != nil {
		return fmt.Errorf("failed to set config %s: %w", key, err)
	}

	return nil
}

// SetWithMetadata stores a configuration value along with its metadata
func (s *Store) SetWithMetadata(ctx context.Context, key string, value any, metadata map[string]any) error {
	jsonValue, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("failed to marshal config value: %w", err)
	}

	var jsonMetadata []byte
	if metadata != nil {
		jsonMetadata, err = json.Marshal(metadata)
		if err != nil {
			return fmt.Errorf("failed to marshal config metadata: %w", err)
		}
	}

	_, err = s.queries.SetConfig(ctx, generated.SetConfigParams{
		Key:     key,
		Value:   jsonValue,
		Column3: jsonMetadata,
	})
	if err != nil {
		return fmt.Errorf("failed to set config %s: %w", key, err)
	}

	return nil
}

// Delete removes a configuration value
func (s *Store) Delete(ctx context.Context, key string) error {
	if err := s.queries.DeleteConfig(ctx, key); err != nil {
		return fmt.Errorf("failed to delete config %s: %w", key, err)
	}
	return nil
}

// GetString retrieves a string configuration value
func (s *Store) GetString(ctx context.Context, key string) (string, error) {
	raw, err := s.Get(ctx, key)
	if err != nil {
		return "", err
	}

	var value string
	if err := json.Unmarshal(raw, &value); err != nil {
		return "", fmt.Errorf("failed to unmarshal string config %s: %w", key, err)
	}

	return value, nil
}

// GetInt retrieves an integer configuration value
func (s *Store) GetInt(ctx context.Context, key string) (int, error) {
	raw, err := s.Get(ctx, key)
	if err != nil {
		return 0, err
	}

	var value int
	if err := json.Unmarshal(raw, &value); err != nil {
		return 0, fmt.Errorf("failed to unmarshal int config %s: %w", key, err)
	}

	return value, nil
}

// GetBool retrieves a boolean configuration value
func (s *Store) GetBool(ctx context.Context, key string) (bool, error) {
	raw, err := s.Get(ctx, key)
	if err != nil {
		return false, err
	}

	var value bool
	if err := json.Unmarshal(raw, &value); err != nil {
		return false, fmt.Errorf("failed to unmarshal bool config %s: %w", key, err)
	}

	return value, nil
}

// GetMap retrieves a map configuration value
func (s *Store) GetMap(ctx context.Context, key string) (map[string]any, error) {
	raw, err := s.Get(ctx, key)
	if err != nil {
		return nil, err
	}

	var value map[string]any
	if err := json.Unmarshal(raw, &value); err != nil {
		return nil, fmt.Errorf("failed to unmarshal map config %s: %w", key, err)
	}

	return value, nil
}

// GetAll retrieves all configuration values
func (s *Store) GetAll(ctx context.Context) (map[string]json.RawMessage, error) {
	configs, err := s.queries.GetAllConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get all config: %w", err)
	}

	result := make(map[string]json.RawMessage, len(configs))
	for _, cfg := range configs {
		result[cfg.Key] = cfg.Value
	}

	return result, nil
}

// GetAllWithMetadata retrieves all configuration entries including metadata
func (s *Store) GetAllWithMetadata(ctx context.Context) ([]generated.Config, error) {
	configs, err := s.queries.GetAllConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get all config: %w", err)
	}
	return configs, nil
}

// GetByPrefix retrieves all configuration values with a given prefix
func (s *Store) GetByPrefix(ctx context.Context, prefix string) (map[string]json.RawMessage, error) {
	configs, err := s.queries.GetConfigByPrefix(ctx, &prefix)
	if err != nil {
		return nil, fmt.Errorf("failed to get config by prefix %s: %w", prefix, err)
	}

	result := make(map[string]json.RawMessage, len(configs))
	for _, cfg := range configs {
		result[cfg.Key] = cfg.Value
	}

	return result, nil
}

// SetString stores a string configuration value
func (s *Store) SetString(ctx context.Context, key, value string) error {
	return s.Set(ctx, key, value)
}

// SetInt stores an integer configuration value
func (s *Store) SetInt(ctx context.Context, key string, value int) error {
	return s.Set(ctx, key, value)
}

// SetBool stores a boolean configuration value
func (s *Store) SetBool(ctx context.Context, key string, value bool) error {
	return s.Set(ctx, key, value)
}

// SetMap stores a map configuration value
func (s *Store) SetMap(ctx context.Context, key string, value map[string]any) error {
	return s.Set(ctx, key, value)
}

// GetOrDefault retrieves a string value or returns a default
func (s *Store) GetOrDefault(ctx context.Context, key, defaultValue string) string {
	value, err := s.GetString(ctx, key)
	if err != nil {
		return defaultValue
	}
	return value
}

// GetIntOrDefault retrieves an integer value or returns a default
func (s *Store) GetIntOrDefault(ctx context.Context, key string, defaultValue int) int {
	value, err := s.GetInt(ctx, key)
	if err != nil {
		return defaultValue
	}
	return value
}

// GetBoolOrDefault retrieves a boolean value or returns a default
func (s *Store) GetBoolOrDefault(ctx context.Context, key string, defaultValue bool) bool {
	value, err := s.GetBool(ctx, key)
	if err != nil {
		return defaultValue
	}
	return value
}

// ParseAndSetFromString parses a string value and stores it
// Attempts to detect the type (int, bool, string)
func (s *Store) ParseAndSetFromString(ctx context.Context, key, valueStr string) error {
	// Try int
	if intVal, err := strconv.Atoi(valueStr); err == nil {
		return s.SetInt(ctx, key, intVal)
	}

	// Try bool
	if boolVal, err := strconv.ParseBool(valueStr); err == nil {
		return s.SetBool(ctx, key, boolVal)
	}

	// Default to string
	return s.SetString(ctx, key, valueStr)
}
