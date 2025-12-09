import { Routes, Route } from "react-router-dom";
import { usePluginUIManifests } from "@/lib/api/plugins";
import { PluginPageLoader } from "./PluginPageLoader";

/**
 * PluginRoutes dynamically renders routes for all enabled plugins.
 *
 * This component fetches plugin UI manifests and creates a Route for each
 * plugin-defined route. The routes are rendered using PluginPageLoader which
 * handles lazy-loading the plugin's JavaScript bundle.
 */
export const PluginRoutes: React.FC = () => {
  const { data: manifests, isLoading } = usePluginUIManifests();

  if (isLoading || !manifests) {
    return null;
  }

  return (
    <Routes>
      {manifests.flatMap((manifest) =>
        manifest.routes.map((route) => (
          <Route
            key={`${manifest.id}-${route.path}`}
            path={route.path}
            element={<PluginPageLoader bundleUrl={route.bundleUrl} />}
          />
        ))
      )}
    </Routes>
  );
};
