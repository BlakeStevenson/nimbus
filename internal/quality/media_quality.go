package quality

import (
	"context"
	"database/sql"
	"fmt"
)

// Media Quality Operations

// GetMediaQuality gets the quality information for a media item
func (s *Service) GetMediaQuality(ctx context.Context, mediaItemID int64) (*MediaQuality, error) {
	query := `
		SELECT mq.id, mq.media_item_id, mq.media_file_id, mq.quality_id, mq.profile_id,
		       mq.detected_quality, mq.resolution, mq.source, mq.codec_video, mq.codec_audio,
		       mq.is_proper, mq.is_repack, mq.is_remux, mq.revision_version,
		       mq.upgrade_allowed, mq.cutoff_met, mq.created_at, mq.updated_at,
		       qd.id, qd.name, qd.title, qd.resolution, qd.source, qd.modifier,
		       qd.min_size, qd.max_size, qd.weight, qd.created_at, qd.updated_at,
		       qp.id, qp.name, qp.description, qp.cutoff_quality_id, qp.upgrade_allowed,
		       qp.created_at, qp.updated_at
		FROM media_quality mq
		LEFT JOIN quality_definitions qd ON mq.quality_id = qd.id
		LEFT JOIN quality_profiles qp ON mq.profile_id = qp.id
		WHERE mq.media_item_id = $1
		LIMIT 1
	`

	var mq MediaQuality

	// Use nullable types for LEFT JOIN columns
	var qualityID, qualityResolution, qualityMinSize, qualityMaxSize sql.NullInt64
	var qualityName, qualityTitle, qualitySource, qualityModifier sql.NullString
	var qualityWeight sql.NullInt64
	var qualityCreatedAt, qualityUpdatedAt sql.NullTime

	var profileID, profileCutoffQualityID sql.NullInt64
	var profileName, profileDescription sql.NullString
	var profileUpgradeAllowed sql.NullBool
	var profileCreatedAt, profileUpdatedAt sql.NullTime

	err := s.db.QueryRow(ctx, query, mediaItemID).Scan(
		&mq.ID, &mq.MediaItemID, &mq.MediaFileID, &mq.QualityID, &mq.ProfileID,
		&mq.DetectedQuality, &mq.Resolution, &mq.Source, &mq.CodecVideo, &mq.CodecAudio,
		&mq.IsProper, &mq.IsRepack, &mq.IsRemux, &mq.RevisionVersion,
		&mq.UpgradeAllowed, &mq.CutoffMet, &mq.CreatedAt, &mq.UpdatedAt,
		&qualityID, &qualityName, &qualityTitle, &qualityResolution, &qualitySource,
		&qualityModifier, &qualityMinSize, &qualityMaxSize, &qualityWeight,
		&qualityCreatedAt, &qualityUpdatedAt,
		&profileID, &profileName, &profileDescription, &profileCutoffQualityID,
		&profileUpgradeAllowed, &profileCreatedAt, &profileUpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("media quality not found")
		}
		return nil, fmt.Errorf("failed to get media quality: %w", err)
	}

	// Populate quality if present
	if qualityID.Valid {
		quality := &QualityDefinition{
			ID:     int(qualityID.Int64),
			Weight: int(qualityWeight.Int64),
		}
		if qualityName.Valid {
			quality.Name = qualityName.String
		}
		if qualityTitle.Valid {
			quality.Title = qualityTitle.String
		}
		if qualityResolution.Valid {
			res := int(qualityResolution.Int64)
			quality.Resolution = &res
		}
		if qualitySource.Valid {
			quality.Source = &qualitySource.String
		}
		if qualityModifier.Valid {
			quality.Modifier = &qualityModifier.String
		}
		if qualityMinSize.Valid {
			size := int64(qualityMinSize.Int64)
			quality.MinSize = &size
		}
		if qualityMaxSize.Valid {
			size := int64(qualityMaxSize.Int64)
			quality.MaxSize = &size
		}
		if qualityCreatedAt.Valid {
			quality.CreatedAt = qualityCreatedAt.Time
		}
		if qualityUpdatedAt.Valid {
			quality.UpdatedAt = qualityUpdatedAt.Time
		}
		mq.Quality = quality
	}

	// Populate profile if present
	if profileID.Valid {
		profile := &QualityProfile{
			ID: int(profileID.Int64),
		}
		if profileName.Valid {
			profile.Name = profileName.String
		}
		if profileDescription.Valid {
			profile.Description = &profileDescription.String
		}
		if profileCutoffQualityID.Valid {
			cutoff := int(profileCutoffQualityID.Int64)
			profile.CutoffQualityID = &cutoff
		}
		if profileUpgradeAllowed.Valid {
			profile.UpgradeAllowed = profileUpgradeAllowed.Bool
		}
		if profileCreatedAt.Valid {
			profile.CreatedAt = profileCreatedAt.Time
		}
		if profileUpdatedAt.Valid {
			profile.UpdatedAt = profileUpdatedAt.Time
		}
		mq.Profile = profile
	}

	return &mq, nil
}

