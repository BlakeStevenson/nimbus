import { useEffect, useState } from "react";
import { Link } from "react-router-dom";
import {
  Download as DownloadIcon,
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
  metadata?: {
    media_id?: number;
    media_title?: string;
    media_kind?: string;
    season?: number;
    episode?: number;
    [key: string]: any;
  };
}

interface DownloaderInfo {
  id: string;
  name: string;
  version: string;
  description: string;
}

interface MediaItem {
  id: number;
  title: string;
  kind: string;
  metadata?: {
    season?: number;
    episode?: number;
    [key: string]: any;
  };
  parent_id?: number;
}

export default function DownloadsPage() {
  const [downloads, setDownloads] = useState<Download[]>([]);
  const [downloaders, setDownloaders] = useState<DownloaderInfo[]>([]);
  const [mediaItems, setMediaItems] = useState<Map<number, MediaItem>>(
    new Map(),
  );
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [filterStatus, setFilterStatus] = useState<string>("active");
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

  const fetchMediaItem = async (mediaId: number) => {
    try {
      const response = await fetch(`/api/media/${mediaId}`, {
        credentials: "include",
      });
      if (response.ok) {
        const item = await response.json();
        return item;
      }
    } catch (err) {
      console.error("Error fetching media item:", err);
    }
    return null;
  };

  const fetchDownloads = async () => {
    try {
      const params = new URLSearchParams();
      if (filterStatus === "active") {
        // Fetch active statuses: queued, downloading, processing
        const response = await fetch(`/api/downloads`, {
          credentials: "include",
        });
        if (!response.ok) throw new Error("Failed to fetch downloads");
        const data = await response.json();
        const activeDownloads = (data.downloads || []).filter((d: Download) =>
          ["queued", "downloading", "processing"].includes(d.status),
        );
        setDownloads(activeDownloads);

        // Fetch media items for downloads with media_id
        const mediaIds = activeDownloads
          .filter((d: Download) => d.metadata?.media_id)
          .map((d: Download) => d.metadata!.media_id!);
        await fetchMediaItems(mediaIds);
      } else if (filterStatus !== "all") {
        params.append("status", filterStatus);
        const response = await fetch(`/api/downloads?${params}`, {
          credentials: "include",
        });
        if (!response.ok) throw new Error("Failed to fetch downloads");
        const data = await response.json();
        setDownloads(data.downloads || []);

        // Fetch media items
        const mediaIds = (data.downloads || [])
          .filter((d: Download) => d.metadata?.media_id)
          .map((d: Download) => d.metadata!.media_id!);
        await fetchMediaItems(mediaIds);
      } else {
        const response = await fetch(`/api/downloads`, {
          credentials: "include",
        });
        if (!response.ok) throw new Error("Failed to fetch downloads");
        const data = await response.json();
        setDownloads(data.downloads || []);

        // Fetch media items
        const mediaIds = (data.downloads || [])
          .filter((d: Download) => d.metadata?.media_id)
          .map((d: Download) => d.metadata!.media_id!);
        await fetchMediaItems(mediaIds);
      }

      if (filterPlugin !== "all") {
        setDownloads((prev) =>
          prev.filter((d) => d.plugin_id === filterPlugin),
        );
      }

      setError(null);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to load downloads");
    } finally {
      setLoading(false);
    }
  };

  const fetchMediaItems = async (mediaIds: number[]) => {
    const uniqueIds = [...new Set(mediaIds)];
    const newMediaItems = new Map(mediaItems);

    for (const id of uniqueIds) {
      if (!newMediaItems.has(id)) {
        const item = await fetchMediaItem(id);
        if (item) {
          newMediaItems.set(id, item);

          // If it's an episode or season, fetch parent (season or series)
          if (item.parent_id) {
            const parent = await fetchMediaItem(item.parent_id);
            if (parent) {
              newMediaItems.set(parent.id, parent);

              // If parent has a parent (series for episode), fetch that too
              if (parent.parent_id) {
                const grandparent = await fetchMediaItem(parent.parent_id);
                if (grandparent) {
                  newMediaItems.set(grandparent.id, grandparent);
                }
              }
            }
          }
        }
      }
    }

    setMediaItems(newMediaItems);
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

  const getMediaDisplayName = (download: Download) => {
    if (!download.metadata?.media_id) return null;

    const mediaItem = mediaItems.get(download.metadata.media_id);
    if (!mediaItem) return null;

    // For TV episodes
    if (
      mediaItem.kind === "tv_episode" ||
      download.metadata.media_kind === "tv_episode"
    ) {
      const season = download.metadata.season || mediaItem.metadata?.season;
      const episode = download.metadata.episode || mediaItem.metadata?.episode;
      const episodeTitle = mediaItem.title;

      // Get series name from parent chain
      let seriesName = mediaItem.title;
      if (mediaItem.parent_id) {
        const seasonItem = mediaItems.get(mediaItem.parent_id);
        if (seasonItem?.parent_id) {
          const seriesItem = mediaItems.get(seasonItem.parent_id);
          if (seriesItem) {
            seriesName = seriesItem.title;
          }
        }
      }

      if (season !== undefined && episode !== undefined) {
        return `${seriesName} - S${season}E${episode} - ${episodeTitle}`;
      }
      return `${seriesName} - ${episodeTitle}`;
    }

    // For TV seasons
    if (
      mediaItem.kind === "tv_season" ||
      download.metadata.media_kind === "tv_season"
    ) {
      const season = download.metadata.season || mediaItem.metadata?.season;

      // Get series name from parent
      let seriesName = mediaItem.title;
      if (mediaItem.parent_id) {
        const seriesItem = mediaItems.get(mediaItem.parent_id);
        if (seriesItem) {
          seriesName = seriesItem.title;
        }
      }

      if (season !== undefined) {
        return `${seriesName} - Season ${season}`;
      }
      return `${seriesName} - ${mediaItem.title}`;
    }

    // For movies or other media
    return mediaItem.title;
  };

  // Calculate stats from all downloads (not filtered)
  const [allDownloads, setAllDownloads] = useState<Download[]>([]);

  useEffect(() => {
    // Fetch all downloads for stats
    const fetchAllDownloads = async () => {
      try {
        const response = await fetch(`/api/downloads`, {
          credentials: "include",
        });
        if (response.ok) {
          const data = await response.json();
          setAllDownloads(data.downloads || []);
        }
      } catch (err) {
        console.error("Error fetching all downloads:", err);
      }
    };
    fetchAllDownloads();
    const interval = setInterval(fetchAllDownloads, 2000);
    return () => clearInterval(interval);
  }, []);

  const activeDownloads = allDownloads.filter((d) =>
    ["queued", "downloading", "processing"].includes(d.status),
  );
  const completedDownloads = allDownloads.filter(
    (d) => d.status === "completed",
  );
  const failedDownloads = allDownloads.filter((d) => d.status === "failed");

  if (loading && downloads.length === 0) {
    return (
      <div className="flex items-center justify-center h-64">
        <Loader2 className="w-8 h-8 animate-spin text-primary" />
      </div>
    );
  }

  return (
    <div className="p-6 max-w-7xl mx-auto">
      {/* Header with filters */}
      <div className="mb-6 flex items-start justify-between gap-6">
        <div>
          <h1 className="text-3xl font-bold flex items-center gap-3">
            <DownloadIcon className="w-8 h-8" />
            Downloads Queue
          </h1>
          <p className="mt-2 text-muted-foreground">
            Manage all downloads across different downloaders
          </p>
        </div>

        {/* Filters on the right */}
        <div className="flex gap-3">
          <div>
            <label className="block text-xs font-medium mb-1 text-muted-foreground">
              Status
            </label>
            <select
              value={filterStatus}
              onChange={(e) => setFilterStatus(e.target.value)}
              className="block px-3 py-2 bg-background border rounded-md focus:ring-2 focus:ring-primary focus:border-primary text-sm"
            >
              <option value="active">Active</option>
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
            <label className="block text-xs font-medium mb-1 text-muted-foreground">
              Downloader
            </label>
            <select
              value={filterPlugin}
              onChange={(e) => setFilterPlugin(e.target.value)}
              className="block px-3 py-2 bg-background border rounded-md focus:ring-2 focus:ring-primary focus:border-primary text-sm"
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

      {/* Compact Stats Cards - Clickable */}
      <div className="grid grid-cols-4 gap-3 mb-6">
        <button
          onClick={() => setFilterStatus("active")}
          className={`bg-blue-100 dark:bg-blue-950 border border-blue-200 dark:border-blue-900 rounded-lg p-3 text-left transition-all hover:shadow-md ${
            filterStatus === "active" ? "ring-2 ring-blue-500" : ""
          }`}
        >
          <div className="text-xs text-blue-700 dark:text-blue-400">Active</div>
          <div className="text-xl font-bold text-blue-800 dark:text-blue-300">
            {activeDownloads.length}
          </div>
        </button>

        <button
          onClick={() => setFilterStatus("all")}
          className={`bg-card border rounded-lg p-3 text-left transition-all hover:shadow-md ${
            filterStatus === "all" ? "ring-2 ring-primary" : ""
          }`}
        >
          <div className="text-xs text-muted-foreground">Total</div>
          <div className="text-xl font-bold">{allDownloads.length}</div>
        </button>

        <button
          onClick={() => setFilterStatus("completed")}
          className={`bg-green-100 dark:bg-green-950 border border-green-200 dark:border-green-900 rounded-lg p-3 text-left transition-all hover:shadow-md ${
            filterStatus === "completed" ? "ring-2 ring-green-500" : ""
          }`}
        >
          <div className="text-xs text-green-700 dark:text-green-400">
            Completed
          </div>
          <div className="text-xl font-bold text-green-800 dark:text-green-300">
            {completedDownloads.length}
          </div>
        </button>

        <button
          onClick={() => setFilterStatus("failed")}
          className={`bg-red-100 dark:bg-red-950 border border-red-200 dark:border-red-900 rounded-lg p-3 text-left transition-all hover:shadow-md ${
            filterStatus === "failed" ? "ring-2 ring-red-500" : ""
          }`}
        >
          <div className="text-xs text-red-700 dark:text-red-400">Failed</div>
          <div className="text-xl font-bold text-red-800 dark:text-red-300">
            {failedDownloads.length}
          </div>
        </button>
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
          <DownloadIcon className="w-16 h-16 mx-auto text-muted-foreground mb-4" />
          <h3 className="text-lg font-medium mb-2">No downloads</h3>
          <p className="text-muted-foreground">
            {filterStatus === "active"
              ? "No active downloads at the moment"
              : "No downloads match the selected filters"}
          </p>
        </div>
      ) : (
        <div className="space-y-3">
          {downloads.map((download) => {
            const mediaDisplayName = getMediaDisplayName(download);
            const mediaId = download.metadata?.media_id;

            return (
              <div
                key={download.id}
                className="bg-card border rounded-lg overflow-hidden"
              >
                <div className="p-3">
                  <div className="flex items-start justify-between mb-2">
                    <div className="flex-1 min-w-0">
                      {/* Media name as link if available */}
                      {mediaDisplayName && mediaId ? (
                        <>
                          <Link
                            to={`/media/${mediaId}`}
                            className="text-base font-semibold hover:underline"
                          >
                            {mediaDisplayName}
                          </Link>
                          <div className="text-sm text-muted-foreground truncate mt-0.5">
                            {download.name}
                          </div>
                        </>
                      ) : (
                        <h3 className="text-base font-semibold truncate">
                          {download.name}
                        </h3>
                      )}

                      <div className="flex items-center gap-2 mt-1 flex-wrap">
                        <span
                          className={`inline-flex items-center px-2 py-0.5 rounded-full text-xs font-medium ${getStatusColor(
                            download.status,
                          )}`}
                        >
                          {download.status.toUpperCase()}
                        </span>
                        <span className="text-xs text-muted-foreground">
                          {downloaders.find((d) => d.id === download.plugin_id)
                            ?.name || download.plugin_id}
                        </span>
                        {download.queue_position !== null &&
                          download.queue_position !== undefined && (
                            <span className="text-xs text-muted-foreground">
                              Queue: #{download.queue_position}
                            </span>
                          )}
                        {download.started_at && (
                          <span className="text-xs text-muted-foreground">
                            {formatDate(download.started_at)}
                          </span>
                        )}
                      </div>
                    </div>

                    <div className="flex items-center gap-1.5 ml-4">
                      {download.status === "downloading" && (
                        <button
                          onClick={() => handlePause(download)}
                          className="p-1.5 text-yellow-700 dark:text-yellow-400 hover:bg-yellow-100 dark:hover:bg-yellow-950 rounded transition-colors"
                          title="Pause"
                        >
                          <Pause className="w-4 h-4" />
                        </button>
                      )}
                      {download.status === "paused" && (
                        <button
                          onClick={() => handleResume(download)}
                          className="p-1.5 text-green-700 dark:text-green-400 hover:bg-green-100 dark:hover:bg-green-950 rounded transition-colors"
                          title="Resume"
                        >
                          <Play className="w-4 h-4" />
                        </button>
                      )}
                      {download.status === "failed" && (
                        <button
                          onClick={() => handleRetry(download)}
                          className="p-1.5 text-blue-700 dark:text-blue-400 hover:bg-blue-100 dark:hover:bg-blue-950 rounded transition-colors"
                          title="Retry"
                        >
                          <RefreshCw className="w-4 h-4" />
                        </button>
                      )}
                      <button
                        onClick={() => handleDelete(download)}
                        className="p-1.5 text-red-700 dark:text-red-400 hover:bg-red-100 dark:hover:bg-red-950 rounded transition-colors"
                        title="Delete"
                      >
                        <Trash2 className="w-4 h-4" />
                      </button>
                    </div>
                  </div>

                  {/* Progress Bar */}
                  {["downloading", "processing", "queued"].includes(
                    download.status,
                  ) && (
                    <div className="mb-2">
                      <div className="flex items-center justify-between text-xs text-muted-foreground mb-1">
                        <div className="flex items-center gap-2">
                          <span>{download.progress.toFixed(1)}%</span>
                          {download.speed && download.speed > 0 && (
                            <span>{formatBytes(download.speed)}/s</span>
                          )}
                        </div>
                        <span className="text-xs">
                          {formatBytes(download.downloaded_bytes)}
                          {download.total_bytes &&
                            ` / ${formatBytes(download.total_bytes)}`}
                        </span>
                      </div>
                      <div className="w-full bg-secondary rounded-full h-1.5">
                        <div
                          className="bg-primary h-1.5 rounded-full transition-all duration-300"
                          style={{ width: `${download.progress}%` }}
                        />
                      </div>
                    </div>
                  )}

                  {/* Error Message */}
                  {download.error_message && (
                    <div className="mb-2 p-2 bg-red-100 dark:bg-red-950 border border-red-300 dark:border-red-800 rounded flex items-start gap-2">
                      <AlertCircle className="w-3.5 h-3.5 text-red-700 dark:text-red-400 flex-shrink-0 mt-0.5" />
                      <span className="text-xs text-red-900 dark:text-red-300">
                        {download.error_message}
                      </span>
                    </div>
                  )}
                </div>
              </div>
            );
          })}
        </div>
      )}
    </div>
  );
}
