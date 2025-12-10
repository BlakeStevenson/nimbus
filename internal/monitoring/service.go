package monitoring

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Service handles monitoring operations
type Service struct {
	db *pgxpool.Pool
}

// NewService creates a new monitoring service
func NewService(db *pgxpool.Pool) *Service {
	return &Service{
		db: db,
	}
}

// ========================
// Monitoring Rules
// ========================

// CreateMonitoringRule creates a new monitoring rule (or updates if exists)
func (s *Service) CreateMonitoringRule(ctx context.Context, params CreateMonitoringRuleParams) (*MonitoringRule, error) {
	query := `
		INSERT INTO monitoring_rules (
			media_item_id, enabled, quality_profile_id, monitor_mode,
			search_on_add, automatic_search, backlog_search,
			prefer_season_packs, minimum_seeders, tags,
			search_interval_minutes, created_by_user_id
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		ON CONFLICT (media_item_id) DO UPDATE SET
			enabled = EXCLUDED.enabled,
			quality_profile_id = EXCLUDED.quality_profile_id,
			monitor_mode = EXCLUDED.monitor_mode,
			search_on_add = EXCLUDED.search_on_add,
			automatic_search = EXCLUDED.automatic_search,
			backlog_search = EXCLUDED.backlog_search,
			prefer_season_packs = EXCLUDED.prefer_season_packs,
			minimum_seeders = EXCLUDED.minimum_seeders,
			tags = EXCLUDED.tags,
			search_interval_minutes = EXCLUDED.search_interval_minutes
		RETURNING id, media_item_id, enabled, quality_profile_id, monitor_mode,
		          search_on_add, automatic_search, backlog_search,
		          prefer_season_packs, minimum_seeders, tags,
		          search_interval_minutes, last_search_at, next_search_at,
		          search_count, items_found_count, items_grabbed_count,
		          created_at, updated_at, created_by_user_id
	`

	var rule MonitoringRule
	err := s.db.QueryRow(ctx, query,
		params.MediaItemID, params.Enabled, params.QualityProfileID, params.MonitorMode,
		params.SearchOnAdd, params.AutomaticSearch, params.BacklogSearch,
		params.PreferSeasonPacks, params.MinimumSeeders, params.Tags,
		params.SearchIntervalMinutes, params.CreatedByUserID,
	).Scan(
		&rule.ID, &rule.MediaItemID, &rule.Enabled, &rule.QualityProfile, &rule.MonitorMode,
		&rule.SearchOnAdd, &rule.AutomaticSearch, &rule.BacklogSearch,
		&rule.PreferSeasonPacks, &rule.MinimumSeeders, &rule.Tags,
		&rule.SearchIntervalMinutes, &rule.LastSearchAt, &rule.NextSearchAt,
		&rule.SearchCount, &rule.ItemsFoundCount, &rule.ItemsGrabbedCount,
		&rule.CreatedAt, &rule.UpdatedAt, &rule.CreatedByUser,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create monitoring rule: %w", err)
	}

	return &rule, nil
}

// GetMonitoringRule gets a monitoring rule by ID
func (s *Service) GetMonitoringRule(ctx context.Context, id int64) (*MonitoringRule, error) {
	query := `
		SELECT id, media_item_id, enabled, quality_profile_id, monitor_mode,
		       search_on_add, automatic_search, backlog_search,
		       prefer_season_packs, minimum_seeders, tags,
		       search_interval_minutes, last_search_at, next_search_at,
		       search_count, items_found_count, items_grabbed_count,
		       created_at, updated_at, created_by_user_id
		FROM monitoring_rules
		WHERE id = $1
	`

	var rule MonitoringRule
	err := s.db.QueryRow(ctx, query, id).Scan(
		&rule.ID, &rule.MediaItemID, &rule.Enabled, &rule.QualityProfile, &rule.MonitorMode,
		&rule.SearchOnAdd, &rule.AutomaticSearch, &rule.BacklogSearch,
		&rule.PreferSeasonPacks, &rule.MinimumSeeders, &rule.Tags,
		&rule.SearchIntervalMinutes, &rule.LastSearchAt, &rule.NextSearchAt,
		&rule.SearchCount, &rule.ItemsFoundCount, &rule.ItemsGrabbedCount,
		&rule.CreatedAt, &rule.UpdatedAt, &rule.CreatedByUser,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("monitoring rule not found")
		}
		return nil, fmt.Errorf("failed to get monitoring rule: %w", err)
	}

	return &rule, nil
}

