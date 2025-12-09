/**
 * Library Scanner API
 *
 * This module provides functions to interact with the library scanner API.
 * The scanner crawls the media library directory, parses filenames, and imports
 * media items into the database.
 *
 * Features:
 * - Start/stop library scans
 * - Monitor scan progress in real-time
 * - View scan logs and errors
 * - Reset scanner state
 */

import { apiGet, apiPost } from '../api-client';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';

// =============================================================================
// Types
// =============================================================================

export interface LogEntry {
  timestamp: string;
  message: string;
  level?: string;
}

export interface ScanStatus {
  running: boolean;
  started_at: string | null;
  finished_at: string | null;
  files_scanned: number;
  items_created: number;
  items_updated: number;
  errors: LogEntry[];
  log: LogEntry[];
}

export interface ScanStartResponse {
  status: string;
  message: string;
}

// =============================================================================
// API Functions
// =============================================================================

/**
 * Start a new library scan
 *
 * Initiates a background scan of the library directory. The scan will:
 * 1. Walk the filesystem to find all media files
 * 2. Parse filenames to extract metadata
 * 3. Create or update media items in the database
 * 4. Track progress and log activity
 *
 * Returns 409 Conflict if a scan is already running.
 *
 * @returns Promise that resolves when the scan has been started
 */
export async function startLibraryScan(): Promise<ScanStartResponse> {
  return apiPost<ScanStartResponse>('/api/library/scan', {});
}

/**
 * Get the current scanner status
 *
 * Returns real-time information about the scanner including:
 * - Whether a scan is currently running
 * - Progress counters (files scanned, items created/updated)
 * - Start and finish timestamps
 * - Error log
 * - Activity log
 *
 * This endpoint can be polled to monitor scan progress.
 *
 * @returns Promise that resolves with the current scan status
 */
export async function getLibraryScanStatus(): Promise<ScanStatus> {
  return apiGet<ScanStatus>('/api/library/scan/status');
}

/**
 * Stop a running library scan
 *
 * Attempts to gracefully stop the current scan. The scanner will finish
 * processing the current file and then exit.
 *
 * @returns Promise that resolves when the stop request has been sent
 */
export async function stopLibraryScan(): Promise<ScanStartResponse> {
  return apiPost<ScanStartResponse>('/api/library/scan/stop', {});
}

/**
 * Reset the scanner state
 *
 * Clears all scanner state including:
 * - Progress counters (files scanned, items created/updated)
 * - Error log
 * - Activity log
 * - Running flag
 *
 * Use this to clean up after a failed scan or to clear old logs.
 *
 * WARNING: This cannot be undone.
 *
 * @returns Promise that resolves when the reset is complete
 */
export async function resetScannerState(): Promise<ScanStartResponse> {
  return apiPost<ScanStartResponse>('/api/library/scan/reset', {});
}

// =============================================================================
// React Query Hooks
// =============================================================================

/**
 * Hook to fetch the current scan status
 *
 * By default, this hook will refetch the status every 5 seconds when a scan
 * is running, allowing the UI to show real-time progress.
 *
 * @param options - Optional query options
 * @returns Query result with scan status
 *
 * @example
 * ```tsx
 * const { data: status, isLoading } = useLibraryScanStatus();
 *
 * if (status?.running) {
 *   return <div>Scanning... {status.files_scanned} files processed</div>;
 * }
 * ```
 */
export function useLibraryScanStatus(options?: { refetchInterval?: number }) {
  return useQuery({
    queryKey: ['library', 'scan-status'],
    queryFn: getLibraryScanStatus,
    refetchInterval: options?.refetchInterval ?? 5000, // Poll every 5 seconds
    refetchIntervalInBackground: false,
  });
}

/**
 * Hook to start a library scan
 *
 * Returns a mutation that starts a new library scan. On success, it will
 * invalidate the scan status query to trigger a refetch.
 *
 * @returns Mutation result
 *
 * @example
 * ```tsx
 * const startScan = useStartLibraryScan();
 *
 * <button
 *   onClick={() => startScan.mutate()}
 *   disabled={startScan.isPending}
 * >
 *   {startScan.isPending ? 'Starting...' : 'Start Scan'}
 * </button>
 * ```
 */
export function useStartLibraryScan() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: startLibraryScan,
    onSuccess: () => {
      // Invalidate status query to immediately show the scan as running
      queryClient.invalidateQueries({ queryKey: ['library', 'scan-status'] });
    },
  });
}

/**
 * Hook to stop a library scan
 *
 * Returns a mutation that stops the currently running scan.
 *
 * @returns Mutation result
 */
export function useStopLibraryScan() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: stopLibraryScan,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['library', 'scan-status'] });
    },
  });
}

/**
 * Hook to reset scanner state
 *
 * Returns a mutation that resets all scanner state including logs and counters.
 *
 * @returns Mutation result
 */
export function useResetScannerState() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: resetScannerState,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['library', 'scan-status'] });
    },
  });
}
