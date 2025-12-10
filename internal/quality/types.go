package quality

import "time"

// QualityDefinition represents a specific quality level with detailed specifications
type QualityDefinition struct {
	ID         int       `json:"id" db:"id"`
	Name       string    `json:"name" db:"name"`
	Title      string    `json:"title" db:"title"`
	Resolution *int      `json:"resolution,omitempty" db:"resolution"`
	Source     *string   `json:"source,omitempty" db:"source"`
	Modifier   *string   `json:"modifier,omitempty" db:"modifier"`
	MinSize    *int64    `json:"min_size,omitempty" db:"min_size"`
	MaxSize    *int64    `json:"max_size,omitempty" db:"max_size"`
	Weight     int       `json:"weight" db:"weight"`
	CreatedAt  time.Time `json:"created_at" db:"created_at"`
	UpdatedAt  time.Time `json:"updated_at" db:"updated_at"`
}

// QualityProfile represents a user-defined quality profile with ordered preferences
type QualityProfile struct {
	ID              int                  `json:"id" db:"id"`
	Name            string               `json:"name" db:"name"`
	Description     *string              `json:"description,omitempty" db:"description"`
	CutoffQualityID *int                 `json:"cutoff_quality_id,omitempty" db:"cutoff_quality_id"`
	UpgradeAllowed  bool                 `json:"upgrade_allowed" db:"upgrade_allowed"`
	CreatedAt       time.Time            `json:"created_at" db:"created_at"`
	UpdatedAt       time.Time            `json:"updated_at" db:"updated_at"`
	Items           []QualityProfileItem `json:"items,omitempty"`
	CutoffQuality   *QualityDefinition   `json:"cutoff_quality,omitempty"`
}

// QualityProfileItem represents a quality in a profile with its settings
type QualityProfileItem struct {
	ID        int                `json:"id" db:"id"`
	ProfileID int                `json:"profile_id" db:"profile_id"`
	QualityID int                `json:"quality_id" db:"quality_id"`
	Allowed   bool               `json:"allowed" db:"allowed"`
	SortOrder int                `json:"sort_order" db:"sort_order"`
	CreatedAt time.Time          `json:"created_at" db:"created_at"`
	Quality   *QualityDefinition `json:"quality,omitempty"`
}

// MediaQuality tracks the current quality of a media item
type MediaQuality struct {
	ID              int64              `json:"id" db:"id"`
	MediaItemID     int64              `json:"media_item_id" db:"media_item_id"`
	MediaFileID     *int64             `json:"media_file_id,omitempty" db:"media_file_id"`
	QualityID       *int               `json:"quality_id,omitempty" db:"quality_id"`
	ProfileID       *int               `json:"profile_id,omitempty" db:"profile_id"`
	DetectedQuality *string            `json:"detected_quality,omitempty" db:"detected_quality"`
	Resolution      *int               `json:"resolution,omitempty" db:"resolution"`
	Source          *string            `json:"source,omitempty" db:"source"`
	CodecVideo      *string            `json:"codec_video,omitempty" db:"codec_video"`
	CodecAudio      *string            `json:"codec_audio,omitempty" db:"codec_audio"`
	IsProper        bool               `json:"is_proper" db:"is_proper"`
	IsRepack        bool               `json:"is_repack" db:"is_repack"`
	IsRemux         bool               `json:"is_remux" db:"is_remux"`
	RevisionVersion int                `json:"revision_version" db:"revision_version"`
	UpgradeAllowed  bool               `json:"upgrade_allowed" db:"upgrade_allowed"`
	CutoffMet       bool               `json:"cutoff_met" db:"cutoff_met"`
	CreatedAt       time.Time          `json:"created_at" db:"created_at"`
	UpdatedAt       time.Time          `json:"updated_at" db:"updated_at"`
	Quality         *QualityDefinition `json:"quality,omitempty"`
	Profile         *QualityProfile    `json:"profile,omitempty"`
}