// GetMonitoringRuleByMediaItem gets a monitoring rule by media item ID
func (s *Service) GetMonitoringRuleByMediaItem(ctx context.Context, mediaItemID int64) (*MonitoringRule, error) {
	query := `
		SELECT id, media_item_id, enabled, quality_profile_id, monitor_mode,
		       search_on_add, automatic_search, backlog_search,
		       prefer_season_packs, minimum_seeders, tags,
		       search_interval_minutes, last_search_at, next_search_at,
		       search_count, items_found_count, items_grabbed_count,
		       created_at, updated_at, created_by_user_id
		FROM monitoring_rules
		WHERE media_item_id = $1
	`

	var rule MonitoringRule
	err := s.db.QueryRow(ctx, query, mediaItemID).Scan(
		&rule.ID, &rule.MediaItemID, &rule.Enabled, &rule.QualityProfile, &rule.MonitorMode,
		&rule.SearchOnAdd, &rule.AutomaticSearch, &rule.BacklogSearch,
		&rule.PreferSeasonPacks, &rule.MinimumSeeders, &rule.Tags,
		&rule.SearchIntervalMinutes, &rule.LastSearchAt, &rule.NextSearchAt,
		&rule.SearchCount, &rule.ItemsFoundCount, &rule.ItemsGrabbedCount,
		&rule.CreatedAt, &rule.UpdatedAt, &rule.CreatedByUser,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("monitoring rule not found")
		}
		return nil, fmt.Errorf("failed to get monitoring rule: %w", err)
	}

	return &rule, nil
}

// ListMonitoringRules lists all monitoring rules with optional filters
func (s *Service) ListMonitoringRules(ctx context.Context, enabledOnly bool) ([]MonitoringRule, error) {
	query := `
		SELECT id, media_item_id, enabled, quality_profile_id, monitor_mode,
		       search_on_add, automatic_search, backlog_search,
		       prefer_season_packs, minimum_seeders, tags,
		       search_interval_minutes, last_search_at, next_search_at,
		       search_count, items_found_count, items_grabbed_count,
		       created_at, updated_at, created_by_user_id
		FROM monitoring_rules
	`

	if enabledOnly {
		query += " WHERE enabled = true"
	}

	query += " ORDER BY created_at DESC"

	rows, err := s.db.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list monitoring rules: %w", err)
	}
	defer rows.Close()

	var rules []MonitoringRule
	for rows.Next() {
		var rule MonitoringRule
		err := rows.Scan(
			&rule.ID, &rule.MediaItemID, &rule.Enabled, &rule.QualityProfile, &rule.MonitorMode,
			&rule.SearchOnAdd, &rule.AutomaticSearch, &rule.BacklogSearch,
			&rule.PreferSeasonPacks, &rule.MinimumSeeders, &rule.Tags,
			&rule.SearchIntervalMinutes, &rule.LastSearchAt, &rule.NextSearchAt,
			&rule.SearchCount, &rule.ItemsFoundCount, &rule.ItemsGrabbedCount,
			&rule.CreatedAt, &rule.UpdatedAt, &rule.CreatedByUser,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan monitoring rule: %w", err)
		}
		rules = append(rules, rule)
	}

	return rules, rows.Err()
}

