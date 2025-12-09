import { useState, useEffect } from "react";
import {
  ChevronsUp,
  ChevronUp,
  ChevronDown,
  ChevronsDown,
  Trash2,
  X,
} from "lucide-react";

// Alert Modal Component
interface AlertModalProps {
  isOpen: boolean;
  title: string;
  message: string;
  onClose: () => void;
}

function AlertModal({ isOpen, title, message, onClose }: AlertModalProps) {
  if (!isOpen) return null;

  return (
    <div className="fixed inset-0 bg-black/50 flex items-center justify-center p-4 z-50">
      <div className="bg-card border rounded-lg p-6 max-w-md w-full space-y-4">
        <h3 className="text-lg font-semibold">{title}</h3>
        <p className="text-sm text-muted-foreground">{message}</p>
        <div className="flex justify-end">
          <button
            className="px-4 py-2 bg-primary text-primary-foreground rounded-md hover:bg-primary/90"
            onClick={onClose}
          >
            OK
          </button>
        </div>
      </div>
    </div>
  );
}

// Confirm Modal Component
interface ConfirmModalProps {
  isOpen: boolean;
  title: string;
  message: string;
  onConfirm: () => void;
  onCancel: () => void;
}

function ConfirmModal({
  isOpen,
  title,
  message,
  onConfirm,
  onCancel,
}: ConfirmModalProps) {
  if (!isOpen) return null;

  return (
    <div className="fixed inset-0 bg-black/50 flex items-center justify-center p-4 z-50">
      <div className="bg-card border rounded-lg p-6 max-w-md w-full space-y-4">
        <h3 className="text-lg font-semibold">{title}</h3>
        <p className="text-sm text-muted-foreground">{message}</p>
        <div className="flex justify-end space-x-2">
          <button
            className="px-4 py-2 bg-secondary text-secondary-foreground rounded-md hover:bg-secondary/90"
            onClick={onCancel}
          >
            Cancel
          </button>
          <button
            className="px-4 py-2 bg-red-500 text-white rounded-md hover:bg-red-600"
            onClick={onConfirm}
          >
            Confirm
          </button>
        </div>
      </div>
    </div>
  );
}

interface Server {
  id: string;
  name: string;
  host: string;
  port: number;
  username: string;
  password: string;
  use_ssl: boolean;
  enabled: boolean;
  connections: number;
  priority: number;
}

interface Download {
  id: string;
  name: string;
  status: string;
  progress: number;
  total_bytes: number;
  downloaded_bytes: number;
  speed: number;
  eta: number;
  added_at: string;
  started_at?: string;
  completed_at?: string;
  error?: string;
}

