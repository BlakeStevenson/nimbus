package monitoring

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Scheduler manages background job execution
type Scheduler struct {
	db            *pgxpool.Pool
	monitoringSvc *Service
	stopChan      chan struct{}
	running       bool
	jobHandlers   map[string]JobHandler
	tickInterval  time.Duration
}

// JobHandler is a function that handles a job execution
type JobHandler func(ctx context.Context, job *SchedulerJob) error

// NewScheduler creates a new scheduler
func NewScheduler(db *pgxpool.Pool, monitoringSvc *Service) *Scheduler {
	return &Scheduler{
		db:            db,
		monitoringSvc: monitoringSvc,
		stopChan:      make(chan struct{}),
		jobHandlers:   make(map[string]JobHandler),
		tickInterval:  30 * time.Second, // Check for jobs every 30 seconds
	}
}

// RegisterJobHandler registers a handler for a job type
func (s *Scheduler) RegisterJobHandler(jobName string, handler JobHandler) {
	s.jobHandlers[jobName] = handler
}

// Start starts the scheduler
func (s *Scheduler) Start(ctx context.Context) error {
	if s.running {
		return fmt.Errorf("scheduler already running")
	}

	s.running = true

	// Register default job handlers
	s.registerDefaultHandlers()

	go s.run(ctx)

	return nil
}

// Stop stops the scheduler
func (s *Scheduler) Stop() {
	if !s.running {
		return
	}

	close(s.stopChan)
	s.running = false
}

// run is the main scheduler loop
func (s *Scheduler) run(ctx context.Context) {
	ticker := time.NewTicker(s.tickInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-s.stopChan:
			return
		case <-ticker.C:
			s.processDueJobs(ctx)
		}
	}
}

// processDueJobs processes jobs that are due to run
func (s *Scheduler) processDueJobs(ctx context.Context) {
	jobs, err := s.GetDueJobs(ctx)
	if err != nil {
		fmt.Printf("failed to get due jobs: %v\n", err)
		return
	}

	for _, job := range jobs {
		// Execute job in goroutine to avoid blocking
		go s.executeJob(ctx, &job)
	}
}

// executeJob executes a single job
func (s *Scheduler) executeJob(ctx context.Context, job *SchedulerJob) {
	// Mark job as running
	if err := s.markJobRunning(ctx, job.ID, true); err != nil {
		fmt.Printf("failed to mark job as running: %v\n", err)
		return
	}

	startTime := time.Now()

	// Create job history entry
	historyID, err := s.createJobHistory(ctx, job.ID, startTime)
	if err != nil {
		fmt.Printf("failed to create job history: %v\n", err)
		s.markJobRunning(ctx, job.ID, false)
		return
	}

	// Execute job handler
	var execErr error
	handler, ok := s.jobHandlers[job.JobName]
	if !ok {
		execErr = fmt.Errorf("no handler registered for job: %s", job.JobName)
	} else {
		execErr = handler(ctx, job)
	}

	finishTime := time.Now()
	duration := int(finishTime.Sub(startTime).Milliseconds())

	// Update job history
	status := JobStatusSuccess
	var errorMsg *string
	if execErr != nil {
		status = JobStatusFailed
		msg := execErr.Error()
		errorMsg = &msg
	}

	if err := s.updateJobHistory(ctx, historyID, finishTime, duration, status, errorMsg); err != nil {
		fmt.Printf("failed to update job history: %v\n", err)
	}

	// Update job status
	if err := s.updateJobStatus(ctx, job.ID, finishTime, duration, status, errorMsg); err != nil {
		fmt.Printf("failed to update job status: %v\n", err)
	}

	// Mark job as not running
	if err := s.markJobRunning(ctx, job.ID, false); err != nil {
		fmt.Printf("failed to mark job as not running: %v\n", err)
	}
}

