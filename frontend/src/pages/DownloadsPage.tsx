import { useEffect, useState } from "react";
import {
  Download,
  Loader2,
  Pause,
  Play,
  Trash2,
  RefreshCw,
  AlertCircle,
} from "lucide-react";

interface Download {
  id: string;
  plugin_id: string;
  name: string;
  status:
    | "queued"
    | "downloading"
    | "processing"
    | "paused"
    | "completed"
    | "failed"
    | "cancelled";
  progress: number;
  total_bytes?: number;
  downloaded_bytes: number;
  speed?: number;
  destination_path?: string;
  error_message?: string;
  queue_position?: number;
  priority: number;
  created_at?: string;
  added_at?: string;
  started_at?: string;
  completed_at?: string;
}

interface DownloaderInfo {
  id: string;
  name: string;
  version: string;
  description: string;
}

export default function DownloadsPage() {
  const [downloads, setDownloads] = useState<Download[]>([]);
  const [downloaders, setDownloaders] = useState<DownloaderInfo[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [filterStatus, setFilterStatus] = useState<string>("all");
  const [filterPlugin, setFilterPlugin] = useState<string>("all");

  useEffect(() => {
    fetchDownloaders();
    fetchDownloads();

    // Poll for updates every 2 seconds
    const interval = setInterval(fetchDownloads, 2000);
    return () => clearInterval(interval);
  }, [filterStatus, filterPlugin]);

  const fetchDownloaders = async () => {
    try {
      const response = await fetch("/api/downloaders", {
        credentials: "include",
      });

      if (!response.ok) throw new Error("Failed to fetch downloaders");

      const data = await response.json();
      setDownloaders(data.downloaders || []);
    } catch (err) {
      console.error("Error fetching downloaders:", err);
    }
  };

  const fetchDownloads = async () => {
    try {
      const params = new URLSearchParams();
      if (filterStatus !== "all") params.append("status", filterStatus);
      if (filterPlugin !== "all") params.append("plugin_id", filterPlugin);

      const response = await fetch(`/api/downloads?${params}`, {
        credentials: "include",
      });

      if (!response.ok) throw new Error("Failed to fetch downloads");

      const data = await response.json();
      setDownloads(data.downloads || []);
      setError(null);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to load downloads");
    } finally {
      setLoading(false);
    }
  };

  const handlePause = async (download: Download) => {
    try {
      const response = await fetch(
        `/api/downloads/${download.plugin_id}/${download.id}/pause`,
        {
          method: "POST",
          credentials: "include",
        },
      );

      if (!response.ok) throw new Error("Failed to pause download");
      fetchDownloads();
    } catch (err) {
      console.error("Error pausing download:", err);
    }
  };

  const handleResume = async (download: Download) => {
    try {
      const response = await fetch(
        `/api/downloads/${download.plugin_id}/${download.id}/resume`,
        {
          method: "POST",
          credentials: "include",
        },
      );

      if (!response.ok) throw new Error("Failed to resume download");
      fetchDownloads();
    } catch (err) {
      console.error("Error resuming download:", err);
    }
  };

  const handleRetry = async (download: Download) => {
    try {
      const response = await fetch(
        `/api/downloads/${download.plugin_id}/${download.id}/retry`,
        {
          method: "POST",
          credentials: "include",
        },
      );

      if (!response.ok) throw new Error("Failed to retry download");
      fetchDownloads();
    } catch (err) {
      console.error("Error retrying download:", err);
    }
  };

  const handleDelete = async (download: Download) => {
    if (!confirm(`Delete download "${download.name}"?`)) return;

    try {
      const response = await fetch(
        `/api/downloads/${download.plugin_id}/${download.id}`,
        {
          method: "DELETE",
          credentials: "include",
        },
      );

      if (!response.ok) throw new Error("Failed to delete download");
      fetchDownloads();
    } catch (err) {
      console.error("Error deleting download:", err);
    }
  };

  const formatBytes = (bytes?: number) => {
    if (!bytes) return "Unknown";
    const sizes = ["B", "KB", "MB", "GB", "TB"];
    const i = Math.floor(Math.log(bytes) / Math.log(1024));
    return `${(bytes / Math.pow(1024, i)).toFixed(2)} ${sizes[i]}`;
  };

  const formatDate = (dateString?: string) => {
    if (!dateString) return "N/A";
    return new Date(dateString).toLocaleString();
  };

  const getStatusColor = (status: Download["status"]) => {
    switch (status) {
      case "completed":
        return "text-green-700 dark:text-green-400 bg-green-100 dark:bg-green-950";
      case "downloading":
        return "text-blue-700 dark:text-blue-400 bg-blue-100 dark:bg-blue-950";
      case "failed":
        return "text-red-700 dark:text-red-400 bg-red-100 dark:bg-red-950";
      case "paused":
        return "text-yellow-700 dark:text-yellow-400 bg-yellow-100 dark:bg-yellow-950";
      case "queued":
        return "text-gray-700 dark:text-gray-400 bg-gray-100 dark:bg-gray-800";
      case "processing":
        return "text-purple-700 dark:text-purple-400 bg-purple-100 dark:bg-purple-950";
      case "cancelled":
        return "text-orange-700 dark:text-orange-400 bg-orange-100 dark:bg-orange-950";
      default:
        return "text-gray-700 dark:text-gray-400 bg-gray-100 dark:bg-gray-800";
    }
  };

  const activeDownloads = downloads.filter((d) =>
    ["queued", "downloading", "processing"].includes(d.status),
  );
  const completedDownloads = downloads.filter((d) => d.status === "completed");
  const failedDownloads = downloads.filter((d) => d.status === "failed");

  if (loading && downloads.length === 0) {
    return (
      <div className="flex items-center justify-center h-64">
        <Loader2 className="w-8 h-8 animate-spin text-primary" />
      </div>
    );
  }

  return (
    <div className="p-6 max-w-7xl mx-auto">
      <div className="mb-6">
        <h1 className="text-3xl font-bold flex items-center gap-3">
          <Download className="w-8 h-8" />
          Downloads Queue
        </h1>
        <p className="mt-2 text-muted-foreground">
          Manage all downloads across different downloaders
        </p>
      </div>

      {/* Stats */}
      <div className="grid grid-cols-1 md:grid-cols-4 gap-4 mb-6">
        <div className="bg-card border rounded-lg p-4">
          <div className="text-sm text-muted-foreground">Total Downloads</div>
          <div className="text-2xl font-bold">{downloads.length}</div>
        </div>
        <div className="bg-blue-100 dark:bg-blue-950 border border-blue-200 dark:border-blue-900 rounded-lg p-4">
          <div className="text-sm text-blue-700 dark:text-blue-400">Active</div>
          <div className="text-2xl font-bold text-blue-800 dark:text-blue-300">
            {activeDownloads.length}
          </div>
        </div>
        <div className="bg-green-100 dark:bg-green-950 border border-green-200 dark:border-green-900 rounded-lg p-4">
          <div className="text-sm text-green-700 dark:text-green-400">
            Completed
          </div>
          <div className="text-2xl font-bold text-green-800 dark:text-green-300">
            {completedDownloads.length}
          </div>
        </div>
        <div className="bg-red-100 dark:bg-red-950 border border-red-200 dark:border-red-900 rounded-lg p-4">
          <div className="text-sm text-red-700 dark:text-red-400">Failed</div>
          <div className="text-2xl font-bold text-red-800 dark:text-red-300">
            {failedDownloads.length}
          </div>
        </div>
      </div>

      {/* Filters */}
      <div className="bg-card border rounded-lg p-4 mb-6">
        <div className="flex flex-wrap gap-4">
          <div>
            <label className="block text-sm font-medium mb-1">Status</label>
            <select
              value={filterStatus}
              onChange={(e) => setFilterStatus(e.target.value)}
              className="block w-full px-3 py-2 bg-background border rounded-md focus:ring-2 focus:ring-primary focus:border-primary"
            >
              <option value="all">All Statuses</option>
              <option value="queued">Queued</option>
              <option value="downloading">Downloading</option>
              <option value="processing">Processing</option>
              <option value="paused">Paused</option>
              <option value="completed">Completed</option>
              <option value="failed">Failed</option>
              <option value="cancelled">Cancelled</option>
            </select>
          </div>

          <div>
            <label className="block text-sm font-medium mb-1">Downloader</label>
            <select
              value={filterPlugin}
              onChange={(e) => setFilterPlugin(e.target.value)}
              className="block w-full px-3 py-2 bg-background border rounded-md focus:ring-2 focus:ring-primary focus:border-primary"
            >
              <option value="all">All Downloaders</option>
              {downloaders.map((d) => (
                <option key={d.id} value={d.id}>
                  {d.name}
                </option>
              ))}
            </select>
          </div>
        </div>
      </div>

      {/* Error Display */}
      {error && (
        <div className="bg-red-100 dark:bg-red-950 border border-red-300 dark:border-red-800 rounded-lg p-4 mb-6 flex items-center gap-3">
          <AlertCircle className="w-5 h-5 text-red-700 dark:text-red-400 flex-shrink-0" />
          <span className="text-red-900 dark:text-red-300">{error}</span>
        </div>
      )}

      {/* Downloads List */}
      {downloads.length === 0 ? (
        <div className="bg-card border rounded-lg p-12 text-center">
          <Download className="w-16 h-16 mx-auto text-muted-foreground mb-4" />
          <h3 className="text-lg font-medium mb-2">No downloads</h3>
          <p className="text-muted-foreground">
            Downloads will appear here when you start downloading content
          </p>
        </div>
      ) : (
        <div className="space-y-4">
          {downloads.map((download) => (
            <div
              key={download.id}
              className="bg-card border rounded-lg overflow-hidden"
            >
              <div className="p-4">
                <div className="flex items-start justify-between mb-3">
                  <div className="flex-1 min-w-0">
                    <h3 className="text-lg font-semibold truncate">
                      {download.name}
                    </h3>
                    <div className="flex items-center gap-2 mt-1 flex-wrap">
                      <span
                        className={`inline-flex items-center px-2 py-1 rounded-full text-xs font-medium ${getStatusColor(
                          download.status,
                        )}`}
                      >
                        {download.status.toUpperCase()}
                      </span>
                      <span className="text-sm text-muted-foreground">
                        {downloaders.find((d) => d.id === download.plugin_id)
                          ?.name || download.plugin_id}
                      </span>
                      {download.queue_position !== null &&
                        download.queue_position !== undefined && (
                          <span className="text-sm text-muted-foreground">
                            Queue: #{download.queue_position}
                          </span>
                        )}
                    </div>
                  </div>

                  <div className="flex items-center gap-2 ml-4">
                    {download.status === "downloading" && (
                      <button
                        onClick={() => handlePause(download)}
                        className="p-2 text-yellow-700 dark:text-yellow-400 hover:bg-yellow-100 dark:hover:bg-yellow-950 rounded transition-colors"
                        title="Pause"
                      >
                        <Pause className="w-5 h-5" />
                      </button>
                    )}
                    {download.status === "paused" && (
                      <button
                        onClick={() => handleResume(download)}
                        className="p-2 text-green-700 dark:text-green-400 hover:bg-green-100 dark:hover:bg-green-950 rounded transition-colors"
                        title="Resume"
                      >
                        <Play className="w-5 h-5" />
                      </button>
                    )}
                    {download.status === "failed" && (
                      <button
                        onClick={() => handleRetry(download)}
                        className="p-2 text-blue-700 dark:text-blue-400 hover:bg-blue-100 dark:hover:bg-blue-950 rounded transition-colors"
                        title="Retry"
                      >
                        <RefreshCw className="w-5 h-5" />
                      </button>
                    )}
                    <button
                      onClick={() => handleDelete(download)}
                      className="p-2 text-red-700 dark:text-red-400 hover:bg-red-100 dark:hover:bg-red-950 rounded transition-colors"
                      title="Delete"
                    >
                      <Trash2 className="w-5 h-5" />
                    </button>
                  </div>
                </div>

                {/* Progress Bar */}
                {["downloading", "processing", "queued"].includes(
                  download.status,
                ) && (
                  <div className="mb-3">
                    <div className="flex items-center justify-between text-sm text-muted-foreground mb-1">
                      <div className="flex items-center gap-3">
                        <span>Progress: {download.progress.toFixed(1)}%</span>
                        {download.speed && download.speed > 0 && (
                          <span className="text-xs">
                            {formatBytes(download.speed)}/s
                          </span>
                        )}
                      </div>
                      <span>
                        {formatBytes(download.downloaded_bytes)}
                        {download.total_bytes &&
                          ` / ${formatBytes(download.total_bytes)}`}
                      </span>
                    </div>
                    <div className="w-full bg-secondary rounded-full h-2">
                      <div
                        className="bg-primary h-2 rounded-full transition-all duration-300"
                        style={{ width: `${download.progress}%` }}
                      />
                    </div>
                  </div>
                )}

                {/* Error Message */}
                {download.error_message && (
                  <div className="mb-3 p-3 bg-red-100 dark:bg-red-950 border border-red-300 dark:border-red-800 rounded flex items-start gap-2">
                    <AlertCircle className="w-4 h-4 text-red-700 dark:text-red-400 flex-shrink-0 mt-0.5" />
                    <span className="text-sm text-red-900 dark:text-red-300">
                      {download.error_message}
                    </span>
                  </div>
                )}

                {/* Details */}
                <div className="grid grid-cols-2 md:grid-cols-4 gap-4 text-sm text-muted-foreground">
                  <div>
                    <div className="font-medium text-foreground">Created</div>
                    <div>
                      {formatDate(download.created_at || download.added_at)}
                    </div>
                  </div>
                  {download.started_at && (
                    <div>
                      <div className="font-medium text-foreground">Started</div>
                      <div>{formatDate(download.started_at)}</div>
                    </div>
                  )}
                  {download.completed_at && (
                    <div>
                      <div className="font-medium text-foreground">
                        Completed
                      </div>
                      <div>{formatDate(download.completed_at)}</div>
                    </div>
                  )}
                  {download.destination_path && (
                    <div className="col-span-2">
                      <div className="font-medium text-foreground">
                        Destination
                      </div>
                      <div
                        className="truncate"
                        title={download.destination_path}
                      >
                        {download.destination_path}
                      </div>
                    </div>
                  )}
                </div>
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
