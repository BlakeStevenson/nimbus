import { useLocation } from "react-router-dom";
import { usePluginUIManifests } from "@/lib/api/plugins";
import { PluginPageLoader } from "./PluginPageLoader";

/**
 * DynamicPluginRoute component that renders plugin pages based on the current path.
 * This is used as a catch-all route element that checks if the current path
 * matches any plugin route.
 */
export const DynamicPluginRoute: React.FC = () => {
  const location = useLocation();
  const { data: manifests, isLoading } = usePluginUIManifests();

  if (isLoading) {
    return (
      <div className="flex items-center justify-center h-64">
        <div className="text-center">
          <div className="inline-block animate-spin rounded-full h-8 w-8 border-b-2 border-gray-900 mb-4"></div>
          <p className="text-gray-600">Loading plugins...</p>
        </div>
      </div>
    );
  }

  if (!manifests || manifests.length === 0) {
    return (
      <div className="p-6">
        <div className="bg-gray-50 border border-gray-200 rounded-md p-4">
          <p className="text-gray-600">No plugin found for this route.</p>
        </div>
      </div>
    );
  }

  // Find the matching plugin route based on current path
  const currentPath = location.pathname;

  for (const manifest of manifests) {
    for (const route of manifest.routes) {
      // Simple path matching (you might want to use a more sophisticated matcher)
      if (currentPath.startsWith(route.path) || currentPath === route.path) {
        return (
          <PluginPageLoader key={currentPath} bundleUrl={route.bundleUrl} />
        );
      }
    }
  }

  return (
    <div className="p-6">
      <div className="bg-gray-50 border border-gray-200 rounded-md p-4">
        <p className="text-gray-600">No plugin found for this route.</p>
      </div>
    </div>
  );
};