// GetDueJobs gets jobs that are due to run
func (s *Scheduler) GetDueJobs(ctx context.Context) ([]SchedulerJob, error) {
	query := `
		SELECT id, job_name, job_type, cron_expression, interval_minutes,
		       next_run_at, last_run_at, last_run_duration_ms, enabled, running,
		       total_runs, consecutive_failures, last_status, last_error, config,
		       created_at, updated_at
		FROM scheduler_jobs
		WHERE enabled = true
		  AND running = false
		  AND (next_run_at IS NULL OR next_run_at <= NOW())
		ORDER BY next_run_at ASC NULLS FIRST
		LIMIT 10
	`

	rows, err := s.db.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to get due jobs: %w", err)
	}
	defer rows.Close()

	var jobs []SchedulerJob
	for rows.Next() {
		var job SchedulerJob
		var configJSON []byte

		err := rows.Scan(
			&job.ID, &job.JobName, &job.JobType, &job.CronExpression, &job.IntervalMinutes,
			&job.NextRunAt, &job.LastRunAt, &job.LastRunDurationMs, &job.Enabled, &job.Running,
			&job.TotalRuns, &job.ConsecutiveFailures, &job.LastStatus, &job.LastError, &configJSON,
			&job.CreatedAt, &job.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan job: %w", err)
		}

		if len(configJSON) > 0 {
			if err := json.Unmarshal(configJSON, &job.Config); err != nil {
				return nil, fmt.Errorf("failed to unmarshal config: %w", err)
			}
		}

		jobs = append(jobs, job)
	}

	return jobs, rows.Err()
}

// markJobRunning marks a job as running or not running
func (s *Scheduler) markJobRunning(ctx context.Context, jobID int64, running bool) error {
	query := `UPDATE scheduler_jobs SET running = $1 WHERE id = $2`
	_, err := s.db.Exec(ctx, query, running, jobID)
	return err
}

// createJobHistory creates a job history entry
func (s *Scheduler) createJobHistory(ctx context.Context, jobID int64, startTime time.Time) (int64, error) {
	query := `
		INSERT INTO scheduler_job_history (job_id, started_at, status)
		VALUES ($1, $2, $3)
		RETURNING id
	`

	var id int64
	err := s.db.QueryRow(ctx, query, jobID, startTime, JobStatusSuccess).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("failed to create job history: %w", err)
	}

	return id, nil
}

// updateJobHistory updates a job history entry
func (s *Scheduler) updateJobHistory(ctx context.Context, historyID int64, finishTime time.Time, durationMs int, status JobStatus, errorMsg *string) error {
	query := `
		UPDATE scheduler_job_history
		SET finished_at = $1,
		    duration_ms = $2,
		    status = $3,
		    error_message = $4
		WHERE id = $5
	`

	_, err := s.db.Exec(ctx, query, finishTime, durationMs, status, errorMsg, historyID)
	return err
}

// updateJobStatus updates job status after execution
func (s *Scheduler) updateJobStatus(ctx context.Context, jobID int64, finishTime time.Time, durationMs int, status JobStatus, errorMsg *string) error {
	query := `
		UPDATE scheduler_jobs
		SET last_run_at = $1,
		    last_run_duration_ms = $2,
		    last_status = $3,
		    last_error = $4,
		    next_run_at = CASE
		        WHEN interval_minutes IS NOT NULL THEN $1::timestamptz + (interval_minutes || ' minutes')::INTERVAL
		        ELSE next_run_at
		    END,
		    total_runs = total_runs + 1,
		    consecutive_failures = CASE
		        WHEN $3 = 'failed' THEN consecutive_failures + 1
		        ELSE 0
		    END
		WHERE id = $5
	`

	_, err := s.db.Exec(ctx, query, finishTime, durationMs, status, errorMsg, jobID)
	return err
}

// registerDefaultHandlers registers default job handlers
func (s *Scheduler) registerDefaultHandlers() {
	// RSS sync handler
	s.RegisterJobHandler("rss_sync", s.handleRSSSync)

	// Backlog search handler
	s.RegisterJobHandler("backlog_search", s.handleBacklogSearch)

	// Calendar update handler
	s.RegisterJobHandler("calendar_update", s.handleCalendarUpdate)

	// Monitoring check handler
	s.RegisterJobHandler("monitoring_check", s.handleMonitoringCheck)

	// Download cleanup handler
	s.RegisterJobHandler("download_cleanup", s.handleDownloadCleanup)

	// Blocklist cleanup handler
	s.RegisterJobHandler("blocklist_cleanup", s.handleBlocklistCleanup)

	// Quality upgrade search handler
	s.RegisterJobHandler("quality_upgrade_search", s.handleQualityUpgradeSearch)
}