// SetMediaQuality sets or updates the quality information for a media item
func (s *Service) SetMediaQuality(ctx context.Context, mediaItemID int64, mediaFileID *int64, detectedInfo *DetectedQualityInfo) (*MediaQuality, error) {
	var qualityID *int
	if detectedInfo.Quality != nil {
		qualityID = &detectedInfo.Quality.ID
	}

	query := `
		INSERT INTO media_quality (
			media_item_id, media_file_id, quality_id, detected_quality, resolution, source,
			codec_video, codec_audio, is_proper, is_repack, is_remux
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		ON CONFLICT (media_item_id, media_file_id)
		DO UPDATE SET
			quality_id = EXCLUDED.quality_id,
			detected_quality = EXCLUDED.detected_quality,
			resolution = EXCLUDED.resolution,
			source = EXCLUDED.source,
			codec_video = EXCLUDED.codec_video,
			codec_audio = EXCLUDED.codec_audio,
			is_proper = EXCLUDED.is_proper,
			is_repack = EXCLUDED.is_repack,
			is_remux = EXCLUDED.is_remux,
			revision_version = CASE
				WHEN media_quality.is_proper = false AND EXCLUDED.is_proper = true THEN media_quality.revision_version + 1
				WHEN media_quality.is_repack = false AND EXCLUDED.is_repack = true THEN media_quality.revision_version + 1
				ELSE media_quality.revision_version
			END
		RETURNING id, media_item_id, media_file_id, quality_id, profile_id,
		          detected_quality, resolution, source, codec_video, codec_audio,
		          is_proper, is_repack, is_remux, revision_version,
		          upgrade_allowed, cutoff_met, created_at, updated_at
	`

	var mq MediaQuality
	err := s.db.QueryRow(ctx, query,
		mediaItemID, mediaFileID, qualityID, detectedInfo.QualityName, detectedInfo.Resolution,
		detectedInfo.Source, detectedInfo.CodecVideo, detectedInfo.CodecAudio,
		detectedInfo.IsProper, detectedInfo.IsRepack, detectedInfo.IsRemux,
	).Scan(
		&mq.ID, &mq.MediaItemID, &mq.MediaFileID, &mq.QualityID, &mq.ProfileID,
		&mq.DetectedQuality, &mq.Resolution, &mq.Source, &mq.CodecVideo, &mq.CodecAudio,
		&mq.IsProper, &mq.IsRepack, &mq.IsRemux, &mq.RevisionVersion,
		&mq.UpgradeAllowed, &mq.CutoffMet, &mq.CreatedAt, &mq.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to set media quality: %w", err)
	}

	return &mq, nil
}

// AssignProfileToMedia assigns a quality profile to a media item
func (s *Service) AssignProfileToMedia(ctx context.Context, mediaItemID int64, profileID int) error {
	// Verify profile exists
	profile, err := s.GetQualityProfile(ctx, profileID)
	if err != nil {
		return fmt.Errorf("failed to get quality profile: %w", err)
	}

	// Try to update existing record first
	updateQuery := `
		UPDATE media_quality
		SET profile_id = $1, upgrade_allowed = $2, updated_at = NOW()
		WHERE media_item_id = $3 AND media_file_id IS NULL
	`
	result, err := s.db.Exec(ctx, updateQuery, profileID, profile.UpgradeAllowed, mediaItemID)
	if err != nil {
		return fmt.Errorf("failed to update profile: %w", err)
	}

	// If no rows were updated, insert a new record
	if result.RowsAffected() == 0 {
		insertQuery := `
			INSERT INTO media_quality (media_item_id, media_file_id, profile_id, upgrade_allowed)
			VALUES ($1, NULL, $2, $3)
		`
		_, err := s.db.Exec(ctx, insertQuery, mediaItemID, profileID, profile.UpgradeAllowed)
		if err != nil {
			return fmt.Errorf("failed to insert profile: %w", err)
		}
	}

	// Get the media quality record to check cutoff
	mq, err := s.GetMediaQuality(ctx, mediaItemID)
	if err != nil {
		// Ignore error if quality record doesn't exist yet
		return nil
	}

	// Check if cutoff is met
	if mq != nil && mq.QualityID != nil && profile.CutoffQualityID != nil {
		cutoffMet, err := s.isCutoffMet(ctx, *mq.QualityID, *profile.CutoffQualityID)
		if err != nil {
			return fmt.Errorf("failed to check cutoff: %w", err)
		}

		if cutoffMet {
			_, err := s.db.Exec(ctx, `UPDATE media_quality SET cutoff_met = true WHERE media_item_id = $1`, mediaItemID)
			if err != nil {
				return fmt.Errorf("failed to update cutoff status: %w", err)
			}
		}
	}

	return nil
}

