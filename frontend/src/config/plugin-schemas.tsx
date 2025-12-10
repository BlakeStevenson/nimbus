import { ReactNode } from "react";

export interface PluginFieldSchema {
  key: string;
  label: string;
  type: "text" | "number" | "password" | "boolean" | "textarea";
  required?: boolean;
  placeholder?: string;
  description?: string;
  min?: number;
  max?: number;
  isArray?: boolean;
}

export interface PluginObjectListConfig {
  schema: PluginFieldSchema[];
  title: string;
  description?: string;
  itemName: string;
  defaultItem: any;
  renderBadges?: (item: any) => ReactNode;
  renderSummary?: (item: any) => string;
}

// NZB Downloader Server Schema
export const nzbDownloaderServerSchema: PluginObjectListConfig = {
  schema: [
    {
      key: "name",
      label: "Server Name",
      type: "text",
      required: true,
      placeholder: "My Usenet Server",
    },
    {
      key: "host",
      label: "Hostname",
      type: "text",
      required: true,
      placeholder: "news.example.com",
    },
    {
      key: "port",
      label: "Port",
      type: "number",
      required: true,
      min: 1,
      max: 65535,
    },
    {
      key: "connections",
      label: "Connections",
      type: "number",
      description: "Number of concurrent connections",
      min: 1,
      max: 50,
    },
    {
      key: "username",
      label: "Username",
      type: "text",
      placeholder: "username",
    },
    {
      key: "password",
      label: "Password",
      type: "password",
      placeholder: "password",
    },
    {
      key: "priority",
      label: "Priority",
      type: "number",
      description: "Lower numbers = higher priority",
    },
    {
      key: "use_ssl",
      label: "Use SSL/TLS",
      type: "boolean",
    },
    {
      key: "enabled",
      label: "Enabled",
      type: "boolean",
    },
  ],
  title: "NNTP Servers",
  description: "Configure your Usenet server connections",
  itemName: "Server",
  defaultItem: {
    name: "",
    host: "",
    port: 563,
    username: "",
    password: "",
    use_ssl: true,
    enabled: true,
    connections: 10,
    priority: 0,
  },
  renderBadges: (server) => (
    <>
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
    </>
  ),
  renderSummary: (server) =>
    `${server.host}:${server.port} • ${server.connections} connections • Priority: ${server.priority}`,
};

// Usenet Indexer Schema
export const usenetIndexerSchema: PluginObjectListConfig = {
  schema: [
    {
      key: "name",
      label: "Indexer Name",
      type: "text",
      required: true,
      placeholder: "NZBGeek",
    },
    {
      key: "url",
      label: "API URL",
      type: "text",
      required: true,
      placeholder: "https://api.nzbgeek.info",
    },
    {
      key: "api_key",
      label: "API Key",
      type: "password",
      required: true,
      placeholder: "your-api-key-here",
    },
    {
      key: "tv_categories",
      label: "TV Categories",
      type: "text",
      description: "Comma-separated category IDs (e.g., 5030,5040)",
      placeholder: "5030,5040",
      isArray: true,
    },
    {
      key: "movie_categories",
      label: "Movie Categories",
      type: "text",
      description: "Comma-separated category IDs (e.g., 2040,2045)",
      placeholder: "2040,2045",
      isArray: true,
    },
    {
      key: "priority",
      label: "Priority",
      type: "number",
      description: "Lower numbers = higher priority",
    },
    {
      key: "enabled",
      label: "Enabled",
      type: "boolean",
    },
  ],
  title: "Newznab Indexers",
  description:
    "Configure Newznab-compatible Usenet indexers for searching releases",
  itemName: "Indexer",
  defaultItem: {
    name: "",
    url: "",
    api_key: "",
    tv_categories: ["5030", "5040"],
    movie_categories: ["2040", "2045"],
    enabled: true,
    priority: 0,
  },
  renderBadges: (indexer) => (
    <>
      {!indexer.enabled && (
        <span className="text-xs px-2 py-0.5 bg-red-500/10 text-red-600 rounded">
          Disabled
        </span>
      )}
    </>
  ),
  renderSummary: (indexer) => {
    const tvCats =
      typeof indexer.tv_categories === "string"
        ? indexer.tv_categories
        : (indexer.tv_categories || []).join(", ");
    const movieCats =
      typeof indexer.movie_categories === "string"
        ? indexer.movie_categories
        : (indexer.movie_categories || []).join(", ");
    return `${indexer.url} • TV: ${tvCats} • Movies: ${movieCats}`;
  },
};

// Schema registry - maps plugin ID + field key to schema config
export const pluginSchemas: Record<string, PluginObjectListConfig> = {
  "nzb-downloader:plugins.nzb-downloader.servers": nzbDownloaderServerSchema,
  "usenet-indexer:plugins.usenet-indexer.indexers": usenetIndexerSchema,
};
