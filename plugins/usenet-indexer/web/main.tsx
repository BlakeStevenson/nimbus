import { useState, useEffect } from "react";

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

interface Indexer {
  id: string;
  name: string;
  url: string;
  api_key: string;
  enabled: boolean;
  priority: number;
  tv_categories: string[];
  movie_categories: string[];
}

interface Release {
  id: string;
  title: string;
  size: number;
  publishDate: string;
  category: string;
  downloadUrl: string;
  attributes: Record<string, string>;
}

interface SearchResult {
  releases: Release[];
  count: number;
}

export default function UsenetIndexerPage() {
  const [indexers, setIndexers] = useState<Indexer[]>([]);
  const [loading, setLoading] = useState(false);
  const [showIndexerForm, setShowIndexerForm] = useState(false);
  const [editingIndexer, setEditingIndexer] = useState<Indexer | null>(null);
  const [searchQuery, setSearchQuery] = useState("");
  const [searchType, setSearchType] = useState<"general" | "tv" | "movie">(
    "general",
  );
  const [searchResults, setSearchResults] = useState<Release[] | null>(null);
  const [searching, setSearching] = useState(false);

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
    loadIndexers();
  }, []);

  const loadIndexers = async () => {
    setLoading(true);
    try {
      const response = await fetch("/api/plugins/usenet-indexer/indexers", {
        credentials: "include",
      });
      if (response.ok) {
        const data = await response.json();
        setIndexers(data.indexers || []);
      }
    } catch (error) {
      console.error("Failed to load indexers:", error);
    } finally {
      setLoading(false);
    }
  };

  const saveIndexer = async (indexer: Indexer) => {
    try {
      const isUpdate = indexer.id && indexer.id !== "";
      const url = isUpdate
        ? `/api/plugins/usenet-indexer/indexers/${indexer.id}`
        : "/api/plugins/usenet-indexer/indexers";
      const method = isUpdate ? "PUT" : "POST";

      const response = await fetch(url, {
        method,
        headers: { "Content-Type": "application/json" },
        credentials: "include",
        body: JSON.stringify(indexer),
      });

      if (response.ok) {
        await loadIndexers();
        setShowIndexerForm(false);
        setEditingIndexer(null);
      } else {
        const errorData = await response
          .json()
          .catch(() => ({ error: "Unknown error" }));
        showAlert(
          "Save Failed",
          `Failed to save indexer: ${errorData.error || response.statusText}`,
        );
      }
    } catch (error: any) {
      console.error("Failed to save indexer:", error);
      showAlert("Save Failed", `Failed to save indexer: ${error.message}`);
    }
  };

  const testIndexer = async (indexerId: string) => {
    try {
      const response = await fetch(
        `/api/plugins/usenet-indexer/indexers/${indexerId}/test`,
        {
          method: "POST",
          credentials: "include",
        },
      );
      const data = await response.json();
      if (data.success) {
        showAlert("Connection Successful", data.message);
      } else {
        showAlert("Connection Failed", data.error);
      }
    } catch (error: any) {
      showAlert("Test Failed", error.message);
    }
  };

  const performSearch = async () => {
    if (!searchQuery.trim()) return;

    setSearching(true);
    setSearchResults(null);
    try {
      let endpoint = "/api/plugins/usenet-indexer/search";
      if (searchType === "tv") {
        endpoint = "/api/plugins/usenet-indexer/search/tv";
      } else if (searchType === "movie") {
        endpoint = "/api/plugins/usenet-indexer/search/movie";
      }

      const response = await fetch(
        `${endpoint}?q=${encodeURIComponent(searchQuery)}&limit=50`,
        { credentials: "include" },
      );
      if (response.ok) {
        const data: SearchResult = await response.json();
        setSearchResults(data.releases);
      } else {
        showAlert("Search Failed", "Failed to search indexers");
      }
    } catch (error) {
      console.error("Search failed:", error);
      showAlert("Search Failed", "Failed to search indexers");
    } finally {
      setSearching(false);
    }
  };

  const formatBytes = (bytes: number): string => {
    if (bytes === 0) return "0 B";
    const k = 1024;
    const sizes = ["B", "KB", "MB", "GB", "TB"];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return Math.round((bytes / Math.pow(k, i)) * 100) / 100 + " " + sizes[i];
  };

  return (
    <div className="space-y-6 p-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-3xl font-bold">Usenet Indexer</h1>
          <p className="text-muted-foreground">
            Configure Newznab-compatible Usenet indexers
          </p>
        </div>
        <button
          className="px-4 py-2 bg-primary text-primary-foreground rounded-md hover:bg-primary/90"
          onClick={() => {
            setEditingIndexer({
              id: "",
              name: "",
              url: "",
              api_key: "",
              enabled: true,
              priority: 0,
              tv_categories: [],
              movie_categories: [],
            });
            setShowIndexerForm(true);
          }}
        >
          Add Indexer
        </button>
      </div>

      {/* Indexers List */}
      <div className="bg-card border rounded-lg p-6 space-y-4">
        <h2 className="text-xl font-semibold">Indexers ({indexers.length})</h2>

        {indexers.length === 0 && !showIndexerForm ? (
          <p className="text-muted-foreground text-center py-4">
            No indexers configured. Click "Add Indexer" to get started.
          </p>
        ) : (
          <div className="space-y-2">
            {indexers.map((indexer) => (
              <div
                key={indexer.id}
                className="flex items-center justify-between p-4 bg-muted rounded-md"
              >
                <div className="flex-1">
                  <div className="flex items-center space-x-2">
                    <h4 className="font-medium">{indexer.name}</h4>
                    {!indexer.enabled && (
                      <span className="text-xs px-2 py-1 bg-red-500/10 text-red-600 rounded">
                        Disabled
                      </span>
                    )}
                  </div>
                  <p className="text-sm text-muted-foreground">{indexer.url}</p>
                </div>
                <div className="flex space-x-2">
                  <button
                    className="px-3 py-1 text-sm bg-blue-500 text-white rounded-md hover:bg-blue-600"
                    onClick={() => testIndexer(indexer.id)}
                  >
                    Test
                  </button>
                  <button
                    className="px-3 py-1 text-sm bg-secondary text-secondary-foreground rounded-md hover:bg-secondary/90"
                    onClick={() => {
                      setEditingIndexer(indexer);
                      setShowIndexerForm(true);
                    }}
                  >
                    Edit
                  </button>
                  <button
                    className="px-3 py-1 text-sm bg-red-500 text-white rounded-md hover:bg-red-600"
                    onClick={() => {
                      showConfirm(
                        "Delete Indexer",
                        `Are you sure you want to delete indexer "${indexer.name}"?`,
                        async () => {
                          try {
                            const response = await fetch(
                              `/api/plugins/usenet-indexer/indexers/${indexer.id}`,
                              {
                                method: "DELETE",
                                credentials: "include",
                              },
                            );
                            if (response.ok) {
                              await loadIndexers();
                            } else {
                              showAlert(
                                "Delete Failed",
                                "Failed to delete indexer",
                              );
                            }
                          } catch (error) {
                            console.error("Failed to delete indexer:", error);
                            showAlert(
                              "Delete Failed",
                              "Failed to delete indexer",
                            );
                          }
                          setConfirmModal({ ...confirmModal, isOpen: false });
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

        {/* Indexer Form */}
        {showIndexerForm && editingIndexer && (
          <div className="bg-background border rounded-lg p-4 space-y-4">
            <h4 className="text-lg font-semibold">
              {editingIndexer.id ? "Edit Indexer" : "New Indexer"}
            </h4>

            <div className="grid grid-cols-2 gap-4">
              <div className="space-y-2">
                <label className="block text-sm font-medium">Name</label>
                <input
                  type="text"
                  className="w-full px-3 py-2 bg-background border rounded-md"
                  placeholder="My Indexer"
                  value={editingIndexer.name}
                  onChange={(e) =>
                    setEditingIndexer({
                      ...editingIndexer,
                      name: e.target.value,
                    })
                  }
                />
              </div>

              <div className="space-y-2">
                <label className="block text-sm font-medium">API URL</label>
                <input
                  type="text"
                  className="w-full px-3 py-2 bg-background border rounded-md"
                  placeholder="https://indexer.example.com"
                  value={editingIndexer.url}
                  onChange={(e) =>
                    setEditingIndexer({
                      ...editingIndexer,
                      url: e.target.value,
                    })
                  }
                />
              </div>

              <div className="space-y-2 col-span-2">
                <label className="block text-sm font-medium">API Key</label>
                <input
                  type="password"
                  className="w-full px-3 py-2 bg-background border rounded-md"
                  placeholder="Your API key"
                  value={editingIndexer.api_key}
                  onChange={(e) =>
                    setEditingIndexer({
                      ...editingIndexer,
                      api_key: e.target.value,
                    })
                  }
                />
              </div>

              <div className="space-y-2">
                <label className="block text-sm font-medium">
                  TV Categories
                </label>
                <input
                  type="text"
                  className="w-full px-3 py-2 bg-background border rounded-md"
                  placeholder="5030,5040"
                  value={editingIndexer.tv_categories.join(",")}
                  onChange={(e) =>
                    setEditingIndexer({
                      ...editingIndexer,
                      tv_categories: e.target.value
                        .split(",")
                        .filter((c) => c.trim()),
                    })
                  }
                />
                <p className="text-xs text-muted-foreground">
                  Comma-separated Newznab category IDs
                </p>
              </div>

              <div className="space-y-2">
                <label className="block text-sm font-medium">
                  Movie Categories
                </label>
                <input
                  type="text"
                  className="w-full px-3 py-2 bg-background border rounded-md"
                  placeholder="2000,2040"
                  value={editingIndexer.movie_categories.join(",")}
                  onChange={(e) =>
                    setEditingIndexer({
                      ...editingIndexer,
                      movie_categories: e.target.value
                        .split(",")
                        .filter((c) => c.trim()),
                    })
                  }
                />
                <p className="text-xs text-muted-foreground">
                  Comma-separated Newznab category IDs
                </p>
              </div>

              <div className="space-y-2">
                <label className="block text-sm font-medium">Priority</label>
                <input
                  type="number"
                  className="w-full px-3 py-2 bg-background border rounded-md"
                  value={editingIndexer.priority}
                  onChange={(e) =>
                    setEditingIndexer({
                      ...editingIndexer,
                      priority: parseInt(e.target.value) || 0,
                    })
                  }
                />
                <p className="text-xs text-muted-foreground">
                  Higher priority indexers are searched first
                </p>
              </div>
            </div>

            <div className="flex items-center space-x-2">
              <input
                type="checkbox"
                id="indexer-enabled"
                checked={editingIndexer.enabled}
                onChange={(e) =>
                  setEditingIndexer({
                    ...editingIndexer,
                    enabled: e.target.checked,
                  })
                }
                className="rounded"
              />
              <label htmlFor="indexer-enabled" className="text-sm font-medium">
                Enabled
              </label>
            </div>

            <div className="flex space-x-3">
              <button
                className="px-4 py-2 bg-primary text-primary-foreground rounded-md hover:bg-primary/90"
                onClick={() => {
                  saveIndexer(editingIndexer);
                }}
              >
                Save
              </button>
              <button
                className="px-4 py-2 bg-secondary text-secondary-foreground rounded-md hover:bg-secondary/90"
                onClick={() => {
                  setShowIndexerForm(false);
                  setEditingIndexer(null);
                }}
              >
                Cancel
              </button>
            </div>
          </div>
        )}
      </div>

      {/* Search Section */}
      <div className="bg-card border rounded-lg p-6 space-y-4">
        <h2 className="text-xl font-semibold">Search Releases</h2>

        {/* Search Type */}
        <div className="space-y-2">
          <label className="block text-sm font-medium">Search Type</label>
          <select
            className="w-full px-3 py-2 bg-background border rounded-md"
            value={searchType}
            onChange={(e) =>
              setSearchType(e.target.value as "general" | "tv" | "movie")
            }
          >
            <option value="general">General Search</option>
            <option value="tv">TV Shows</option>
            <option value="movie">Movies</option>
          </select>
        </div>

        {/* Search Query */}
        <div className="space-y-2">
          <label className="block text-sm font-medium">Search Query</label>
          <div className="flex space-x-2">
            <input
              type="text"
              className="flex-1 px-3 py-2 bg-background border rounded-md"
              placeholder="Enter search query..."
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
              onKeyPress={(e) => e.key === "Enter" && performSearch()}
            />
            <button
              className="px-4 py-2 bg-primary text-primary-foreground rounded-md hover:bg-primary/90 disabled:opacity-50"
              onClick={performSearch}
              disabled={searching || !searchQuery.trim()}
            >
              {searching ? "Searching..." : "Search"}
            </button>
          </div>
        </div>

        {/* Search Results */}
        {searchResults && (
          <div className="space-y-2 pt-4">
            <h3 className="font-medium">
              Results ({searchResults.length} releases)
            </h3>
            <div className="space-y-2 max-h-96 overflow-y-auto">
              {searchResults.map((release) => (
                <div
                  key={release.id}
                  className="p-3 bg-muted rounded-md space-y-1"
                >
                  <div className="font-medium text-sm">{release.title}</div>
                  <div className="flex items-center space-x-4 text-xs text-muted-foreground">
                    <span>Size: {formatBytes(release.size)}</span>
                    <span>Category: {release.category}</span>
                    {release.publishDate && (
                      <span>
                        Published:{" "}
                        {new Date(release.publishDate).toLocaleDateString()}
                      </span>
                    )}
                  </div>
                  {release.downloadUrl && (
                    <a
                      href={release.downloadUrl}
                      className="text-xs text-blue-500 hover:underline"
                      target="_blank"
                      rel="noopener noreferrer"
                    >
                      Download
                    </a>
                  )}
                </div>
              ))}
            </div>
          </div>
        )}
      </div>

      {/* Common Newznab Categories Reference */}
      <div className="bg-card border rounded-lg p-6">
        <h2 className="text-xl font-semibold mb-4">
          Common Newznab Categories
        </h2>
        <div className="grid grid-cols-2 gap-4 text-sm">
          <div>
            <h3 className="font-medium mb-2">TV Shows</h3>
            <ul className="space-y-1 text-muted-foreground">
              <li>5000 - TV (All)</li>
              <li>5030 - TV HD</li>
              <li>5040 - TV SD</li>
              <li>5070 - TV Anime</li>
            </ul>
          </div>
          <div>
            <h3 className="font-medium mb-2">Movies</h3>
            <ul className="space-y-1 text-muted-foreground">
              <li>2000 - Movies (All)</li>
              <li>2010 - Movies Foreign</li>
              <li>2020 - Movies Other</li>
              <li>2030 - Movies SD</li>
              <li>2040 - Movies HD</li>
              <li>2045 - Movies UHD</li>
              <li>2050 - Movies BluRay</li>
              <li>2060 - Movies 3D</li>
            </ul>
          </div>
        </div>
      </div>

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