// CheckUpgradeAvailable checks if a quality upgrade is available for media
func (s *Service) CheckUpgradeAvailable(ctx context.Context, mediaItemID int64, availableQualityID int) (*QualityUpgradeCheckResult, error) {
	// Get current media quality
	currentMQ, err := s.GetMediaQuality(ctx, mediaItemID)
	if err != nil {
		return &QualityUpgradeCheckResult{
			CanUpgrade: true,
			Reason:     "no existing quality",
		}, nil
	}

	// Check if upgrades are allowed
	if !currentMQ.UpgradeAllowed {
		return &QualityUpgradeCheckResult{
			CanUpgrade: false,
			Reason:     "upgrades disabled for this media",
		}, nil
	}

	// Check if cutoff is already met
	if currentMQ.CutoffMet {
		return &QualityUpgradeCheckResult{
			CanUpgrade: false,
			Reason:     "cutoff quality already met",
		}, nil
	}

	// Get current quality
	if currentMQ.QualityID == nil {
		return &QualityUpgradeCheckResult{
			CanUpgrade: true,
			Reason:     "no current quality set",
		}, nil
	}

	currentQuality, err := s.GetQualityDefinition(ctx, *currentMQ.QualityID)
	if err != nil {
		return nil, fmt.Errorf("failed to get current quality: %w", err)
	}

	// Get available quality
	availableQuality, err := s.GetQualityDefinition(ctx, availableQualityID)
	if err != nil {
		return nil, fmt.Errorf("failed to get available quality: %w", err)
	}

	// Compare qualities
	comparison := s.CompareQuality(currentQuality, availableQuality)

	if comparison == QualityBetter {
		// Check if profile allows this quality
		if currentMQ.ProfileID != nil {
			allowed, err := s.isQualityAllowedInProfile(ctx, *currentMQ.ProfileID, availableQualityID)
			if err != nil {
				return nil, fmt.Errorf("failed to check profile: %w", err)
			}

			if !allowed {
				return &QualityUpgradeCheckResult{
					CanUpgrade:       false,
					CurrentQuality:   currentQuality,
					AvailableQuality: availableQuality,
					Reason:           "available quality not allowed in profile",
				}, nil
			}
		}

		return &QualityUpgradeCheckResult{
			CanUpgrade:       true,
			CurrentQuality:   currentQuality,
			AvailableQuality: availableQuality,
			Reason:           "better quality available",
		}, nil
	}

	return &QualityUpgradeCheckResult{
		CanUpgrade:       false,
		CurrentQuality:   currentQuality,
		AvailableQuality: availableQuality,
		Reason:           "available quality not better than current",
	}, nil
}

// CompareQuality compares two quality definitions
func (s *Service) CompareQuality(current, available *QualityDefinition) QualityComparisonResult {
	if current.Weight < available.Weight {
		return QualityBetter
	} else if current.Weight > available.Weight {
		return QualityWorse
	}
	return QualitySame
}

// RecordQualityUpgrade records a quality upgrade in history
func (s *Service) RecordQualityUpgrade(ctx context.Context, mediaItemID int64, oldQualityID, newQualityID *int, oldFileID, newFileID *int64, downloadID *string, reason string, userID *int64) error {
	var oldFileSize, newFileSize *int64

	// Get old file size if available
	if oldFileID != nil {
		var size int64
		err := s.db.QueryRow(ctx, `SELECT size FROM media_files WHERE id = $1`, *oldFileID).Scan(&size)
		if err == nil {
			oldFileSize = &size
		}
	}

	// Get new file size if available
	if newFileID != nil {
		var size int64
		err := s.db.QueryRow(ctx, `SELECT size FROM media_files WHERE id = $1`, *newFileID).Scan(&size)
		if err == nil {
			newFileSize = &size
		}
	}

	query := `
		INSERT INTO quality_upgrade_history (
			media_item_id, old_quality_id, new_quality_id, old_file_id, new_file_id,
			download_id, reason, old_file_size, new_file_size, created_by_user_id
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`

	_, err := s.db.Exec(ctx, query,
		mediaItemID, oldQualityID, newQualityID, oldFileID, newFileID,
		downloadID, reason, oldFileSize, newFileSize, userID,
	)
	if err != nil {
		return fmt.Errorf("failed to record quality upgrade: %w", err)
	}

	return nil
}

