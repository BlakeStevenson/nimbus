package monitoring

import (
	"time"
)

// MonitorMode defines how a series/season should be monitored
type MonitorMode string

const (
	MonitorModeAll          MonitorMode = "all"           // Monitor all episodes
	MonitorModeFuture       MonitorMode = "future"        // Monitor future episodes only
	MonitorModeMissing      MonitorMode = "missing"       // Monitor missing episodes only
	MonitorModeExisting     MonitorMode = "existing"      // Monitor existing episodes only
	MonitorModeFirstSeason  MonitorMode = "first_season"  // Monitor first season only
	MonitorModeLatestSeason MonitorMode = "latest_season" // Monitor latest season only
	MonitorModePilot        MonitorMode = "pilot"         // Monitor pilot episode only
	MonitorModeNone         MonitorMode = "none"          // Don't monitor
)

// SearchType defines the type of search
type SearchType string

const (
	SearchTypeAutomatic SearchType = "automatic" // Automatic scheduled search
	SearchTypeManual    SearchType = "manual"    // Manual user-initiated search
	SearchTypeRSS       SearchType = "rss"       // RSS feed sync
	SearchTypeBacklog   SearchType = "backlog"   // Backlog search for missing items
)

// TriggerSource defines what triggered a search
type TriggerSource string

const (
	TriggerSourceUser      TriggerSource = "user"          // User action
	TriggerSourceScheduler TriggerSource = "scheduler"     // Scheduler job
	TriggerSourceRSSSync   TriggerSource = "rss_sync"      // RSS sync
	TriggerSourceMissing   TriggerSource = "missing_check" // Missing items check
)

// SearchStatus defines the status of a search
type SearchStatus string

const (
	SearchStatusPending   SearchStatus = "pending"   // Search queued
	SearchStatusCompleted SearchStatus = "completed" // Search completed
	SearchStatusFailed    SearchStatus = "failed"    // Search failed
)

// BlockReason defines why a release was blocked
type BlockReason string

const (
	BlockReasonQuality        BlockReason = "quality"         // Quality doesn't meet profile
	BlockReasonFake           BlockReason = "fake"            // Suspected fake release
	BlockReasonCorrupted      BlockReason = "corrupted"       // Corrupted file
	BlockReasonFailedDownload BlockReason = "failed_download" // Download failed
	BlockReasonManual         BlockReason = "manual"          // Manually blocked
	BlockReasonDuplicate      BlockReason = "duplicate"       // Duplicate release
	BlockReasonSize           BlockReason = "size"            // File size issues
	BlockReasonIndexer        BlockReason = "indexer"         // Indexer-related issue
)

// EventType defines calendar event types
type EventType string

const (
	EventTypeAirDate         EventType = "air_date"         // Episode air date
	EventTypeDigitalRelease  EventType = "digital_release"  // Digital release date
	EventTypePhysicalRelease EventType = "physical_release" // Physical media release date
)

// JobType defines scheduler job types
type JobType string

const (
	JobTypeRecurring JobType = "recurring" // Recurring job
	JobTypeOneTime   JobType = "one_time"  // One-time job
)

// JobStatus defines job execution status
type JobStatus string

const (
	JobStatusSuccess JobStatus = "success" // Job succeeded
	JobStatusFailed  JobStatus = "failed"  // Job failed
	JobStatusSkipped JobStatus = "skipped" // Job was skipped
)

// MonitoringRule represents a monitoring rule for a media item
type MonitoringRule struct {
	ID             int64       `json:"id"`
	MediaItemID    int64       `json:"media_item_id"`
	Enabled        bool        `json:"enabled"`
	QualityProfile *int        `json:"quality_profile_id"`
	MonitorMode    MonitorMode `json:"monitor_mode"`

	// Search settings
	SearchOnAdd     bool `json:"search_on_add"`
	AutomaticSearch bool `json:"automatic_search"`
	BacklogSearch   bool `json:"backlog_search"`

	// Release preferences
	PreferSeasonPacks bool     `json:"prefer_season_packs"`
	MinimumSeeders    int      `json:"minimum_seeders"`
	Tags              []string `json:"tags"`

	// Schedule
	SearchIntervalMinutes int        `json:"search_interval_minutes"`
	LastSearchAt          *time.Time `json:"last_search_at"`
	NextSearchAt          *time.Time `json:"next_search_at"`

	// Statistics
	SearchCount       int `json:"search_count"`
	ItemsFoundCount   int `json:"items_found_count"`
	ItemsGrabbedCount int `json:"items_grabbed_count"`

	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
	CreatedByUser *int64    `json:"created_by_user_id"`
}

