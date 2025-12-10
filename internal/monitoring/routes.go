package monitoring

import (
	"github.com/go-chi/chi/v5"
)

// SetupRoutes configures monitoring routes
func SetupRoutes(r chi.Router, handler *Handler) {
	// Monitoring rules
	r.Route("/monitoring", func(r chi.Router) {
		r.Get("/", handler.ListMonitoringRules)
		r.Post("/", handler.CreateMonitoringRule)
		r.Get("/{id}", handler.GetMonitoringRule)
		r.Put("/{id}", handler.UpdateMonitoringRule)
		r.Delete("/{id}", handler.DeleteMonitoringRule)

		// Statistics
		r.Get("/stats", handler.GetMonitoringStats)

		// Missing episodes/wanted items
		r.Get("/missing", handler.GetMissingEpisodes)
	})

	// Media-specific monitoring routes
	r.Route("/media/{mediaId}/monitoring", func(r chi.Router) {
		r.Get("/", handler.GetMonitoringRuleByMediaItem)
		r.Get("/history", handler.GetSearchHistory)
	})

	// Calendar
	r.Get("/calendar", handler.GetCalendarEvents)

	// Blocklist
	r.Post("/blocklist", handler.CreateBlocklistEntry)

	// Scheduler jobs
	r.Route("/scheduler", func(r chi.Router) {
		r.Get("/jobs", handler.ListSchedulerJobs)
		r.Get("/jobs/{id}", handler.GetSchedulerJob)
		r.Post("/jobs/{id}/trigger", handler.TriggerSchedulerJob)
	})
}
