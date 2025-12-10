import React, { Suspense, lazy } from "react";

interface PluginPageLoaderProps {
  bundleUrl: string;
}

/**
 * PluginPageLoader dynamically loads a plugin's React component from a remote bundle.
 *
 * This component uses React.lazy() with dynamic imports to load plugin code at runtime.
 * The bundleUrl should point to a JavaScript module that exports a default React component.
 *
 * Example bundle structure:
 * ```js
 * // Plugin bundle (e.g., /plugins/sonarr-compat/main.js)
 * export default function SonarrPlugin() {
 *   return <div>Sonarr Plugin UI</div>;
 * }
 * ```
 */
export const PluginPageLoader: React.FC<PluginPageLoaderProps> = ({
  bundleUrl,
}) => {
  // For development, use a workaround to load plugin components
  // In production, this would load pre-built bundles
  const Component = React.useMemo(() => {
    // Map known plugins to their modules
    const pluginModules: Record<string, () => Promise<any>> = {
      "/src/plugins-example-plugin.tsx": () =>
        import("@/plugins-example-plugin"),
    };

    const loader = pluginModules[bundleUrl];

    if (!loader) {
      // Fallback for unknown plugins
      return () => (
        <div className="p-6">
          <div className="bg-red-50 border border-red-200 rounded-md p-4">
            <h3 className="text-red-800 font-semibold mb-2">
              Failed to Load Plugin
            </h3>
            <p className="text-red-600 text-sm">
              Could not load plugin bundle from: <code>{bundleUrl}</code>
            </p>
            <p className="text-red-600 text-sm mt-2">
              Plugin module not found. Available plugins need to be registered
              in PluginPageLoader.
            </p>
          </div>
        </div>
      );
    }

    return lazy(loader);
  }, [bundleUrl]);

  return (
    <Suspense
      fallback={
        <div className="flex items-center justify-center h-64">
          <div className="text-center">
            <div className="inline-block animate-spin rounded-full h-8 w-8 border-b-2 border-gray-900 mb-4"></div>
            <p className="text-gray-600">Loading plugin...</p>
          </div>
        </div>
      }
    >
      <Component />
    </Suspense>
  );
};