// UpdateMonitoringRule updates a monitoring rule
func (s *Service) UpdateMonitoringRule(ctx context.Context, id int64, params UpdateMonitoringRuleParams) (*MonitoringRule, error) {
	query := `
		UPDATE monitoring_rules
		SET enabled = COALESCE($1, enabled),
		    quality_profile_id = COALESCE($2, quality_profile_id),
		    monitor_mode = COALESCE($3, monitor_mode),
		    search_on_add = COALESCE($4, search_on_add),
		    automatic_search = COALESCE($5, automatic_search),
		    backlog_search = COALESCE($6, backlog_search),
		    prefer_season_packs = COALESCE($7, prefer_season_packs),
		    minimum_seeders = COALESCE($8, minimum_seeders),
		    tags = COALESCE($9, tags),
		    search_interval_minutes = COALESCE($10, search_interval_minutes)
		WHERE id = $11
		RETURNING id, media_item_id, enabled, quality_profile_id, monitor_mode,
		          search_on_add, automatic_search, backlog_search,
		          prefer_season_packs, minimum_seeders, tags,
		          search_interval_minutes, last_search_at, next_search_at,
		          search_count, items_found_count, items_grabbed_count,
		          created_at, updated_at, created_by_user_id
	`

	var rule MonitoringRule

	err := s.db.QueryRow(ctx, query,
		params.Enabled, params.QualityProfileID, params.MonitorMode,
		params.SearchOnAdd, params.AutomaticSearch, params.BacklogSearch,
		params.PreferSeasonPacks, params.MinimumSeeders, params.Tags,
		params.SearchIntervalMinutes, id,
	).Scan(
		&rule.ID, &rule.MediaItemID, &rule.Enabled, &rule.QualityProfile, &rule.MonitorMode,
		&rule.SearchOnAdd, &rule.AutomaticSearch, &rule.BacklogSearch,
		&rule.PreferSeasonPacks, &rule.MinimumSeeders, &rule.Tags,
		&rule.SearchIntervalMinutes, &rule.LastSearchAt, &rule.NextSearchAt,
		&rule.SearchCount, &rule.ItemsFoundCount, &rule.ItemsGrabbedCount,
		&rule.CreatedAt, &rule.UpdatedAt, &rule.CreatedByUser,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("monitoring rule not found")
		}
		return nil, fmt.Errorf("failed to update monitoring rule: %w", err)
	}

	return &rule, nil
}

// DeleteMonitoringRule deletes a monitoring rule
func (s *Service) DeleteMonitoringRule(ctx context.Context, id int64) error {
	query := `DELETE FROM monitoring_rules WHERE id = $1`

	result, err := s.db.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete monitoring rule: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("monitoring rule not found")
	}

	return nil
}

// GetMonitoringRulesDueForSearch returns monitoring rules that need to be searched
func (s *Service) GetMonitoringRulesDueForSearch(ctx context.Context) ([]MonitoringRule, error) {
	query := `
		SELECT id, media_item_id, enabled, quality_profile_id, monitor_mode,
		       search_on_add, automatic_search, backlog_search,
		       prefer_season_packs, minimum_seeders, tags,
		       search_interval_minutes, last_search_at, next_search_at,
		       search_count, items_found_count, items_grabbed_count,
		       created_at, updated_at, created_by_user_id
		FROM monitoring_rules
		WHERE enabled = true
		  AND automatic_search = true
		  AND (next_search_at IS NULL OR next_search_at <= NOW())
		ORDER BY next_search_at ASC NULLS FIRST
		LIMIT 100
	`

	rows, err := s.db.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to get monitoring rules due for search: %w", err)
	}
	defer rows.Close()

	var rules []MonitoringRule
	for rows.Next() {
		var rule MonitoringRule
		err := rows.Scan(
			&rule.ID, &rule.MediaItemID, &rule.Enabled, &rule.QualityProfile, &rule.MonitorMode,
			&rule.SearchOnAdd, &rule.AutomaticSearch, &rule.BacklogSearch,
			&rule.PreferSeasonPacks, &rule.MinimumSeeders, &rule.Tags,
			&rule.SearchIntervalMinutes, &rule.LastSearchAt, &rule.NextSearchAt,
			&rule.SearchCount, &rule.ItemsFoundCount, &rule.ItemsGrabbedCount,
			&rule.CreatedAt, &rule.UpdatedAt, &rule.CreatedByUser,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan monitoring rule: %w", err)
		}
		rules = append(rules, rule)
	}

	return rules, rows.Err()
}