// EpisodeMonitoring tracks monitoring for individual episodes
type EpisodeMonitoring struct {
	ID           int64      `json:"id"`
	MediaItemID  int64      `json:"media_item_id"`
	Monitored    bool       `json:"monitored"`
	HasFile      bool       `json:"has_file"`
	FileID       *int64     `json:"file_id"`
	AirDate      *time.Time `json:"air_date"`
	AirDateUTC   *time.Time `json:"air_date_utc"`
	SearchCount  int        `json:"search_count"`
	LastSearchAt *time.Time `json:"last_search_at"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

// SearchHistory tracks search executions
type SearchHistory struct {
	ID               int64          `json:"id"`
	MonitoringRuleID *int64         `json:"monitoring_rule_id"`
	MediaItemID      int64          `json:"media_item_id"`
	SearchType       SearchType     `json:"search_type"`
	TriggerSource    *TriggerSource `json:"trigger_source"`
	Query            *string        `json:"query"`

	// Results
	ResultsFound    int     `json:"results_found"`
	ResultsApproved int     `json:"results_approved"`
	ResultsRejected int     `json:"results_rejected"`
	DownloadGrabbed bool    `json:"download_grabbed"`
	DownloadID      *string `json:"download_id"`

	// Timing
	SearchDurationMs *int `json:"search_duration_ms"`

	// Status
	Status       SearchStatus `json:"status"`
	ErrorMessage *string      `json:"error_message"`

	Metadata      map[string]interface{} `json:"metadata"`
	CreatedAt     time.Time              `json:"created_at"`
	CreatedByUser *int64                 `json:"created_by_user_id"`
}

// BlocklistEntry represents a blocked release
type BlocklistEntry struct {
	ID              int64       `json:"id"`
	MediaItemID     *int64      `json:"media_item_id"`
	ReleaseHash     string      `json:"release_hash"`
	ReleaseTitle    string      `json:"release_title"`
	IndexerID       *string     `json:"indexer_id"`
	Reason          BlockReason `json:"reason"`
	Message         *string     `json:"message"`
	Permanent       bool        `json:"permanent"`
	ExpiresAt       *time.Time  `json:"expires_at"`
	DownloadID      *string     `json:"download_id"`
	SearchHistoryID *int64      `json:"search_history_id"`
	CreatedAt       time.Time   `json:"created_at"`
	CreatedByUser   *int64      `json:"created_by_user_id"`
}

// RSSSyncState tracks RSS feed synchronization state
type RSSSyncState struct {
	ID                  int64      `json:"id"`
	IndexerID           string     `json:"indexer_id"`
	LastSyncAt          *time.Time `json:"last_sync_at"`
	NextSyncAt          *time.Time `json:"next_sync_at"`
	SyncIntervalMinutes int        `json:"sync_interval_minutes"`
	TotalSyncs          int        `json:"total_syncs"`
	TotalItemsFound     int        `json:"total_items_found"`
	TotalItemsGrabbed   int        `json:"total_items_grabbed"`
	ConsecutiveFailures int        `json:"consecutive_failures"`
	LastError           *string    `json:"last_error"`
	Enabled             bool       `json:"enabled"`
	CreatedAt           time.Time  `json:"created_at"`
	UpdatedAt           time.Time  `json:"updated_at"`
}

// CalendarEvent represents an upcoming or recent release
type CalendarEvent struct {
	ID               int64                  `json:"id"`
	MediaItemID      int64                  `json:"media_item_id"`
	EventType        EventType              `json:"event_type"`
	EventDate        time.Time              `json:"event_date"`
	EventDateTimeUTC *time.Time             `json:"event_datetime_utc"`
	Monitored        bool                   `json:"monitored"`
	HasFile          bool                   `json:"has_file"`
	Downloaded       bool                   `json:"downloaded"`
	Title            string                 `json:"title"`
	ParentTitle      *string                `json:"parent_title"`
	Metadata         map[string]interface{} `json:"metadata"`
	CreatedAt        time.Time              `json:"created_at"`
	UpdatedAt        time.Time              `json:"updated_at"`
}

// SchedulerJob represents a background job
type SchedulerJob struct {
	ID                  int64                  `json:"id"`
	JobName             string                 `json:"job_name"`
	JobType             JobType                `json:"job_type"`
	CronExpression      *string                `json:"cron_expression"`
	IntervalMinutes     *int                   `json:"interval_minutes"`
	NextRunAt           *time.Time             `json:"next_run_at"`
	LastRunAt           *time.Time             `json:"last_run_at"`
	LastRunDurationMs   *int                   `json:"last_run_duration_ms"`
	Enabled             bool                   `json:"enabled"`
	Running             bool                   `json:"running"`
	TotalRuns           int                    `json:"total_runs"`
	ConsecutiveFailures int                    `json:"consecutive_failures"`
	LastStatus          *JobStatus             `json:"last_status"`
	LastError           *string                `json:"last_error"`
	Config              map[string]interface{} `json:"config"`
	CreatedAt           time.Time              `json:"created_at"`
	UpdatedAt           time.Time              `json:"updated_at"`
}

// SchedulerJobHistory tracks job execution history
type SchedulerJobHistory struct {
	ID             int64                  `json:"id"`
	JobID          int64                  `json:"job_id"`
	StartedAt      time.Time              `json:"started_at"`
	FinishedAt     *time.Time             `json:"finished_at"`
	DurationMs     *int                   `json:"duration_ms"`
	Status         JobStatus              `json:"status"`
	ErrorMessage   *string                `json:"error_message"`
	ItemsProcessed int                    `json:"items_processed"`
	LogEntries     []interface{}          `json:"log_entries"`
	Metadata       map[string]interface{} `json:"metadata"`
	CreatedAt      time.Time              `json:"created_at"`
}

// CreateMonitoringRuleParams defines parameters for creating a monitoring rule
type CreateMonitoringRuleParams struct {
	MediaItemID           int64       `json:"media_item_id"`
	Enabled               bool        `json:"enabled"`
	QualityProfileID      *int        `json:"quality_profile_id"`
	MonitorMode           MonitorMode `json:"monitor_mode"`
	SearchOnAdd           bool        `json:"search_on_add"`
	AutomaticSearch       bool        `json:"automatic_search"`
	BacklogSearch         bool        `json:"backlog_search"`
	PreferSeasonPacks     bool        `json:"prefer_season_packs"`
	MinimumSeeders        int         `json:"minimum_seeders"`
	Tags                  []string    `json:"tags"`
	SearchIntervalMinutes int         `json:"search_interval_minutes"`
	CreatedByUserID       *int64      `json:"created_by_user_id"`
}

// UpdateMonitoringRuleParams defines parameters for updating a monitoring rule
type UpdateMonitoringRuleParams struct {
	Enabled               *bool        `json:"enabled"`
	QualityProfileID      *int         `json:"quality_profile_id"`
	MonitorMode           *MonitorMode `json:"monitor_mode"`
	SearchOnAdd           *bool        `json:"search_on_add"`
	AutomaticSearch       *bool        `json:"automatic_search"`
	BacklogSearch         *bool        `json:"backlog_search"`
	PreferSeasonPacks     *bool        `json:"prefer_season_packs"`
	MinimumSeeders        *int         `json:"minimum_seeders"`
	Tags                  []string     `json:"tags"`
	SearchIntervalMinutes *int         `json:"search_interval_minutes"`
}

// CreateBlocklistEntryParams defines parameters for creating a blocklist entry
type CreateBlocklistEntryParams struct {
	MediaItemID     *int64      `json:"media_item_id"`
	ReleaseHash     string      `json:"release_hash"`
	ReleaseTitle    string      `json:"release_title"`
	IndexerID       *string     `json:"indexer_id"`
	Reason          BlockReason `json:"reason"`
	Message         *string     `json:"message"`
	Permanent       bool        `json:"permanent"`
	ExpiresAt       *time.Time  `json:"expires_at"`
	DownloadID      *string     `json:"download_id"`
	SearchHistoryID *int64      `json:"search_history_id"`
	CreatedByUserID *int64      `json:"created_by_user_id"`
}

// MonitoringStats represents monitoring statistics
type MonitoringStats struct {
	TotalMonitored      int `json:"total_monitored"`
	TotalEnabled        int `json:"total_enabled"`
	TotalMissing        int `json:"total_missing"`
	TotalDownloading    int `json:"total_downloading"`
	SearchesLast24Hours int `json:"searches_last_24_hours"`
	GrabbedLast24Hours  int `json:"grabbed_last_24_hours"`
}