// GetQualityUpgradeHistory gets the upgrade history for a media item
func (s *Service) GetQualityUpgradeHistory(ctx context.Context, mediaItemID int64) ([]QualityUpgradeHistory, error) {
	query := `
		SELECT quh.id, quh.media_item_id, quh.old_quality_id, quh.new_quality_id,
		       quh.old_file_id, quh.new_file_id, quh.download_id, quh.reason,
		       quh.old_file_size, quh.new_file_size, quh.created_at, quh.created_by_user_id,
		       oq.id, oq.name, oq.title, oq.resolution, oq.source, oq.modifier,
		       oq.min_size, oq.max_size, oq.weight, oq.created_at, oq.updated_at,
		       nq.id, nq.name, nq.title, nq.resolution, nq.source, nq.modifier,
		       nq.min_size, nq.max_size, nq.weight, nq.created_at, nq.updated_at
		FROM quality_upgrade_history quh
		LEFT JOIN quality_definitions oq ON quh.old_quality_id = oq.id
		LEFT JOIN quality_definitions nq ON quh.new_quality_id = nq.id
		WHERE quh.media_item_id = $1
		ORDER BY quh.created_at DESC
	`

	rows, err := s.db.Query(ctx, query, mediaItemID)
	if err != nil {
		return nil, fmt.Errorf("failed to get upgrade history: %w", err)
	}
	defer rows.Close()

	var history []QualityUpgradeHistory
	for rows.Next() {
		var h QualityUpgradeHistory
		var oldQuality, newQuality QualityDefinition

		err := rows.Scan(
			&h.ID, &h.MediaItemID, &h.OldQualityID, &h.NewQualityID,
			&h.OldFileID, &h.NewFileID, &h.DownloadID, &h.Reason,
			&h.OldFileSize, &h.NewFileSize, &h.CreatedAt, &h.CreatedByUserID,
			&oldQuality.ID, &oldQuality.Name, &oldQuality.Title, &oldQuality.Resolution,
			&oldQuality.Source, &oldQuality.Modifier, &oldQuality.MinSize, &oldQuality.MaxSize,
			&oldQuality.Weight, &oldQuality.CreatedAt, &oldQuality.UpdatedAt,
			&newQuality.ID, &newQuality.Name, &newQuality.Title, &newQuality.Resolution,
			&newQuality.Source, &newQuality.Modifier, &newQuality.MinSize, &newQuality.MaxSize,
			&newQuality.Weight, &newQuality.CreatedAt, &newQuality.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan upgrade history: %w", err)
		}

		if h.OldQualityID != nil {
			h.OldQuality = &oldQuality
		}
		if h.NewQualityID != nil {
			h.NewQuality = &newQuality
		}

		history = append(history, h)
	}

	return history, rows.Err()
}

// ListMediaForUpgrade lists media items that are eligible for quality upgrades
func (s *Service) ListMediaForUpgrade(ctx context.Context, profileID *int) ([]int64, error) {
	query := `
		SELECT media_item_id
		FROM media_quality
		WHERE upgrade_allowed = true
		  AND cutoff_met = false
		  AND ($1::int IS NULL OR profile_id = $1)
		ORDER BY media_item_id
	`

	rows, err := s.db.Query(ctx, query, profileID)
	if err != nil {
		return nil, fmt.Errorf("failed to list media for upgrade: %w", err)
	}
	defer rows.Close()

	var mediaIDs []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("failed to scan media ID: %w", err)
		}
		mediaIDs = append(mediaIDs, id)
	}

	return mediaIDs, rows.Err()
}

// Helper functions

func (s *Service) isCutoffMet(ctx context.Context, currentQualityID, cutoffQualityID int) (bool, error) {
	current, err := s.GetQualityDefinition(ctx, currentQualityID)
	if err != nil {
		return false, err
	}

	cutoff, err := s.GetQualityDefinition(ctx, cutoffQualityID)
	if err != nil {
		return false, err
	}

	return current.Weight >= cutoff.Weight, nil
}

func (s *Service) isQualityAllowedInProfile(ctx context.Context, profileID, qualityID int) (bool, error) {
	query := `
		SELECT allowed
		FROM quality_profile_items
		WHERE profile_id = $1 AND quality_id = $2
	`

	var allowed bool
	err := s.db.QueryRow(ctx, query, profileID, qualityID).Scan(&allowed)
	if err != nil {
		if err == sql.ErrNoRows {
			return false, nil
		}
		return false, err
	}

	return allowed, nil
}