// UpdateMonitoringRuleSearchTime updates the search timestamps for a monitoring rule
func (s *Service) UpdateMonitoringRuleSearchTime(ctx context.Context, id int64) error {
	query := `
		UPDATE monitoring_rules
		SET last_search_at = NOW(),
		    next_search_at = NOW() + (search_interval_minutes || ' minutes')::INTERVAL,
		    search_count = search_count + 1
		WHERE id = $1
	`

	_, err := s.db.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to update monitoring rule search time: %w", err)
	}

	return nil
}

// ========================
// Episode Monitoring
// ========================

// CreateEpisodeMonitoring creates episode monitoring record
func (s *Service) CreateEpisodeMonitoring(ctx context.Context, mediaItemID int64, monitored bool) (*EpisodeMonitoring, error) {
	query := `
		INSERT INTO episode_monitoring (media_item_id, monitored)
		VALUES ($1, $2)
		ON CONFLICT (media_item_id) DO UPDATE
		SET monitored = EXCLUDED.monitored
		RETURNING id, media_item_id, monitored, has_file, file_id,
		          air_date, air_date_utc, search_count, last_search_at,
		          created_at, updated_at
	`

	var em EpisodeMonitoring
	err := s.db.QueryRow(ctx, query, mediaItemID, monitored).Scan(
		&em.ID, &em.MediaItemID, &em.Monitored, &em.HasFile, &em.FileID,
		&em.AirDate, &em.AirDateUTC, &em.SearchCount, &em.LastSearchAt,
		&em.CreatedAt, &em.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create episode monitoring: %w", err)
	}

	return &em, nil
}

// GetMissingEpisodes returns monitored episodes without files
func (s *Service) GetMissingEpisodes(ctx context.Context, limit int) ([]EpisodeMonitoring, error) {
	query := `
		SELECT id, media_item_id, monitored, has_file, file_id,
		       air_date, air_date_utc, search_count, last_search_at,
		       created_at, updated_at
		FROM episode_monitoring
		WHERE monitored = true
		  AND has_file = false
		  AND (air_date IS NULL OR air_date <= CURRENT_DATE)
		ORDER BY air_date DESC NULLS LAST
		LIMIT $1
	`

	rows, err := s.db.Query(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get missing episodes: %w", err)
	}
	defer rows.Close()

	var episodes []EpisodeMonitoring
	for rows.Next() {
		var em EpisodeMonitoring
		err := rows.Scan(
			&em.ID, &em.MediaItemID, &em.Monitored, &em.HasFile, &em.FileID,
			&em.AirDate, &em.AirDateUTC, &em.SearchCount, &em.LastSearchAt,
			&em.CreatedAt, &em.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan episode monitoring: %w", err)
		}
		episodes = append(episodes, em)
	}

	return episodes, rows.Err()
}

// ========================
// Search History
// ========================

// CreateSearchHistory creates a search history record
func (s *Service) CreateSearchHistory(ctx context.Context, history *SearchHistory) (*SearchHistory, error) {
	metadataJSON, err := json.Marshal(history.Metadata)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal metadata: %w", err)
	}

	query := `
		INSERT INTO search_history (
			monitoring_rule_id, media_item_id, search_type, trigger_source, query,
			results_found, results_approved, results_rejected, download_grabbed, download_id,
			search_duration_ms, status, error_message, metadata, created_by_user_id
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
		RETURNING id, created_at
	`

	err = s.db.QueryRow(ctx, query,
		history.MonitoringRuleID, history.MediaItemID, history.SearchType, history.TriggerSource, history.Query,
		history.ResultsFound, history.ResultsApproved, history.ResultsRejected, history.DownloadGrabbed, history.DownloadID,
		history.SearchDurationMs, history.Status, history.ErrorMessage, metadataJSON, history.CreatedByUser,
	).Scan(&history.ID, &history.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to create search history: %w", err)
	}

	return history, nil
}

