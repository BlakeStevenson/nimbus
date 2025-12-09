package media

import (
	"encoding/json"
	"time"
)

// MediaKind represents the type of media item
type MediaKind string

const (
	MediaKindMovie       MediaKind = "movie"
	MediaKindTVSeries    MediaKind = "tv_series"
	MediaKindTVSeason    MediaKind = "tv_season"
	MediaKindTVEpisode   MediaKind = "tv_episode"
	MediaKindMusicArtist MediaKind = "music_artist"
	MediaKindMusicAlbum  MediaKind = "music_album"
	MediaKindMusicTrack  MediaKind = "music_track"
	MediaKindBook        MediaKind = "book"
	MediaKindBookSeries  MediaKind = "book_series"
)

// MediaItem represents a generic media item
type MediaItem struct {
	ID          int64                  `json:"id"`
	Kind        MediaKind              `json:"kind"`
	Title       string                 `json:"title"`
	SortTitle   string                 `json:"sort_title"`
	Year        *int32                 `json:"year,omitempty"`
	ExternalIDs map[string]interface{} `json:"external_ids"`
	Metadata    map[string]interface{} `json:"metadata"`
	ParentID    *int64                 `json:"parent_id,omitempty"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
}

// CreateMediaParams holds parameters for creating a media item
type CreateMediaParams struct {
	Kind        MediaKind              `json:"kind"`
	Title       string                 `json:"title"`
	SortTitle   string                 `json:"sort_title,omitempty"`
	Year        *int32                 `json:"year,omitempty"`
	ExternalIDs map[string]interface{} `json:"external_ids,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	ParentID    *int64                 `json:"parent_id,omitempty"`
}

// UpdateMediaParams holds parameters for updating a media item
type UpdateMediaParams struct {
	Title       *string                `json:"title,omitempty"`
	SortTitle   *string                `json:"sort_title,omitempty"`
	Year        *int32                 `json:"year,omitempty"`
	ExternalIDs map[string]interface{} `json:"external_ids,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	ParentID    *int64                 `json:"parent_id,omitempty"`
}

// MediaFilter holds filter parameters for listing media items
type MediaFilter struct {
	Kind         *MediaKind `json:"kind,omitempty"`
	Search       *string    `json:"search,omitempty"`
	ParentID     *int64     `json:"parent_id,omitempty"`
	TopLevelOnly bool       `json:"top_level_only,omitempty"` // Exclude items with parents
	Limit        int32      `json:"limit"`
	Offset       int32      `json:"offset"`
}

// MediaList represents a paginated list of media items
type MediaList struct {
	Items   []*MediaItem `json:"items"`
	Total   int64        `json:"total"`
	Limit   int32        `json:"limit"`
	Offset  int32        `json:"offset"`
	HasMore bool         `json:"has_more"`
}

// Validate validates create parameters
func (p *CreateMediaParams) Validate() error {
	if p.Kind == "" {
		return ErrInvalidKind
	}
	if p.Title == "" {
		return ErrTitleRequired
	}
	return nil
}

// Validate validates update parameters
func (p *UpdateMediaParams) Validate() error {
	if p.Title != nil && *p.Title == "" {
		return ErrTitleRequired
	}
	return nil
}

// Validate validates filter parameters
func (f *MediaFilter) Validate() error {
	if f.Limit <= 0 {
		f.Limit = 20
	}
	if f.Limit > 100 {
		f.Limit = 100
	}
	if f.Offset < 0 {
		f.Offset = 0
	}
	return nil
}

// MarshalMap converts a map to JSONB
func MarshalMap(m map[string]interface{}) ([]byte, error) {
	if m == nil {
		return []byte("{}"), nil
	}
	return json.Marshal(m)
}

// UnmarshalMap converts JSONB to a map
func UnmarshalMap(data []byte) (map[string]interface{}, error) {
	var m map[string]interface{}
	if len(data) == 0 {
		return make(map[string]interface{}), nil
	}
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, err
	}
	return m, nil
}
