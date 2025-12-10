package monitoring

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/blakestevenson/nimbus/internal/httputil"
	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"
)

// Handler handles HTTP requests for monitoring
type Handler struct {
	service   *Service
	scheduler *Scheduler
	logger    *zap.Logger
}

// NewHandler creates a new monitoring handler
func NewHandler(service *Service, scheduler *Scheduler, logger *zap.Logger) *Handler {
	return &Handler{
		service:   service,
		scheduler: scheduler,
		logger:    logger,
	}
}

// ========================
// Monitoring Rules
// ========================

// CreateMonitoringRule creates a new monitoring rule
func (h *Handler) CreateMonitoringRule(w http.ResponseWriter, r *http.Request) {
	var params CreateMonitoringRuleParams
	if err := json.NewDecoder(r.Body).Decode(&params); err != nil {
		httputil.RespondErrorMessage(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	rule, err := h.service.CreateMonitoringRule(r.Context(), params)
	if err != nil {
		h.logger.Error("Failed to create monitoring rule", zap.Error(err))
		httputil.RespondErrorMessage(w, http.StatusInternalServerError, "Failed to create monitoring rule")
		return
	}

	httputil.RespondJSON(w, http.StatusCreated, rule)
}

// GetMonitoringRule gets a monitoring rule by ID
func (h *Handler) GetMonitoringRule(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		httputil.RespondErrorMessage(w, http.StatusBadRequest, "Invalid rule ID")
		return
	}

	rule, err := h.service.GetMonitoringRule(r.Context(), id)
	if err != nil {
		h.logger.Error("Failed to get monitoring rule", zap.Error(err))
		httputil.RespondErrorMessage(w, http.StatusNotFound, "Monitoring rule not found")
		return
	}

	httputil.RespondJSON(w, http.StatusOK, rule)
}

// GetMonitoringRuleByMediaItem gets a monitoring rule by media item ID
func (h *Handler) GetMonitoringRuleByMediaItem(w http.ResponseWriter, r *http.Request) {
	mediaIDStr := chi.URLParam(r, "mediaId")
	mediaID, err := strconv.ParseInt(mediaIDStr, 10, 64)
	if err != nil {
		httputil.RespondErrorMessage(w, http.StatusBadRequest, "Invalid media item ID")
		return
	}

	rule, err := h.service.GetMonitoringRuleByMediaItem(r.Context(), mediaID)
	if err != nil {
		h.logger.Error("Failed to get monitoring rule by media item", zap.Error(err))
		httputil.RespondErrorMessage(w, http.StatusNotFound, "Monitoring rule not found")
		return
	}

	httputil.RespondJSON(w, http.StatusOK, rule)
}

// ListMonitoringRules lists all monitoring rules
func (h *Handler) ListMonitoringRules(w http.ResponseWriter, r *http.Request) {
	enabledOnlyStr := r.URL.Query().Get("enabled")
	enabledOnly := enabledOnlyStr == "true"

	rules, err := h.service.ListMonitoringRules(r.Context(), enabledOnly)
	if err != nil {
		h.logger.Error("Failed to list monitoring rules", zap.Error(err))
		httputil.RespondErrorMessage(w, http.StatusInternalServerError, "Failed to list monitoring rules")
		return
	}

	httputil.RespondJSON(w, http.StatusOK, rules)
}

// UpdateMonitoringRule updates a monitoring rule
func (h *Handler) UpdateMonitoringRule(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		httputil.RespondErrorMessage(w, http.StatusBadRequest, "Invalid rule ID")
		return
	}

	var params UpdateMonitoringRuleParams
	if err := json.NewDecoder(r.Body).Decode(&params); err != nil {
		httputil.RespondErrorMessage(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	rule, err := h.service.UpdateMonitoringRule(r.Context(), id, params)
	if err != nil {
		h.logger.Error("Failed to update monitoring rule", zap.Error(err))
		httputil.RespondErrorMessage(w, http.StatusInternalServerError, "Failed to update monitoring rule")
		return
	}

	httputil.RespondJSON(w, http.StatusOK, rule)
}

// DeleteMonitoringRule deletes a monitoring rule
func (h *Handler) DeleteMonitoringRule(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		httputil.RespondErrorMessage(w, http.StatusBadRequest, "Invalid rule ID")
		return
	}

	if err := h.service.DeleteMonitoringRule(r.Context(), id); err != nil {
		h.logger.Error("Failed to delete monitoring rule", zap.Error(err))
		httputil.RespondErrorMessage(w, http.StatusInternalServerError, "Failed to delete monitoring rule")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// ========================
// Episode Monitoring
// ========================

// GetMissingEpisodes gets missing episodes
func (h *Handler) GetMissingEpisodes(w http.ResponseWriter, r *http.Request) {
	limitStr := r.URL.Query().Get("limit")
	limit := 50
	if limitStr != "" {
		if parsedLimit, err := strconv.Atoi(limitStr); err == nil && parsedLimit > 0 {
			limit = parsedLimit
		}
	}

	episodes, err := h.service.GetMissingEpisodes(r.Context(), limit)
	if err != nil {
		h.logger.Error("Failed to get missing episodes", zap.Error(err))
		httputil.RespondErrorMessage(w, http.StatusInternalServerError, "Failed to get missing episodes")
		return
	}

	httputil.RespondJSON(w, http.StatusOK, episodes)
}

// ========================
// Search History
// ========================

// GetSearchHistory gets search history for a media item
func (h *Handler) GetSearchHistory(w http.ResponseWriter, r *http.Request) {
	mediaIDStr := chi.URLParam(r, "mediaId")
	mediaID, err := strconv.ParseInt(mediaIDStr, 10, 64)
	if err != nil {
		httputil.RespondErrorMessage(w, http.StatusBadRequest, "Invalid media item ID")
		return
	}

	limitStr := r.URL.Query().Get("limit")
	limit := 20
	if limitStr != "" {
		if parsedLimit, err := strconv.Atoi(limitStr); err == nil && parsedLimit > 0 {
			limit = parsedLimit
		}
	}

	history, err := h.service.GetSearchHistory(r.Context(), mediaID, limit)
	if err != nil {
		h.logger.Error("Failed to get search history", zap.Error(err))
		httputil.RespondErrorMessage(w, http.StatusInternalServerError, "Failed to get search history")
		return
	}

	httputil.RespondJSON(w, http.StatusOK, history)
}

// ========================
// Blocklist
// ========================

// CreateBlocklistEntry creates a blocklist entry
func (h *Handler) CreateBlocklistEntry(w http.ResponseWriter, r *http.Request) {
	var params CreateBlocklistEntryParams
	if err := json.NewDecoder(r.Body).Decode(&params); err != nil {
		httputil.RespondErrorMessage(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	entry, err := h.service.CreateBlocklistEntry(r.Context(), params)
	if err != nil {
		h.logger.Error("Failed to create blocklist entry", zap.Error(err))
		httputil.RespondErrorMessage(w, http.StatusInternalServerError, "Failed to create blocklist entry")
		return
	}

	httputil.RespondJSON(w, http.StatusCreated, entry)
}

// ========================
// Calendar
// ========================

// GetCalendarEvents gets calendar events
func (h *Handler) GetCalendarEvents(w http.ResponseWriter, r *http.Request) {
	// Parse date range from query params
	startDateStr := r.URL.Query().Get("start")
	endDateStr := r.URL.Query().Get("end")
	monitoredOnlyStr := r.URL.Query().Get("monitored")

	// Default to 30 days before and after today
	now := time.Now()
	startDate := now.AddDate(0, 0, -30)
	endDate := now.AddDate(0, 0, 30)

	if startDateStr != "" {
		if parsed, err := time.Parse("2006-01-02", startDateStr); err == nil {
			startDate = parsed
		}
	}

	if endDateStr != "" {
		if parsed, err := time.Parse("2006-01-02", endDateStr); err == nil {
			endDate = parsed
		}
	}

	monitoredOnly := monitoredOnlyStr == "true"

	events, err := h.service.GetCalendarEvents(r.Context(), startDate, endDate, monitoredOnly)
	if err != nil {
		h.logger.Error("Failed to get calendar events", zap.Error(err))
		httputil.RespondErrorMessage(w, http.StatusInternalServerError, "Failed to get calendar events")
		return
	}

	httputil.RespondJSON(w, http.StatusOK, events)
}

// ========================
// Statistics
// ========================

// GetMonitoringStats gets monitoring statistics
func (h *Handler) GetMonitoringStats(w http.ResponseWriter, r *http.Request) {
	stats, err := h.service.GetMonitoringStats(r.Context())
	if err != nil {
		h.logger.Error("Failed to get monitoring stats", zap.Error(err))
		httputil.RespondErrorMessage(w, http.StatusInternalServerError, "Failed to get monitoring stats")
		return
	}

	httputil.RespondJSON(w, http.StatusOK, stats)
}

// ========================
// Scheduler Jobs
// ========================

// ListSchedulerJobs lists all scheduler jobs
func (h *Handler) ListSchedulerJobs(w http.ResponseWriter, r *http.Request) {
	jobs, err := h.scheduler.ListJobs(r.Context())
	if err != nil {
		h.logger.Error("Failed to list scheduler jobs", zap.Error(err))
		httputil.RespondErrorMessage(w, http.StatusInternalServerError, "Failed to list scheduler jobs")
		return
	}

	httputil.RespondJSON(w, http.StatusOK, jobs)
}

// GetSchedulerJob gets a scheduler job by ID
func (h *Handler) GetSchedulerJob(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		httputil.RespondErrorMessage(w, http.StatusBadRequest, "Invalid job ID")
		return
	}

	job, err := h.scheduler.GetJob(r.Context(), id)
	if err != nil {
		h.logger.Error("Failed to get scheduler job", zap.Error(err))
		httputil.RespondErrorMessage(w, http.StatusNotFound, "Scheduler job not found")
		return
	}

	httputil.RespondJSON(w, http.StatusOK, job)
}

// TriggerSchedulerJob manually triggers a scheduler job
func (h *Handler) TriggerSchedulerJob(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		httputil.RespondErrorMessage(w, http.StatusBadRequest, "Invalid job ID")
		return
	}

	if err := h.scheduler.TriggerJob(r.Context(), id); err != nil {
		h.logger.Error("Failed to trigger scheduler job", zap.Error(err))
		httputil.RespondErrorMessage(w, http.StatusInternalServerError, "Failed to trigger scheduler job")
		return
	}

	httputil.RespondJSON(w, http.StatusOK, map[string]string{
		"message": "Job triggered successfully",
	})
}