// ========================
// Job Handlers
// ========================

// handleRSSSync handles RSS feed synchronization
func (s *Scheduler) handleRSSSync(ctx context.Context, job *SchedulerJob) error {
	// TODO: Implement RSS sync logic
	// This would:
	// 1. Get all enabled indexers
	// 2. Fetch RSS feeds from each
	// 3. Parse new releases
	// 4. Match against monitored items
	// 5. Download if quality matches

	fmt.Printf("Executing RSS sync job\n")
	return nil
}

// handleBacklogSearch handles backlog searching for missing items
func (s *Scheduler) handleBacklogSearch(ctx context.Context, job *SchedulerJob) error {
	// Get configuration
	maxItems := 50
	if val, ok := job.Config["max_items_per_run"].(float64); ok {
		maxItems = int(val)
	}

	// Get missing episodes
	missingEpisodes, err := s.monitoringSvc.GetMissingEpisodes(ctx, maxItems)
	if err != nil {
		return fmt.Errorf("failed to get missing episodes: %w", err)
	}

	fmt.Printf("Backlog search: found %d missing episodes\n", len(missingEpisodes))

	// TODO: Implement actual search logic
	// For each missing episode:
	// 1. Get monitoring rule
	// 2. Search indexers
	// 3. Filter by quality profile
	// 4. Download best match

	return nil
}

// handleCalendarUpdate handles calendar event updates
func (s *Scheduler) handleCalendarUpdate(ctx context.Context, job *SchedulerJob) error {
	// TODO: Implement calendar update logic
	// This would:
	// 1. Query TVDB/TMDB for upcoming episodes
	// 2. Update calendar_events table
	// 3. Create episode_monitoring records for new episodes

	fmt.Printf("Executing calendar update job\n")
	return nil
}

// handleMonitoringCheck handles monitoring checks for new releases
func (s *Scheduler) handleMonitoringCheck(ctx context.Context, job *SchedulerJob) error {
	// Get monitoring rules due for search
	rules, err := s.monitoringSvc.GetMonitoringRulesDueForSearch(ctx)
	if err != nil {
		return fmt.Errorf("failed to get monitoring rules: %w", err)
	}

	fmt.Printf("Monitoring check: found %d rules due for search\n", len(rules))

	// TODO: Implement monitoring check logic
	// For each rule:
	// 1. Search indexers for new releases
	// 2. Filter by quality profile
	// 3. Check blocklist
	// 4. Download if match found
	// 5. Update monitoring rule timestamps

	return nil
}

// handleDownloadCleanup handles download cleanup
func (s *Scheduler) handleDownloadCleanup(ctx context.Context, job *SchedulerJob) error {
	keepCompletedDays := 30
	if val, ok := job.Config["keep_completed_days"].(float64); ok {
		keepCompletedDays = int(val)
	}

	keepFailedDays := 7
	if val, ok := job.Config["keep_failed_days"].(float64); ok {
		keepFailedDays = int(val)
	}

	// Delete old completed downloads
	query := `
		DELETE FROM downloads
		WHERE status = 'completed'
		  AND completed_at < NOW() - ($1 || ' days')::INTERVAL
	`
	result, err := s.db.Exec(ctx, query, keepCompletedDays)
	if err != nil {
		return fmt.Errorf("failed to delete completed downloads: %w", err)
	}

	completedDeleted := result.RowsAffected()

	// Delete old failed downloads
	query = `
		DELETE FROM downloads
		WHERE status = 'failed'
		  AND updated_at < NOW() - ($1 || ' days')::INTERVAL
	`
	result, err = s.db.Exec(ctx, query, keepFailedDays)
	if err != nil {
		return fmt.Errorf("failed to delete failed downloads: %w", err)
	}

	failedDeleted := result.RowsAffected()

	fmt.Printf("Download cleanup: deleted %d completed, %d failed downloads\n", completedDeleted, failedDeleted)
	return nil
}

