package quality

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Service handles quality profile operations
type Service struct {
	db       *pgxpool.Pool
	detector *Detector
}

// NewService creates a new quality service
func NewService(db *pgxpool.Pool) *Service {
	return &Service{
		db:       db,
		detector: NewDetector(),
	}
}

// Quality Definitions

// ListQualityDefinitions lists all quality definitions
func (s *Service) ListQualityDefinitions(ctx context.Context) ([]QualityDefinition, error) {
	query := `
		SELECT id, name, title, resolution, source, modifier, min_size, max_size, weight, created_at, updated_at
		FROM quality_definitions
		ORDER BY weight DESC, name
	`

	rows, err := s.db.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list quality definitions: %w", err)
	}
	defer rows.Close()

	var definitions []QualityDefinition
	for rows.Next() {
		var def QualityDefinition
		err := rows.Scan(
			&def.ID, &def.Name, &def.Title, &def.Resolution, &def.Source, &def.Modifier,
			&def.MinSize, &def.MaxSize, &def.Weight, &def.CreatedAt, &def.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan quality definition: %w", err)
		}
		definitions = append(definitions, def)
	}

	return definitions, rows.Err()
}

// GetQualityDefinition gets a quality definition by ID
func (s *Service) GetQualityDefinition(ctx context.Context, id int) (*QualityDefinition, error) {
	query := `
		SELECT id, name, title, resolution, source, modifier, min_size, max_size, weight, created_at, updated_at
		FROM quality_definitions
		WHERE id = $1
	`

	var def QualityDefinition
	err := s.db.QueryRow(ctx, query, id).Scan(
		&def.ID, &def.Name, &def.Title, &def.Resolution, &def.Source, &def.Modifier,
		&def.MinSize, &def.MaxSize, &def.Weight, &def.CreatedAt, &def.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("quality definition not found")
		}
		return nil, fmt.Errorf("failed to get quality definition: %w", err)
	}

	return &def, nil
}

// GetQualityDefinitionByName gets a quality definition by name
func (s *Service) GetQualityDefinitionByName(ctx context.Context, name string) (*QualityDefinition, error) {
	query := `
		SELECT id, name, title, resolution, source, modifier, min_size, max_size, weight, created_at, updated_at
		FROM quality_definitions
		WHERE name = $1
	`

	var def QualityDefinition
	err := s.db.QueryRow(ctx, query, name).Scan(
		&def.ID, &def.Name, &def.Title, &def.Resolution, &def.Source, &def.Modifier,
		&def.MinSize, &def.MaxSize, &def.Weight, &def.CreatedAt, &def.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("quality definition not found")
		}
		return nil, fmt.Errorf("failed to get quality definition: %w", err)
	}

	return &def, nil
}

// CreateQualityDefinition creates a new quality definition
func (s *Service) CreateQualityDefinition(ctx context.Context, params CreateQualityDefinitionParams) (*QualityDefinition, error) {
	query := `
		INSERT INTO quality_definitions (name, title, resolution, source, modifier, min_size, max_size, weight)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, name, title, resolution, source, modifier, min_size, max_size, weight, created_at, updated_at
	`

	var def QualityDefinition
	err := s.db.QueryRow(ctx, query,
		params.Name, params.Title, params.Resolution, params.Source, params.Modifier,
		params.MinSize, params.MaxSize, params.Weight,
	).Scan(
		&def.ID, &def.Name, &def.Title, &def.Resolution, &def.Source, &def.Modifier,
		&def.MinSize, &def.MaxSize, &def.Weight, &def.CreatedAt, &def.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create quality definition: %w", err)
	}

	return &def, nil
}

// UpdateQualityDefinition updates a quality definition
func (s *Service) UpdateQualityDefinition(ctx context.Context, id int, params UpdateQualityDefinitionParams) (*QualityDefinition, error) {
	query := `
		UPDATE quality_definitions
		SET title = COALESCE($1, title),
			resolution = COALESCE($2, resolution),
			source = COALESCE($3, source),
			modifier = COALESCE($4, modifier),
			min_size = COALESCE($5, min_size),
			max_size = COALESCE($6, max_size),
			weight = COALESCE($7, weight)
		WHERE id = $8
		RETURNING id, name, title, resolution, source, modifier, min_size, max_size, weight, created_at, updated_at
	`

	var def QualityDefinition
	err := s.db.QueryRow(ctx, query,
		params.Title, params.Resolution, params.Source, params.Modifier,
		params.MinSize, params.MaxSize, params.Weight, id,
	).Scan(
		&def.ID, &def.Name, &def.Title, &def.Resolution, &def.Source, &def.Modifier,
		&def.MinSize, &def.MaxSize, &def.Weight, &def.CreatedAt, &def.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("quality definition not found")
		}
		return nil, fmt.Errorf("failed to update quality definition: %w", err)
	}

	return &def, nil
}