// GetSearchHistory gets search history for a media item
func (s *Service) GetSearchHistory(ctx context.Context, mediaItemID int64, limit int) ([]SearchHistory, error) {
	query := `
		SELECT id, monitoring_rule_id, media_item_id, search_type, trigger_source, query,
		       results_found, results_approved, results_rejected, download_grabbed, download_id,
		       search_duration_ms, status, error_message, metadata, created_at, created_by_user_id
		FROM search_history
		WHERE media_item_id = $1
		ORDER BY created_at DESC
		LIMIT $2
	`

	rows, err := s.db.Query(ctx, query, mediaItemID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get search history: %w", err)
	}
	defer rows.Close()

	var histories []SearchHistory
	for rows.Next() {
		var history SearchHistory
		var metadataJSON []byte

		err := rows.Scan(
			&history.ID, &history.MonitoringRuleID, &history.MediaItemID, &history.SearchType,
			&history.TriggerSource, &history.Query, &history.ResultsFound, &history.ResultsApproved,
			&history.ResultsRejected, &history.DownloadGrabbed, &history.DownloadID,
			&history.SearchDurationMs, &history.Status, &history.ErrorMessage,
			&metadataJSON, &history.CreatedAt, &history.CreatedByUser,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan search history: %w", err)
		}

		if len(metadataJSON) > 0 {
			if err := json.Unmarshal(metadataJSON, &history.Metadata); err != nil {
				return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
			}
		}

		histories = append(histories, history)
	}

	return histories, rows.Err()
}

// ========================
// Blocklist
// ========================

// CreateBlocklistEntry creates a blocklist entry
func (s *Service) CreateBlocklistEntry(ctx context.Context, params CreateBlocklistEntryParams) (*BlocklistEntry, error) {
	query := `
		INSERT INTO blocklist (
			media_item_id, release_hash, release_title, indexer_id, reason, message,
			permanent, expires_at, download_id, search_history_id, created_by_user_id
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		ON CONFLICT (release_hash, media_item_id) DO UPDATE
		SET reason = EXCLUDED.reason,
		    message = EXCLUDED.message,
		    permanent = EXCLUDED.permanent,
		    expires_at = EXCLUDED.expires_at
		RETURNING id, media_item_id, release_hash, release_title, indexer_id, reason, message,
		          permanent, expires_at, download_id, search_history_id, created_at, created_by_user_id
	`

	var entry BlocklistEntry
	err := s.db.QueryRow(ctx, query,
		params.MediaItemID, params.ReleaseHash, params.ReleaseTitle, params.IndexerID, params.Reason, params.Message,
		params.Permanent, params.ExpiresAt, params.DownloadID, params.SearchHistoryID, params.CreatedByUserID,
	).Scan(
		&entry.ID, &entry.MediaItemID, &entry.ReleaseHash, &entry.ReleaseTitle, &entry.IndexerID, &entry.Reason, &entry.Message,
		&entry.Permanent, &entry.ExpiresAt, &entry.DownloadID, &entry.SearchHistoryID, &entry.CreatedAt, &entry.CreatedByUser,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create blocklist entry: %w", err)
	}

	return &entry, nil
}