// QualityUpgradeHistory tracks quality upgrades over time
type QualityUpgradeHistory struct {
	ID              int64              `json:"id" db:"id"`
	MediaItemID     int64              `json:"media_item_id" db:"media_item_id"`
	OldQualityID    *int               `json:"old_quality_id,omitempty" db:"old_quality_id"`
	NewQualityID    *int               `json:"new_quality_id,omitempty" db:"new_quality_id"`
	OldFileID       *int64             `json:"old_file_id,omitempty" db:"old_file_id"`
	NewFileID       *int64             `json:"new_file_id,omitempty" db:"new_file_id"`
	DownloadID      *string            `json:"download_id,omitempty" db:"download_id"`
	Reason          *string            `json:"reason,omitempty" db:"reason"`
	OldFileSize     *int64             `json:"old_file_size,omitempty" db:"old_file_size"`
	NewFileSize     *int64             `json:"new_file_size,omitempty" db:"new_file_size"`
	CreatedAt       time.Time          `json:"created_at" db:"created_at"`
	CreatedByUserID *int64             `json:"created_by_user_id,omitempty" db:"created_by_user_id"`
	OldQuality      *QualityDefinition `json:"old_quality,omitempty"`
	NewQuality      *QualityDefinition `json:"new_quality,omitempty"`
}

// DetectedQualityInfo represents quality information detected from a release name
type DetectedQualityInfo struct {
	Quality      *QualityDefinition `json:"quality,omitempty"`
	QualityName  string             `json:"quality_name"`
	Resolution   *int               `json:"resolution,omitempty"`
	Source       *string            `json:"source,omitempty"`
	CodecVideo   *string            `json:"codec_video,omitempty"`
	CodecAudio   *string            `json:"codec_audio,omitempty"`
	IsProper     bool               `json:"is_proper"`
	IsRepack     bool               `json:"is_repack"`
	IsRemux      bool               `json:"is_remux"`
	IsRemastered bool               `json:"is_remastered"`
}

// CreateQualityDefinitionParams represents parameters for creating a quality definition
type CreateQualityDefinitionParams struct {
	Name       string  `json:"name" binding:"required"`
	Title      string  `json:"title" binding:"required"`
	Resolution *int    `json:"resolution,omitempty"`
	Source     *string `json:"source,omitempty"`
	Modifier   *string `json:"modifier,omitempty"`
	MinSize    *int64  `json:"min_size,omitempty"`
	MaxSize    *int64  `json:"max_size,omitempty"`
	Weight     int     `json:"weight"`
}

// UpdateQualityDefinitionParams represents parameters for updating a quality definition
type UpdateQualityDefinitionParams struct {
	Title      *string `json:"title,omitempty"`
	Resolution *int    `json:"resolution,omitempty"`
	Source     *string `json:"source,omitempty"`
	Modifier   *string `json:"modifier,omitempty"`
	MinSize    *int64  `json:"min_size,omitempty"`
	MaxSize    *int64  `json:"max_size,omitempty"`
	Weight     *int    `json:"weight,omitempty"`
}

// CreateQualityProfileParams represents parameters for creating a quality profile
type CreateQualityProfileParams struct {
	Name            string                     `json:"name" binding:"required"`
	Description     *string                    `json:"description,omitempty"`
	CutoffQualityID *int                       `json:"cutoff_quality_id,omitempty"`
	UpgradeAllowed  bool                       `json:"upgrade_allowed"`
	Items           []CreateQualityProfileItem `json:"items,omitempty"`
}

// CreateQualityProfileItem represents a quality item when creating a profile
type CreateQualityProfileItem struct {
	QualityID int  `json:"quality_id" binding:"required"`
	Allowed   bool `json:"allowed"`
	SortOrder int  `json:"sort_order" binding:"required"`
}

// UpdateQualityProfileParams represents parameters for updating a quality profile
type UpdateQualityProfileParams struct {
	Name            *string                    `json:"name,omitempty"`
	Description     *string                    `json:"description,omitempty"`
	CutoffQualityID *int                       `json:"cutoff_quality_id,omitempty"`
	UpgradeAllowed  *bool                      `json:"upgrade_allowed,omitempty"`
	Items           []CreateQualityProfileItem `json:"items,omitempty"`
}

// AssignProfileToMediaParams represents parameters for assigning a profile to media
type AssignProfileToMediaParams struct {
	MediaItemID int `json:"media_item_id" binding:"required"`
	ProfileID   int `json:"profile_id" binding:"required"`
}

// QualityUpgradeCheckResult represents the result of checking if an upgrade is available
type QualityUpgradeCheckResult struct {
	CanUpgrade       bool               `json:"can_upgrade"`
	CurrentQuality   *QualityDefinition `json:"current_quality,omitempty"`
	AvailableQuality *QualityDefinition `json:"available_quality,omitempty"`
	Reason           string             `json:"reason,omitempty"`
}

// QualityComparisonResult represents the result of comparing two qualities
type QualityComparisonResult int

const (
	QualityWorse  QualityComparisonResult = -1
	QualitySame   QualityComparisonResult = 0
	QualityBetter QualityComparisonResult = 1
)
