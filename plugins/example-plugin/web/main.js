// Example Plugin UI - Browser-compatible version
// This uses global React from the parent Nimbus application

(function () {
  // React is available globally from the parent app
  const React = window.React;
  const { useState, useEffect, createElement: h } = React;

  function ExamplePluginPage() {
    const [helloMessage, setHelloMessage] = useState(null);
    const [statusMessage, setStatusMessage] = useState(null);
    const [loading, setLoading] = useState(false);

    const fetchHello = async () => {
      setLoading(true);
      try {
        const response = await fetch("/api/plugins/example/hello");
        const data = await response.json();
        setHelloMessage(data);
      } catch (error) {
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
      } catch (error) {
        console.error("Failed to fetch status:", error);
        setStatusMessage({ error: error.message });
      } finally {
        setLoading(false);
      }
    };

    return h(
      "div",
      { className: "space-y-6 p-6" },

      // Header
      h(
        "div",
        null,
        h("h1", { className: "text-3xl font-bold" }, "Example Plugin"),
        h(
          "p",
          { className: "text-muted-foreground" },
          "This is a demonstration of the Nimbus plugin system",
        ),
      ),

      // Info Card
      h(
        "div",
        { className: "bg-card border rounded-lg p-6 space-y-4" },
        h("h2", { className: "text-xl font-semibold" }, "Plugin Information"),
        h(
          "p",
          { className: "text-sm text-muted-foreground" },
          "ID: example-plugin",
        ),
        h(
          "p",
          { className: "text-sm text-muted-foreground" },
          "Version: 0.1.0",
        ),
        h(
          "p",
          { className: "text-sm text-muted-foreground" },
          "Capabilities: API, UI",
        ),
      ),

      // API Testing Section
      h(
        "div",
        { className: "bg-card border rounded-lg p-6 space-y-4" },
        h("h2", { className: "text-xl font-semibold mb-4" }, "API Endpoints"),

        // Hello Endpoint
        h(
          "div",
          { className: "space-y-2" },
          h("h3", { className: "font-medium" }, "Public Hello Endpoint"),
          h(
            "button",
            {
              className:
                "px-4 py-2 bg-primary text-primary-foreground rounded-md hover:bg-primary/90 disabled:opacity-50",
              onClick: fetchHello,
              disabled: loading,
            },
            loading ? "Loading..." : "Call /api/plugins/example/hello",
          ),
          helloMessage &&
            h(
              "pre",
              { className: "mt-2 p-3 bg-muted rounded text-xs overflow-auto" },
              JSON.stringify(helloMessage, null, 2),
            ),
        ),

        // Status Endpoint
        h(
          "div",
          { className: "space-y-2 pt-4" },
          h(
            "h3",
            { className: "font-medium" },
            "Authenticated Status Endpoint",
          ),
          h(
            "button",
            {
              className:
                "px-4 py-2 bg-secondary text-secondary-foreground rounded-md hover:bg-secondary/90 disabled:opacity-50",
              onClick: fetchStatus,
              disabled: loading,
            },
            loading ? "Loading..." : "Call /api/plugins/example/status",
          ),
          statusMessage &&
            h(
              "pre",
              { className: "mt-2 p-3 bg-muted rounded text-xs overflow-auto" },
              JSON.stringify(statusMessage, null, 2),
            ),
        ),
      ),

      // Features Section
      h(
        "div",
        { className: "bg-card border rounded-lg p-6" },
        h("h2", { className: "text-xl font-semibold mb-4" }, "Plugin Features"),
        h(
          "ul",
          {
            className:
              "list-disc list-inside space-y-2 text-sm text-muted-foreground",
          },
          h("li", null, "Custom API endpoints (public and authenticated)"),
          h("li", null, "UI integration with dynamic routing"),
          h("li", null, "Sidebar navigation item"),
          h("li", null, "Access to Nimbus core services via SDK"),
          h("li", null, "Hot-reload support (enable/disable without restart)"),
        ),
      ),
    );
  }

  // Export as default for dynamic import
  if (typeof module !== "undefined" && module.exports) {
    module.exports = { default: ExamplePluginPage };
  } else {
    window.NimbusPluginExamplePlugin = ExamplePluginPage;
  }
})();