// handleBlocklistCleanup handles blocklist cleanup
func (s *Scheduler) handleBlocklistCleanup(ctx context.Context, job *SchedulerJob) error {
	// Delete expired temporary blocks
	query := `
		DELETE FROM blocklist
		WHERE permanent = false
		  AND expires_at < NOW()
	`

	result, err := s.db.Exec(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to delete expired blocklist entries: %w", err)
	}

	deleted := result.RowsAffected()
	fmt.Printf("Blocklist cleanup: deleted %d expired entries\n", deleted)
	return nil
}

// handleQualityUpgradeSearch handles quality upgrade searching
func (s *Scheduler) handleQualityUpgradeSearch(ctx context.Context, job *SchedulerJob) error {
	// TODO: Implement quality upgrade search logic
	// This would:
	// 1. Find media items with quality below cutoff
	// 2. Search for better quality releases
	// 3. Download if found and better than current

	fmt.Printf("Executing quality upgrade search job\n")
	return nil
}

// ========================
// Job Management
// ========================

// GetJob gets a scheduler job by ID
func (s *Scheduler) GetJob(ctx context.Context, id int64) (*SchedulerJob, error) {
	query := `
		SELECT id, job_name, job_type, cron_expression, interval_minutes,
		       next_run_at, last_run_at, last_run_duration_ms, enabled, running,
		       total_runs, consecutive_failures, last_status, last_error, config,
		       created_at, updated_at
		FROM scheduler_jobs
		WHERE id = $1
	`

	var job SchedulerJob
	var configJSON []byte

	err := s.db.QueryRow(ctx, query, id).Scan(
		&job.ID, &job.JobName, &job.JobType, &job.CronExpression, &job.IntervalMinutes,
		&job.NextRunAt, &job.LastRunAt, &job.LastRunDurationMs, &job.Enabled, &job.Running,
		&job.TotalRuns, &job.ConsecutiveFailures, &job.LastStatus, &job.LastError, &configJSON,
		&job.CreatedAt, &job.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("job not found")
		}
		return nil, fmt.Errorf("failed to get job: %w", err)
	}

	if len(configJSON) > 0 {
		if err := json.Unmarshal(configJSON, &job.Config); err != nil {
			return nil, fmt.Errorf("failed to unmarshal config: %w", err)
		}
	}

	return &job, nil
}

// ListJobs lists all scheduler jobs
func (s *Scheduler) ListJobs(ctx context.Context) ([]SchedulerJob, error) {
	query := `
		SELECT id, job_name, job_type, cron_expression, interval_minutes,
		       next_run_at, last_run_at, last_run_duration_ms, enabled, running,
		       total_runs, consecutive_failures, last_status, last_error, config,
		       created_at, updated_at
		FROM scheduler_jobs
		ORDER BY job_name
	`

	rows, err := s.db.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list jobs: %w", err)
	}
	defer rows.Close()

	var jobs []SchedulerJob
	for rows.Next() {
		var job SchedulerJob
		var configJSON []byte

		err := rows.Scan(
			&job.ID, &job.JobName, &job.JobType, &job.CronExpression, &job.IntervalMinutes,
			&job.NextRunAt, &job.LastRunAt, &job.LastRunDurationMs, &job.Enabled, &job.Running,
			&job.TotalRuns, &job.ConsecutiveFailures, &job.LastStatus, &job.LastError, &configJSON,
			&job.CreatedAt, &job.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan job: %w", err)
		}

		if len(configJSON) > 0 {
			if err := json.Unmarshal(configJSON, &job.Config); err != nil {
				return nil, fmt.Errorf("failed to unmarshal config: %w", err)
			}
		}

		jobs = append(jobs, job)
	}

	return jobs, rows.Err()
}

// TriggerJob manually triggers a job
func (s *Scheduler) TriggerJob(ctx context.Context, jobID int64) error {
	job, err := s.GetJob(ctx, jobID)
	if err != nil {
		return err
	}

	if job.Running {
		return fmt.Errorf("job is already running")
	}

	go s.executeJob(ctx, job)
	return nil
}
