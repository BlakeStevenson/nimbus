/**
 * Library Page
 *
 * This page provides a UI for managing the library scanner. Users can:
 * - Start/stop library scans
 * - Monitor scan progress in real-time
 * - View scan statistics and history
 * - Review errors and logs
 * - Reset scanner state
 */

import { useState } from 'react';
import {
  useLibraryScanStatus,
  useStartLibraryScan,
  useStopLibraryScan,
  useResetScannerState,
} from '@/lib/api/library';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { ScrollArea } from '@/components/ui/scroll-area';
import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert';
import {
  PlayCircle,
  StopCircle,
  RefreshCw,
  AlertCircle,
  CheckCircle,
  Clock,
  FileText,
  Database,
  Loader2
} from 'lucide-react';

export default function LibraryPage() {
  const [showLogs, setShowLogs] = useState(true);
  const [showErrors, setShowErrors] = useState(true);

  // Fetch scan status (polls every 5 seconds)
  const { data: status, isLoading } = useLibraryScanStatus();

  // Mutations
  const startScan = useStartLibraryScan();
  const stopScan = useStopLibraryScan();
  const resetState = useResetScannerState();

  const handleStartScan = () => {
    startScan.mutate(undefined, {
      onError: (error: any) => {
        alert(error.message || 'Failed to start scan');
      },
    });
  };

  const handleStopScan = () => {
    stopScan.mutate(undefined, {
      onError: (error: any) => {
        alert(error.message || 'Failed to stop scan');
      },
    });
  };

  const handleResetState = () => {
    if (confirm('Are you sure you want to reset the scanner state? This will clear all logs and counters.')) {
      resetState.mutate(undefined, {
        onError: (error: any) => {
          alert(error.message || 'Failed to reset scanner state');
        },
      });
    }
  };

  if (isLoading) {
    return (
      <div className="flex items-center justify-center h-64">
        <Loader2 className="h-8 w-8 animate-spin text-primary" />
      </div>
    );
  }

  const isRunning = status?.running ?? false;
  const hasErrors = (status?.errors?.length ?? 0) > 0;

  return (
    <div className="container mx-auto py-8 space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-3xl font-bold">Library Scanner</h1>
          <p className="text-muted-foreground mt-1">
            Scan your media library to import movies, TV shows, music, and books
          </p>
        </div>

        <div className="flex gap-2">
          {isRunning ? (
            <Button
              onClick={handleStopScan}
              disabled={stopScan.isPending}
              variant="destructive"
            >
              {stopScan.isPending ? (
                <>
                  <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                  Stopping...
                </>
              ) : (
                <>
                  <StopCircle className="mr-2 h-4 w-4" />
                  Stop Scan
                </>
              )}
            </Button>
          ) : (
            <Button
              onClick={handleStartScan}
              disabled={startScan.isPending}
            >
              {startScan.isPending ? (
                <>
                  <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                  Starting...
                </>
              ) : (
                <>
                  <PlayCircle className="mr-2 h-4 w-4" />
                  Start Scan
                </>
              )}
            </Button>
          )}

          <Button
            onClick={handleResetState}
            disabled={resetState.isPending || isRunning}
            variant="outline"
          >
            {resetState.isPending ? (
              <>
                <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                Resetting...
              </>
            ) : (
              <>
                <RefreshCw className="mr-2 h-4 w-4" />
                Reset
              </>
            )}
          </Button>
        </div>
      </div>

      {/* Status Card */}
      <Card>
        <CardHeader>
          <div className="flex items-center justify-between">
            <div>
              <CardTitle>Scanner Status</CardTitle>
              <CardDescription>Current scan progress and statistics</CardDescription>
            </div>
            <Badge variant={isRunning ? 'default' : 'secondary'} className="text-sm">
              {isRunning ? (
                <>
                  <Loader2 className="mr-1 h-3 w-3 animate-spin" />
                  Running
                </>
              ) : (
                <>
                  <CheckCircle className="mr-1 h-3 w-3" />
                  Idle
                </>
              )}
            </Badge>
          </div>
        </CardHeader>
        <CardContent>
          <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
            {/* Files Scanned */}
            <div className="flex items-center space-x-3 p-4 bg-muted rounded-lg">
              <div className="p-2 bg-background rounded-md">
                <FileText className="h-5 w-5 text-blue-500" />
              </div>
              <div>
                <p className="text-sm text-muted-foreground">Files Scanned</p>
                <p className="text-2xl font-bold">{status?.files_scanned ?? 0}</p>
              </div>
            </div>

            {/* Items Created */}
            <div className="flex items-center space-x-3 p-4 bg-muted rounded-lg">
              <div className="p-2 bg-background rounded-md">
                <Database className="h-5 w-5 text-green-500" />
              </div>
              <div>
                <p className="text-sm text-muted-foreground">Items Created</p>
                <p className="text-2xl font-bold">{status?.items_created ?? 0}</p>
              </div>
            </div>

            {/* Items Updated */}
            <div className="flex items-center space-x-3 p-4 bg-muted rounded-lg">
              <div className="p-2 bg-background rounded-md">
                <RefreshCw className="h-5 w-5 text-orange-500" />
              </div>
              <div>
                <p className="text-sm text-muted-foreground">Items Updated</p>
                <p className="text-2xl font-bold">{status?.items_updated ?? 0}</p>
              </div>
            </div>
          </div>

          {/* Timestamps */}
          <div className="mt-4 pt-4 border-t grid grid-cols-1 md:grid-cols-2 gap-4">
            <div className="flex items-center space-x-2 text-sm">
              <Clock className="h-4 w-4 text-muted-foreground" />
              <span className="text-muted-foreground">Started:</span>
              <span className="font-medium">
                {status?.started_at
                  ? new Date(status.started_at).toLocaleString()
                  : 'Never'}
              </span>
            </div>
            <div className="flex items-center space-x-2 text-sm">
              <Clock className="h-4 w-4 text-muted-foreground" />
              <span className="text-muted-foreground">Finished:</span>
              <span className="font-medium">
                {status?.finished_at
                  ? new Date(status.finished_at).toLocaleString()
                  : 'N/A'}
              </span>
            </div>
          </div>
        </CardContent>
      </Card>

      {/* Errors Section */}
      {hasErrors && showErrors && (
        <Card className="border-destructive">
          <CardHeader>
            <div className="flex items-center justify-between">
              <div className="flex items-center space-x-2">
                <AlertCircle className="h-5 w-5 text-destructive" />
                <CardTitle>Errors</CardTitle>
              </div>
              <Button
                variant="ghost"
                size="sm"
                onClick={() => setShowErrors(false)}
              >
                Hide
              </Button>
            </div>
            <CardDescription>
              {status?.errors?.length ?? 0} error(s) occurred during the scan
            </CardDescription>
          </CardHeader>
          <CardContent>
            <ScrollArea className="h-64">
              <div className="space-y-2">
                {status?.errors?.map((error, idx) => (
                  <Alert key={idx} variant="destructive">
                    <AlertCircle className="h-4 w-4" />
                    <AlertTitle className="text-xs text-muted-foreground">
                      {new Date(error.timestamp).toLocaleString()}
                    </AlertTitle>
                    <AlertDescription className="font-mono text-sm">
                      {error.message}
                    </AlertDescription>
                  </Alert>
                ))}
              </div>
            </ScrollArea>
          </CardContent>
        </Card>
      )}

      {hasErrors && !showErrors && (
        <Button
          variant="outline"
          onClick={() => setShowErrors(true)}
          className="w-full"
        >
          <AlertCircle className="mr-2 h-4 w-4" />
          Show {status?.errors?.length ?? 0} Error(s)
        </Button>
      )}

      {/* Activity Log */}
      {showLogs && (
        <Card>
          <CardHeader>
            <div className="flex items-center justify-between">
              <div>
                <CardTitle>Activity Log</CardTitle>
                <CardDescription>
                  Recent scanner activity
                </CardDescription>
              </div>
              <Button
                variant="ghost"
                size="sm"
                onClick={() => setShowLogs(false)}
              >
                Hide
              </Button>
            </div>
          </CardHeader>
          <CardContent>
            <ScrollArea className="h-96">
              {status?.log && status.log.length > 0 ? (
                <div className="space-y-2">
                  {status.log.map((entry, idx) => (
                    <div
                      key={idx}
                      className="flex items-start space-x-3 p-3 rounded-lg bg-muted font-mono text-sm"
                    >
                      <Badge
                        variant={entry.level === 'error' ? 'destructive' : 'secondary'}
                        className="mt-0.5"
                      >
                        {entry.level || 'info'}
                      </Badge>
                      <div className="flex-1">
                        <div className="text-xs text-muted-foreground mb-1">
                          {new Date(entry.timestamp).toLocaleString()}
                        </div>
                        <div>{entry.message}</div>
                      </div>
                    </div>
                  ))}
                </div>
              ) : (
                <div className="text-center py-8 text-muted-foreground">
                  No log entries yet
                </div>
              )}
            </ScrollArea>
          </CardContent>
        </Card>
      )}

      {!showLogs && (
        <Button
          variant="outline"
          onClick={() => setShowLogs(true)}
          className="w-full"
        >
          <FileText className="mr-2 h-4 w-4" />
          Show Activity Log
        </Button>
      )}

      {/* Help Section */}
      <Card>
        <CardHeader>
          <CardTitle>How It Works</CardTitle>
        </CardHeader>
        <CardContent className="prose prose-sm dark:prose-invert max-w-none">
          <p>
            The library scanner automatically discovers and imports media files from your configured library directory.
          </p>

          <h4 className="text-sm font-semibold mt-4 mb-2">Supported File Types:</h4>
          <ul className="text-sm space-y-1">
            <li><strong>Movies:</strong> .mkv, .mp4, .avi, .mov, .wmv, .flv, .webm, .m4v</li>
            <li><strong>TV Shows:</strong> Same as movies (parsed by filename pattern)</li>
            <li><strong>Music:</strong> .mp3, .flac, .m4a, .aac, .ogg, .opus, .wma, .wav</li>
            <li><strong>Books:</strong> .epub, .mobi, .azw, .azw3, .pdf, .djvu</li>
          </ul>

          <h4 className="text-sm font-semibold mt-4 mb-2">Filename Patterns:</h4>
          <ul className="text-sm space-y-1">
            <li><strong>Movies:</strong> "Movie Name (2021).mkv" or "Movie.Name.2021.1080p.mkv"</li>
            <li><strong>TV Shows:</strong> "Show.Name.S01E02.mkv" or "Show Name - 1x02.mkv"</li>
            <li><strong>Music:</strong> Artist/Album/01 Track Name.mp3</li>
            <li><strong>Books:</strong> "Book Title - Author Name.epub"</li>
          </ul>

          <p className="text-sm text-muted-foreground mt-4">
            The scanner will automatically create hierarchies for TV shows (series → season → episode)
            and music (artist → album → track).
          </p>
        </CardContent>
      </Card>
    </div>
  );
}