// IsBlocked checks if a release is blocked
func (s *Service) IsBlocked(ctx context.Context, releaseHash string, mediaItemID *int64) (bool, error) {
	query := `
		SELECT COUNT(*) > 0
		FROM blocklist
		WHERE release_hash = $1
		  AND (media_item_id = $2 OR media_item_id IS NULL)
		  AND (permanent = true OR expires_at > NOW())
	`

	var blocked bool
	err := s.db.QueryRow(ctx, query, releaseHash, mediaItemID).Scan(&blocked)
	if err != nil {
		return false, fmt.Errorf("failed to check if blocked: %w", err)
	}

	return blocked, nil
}

// ========================
// Calendar
// ========================

// GetCalendarEvents gets calendar events within a date range
func (s *Service) GetCalendarEvents(ctx context.Context, startDate, endDate time.Time, monitoredOnly bool) ([]CalendarEvent, error) {
	query := `
		SELECT id, media_item_id, event_type, event_date, event_datetime_utc,
		       monitored, has_file, downloaded, title, parent_title, metadata,
		       created_at, updated_at
		FROM calendar_events
		WHERE event_date >= $1 AND event_date <= $2
	`

	if monitoredOnly {
		query += " AND monitored = true"
	}

	query += " ORDER BY event_date ASC, title ASC"

	rows, err := s.db.Query(ctx, query, startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("failed to get calendar events: %w", err)
	}
	defer rows.Close()

	var events []CalendarEvent
	for rows.Next() {
		var event CalendarEvent
		var metadataJSON []byte

		err := rows.Scan(
			&event.ID, &event.MediaItemID, &event.EventType, &event.EventDate, &event.EventDateTimeUTC,
			&event.Monitored, &event.HasFile, &event.Downloaded, &event.Title, &event.ParentTitle,
			&metadataJSON, &event.CreatedAt, &event.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan calendar event: %w", err)
		}

		if len(metadataJSON) > 0 {
			if err := json.Unmarshal(metadataJSON, &event.Metadata); err != nil {
				return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
			}
		}

		events = append(events, event)
	}

	return events, rows.Err()
}

// ========================
// Statistics
// ========================

// GetMonitoringStats gets monitoring statistics
func (s *Service) GetMonitoringStats(ctx context.Context) (*MonitoringStats, error) {
	stats := &MonitoringStats{}

	// Total monitored
	err := s.db.QueryRow(ctx, `SELECT COUNT(*) FROM monitoring_rules`).Scan(&stats.TotalMonitored)
	if err != nil {
		return nil, fmt.Errorf("failed to get total monitored: %w", err)
	}

	// Total enabled
	err = s.db.QueryRow(ctx, `SELECT COUNT(*) FROM monitoring_rules WHERE enabled = true`).Scan(&stats.TotalEnabled)
	if err != nil {
		return nil, fmt.Errorf("failed to get total enabled: %w", err)
	}

	// Total missing
	err = s.db.QueryRow(ctx, `SELECT COUNT(*) FROM episode_monitoring WHERE monitored = true AND has_file = false`).Scan(&stats.TotalMissing)
	if err != nil {
		return nil, fmt.Errorf("failed to get total missing: %w", err)
	}

	// Total downloading
	err = s.db.QueryRow(ctx, `
		SELECT COUNT(DISTINCT d.media_item_id)
		FROM downloads d
		WHERE d.status IN ('queued', 'downloading', 'processing')
	`).Scan(&stats.TotalDownloading)
	if err != nil {
		return nil, fmt.Errorf("failed to get total downloading: %w", err)
	}

	// Searches last 24 hours
	err = s.db.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM search_history
		WHERE created_at >= NOW() - INTERVAL '24 hours'
	`).Scan(&stats.SearchesLast24Hours)
	if err != nil {
		return nil, fmt.Errorf("failed to get searches last 24 hours: %w", err)
	}

	// Grabbed last 24 hours
	err = s.db.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM search_history
		WHERE created_at >= NOW() - INTERVAL '24 hours'
		  AND download_grabbed = true
	`).Scan(&stats.GrabbedLast24Hours)
	if err != nil {
		return nil, fmt.Errorf("failed to get grabbed last 24 hours: %w", err)
	}

	return stats, nil
}
