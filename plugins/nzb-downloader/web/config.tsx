import { useState, useEffect } from "react";

interface NNTPServer {
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

interface NZBDownloaderConfigProps {
  value: any;
  onChange: (value: any) => void;
}

export default function NZBDownloaderConfig({ value, onChange }: NZBDownloaderConfigProps) {
  const [servers, setServers] = useState<NNTPServer[]>([]);
  const [editingServer, setEditingServer] = useState<NNTPServer | null>(null);
  const [showForm, setShowForm] = useState(false);

  useEffect(() => {
    if (value) {
      try {
        const parsed = typeof value === "string" ? JSON.parse(value) : value;
        setServers(Array.isArray(parsed) ? parsed : []);
      } catch {
        setServers([]);
      }
    }
  }, [value]);

  const handleAddServer = () => {
    setEditingServer({
      id: crypto.randomUUID(),
      name: "",
      host: "",
      port: 563,
      username: "",
      password: "",
      use_ssl: true,
      enabled: true,
      connections: 10,
      priority: 0,
    });
    setShowForm(true);
  };

  const handleEditServer = (server: NNTPServer) => {
    setEditingServer(server);
    setShowForm(true);
  };

  const handleSaveServer = () => {
    if (!editingServer) return;

    const updatedServers = servers.some((s) => s.id === editingServer.id)
      ? servers.map((s) => (s.id === editingServer.id ? editingServer : s))
      : [...servers, editingServer];

    setServers(updatedServers);
    onChange(updatedServers);
    setShowForm(false);
    setEditingServer(null);
  };

  const handleDeleteServer = (id: string) => {
    const updatedServers = servers.filter((s) => s.id !== id);
    setServers(updatedServers);
    onChange(updatedServers);
  };

  const handleCancel = () => {
    setShowForm(false);
    setEditingServer(null);
  };

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h3 className="text-lg font-semibold">NNTP Servers</h3>
          <p className="text-sm text-muted-foreground">
            Configure your Usenet server connections
          </p>
        </div>
        {!showForm && (
          <button
            onClick={handleAddServer}
            className="px-4 py-2 bg-primary text-primary-foreground rounded-md hover:bg-primary/90 text-sm"
          >
            + Add Server
          </button>
        )}
      </div>

      {/* Server Form */}
      {showForm && editingServer && (
        <div className="bg-card border border-primary rounded-lg p-6 space-y-4">
          <h4 className="text-lg font-semibold">
            {servers.some((s) => s.id === editingServer.id)
              ? "Edit Server"
              : "New Server"}
          </h4>

          <div className="grid grid-cols-2 gap-4">
            <div className="space-y-2">
              <label className="block text-sm font-medium">Server Name *</label>
              <input
                type="text"
                value={editingServer.name}
                onChange={(e) =>
                  setEditingServer({ ...editingServer, name: e.target.value })
                }
                placeholder="My Usenet Server"
                className="w-full px-3 py-2 bg-background border rounded-md"
              />
            </div>

            <div className="space-y-2">
              <label className="block text-sm font-medium">Hostname *</label>
              <input
                type="text"
                value={editingServer.host}
                onChange={(e) =>
                  setEditingServer({ ...editingServer, host: e.target.value })
                }
                placeholder="news.example.com"
                className="w-full px-3 py-2 bg-background border rounded-md"
              />
            </div>

            <div className="space-y-2">
              <label className="block text-sm font-medium">Port *</label>
              <input
                type="number"
                value={editingServer.port}
                onChange={(e) =>
                  setEditingServer({
                    ...editingServer,
                    port: parseInt(e.target.value) || 563,
                  })
                }
                className="w-full px-3 py-2 bg-background border rounded-md"
              />
            </div>

            <div className="space-y-2">
              <label className="block text-sm font-medium">Connections</label>
              <input
                type="number"
                value={editingServer.connections}
                onChange={(e) =>
                  setEditingServer({
                    ...editingServer,
                    connections: parseInt(e.target.value) || 10,
                  })
                }
                min={1}
                max={50}
                className="w-full px-3 py-2 bg-background border rounded-md"
              />
            </div>

            <div className="space-y-2">
              <label className="block text-sm font-medium">Username</label>
              <input
                type="text"
                value={editingServer.username}
                onChange={(e) =>
                  setEditingServer({
                    ...editingServer,
                    username: e.target.value,
                  })
                }
                placeholder="username"
                className="w-full px-3 py-2 bg-background border rounded-md"
              />
            </div>

            <div className="space-y-2">
              <label className="block text-sm font-medium">Password</label>
              <input
                type="password"
                value={editingServer.password}
                onChange={(e) =>
                  setEditingServer({
                    ...editingServer,
                    password: e.target.value,
                  })
                }
                placeholder="password"
                className="w-full px-3 py-2 bg-background border rounded-md"
              />
            </div>

            <div className="space-y-2">
              <label className="block text-sm font-medium">Priority</label>
              <input
                type="number"
                value={editingServer.priority}
                onChange={(e) =>
                  setEditingServer({
                    ...editingServer,
                    priority: parseInt(e.target.value) || 0,
                  })
                }
                className="w-full px-3 py-2 bg-background border rounded-md"
              />
              <p className="text-xs text-muted-foreground">
                Lower numbers = higher priority
              </p>
            </div>
          </div>

          <div className="flex items-center space-x-6">
            <div className="flex items-center space-x-2">
              <input
                type="checkbox"
                id="use_ssl"
                checked={editingServer.use_ssl}
                onChange={(e) =>
                  setEditingServer({ ...editingServer, use_ssl: e.target.checked })
                }
                className="rounded"
              />
              <label htmlFor="use_ssl" className="text-sm cursor-pointer">
                Use SSL/TLS
              </label>
            </div>

            <div className="flex items-center space-x-2">
              <input
                type="checkbox"
                id="enabled"
                checked={editingServer.enabled}
                onChange={(e) =>
                  setEditingServer({ ...editingServer, enabled: e.target.checked })
                }
                className="rounded"
              />
              <label htmlFor="enabled" className="text-sm cursor-pointer">
                Enabled
              </label>
            </div>
          </div>

          <div className="flex space-x-2 pt-4">
            <button
              onClick={handleSaveServer}
              className="px-4 py-2 bg-primary text-primary-foreground rounded-md hover:bg-primary/90 text-sm"
            >
              Save Server
            </button>
            <button
              onClick={handleCancel}
              className="px-4 py-2 bg-secondary text-secondary-foreground rounded-md hover:bg-secondary/90 text-sm"
            >
              Cancel
            </button>
          </div>
        </div>
      )}

      {/* Server List */}
      {!showForm && (
        <div className="space-y-2">
          {servers.length === 0 ? (
            <div className="bg-card border rounded-lg p-6">
              <p className="text-sm text-muted-foreground text-center">
                No NNTP servers configured. Click "Add Server" to get started.
              </p>
            </div>
          ) : (
            servers.map((server) => (
              <div key={server.id} className="bg-card border rounded-lg p-4">
                <div className="flex items-center justify-between">
                  <div className="flex-1">
                    <div className="flex items-center space-x-2 mb-1">
                      <h4 className="font-medium">{server.name}</h4>
                      {!server.enabled && (
                        <span className="text-xs px-2 py-0.5 bg-red-500/10 text-red-600 rounded">
                          Disabled
                        </span>
                      )}
                      {server.use_ssl && (
                        <span className="text-xs px-2 py-0.5 bg-green-500/10 text-green-600 rounded">
                          SSL
                        </span>
                      )}
                    </div>
                    <p className="text-sm text-muted-foreground">
                      {server.host}:{server.port} • {server.connections}{" "}
                      connections • Priority: {server.priority}
                    </p>
                  </div>
                  <div className="flex space-x-2">
                    <button
                      onClick={() => handleEditServer(server)}
                      className="px-3 py-1 text-sm bg-secondary text-secondary-foreground rounded-md hover:bg-secondary/90"
                    >
                      Edit
                    </button>
                    <button
                      onClick={() => handleDeleteServer(server.id)}
                      className="px-3 py-1 text-sm bg-red-500 text-white rounded-md hover:bg-red-600"
                    >
                      Delete
                    </button>
                  </div>
                </div>
              </div>
            ))
          )}
        </div>
      )}
    </div>
  );
}
