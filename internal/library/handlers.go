package library

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/blakestevenson/nimbus/internal/db/generated"
	"github.com/blakestevenson/nimbus/internal/httputil"

	"go.uber.org/zap"
)

// =============================================================================
// Handler - HTTP handlers for library scanner API
// =============================================================================

type Handler struct {
	queries *generated.Queries
	scanner *Scanner
	logger  *zap.Logger
	rootDir string
}

// NewHandler creates a new library handler
func NewHandler(queries *generated.Queries, logger *zap.Logger, rootDir string) *Handler {
	return &Handler{
		queries: queries,
		scanner: NewScanner(queries, logger, rootDir),
		logger:  logger,
		rootDir: rootDir,
	}
}

// =============================================================================
// StartScan - POST /api/library/scan
// =============================================================================
// Starts a new library scan in the background.
//
// Access: Admin only (enforced by middleware)
//
// Response:
//   - 200 OK: Scan started successfully
//   - 409 Conflict: Scan already in progress
//   - 500 Internal Server Error: Database or other error
//
// Example Response:
//   {
//     "status": "started",
//     "message": "Library scan started in background"
//   }
// =============================================================================

func (h *Handler) StartScan(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Check if scan is already running
	state, err := h.queries.GetScannerState(ctx)
	if err != nil {
		h.logger.Error("failed to get scanner state", zap.Error(err))
		httputil.RespondErrorMessage(w, http.StatusInternalServerError, "Failed to check scanner status")
		return
	}

	if state.Running {
		httputil.RespondErrorMessage(w, http.StatusConflict, "Scan already in progress")
		return
	}

	// Start scan in background goroutine
	go func() {
		// Use background context since the request context will be cancelled
		bgCtx := context.Background()

		if err := h.scanner.Run(bgCtx); err != nil {
			h.logger.Error("scan failed", zap.Error(err))
		}
	}()

	// Return immediate response
	response := map[string]string{
		"status":  "started",
		"message": "Library scan started in background",
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// =============================================================================
// GetScanStatus - GET /api/library/scan/status
// =============================================================================
// Retrieves the current status of the library scanner.
//
// Access: Authenticated users (enforced by middleware)
//
// Response:
//   - 200 OK: Status retrieved successfully
//   - 500 Internal Server Error: Database error
//
// Example Response:
//   {
//     "running": true,
//     "started_at": "2024-01-15T10:30:00Z",
//     "finished_at": null,
//     "files_scanned": 150,
//     "items_created": 120,
//     "items_updated": 30,
//     "errors": [
//       {
//         "timestamp": "2024-01-15T10:31:00Z",
//         "message": "Failed to parse file.mkv"
//       }
//     ],
//     "log": [
//       {
//         "timestamp": "2024-01-15T10:30:00Z",
//         "level": "info",
//         "message": "Scan started"
//       }
//     ]
//   }
// =============================================================================

func (h *Handler) GetScanStatus(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	status, err := h.scanner.GetScanStatus(ctx)
	if err != nil {
		h.logger.Error("failed to get scan status", zap.Error(err))
		httputil.RespondErrorMessage(w, http.StatusInternalServerError, "Failed to get scan status")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(status)
}

// =============================================================================
// StopScan - POST /api/library/scan/stop
// =============================================================================
// Attempts to stop a running scan (if context cancellation is implemented).
// Currently marks the scanner as not running.
//
// Access: Admin only (enforced by middleware)
//
// Response:
//   - 200 OK: Scan stopped (or wasn't running)
//   - 500 Internal Server Error: Database error
// =============================================================================

func (h *Handler) StopScan(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Mark scanner as not running
	if _, err := h.queries.SetScannerRunning(ctx, false); err != nil {
		h.logger.Error("failed to stop scanner", zap.Error(err))
		httputil.RespondErrorMessage(w, http.StatusInternalServerError, "Failed to stop scanner")
		return
	}

	response := map[string]string{
		"status":  "stopped",
		"message": "Scanner stopped",
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// =============================================================================
// ResetScanner - POST /api/library/scan/reset
// =============================================================================
// Resets the scanner state (clears logs, errors, counters).
// Use with caution - this clears all scan history.
//
// Access: Admin only (enforced by middleware)
//
// Response:
//   - 200 OK: Scanner reset
//   - 500 Internal Server Error: Database error
// =============================================================================

func (h *Handler) ResetScanner(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if _, err := h.queries.ResetScannerState(ctx); err != nil {
		h.logger.Error("failed to reset scanner", zap.Error(err))
		httputil.RespondErrorMessage(w, http.StatusInternalServerError, "Failed to reset scanner")
		return
	}

	response := map[string]string{
		"status":  "reset",
		"message": "Scanner state reset",
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}
