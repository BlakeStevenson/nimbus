import { useQuery } from '@tanstack/react-query';
import type { Plugin, PluginUiManifest } from '../types';

// Placeholder hook for future plugin API implementation
export function usePlugins() {
  return useQuery<Plugin[]>({
    queryKey: ['plugins'],
    queryFn: async () => {
      // TODO: Replace with actual API call when backend is ready
      // return apiGet<Plugin[]>('/api/plugins');

      // Mock data for now
      return [
        {
          id: 'example-plugin',
          name: 'Example Plugin',
          version: '1.0.0',
          enabled: true,
          capabilities: ['metadata', 'import'],
          description: 'An example plugin for demonstration',
        },
      ];
    },
    enabled: false, // Disable until backend is ready
  });
}

// Placeholder hook for fetching plugin UI manifests
export function usePluginUiManifests() {
  return useQuery<PluginUiManifest[]>({
    queryKey: ['plugin-ui-manifests'],
    queryFn: async () => {
      // TODO: Fetch from /api/plugins and then /api/plugins/:id/ui-manifest for each
      return [];
    },
    enabled: false, // Disable until backend is ready
  });
}

// Helper hook to get navigation items from plugins
export function usePluginNavItems() {
  const { data: manifests = [] } = usePluginUiManifests();

  return manifests.flatMap(manifest =>
    manifest.navItems.map(item => ({
      ...item,
      pluginId: manifest.id,
    }))
  );
}