// DeleteQualityDefinition deletes a quality definition
func (s *Service) DeleteQualityDefinition(ctx context.Context, id int) error {
	query := `DELETE FROM quality_definitions WHERE id = $1`

	result, err := s.db.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete quality definition: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("quality definition not found")
	}

	return nil
}

// Quality Profiles

// ListQualityProfiles lists all quality profiles
func (s *Service) ListQualityProfiles(ctx context.Context) ([]QualityProfile, error) {
	query := `
		SELECT qp.id, qp.name, qp.description, qp.cutoff_quality_id, qp.upgrade_allowed,
		       qp.created_at, qp.updated_at,
		       qd.id, qd.name, qd.title, qd.resolution, qd.source, qd.modifier,
		       qd.min_size, qd.max_size, qd.weight, qd.created_at, qd.updated_at
		FROM quality_profiles qp
		LEFT JOIN quality_definitions qd ON qp.cutoff_quality_id = qd.id
		ORDER BY qp.name
	`

	rows, err := s.db.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list quality profiles: %w", err)
	}
	defer rows.Close()

	var profiles []QualityProfile
	for rows.Next() {
		var profile QualityProfile
		var cutoffQuality QualityDefinition
		var hasCutoff bool

		err := rows.Scan(
			&profile.ID, &profile.Name, &profile.Description, &profile.CutoffQualityID,
			&profile.UpgradeAllowed, &profile.CreatedAt, &profile.UpdatedAt,
			&cutoffQuality.ID, &cutoffQuality.Name, &cutoffQuality.Title, &cutoffQuality.Resolution,
			&cutoffQuality.Source, &cutoffQuality.Modifier, &cutoffQuality.MinSize, &cutoffQuality.MaxSize,
			&cutoffQuality.Weight, &cutoffQuality.CreatedAt, &cutoffQuality.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan quality profile: %w", err)
		}

		if profile.CutoffQualityID != nil {
			hasCutoff = true
			profile.CutoffQuality = &cutoffQuality
		}

		// Load profile items
		items, err := s.getProfileItems(ctx, profile.ID)
		if err != nil {
			return nil, fmt.Errorf("failed to load profile items: %w", err)
		}
		profile.Items = items

		if !hasCutoff {
			profile.CutoffQuality = nil
		}

		profiles = append(profiles, profile)
	}

	return profiles, rows.Err()
}

// GetQualityProfile gets a quality profile by ID
func (s *Service) GetQualityProfile(ctx context.Context, id int) (*QualityProfile, error) {
	query := `
		SELECT qp.id, qp.name, qp.description, qp.cutoff_quality_id, qp.upgrade_allowed,
		       qp.created_at, qp.updated_at,
		       qd.id, qd.name, qd.title, qd.resolution, qd.source, qd.modifier,
		       qd.min_size, qd.max_size, qd.weight, qd.created_at, qd.updated_at
		FROM quality_profiles qp
		LEFT JOIN quality_definitions qd ON qp.cutoff_quality_id = qd.id
		WHERE qp.id = $1
	`

	var profile QualityProfile
	var cutoffQuality QualityDefinition

	err := s.db.QueryRow(ctx, query, id).Scan(
		&profile.ID, &profile.Name, &profile.Description, &profile.CutoffQualityID,
		&profile.UpgradeAllowed, &profile.CreatedAt, &profile.UpdatedAt,
		&cutoffQuality.ID, &cutoffQuality.Name, &cutoffQuality.Title, &cutoffQuality.Resolution,
		&cutoffQuality.Source, &cutoffQuality.Modifier, &cutoffQuality.MinSize, &cutoffQuality.MaxSize,
		&cutoffQuality.Weight, &cutoffQuality.CreatedAt, &cutoffQuality.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("quality profile not found")
		}
		return nil, fmt.Errorf("failed to get quality profile: %w", err)
	}

	if profile.CutoffQualityID != nil {
		profile.CutoffQuality = &cutoffQuality
	}

	// Load profile items
	items, err := s.getProfileItems(ctx, profile.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to load profile items: %w", err)
	}
	profile.Items = items

	return &profile, nil
}

