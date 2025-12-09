import { useState } from "react";

export default function ExamplePluginPage() {
  const [helloMessage, setHelloMessage] = useState<any>(null);
  const [statusMessage, setStatusMessage] = useState<any>(null);
  const [loading, setLoading] = useState(false);

  const fetchHello = async () => {
    setLoading(true);
    try {
      const response = await fetch("/api/plugins/example/hello");
      const data = await response.json();
      setHelloMessage(data);
    } catch (error: any) {
      console.error("Failed to fetch hello:", error);
      setHelloMessage({ error: error.message });
    } finally {
      setLoading(false);
    }
  };

  const fetchStatus = async () => {
    setLoading(true);
    try {
      const response = await fetch("/api/plugins/example/status", {
        credentials: "include",
      });
      const data = await response.json();
      setStatusMessage(data);
    } catch (error: any) {
      console.error("Failed to fetch status:", error);
      setStatusMessage({ error: error.message });
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="space-y-6 p-6">
      {/* Header */}
      <div>
        <h1 className="text-3xl font-bold">Example Plugin</h1>
        <p className="text-muted-foreground">
          This is a demonstration of the Nimbus plugin system
        </p>
      </div>

      {/* Info Card */}
      <div className="bg-card border rounded-lg p-6 space-y-4">
        <h2 className="text-xl font-semibold">Plugin Information</h2>
        <p className="text-sm text-muted-foreground">ID: example-plugin</p>
        <p className="text-sm text-muted-foreground">Version: 0.1.0</p>
        <p className="text-sm text-muted-foreground">Capabilities: API, UI</p>
      </div>

      {/* API Testing Section */}
      <div className="bg-card border rounded-lg p-6 space-y-4">
        <h2 className="text-xl font-semibold mb-4">API Endpoints</h2>

        {/* Hello Endpoint */}
        <div className="space-y-2">
          <h3 className="font-medium">Public Hello Endpoint</h3>
          <button
            className="px-4 py-2 bg-primary text-primary-foreground rounded-md hover:bg-primary/90 disabled:opacity-50"
            onClick={fetchHello}
            disabled={loading}
          >
            {loading ? "Loading..." : "Call /api/plugins/example/hello"}
          </button>
          {helloMessage && (
            <pre className="mt-2 p-3 bg-muted rounded text-xs overflow-auto">
              {JSON.stringify(helloMessage, null, 2)}
            </pre>
          )}
        </div>

        {/* Status Endpoint */}
        <div className="space-y-2 pt-4">
          <h3 className="font-medium">Authenticated Status Endpoint</h3>
          <button
            className="px-4 py-2 bg-secondary text-secondary-foreground rounded-md hover:bg-secondary/90 disabled:opacity-50"
            onClick={fetchStatus}
            disabled={loading}
          >
            {loading ? "Loading..." : "Call /api/plugins/example/status"}
          </button>
          {statusMessage && (
            <pre className="mt-2 p-3 bg-muted rounded text-xs overflow-auto">
              {JSON.stringify(statusMessage, null, 2)}
            </pre>
          )}
        </div>
      </div>

      {/* Features Section */}
      <div className="bg-card border rounded-lg p-6">
        <h2 className="text-xl font-semibold mb-4">Plugin Features</h2>
        <ul className="list-disc list-inside space-y-2 text-sm text-muted-foreground">
          <li>Custom API endpoints (public and authenticated)</li>
          <li>UI integration with dynamic routing</li>
          <li>Sidebar navigation item</li>
          <li>Access to Nimbus core services via SDK</li>
          <li>Hot-reload support (enable/disable without restart)</li>
        </ul>
      </div>
    </div>
  );
}