export default function NZBDownloaderPage() {
  const [servers, setServers] = useState<Server[]>([]);
  const [downloads, setDownloads] = useState<Download[]>([]);
  const [config, setConfig] = useState({
    download_dir: "/tmp/nzb-downloads",
    connections: 10,
  });
  const [loading, setLoading] = useState(false);
  const [showAddModal, setShowAddModal] = useState(false);
  const [showSettingsModal, setShowSettingsModal] = useState(false);
  const [showServerForm, setShowServerForm] = useState(false);
  const [editingServer, setEditingServer] = useState<Server | null>(null);
  const [nzbUrl, setNzbUrl] = useState("");
  const [nzbFile, setNzbFile] = useState<File | null>(null);
  const [selectedDownloads, setSelectedDownloads] = useState<Set<string>>(
    new Set(),
  );

  // Alert/Confirm modal state
  const [alertModal, setAlertModal] = useState<{
    isOpen: boolean;
    title: string;
    message: string;
  }>({
    isOpen: false,
    title: "",
    message: "",
  });
  const [confirmModal, setConfirmModal] = useState<{
    isOpen: boolean;
    title: string;
    message: string;
    onConfirm: () => void;
  }>({
    isOpen: false,
    title: "",
    message: "",
    onConfirm: () => {},
  });

  const showAlert = (title: string, message: string) => {
    setAlertModal({ isOpen: true, title, message });
  };

  const showConfirm = (
    title: string,
    message: string,
    onConfirm: () => void,
  ) => {
    setConfirmModal({ isOpen: true, title, message, onConfirm });
  };

  useEffect(() => {
    loadServers();
    loadDownloads();
    loadConfig();
  }, []);

  // Smart polling - only poll when there are active downloads
  useEffect(() => {
    const hasActiveDownloads = downloads.some(
      (d) => d.status === "downloading" || d.status === "queued",
    );

    if (!hasActiveDownloads) {
      return; // No polling needed
    }

    // Poll every 1 second when there are active downloads
    const interval = setInterval(loadDownloads, 1000);
    return () => clearInterval(interval);
  }, [downloads]);

  const loadServers = async () => {
    try {
      const response = await fetch("/api/plugins/nzb-downloader/servers", {
        credentials: "include",
      });
      if (response.ok) {
        const data = await response.json();
        setServers(data.servers || []);
      }
    } catch (error) {
      console.error("Failed to load servers:", error);
    }
  };

  const loadDownloads = async () => {
    try {
      const response = await fetch("/api/plugins/nzb-downloader/downloads", {
        credentials: "include",
      });
      if (response.ok) {
        const data = await response.json();
        setDownloads(data.downloads || []);
      }
    } catch (error) {
      console.error("Failed to load downloads:", error);
    }
  };

  const loadConfig = async () => {
    try {
      const response = await fetch("/api/plugins/nzb-downloader/config", {
        credentials: "include",
      });
      if (response.ok) {
        const data = await response.json();
        setConfig({
          download_dir: data.download_dir || "/tmp/nzb-downloads",
          connections: data.connections || 10,
        });
      }
    } catch (error) {
      console.error("Failed to load config:", error);
    }
  };

  const saveServer = async (server: Server) => {
    try {
      const isUpdate = server.id && server.id !== "";
      const url = isUpdate
        ? `/api/plugins/nzb-downloader/servers/${server.id}`
        : "/api/plugins/nzb-downloader/servers";
      const method = isUpdate ? "PUT" : "POST";

      const response = await fetch(url, {
        method,
        headers: { "Content-Type": "application/json" },
        credentials: "include",
        body: JSON.stringify(server),
      });

      if (response.ok) {
        await loadServers();
        setShowServerForm(false);
        setEditingServer(null);
      } else {
        const errorData = await response
          .json()
          .catch(() => ({ error: "Unknown error" }));
        console.error("Server save failed:", errorData);
        showAlert(
          "Server Save Failed",
          `Failed to save server: ${errorData.error || response.statusText}`,
        );
      }
    } catch (error: any) {
      console.error("Failed to save server:", error);
      showAlert(
        "Server Save Failed",
        `Failed to save server: ${error.message}`,
      );
    }
  };

  const addDownload = async () => {
    if (!nzbUrl && !nzbFile) {
      showAlert("Missing Input", "Please provide an NZB URL or file");
      return;
    }

    setLoading(true);
    try {
      let body: string;

      if (nzbUrl) {
        // Send URL with empty name (backend will extract from URL)
        body = JSON.stringify({ url: nzbUrl, name: "" });
      } else if (nzbFile) {
        // Send NZB content with filename
        const nzbContent = await nzbFile.text();
        const filename = nzbFile.name.replace(/\.nzb$/i, ""); // Remove .nzb extension
        body = JSON.stringify({
          nzb: nzbContent,
          name: filename,
        });
      } else {
        showAlert("Missing Input", "Please provide an NZB URL or file");
        return;
      }

      const response = await fetch("/api/plugins/nzb-downloader/downloads", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        credentials: "include",
        body,
      });

      if (response.ok) {
        setNzbUrl("");
        setNzbFile(null);
        setShowAddModal(false);
        await loadDownloads();
      } else {
        const errorData = await response
          .json()
          .catch(() => ({ error: "Unknown error" }));
        showAlert(
          "Download Failed",
          `Failed to add download: ${errorData.error || response.statusText}`,
        );
      }
    } catch (error: any) {
      console.error("Failed to add download:", error);
      showAlert("Download Failed", `Failed to add download: ${error.message}`);
    } finally {
      setLoading(false);
    }
  };

  const formatBytes = (bytes: number): string => {
    if (bytes === 0) return "0 B";
    const k = 1024;
    const sizes = ["B", "KB", "MB", "GB", "TB"];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return Math.round((bytes / Math.pow(k, i)) * 100) / 100 + " " + sizes[i];
  };

  const formatSpeed = (bytesPerSec: number): string => {
    return formatBytes(bytesPerSec) + "/s";
  };

  const formatTime = (seconds: number): string => {
    if (seconds < 60) return `${seconds}s`;
    if (seconds < 3600) return `${Math.floor(seconds / 60)}m ${seconds % 60}s`;
    return `${Math.floor(seconds / 3600)}h ${Math.floor((seconds % 3600) / 60)}m`;
  };

  const getStatusColor = (status: string): string => {
    switch (status) {
      case "downloading":
        return "text-blue-600 bg-blue-500/10";
      case "completed":
        return "text-green-600 bg-green-500/10";
      case "failed":
        return "text-red-600 bg-red-500/10";
      case "paused":
        return "text-yellow-600 bg-yellow-500/10";
      default:
        return "text-gray-600 bg-gray-500/10";
    }
  };

  const toggleSelectDownload = (downloadId: string) => {
    setSelectedDownloads((prev) => {
      const newSet = new Set(prev);
      if (newSet.has(downloadId)) {
        newSet.delete(downloadId);
      } else {
        newSet.add(downloadId);
      }
      return newSet;
    });
  };

  const toggleSelectAll = () => {
    const activeDownloads = downloads.filter((d) => d.status !== "completed");
    if (selectedDownloads.size === activeDownloads.length) {
      setSelectedDownloads(new Set());
    } else {
      setSelectedDownloads(new Set(activeDownloads.map((d) => d.id)));
    }
  };

  const deleteDownload = async (downloadId: string) => {
    try {
      const response = await fetch(
        `/api/plugins/nzb-downloader/downloads/${downloadId}`,
        {
          method: "DELETE",
          credentials: "include",
        },
      );
      if (response.ok) {
        await loadDownloads();
        setSelectedDownloads((prev) => {
          const newSet = new Set(prev);
          newSet.delete(downloadId);
          return newSet;
        });
      } else {
        showAlert("Delete Failed", "Failed to delete download");
      }
    } catch (error) {
      console.error("Failed to delete download:", error);
      showAlert("Delete Failed", "Failed to delete download");
    }
  };

  const deleteSelectedDownloads = () => {
    showConfirm(
      "Delete Downloads",
      `Are you sure you want to delete ${selectedDownloads.size} download(s)?`,
      async () => {
        for (const downloadId of selectedDownloads) {
          await deleteDownload(downloadId);
        }
        setConfirmModal({ ...confirmModal, isOpen: false });
      },
    );
  };

  const moveDownloads = async (direction: "up" | "down" | "top" | "bottom") => {
    if (selectedDownloads.size === 0) return;

    try {
      const downloadIds = Array.from(selectedDownloads);
      const response = await fetch(
        "/api/plugins/nzb-downloader/downloads/move",
        {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          credentials: "include",
          body: JSON.stringify({
            download_ids: downloadIds,
            direction: direction,
          }),
        },
      );

      if (response.ok) {
        await loadDownloads();
      } else {
        showAlert("Move Failed", "Failed to reorder downloads in queue");
      }
    } catch (error) {
      console.error("Failed to move downloads:", error);
      showAlert("Move Failed", "Failed to reorder downloads in queue");
    }
  };

  return (
    <div className="h-full flex flex-col p-6">
      {/* Header */}
      <div className="flex items-center justify-between mb-6">
        <div>
          <h1 className="text-3xl font-bold">NZB Downloader</h1>
          <p className="text-muted-foreground">Download queue</p>
        </div>
        <div className="flex space-x-2">
          <button
            className="px-4 py-2 bg-primary text-primary-foreground rounded-md hover:bg-primary/90 flex items-center space-x-2"
            onClick={() => setShowAddModal(true)}
          >
            <span>+</span>
            <span>Add Download</span>
          </button>
          <button
            className="px-4 py-2 bg-secondary text-secondary-foreground rounded-md hover:bg-secondary/90"
            onClick={() => setShowSettingsModal(true)}
          >
            Settings
          </button>
        </div>
      </div>

      {/* Downloads Queue */}
      <div className="flex-1 bg-card border rounded-lg overflow-hidden flex flex-col">
        <div className="p-4 border-b flex items-center justify-between">
          <div className="flex items-center space-x-3">
            <input
              type="checkbox"
              className="rounded"
              checked={
                selectedDownloads.size > 0 &&
                selectedDownloads.size ===
                  downloads.filter((d) => d.status !== "completed").length
              }
              onChange={toggleSelectAll}
            />
            <h2 className="text-xl font-semibold">
              Queue ({downloads.filter((d) => d.status !== "completed").length})
            </h2>
          </div>

          {/* Toolbar - shown when downloads are selected */}
          {selectedDownloads.size > 0 && (
            <div className="flex items-center space-x-2">
              <span className="text-sm text-muted-foreground mr-2">
                {selectedDownloads.size} selected
              </span>
              <button
                className="px-3 py-1.5 text-sm bg-secondary text-secondary-foreground rounded-md hover:bg-secondary/90 flex items-center space-x-1"
                onClick={() => moveDownloads("top")}
                title="Move to top"
              >
                <ChevronsUp className="h-4 w-4" />
              </button>
              <button
                className="px-3 py-1.5 text-sm bg-secondary text-secondary-foreground rounded-md hover:bg-secondary/90 flex items-center space-x-1"
                onClick={() => moveDownloads("up")}
                title="Move up"
              >
                <ChevronUp className="h-4 w-4" />
              </button>
              <button
                className="px-3 py-1.5 text-sm bg-secondary text-secondary-foreground rounded-md hover:bg-secondary/90 flex items-center space-x-1"
                onClick={() => moveDownloads("down")}
                title="Move down"
              >
                <ChevronDown className="h-4 w-4" />
              </button>
              <button
                className="px-3 py-1.5 text-sm bg-secondary text-secondary-foreground rounded-md hover:bg-secondary/90 flex items-center space-x-1"
                onClick={() => moveDownloads("bottom")}
                title="Move to bottom"
              >
                <ChevronsDown className="h-4 w-4" />
              </button>
              <button
                className="px-3 py-1.5 text-sm bg-red-500 text-white rounded-md hover:bg-red-600 flex items-center space-x-1"
                onClick={deleteSelectedDownloads}
                title="Delete selected"
              >
                <Trash2 className="h-4 w-4" />
                <span>Delete</span>
              </button>
            </div>
          )}
        </div>

        <div className="flex-1 overflow-y-auto p-4">
          {downloads.filter((d) => d.status !== "completed").length === 0 ? (
            <div className="flex flex-col items-center justify-center h-full text-center">
              <p className="text-muted-foreground text-lg mb-2">
                No downloads in queue
              </p>
              <p className="text-sm text-muted-foreground">
                Click "Add Download" to get started
              </p>
            </div>
          ) : (
            <div className="space-y-2">
              {downloads
                .filter((d) => d.status !== "completed")
                .map((download) => (
                  <div
                    key={download.id}
                    className="p-4 bg-muted rounded-md space-y-2"
                  >
                    <div className="flex items-center space-x-3">
                      {/* Checkbox */}
                      <input
                        type="checkbox"
                        className="rounded"
                        checked={selectedDownloads.has(download.id)}
                        onChange={() => toggleSelectDownload(download.id)}
                      />

                      {/* Download info */}
                      <div className="flex-1 min-w-0">
                        <div className="flex items-center justify-between">
                          <div className="flex-1 min-w-0 mr-4">
                            <h3 className="font-medium truncate">
                              {/* Show actual filename or "Unnamed" if it's the generated download-timestamp format */}
                              {/^download-\d+$/.test(download.name)
                                ? "Unnamed Download"
                                : download.name}
                            </h3>
                            <p className="text-xs text-muted-foreground truncate">
                              {download.id}
                            </p>
                          </div>
                          <div className="flex items-center space-x-2">
                            <span
                              className={`text-xs px-2 py-1 rounded whitespace-nowrap ${getStatusColor(
                                download.status,
                              )}`}
                            >
                              {download.status}
                            </span>
                            {/* Delete button for queued/failed downloads */}
                            {(download.status === "queued" ||
                              download.status === "failed") && (
                              <button
                                className="text-red-500 hover:text-red-700"
                                onClick={() => {
                                  showConfirm(
                                    "Delete Download",
                                    `Are you sure you want to delete "${download.name}"?`,
                                    async () => {
                                      await deleteDownload(download.id);
                                      setConfirmModal({
                                        ...confirmModal,
                                        isOpen: false,
                                      });
                                    },
                                  );
                                }}
                                title="Delete download"
                              >
                                <X className="h-4 w-4" />
                              </button>
                            )}
                          </div>
                        </div>
                      </div>
                    </div>

                    {download.status === "downloading" && (
                      <>
                        <div className="w-full bg-background rounded-full h-2">
                          <div
                            className="bg-primary h-2 rounded-full transition-all"
                            style={{ width: `${download.progress}%` }}
                          />
                        </div>
                        <div className="flex items-center justify-between text-xs text-muted-foreground">
                          <span>
                            {formatBytes(download.downloaded_bytes)} /{" "}
                            {formatBytes(download.total_bytes)} (
                            {download.progress.toFixed(1)}%)
                          </span>
                          <span>
                            {formatSpeed(download.speed)} • ETA:{" "}
                            {formatTime(download.eta)}
                          </span>
                        </div>
                      </>
                    )}

                    {download.status === "completed" && (
                      <p className="text-xs text-muted-foreground">
                        Completed{" "}
                        {new Date(download.completed_at!).toLocaleString()}
                      </p>
                    )}

                    {download.error && (
                      <p className="text-xs text-red-600">
                        Error: {download.error}
                      </p>
                    )}
                  </div>
                ))}
            </div>
          )}
        </div>
      </div>

      {/* Add Download Modal */}
      {showAddModal && (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center p-4 z-50">
          <div className="bg-card border rounded-lg p-6 max-w-md w-full space-y-4">
            <h2 className="text-xl font-semibold">Add Download</h2>

            <div className="space-y-2">
              <label className="block text-sm font-medium">NZB URL</label>
              <input
                type="text"
                className="w-full px-3 py-2 bg-background border rounded-md"
                placeholder="https://example.com/file.nzb"
                value={nzbUrl}
                onChange={(e) => setNzbUrl(e.target.value)}
              />
            </div>

            <div className="text-center text-sm text-muted-foreground">OR</div>

            <div className="space-y-2">
              <label className="block text-sm font-medium">
                Upload NZB File
              </label>
              <input
                type="file"
                accept=".nzb"
                className="w-full px-3 py-2 bg-background border rounded-md"
                onChange={(e) => setNzbFile(e.target.files?.[0] || null)}
              />
            </div>

            {servers.length === 0 && (
              <p className="text-xs text-red-600">
                Please configure at least one NNTP server first
              </p>
            )}

            <div className="flex space-x-3 pt-4">
              <button
                className="flex-1 px-4 py-2 bg-primary text-primary-foreground rounded-md hover:bg-primary/90 disabled:opacity-50"
                onClick={addDownload}
                disabled={
                  loading || (!nzbUrl && !nzbFile) || servers.length === 0
                }
              >
                {loading ? "Adding..." : "Add to Queue"}
              </button>
              <button
                className="px-4 py-2 bg-secondary text-secondary-foreground rounded-md hover:bg-secondary/90"
                onClick={() => {
                  setShowAddModal(false);
                  setNzbUrl("");
                  setNzbFile(null);
                }}
              >
                Cancel
              </button>
            </div>
          </div>
        </div>
      )}

      {/* Settings Modal */}
      {showSettingsModal && (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center p-4 z-50">
          <div className="bg-card border rounded-lg p-6 max-w-4xl w-full max-h-[90vh] overflow-y-auto space-y-6">
            <div className="flex items-center justify-between">
              <h2 className="text-2xl font-semibold">Settings</h2>
              <button
                className="px-4 py-2 bg-secondary text-secondary-foreground rounded-md hover:bg-secondary/90"
                onClick={() => {
                  setShowSettingsModal(false);
                  setShowServerForm(false);
                  setEditingServer(null);
                }}
              >
                Close
              </button>
            </div>

            {/* NNTP Servers Section */}
            <div className="space-y-4">
              <div className="flex items-center justify-between">
                <h3 className="text-xl font-semibold">
                  NNTP Servers ({servers.length})
                </h3>
                <button
                  className="px-4 py-2 bg-primary text-primary-foreground rounded-md hover:bg-primary/90 text-sm"
                  onClick={() => {
                    setEditingServer({
                      id: "",
                      name: "",
                      host: "",
                      port: 119,
                      username: "",
                      password: "",
                      use_ssl: false,
                      enabled: true,
                      connections: 10,
                      priority: 0,
                    });
                    setShowServerForm(true);
                  }}
                >
                  Add Server
                </button>
              </div>

              {servers.length === 0 && !showServerForm ? (
                <p className="text-muted-foreground text-center py-4">
                  No NNTP servers configured. Click "Add Server" to get started.
                </p>
              ) : (
                <div className="space-y-2">
                  {servers.map((server) => (
                    <div
                      key={server.id}
                      className="flex items-center justify-between p-4 bg-muted rounded-md"
                    >
                      <div className="flex-1">
                        <div className="flex items-center space-x-2">
                          <h4 className="font-medium">{server.name}</h4>
                          {!server.enabled && (
                            <span className="text-xs px-2 py-1 bg-red-500/10 text-red-600 rounded">
                              Disabled
                            </span>
                          )}
                          {server.use_ssl && (
                            <span className="text-xs px-2 py-1 bg-green-500/10 text-green-600 rounded">
                              SSL
                            </span>
                          )}
                        </div>
                        <p className="text-sm text-muted-foreground">
                          {server.host}:{server.port} • {server.connections}{" "}
                          connections
                        </p>
                      </div>
                      <div className="flex space-x-2">
                        <button
                          className="px-3 py-1 text-sm bg-secondary text-secondary-foreground rounded-md hover:bg-secondary/90"
                          onClick={() => {
                            setEditingServer(server);
                            setShowServerForm(true);
                          }}
                        >
                          Edit
                        </button>
                        <button
                          className="px-3 py-1 text-sm bg-red-500 text-white rounded-md hover:bg-red-600"
                          onClick={() => {
                            showConfirm(
                              "Delete Server",
                              `Are you sure you want to delete server "${server.name}"?`,
                              async () => {
                                try {
                                  const response = await fetch(
                                    `/api/plugins/nzb-downloader/servers/${server.id}`,
                                    {
                                      method: "DELETE",
                                      credentials: "include",
                                    },
                                  );
                                  if (response.ok) {
                                    await loadServers();
                                  } else {
                                    showAlert(
                                      "Delete Failed",
                                      "Failed to delete server",
                                    );
                                  }
                                } catch (error) {
                                  console.error(
                                    "Failed to delete server:",
                                    error,
                                  );
                                  showAlert(
                                    "Delete Failed",
                                    "Failed to delete server",
                                  );
                                }
                                setConfirmModal({
                                  ...confirmModal,
                                  isOpen: false,
                                });
                              },
                            );
                          }}
                        >
                          Delete
                        </button>
                      </div>
                    </div>
                  ))}
                </div>
              )}

              {/* Server Form */}
              {showServerForm && editingServer && (
                <div className="bg-background border rounded-lg p-4 space-y-4">
                  <h4 className="text-lg font-semibold">
                    {editingServer.id ? "Edit Server" : "New Server"}
                  </h4>

                  <div className="grid grid-cols-2 gap-4">
                    <div className="space-y-2">
                      <label className="block text-sm font-medium">Name</label>
                      <input
                        type="text"
                        className="w-full px-3 py-2 bg-background border rounded-md"
                        value={editingServer.name}
                        onChange={(e) =>
                          setEditingServer({
                            ...editingServer,
                            name: e.target.value,
                          })
                        }
                      />
                    </div>

                    <div className="space-y-2">
                      <label className="block text-sm font-medium">Host</label>
                      <input
                        type="text"
                        className="w-full px-3 py-2 bg-background border rounded-md"
                        value={editingServer.host}
                        onChange={(e) =>
                          setEditingServer({
                            ...editingServer,
                            host: e.target.value,
                          })
                        }
                      />
                    </div>

                    <div className="space-y-2">
                      <label className="block text-sm font-medium">Port</label>
                      <input
                        type="number"
                        className="w-full px-3 py-2 bg-background border rounded-md"
                        value={editingServer.port}
                        onChange={(e) =>
                          setEditingServer({
                            ...editingServer,
                            port: parseInt(e.target.value) || 119,
                          })
                        }
                      />
                    </div>

                    <div className="space-y-2">
                      <label className="block text-sm font-medium">
                        Connections
                      </label>
                      <input
                        type="number"
                        className="w-full px-3 py-2 bg-background border rounded-md"
                        value={editingServer.connections}
                        onChange={(e) =>
                          setEditingServer({
                            ...editingServer,
                            connections: parseInt(e.target.value) || 10,
                          })
                        }
                      />
                    </div>

                    <div className="space-y-2">
                      <label className="block text-sm font-medium">
                        Username
                      </label>
                      <input
                        type="text"
                        className="w-full px-3 py-2 bg-background border rounded-md"
                        value={editingServer.username}
                        onChange={(e) =>
                          setEditingServer({
                            ...editingServer,
                            username: e.target.value,
                          })
                        }
                      />
                    </div>

                    <div className="space-y-2">
                      <label className="block text-sm font-medium">
                        Password
                      </label>
                      <input
                        type="password"
                        className="w-full px-3 py-2 bg-background border rounded-md"
                        value={editingServer.password}
                        onChange={(e) =>
                          setEditingServer({
                            ...editingServer,
                            password: e.target.value,
                          })
                        }
                      />
                    </div>
                  </div>

                  <div className="flex items-center space-x-4">
                    <div className="flex items-center space-x-2">
                      <input
                        type="checkbox"
                        id="use-ssl"
                        checked={editingServer.use_ssl}
                        onChange={(e) =>
                          setEditingServer({
                            ...editingServer,
                            use_ssl: e.target.checked,
                          })
                        }
                        className="rounded"
                      />
                      <label htmlFor="use-ssl" className="text-sm font-medium">
                        Use SSL
                      </label>
                    </div>

                    <div className="flex items-center space-x-2">
                      <input
                        type="checkbox"
                        id="enabled"
                        checked={editingServer.enabled}
                        onChange={(e) =>
                          setEditingServer({
                            ...editingServer,
                            enabled: e.target.checked,
                          })
                        }
                        className="rounded"
                      />
                      <label htmlFor="enabled" className="text-sm font-medium">
                        Enabled
                      </label>
                    </div>
                  </div>

                  <div className="flex space-x-3">
                    <button
                      className="px-4 py-2 bg-primary text-primary-foreground rounded-md hover:bg-primary/90"
                      onClick={() => {
                        saveServer(editingServer);
                        setShowServerForm(false);
                        setEditingServer(null);
                      }}
                    >
                      Save
                    </button>
                    <button
                      className="px-4 py-2 bg-secondary text-secondary-foreground rounded-md hover:bg-secondary/90"
                      onClick={() => {
                        setShowServerForm(false);
                        setEditingServer(null);
                      }}
                    >
                      Cancel
                    </button>
                  </div>
                </div>
              )}
            </div>

            {/* Configuration Section */}
            <div className="space-y-4 border-t pt-6">
              <h3 className="text-xl font-semibold">Configuration</h3>

              <div className="space-y-2">
                <label className="block text-sm font-medium">
                  Download Directory
                </label>
                <input
                  type="text"
                  className="w-full px-3 py-2 bg-background border rounded-md"
                  value={config.download_dir}
                  onChange={(e) =>
                    setConfig({ ...config, download_dir: e.target.value })
                  }
                  placeholder="/tmp/nzb-downloads"
                />
              </div>

              <button
                className="px-4 py-2 bg-primary text-primary-foreground rounded-md hover:bg-primary/90"
                onClick={async () => {
                  try {
                    await fetch("/api/plugins/nzb-downloader/config", {
                      method: "POST",
                      headers: { "Content-Type": "application/json" },
                      credentials: "include",
                      body: JSON.stringify(config),
                    });
                    showAlert("Success", "Configuration saved successfully");
                  } catch (error) {
                    showAlert("Save Failed", "Failed to save configuration");
                  }
                }}
              >
                Save Configuration
              </button>
            </div>
          </div>
        </div>
      )}

      {/* Alert Modal */}
      <AlertModal
        isOpen={alertModal.isOpen}
        title={alertModal.title}
        message={alertModal.message}
        onClose={() => setAlertModal({ ...alertModal, isOpen: false })}
      />

      {/* Confirm Modal */}
      <ConfirmModal
        isOpen={confirmModal.isOpen}
        title={confirmModal.title}
        message={confirmModal.message}
        onConfirm={confirmModal.onConfirm}
        onCancel={() => setConfirmModal({ ...confirmModal, isOpen: false })}
      />
    </div>
  );
}