// getProfileItems gets items for a quality profile
func (s *Service) getProfileItems(ctx context.Context, profileID int) ([]QualityProfileItem, error) {
	query := `
		SELECT qpi.id, qpi.profile_id, qpi.quality_id, qpi.allowed, qpi.sort_order, qpi.created_at,
		       qd.id, qd.name, qd.title, qd.resolution, qd.source, qd.modifier,
		       qd.min_size, qd.max_size, qd.weight, qd.created_at, qd.updated_at
		FROM quality_profile_items qpi
		JOIN quality_definitions qd ON qpi.quality_id = qd.id
		WHERE qpi.profile_id = $1
		ORDER BY qpi.sort_order
	`

	rows, err := s.db.Query(ctx, query, profileID)
	if err != nil {
		return nil, fmt.Errorf("failed to get profile items: %w", err)
	}
	defer rows.Close()

	var items []QualityProfileItem
	for rows.Next() {
		var item QualityProfileItem
		var quality QualityDefinition

		err := rows.Scan(
			&item.ID, &item.ProfileID, &item.QualityID, &item.Allowed, &item.SortOrder, &item.CreatedAt,
			&quality.ID, &quality.Name, &quality.Title, &quality.Resolution, &quality.Source,
			&quality.Modifier, &quality.MinSize, &quality.MaxSize, &quality.Weight,
			&quality.CreatedAt, &quality.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan profile item: %w", err)
		}

		item.Quality = &quality
		items = append(items, item)
	}

	return items, rows.Err()
}

// CreateQualityProfile creates a new quality profile
func (s *Service) CreateQualityProfile(ctx context.Context, params CreateQualityProfileParams) (*QualityProfile, error) {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// Insert profile
	query := `
		INSERT INTO quality_profiles (name, description, cutoff_quality_id, upgrade_allowed)
		VALUES ($1, $2, $3, $4)
		RETURNING id, name, description, cutoff_quality_id, upgrade_allowed, created_at, updated_at
	`

	var profile QualityProfile
	err = tx.QueryRow(ctx, query,
		params.Name, params.Description, params.CutoffQualityID, params.UpgradeAllowed,
	).Scan(
		&profile.ID, &profile.Name, &profile.Description, &profile.CutoffQualityID,
		&profile.UpgradeAllowed, &profile.CreatedAt, &profile.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create quality profile: %w", err)
	}

	// Insert profile items
	if len(params.Items) > 0 {
		itemQuery := `
			INSERT INTO quality_profile_items (profile_id, quality_id, allowed, sort_order)
			VALUES ($1, $2, $3, $4)
		`

		for _, item := range params.Items {
			_, err := tx.Exec(ctx, itemQuery, profile.ID, item.QualityID, item.Allowed, item.SortOrder)
			if err != nil {
				return nil, fmt.Errorf("failed to create profile item: %w", err)
			}
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	// Reload with items
	return s.GetQualityProfile(ctx, profile.ID)
}

// UpdateQualityProfile updates a quality profile
func (s *Service) UpdateQualityProfile(ctx context.Context, id int, params UpdateQualityProfileParams) (*QualityProfile, error) {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// Update profile
	query := `
		UPDATE quality_profiles
		SET name = COALESCE($1, name),
			description = COALESCE($2, description),
			cutoff_quality_id = COALESCE($3, cutoff_quality_id),
			upgrade_allowed = COALESCE($4, upgrade_allowed)
		WHERE id = $5
		RETURNING id, name, description, cutoff_quality_id, upgrade_allowed, created_at, updated_at
	`

	var profile QualityProfile
	err = tx.QueryRow(ctx, query,
		params.Name, params.Description, params.CutoffQualityID, params.UpgradeAllowed, id,
	).Scan(
		&profile.ID, &profile.Name, &profile.Description, &profile.CutoffQualityID,
		&profile.UpgradeAllowed, &profile.CreatedAt, &profile.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("quality profile not found")
		}
		return nil, fmt.Errorf("failed to update quality profile: %w", err)
	}

	// Update items if provided
	if params.Items != nil {
		// Delete existing items
		_, err := tx.Exec(ctx, `DELETE FROM quality_profile_items WHERE profile_id = $1`, id)
		if err != nil {
			return nil, fmt.Errorf("failed to delete existing items: %w", err)
		}

		// Insert new items
		if len(params.Items) > 0 {
			itemQuery := `
				INSERT INTO quality_profile_items (profile_id, quality_id, allowed, sort_order)
				VALUES ($1, $2, $3, $4)
			`

			for _, item := range params.Items {
				_, err := tx.Exec(ctx, itemQuery, id, item.QualityID, item.Allowed, item.SortOrder)
				if err != nil {
					return nil, fmt.Errorf("failed to create profile item: %w", err)
				}
			}
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	// Reload with items
	return s.GetQualityProfile(ctx, id)
}

// DeleteQualityProfile deletes a quality profile
func (s *Service) DeleteQualityProfile(ctx context.Context, id int) error {
	query := `DELETE FROM quality_profiles WHERE id = $1`

	result, err := s.db.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete quality profile: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("quality profile not found")
	}

	return nil
}

// DetectQuality detects quality from a release name
func (s *Service) DetectQuality(ctx context.Context, releaseName string) (*DetectedQualityInfo, error) {
	info := s.detector.DetectQuality(releaseName)

	// Try to match to a known quality definition
	definitions, err := s.ListQualityDefinitions(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list quality definitions: %w", err)
	}

	matched := s.detector.MatchQualityDefinition(info, definitions)
	info.Quality = matched

	return info, nil
}
