import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { apiGet, apiPost } from "../api-client";

// ============================================================================
// Types
// ============================================================================

export interface PluginMeta {
  id: string;
  name: string;
  description: string;
  version: string;
  enabled: boolean;
  capabilities: string[];
  created_at?: string;
  updated_at?: string;
}

export interface PluginUINavItem {
  label: string;
  path: string;
  group?: string;
  icon?: string;
}

export interface PluginUIRoute {
  path: string;
  bundleUrl: string;
}

export interface PluginUIManifest {
  id: string;
  displayName: string;
  navItems: PluginUINavItem[];
  routes: PluginUIRoute[];
  configSection?: PluginConfigSection;
}

export interface PluginConfigSection {
  title: string;
  description?: string;
  fields: PluginConfigField[];
}

export interface PluginConfigField {
  key: string;
  label: string;
  description?: string;
  type: string; // text, number, boolean, select, textarea, password, array
  options?: string[];
  defaultValue?: string;
  required: boolean;
  placeholder?: string;
  validation?: PluginConfigFieldValidation;
}

export interface PluginConfigFieldValidation {
  min?: number;
  max?: number;
  pattern?: string;
  errorMessage?: string;
}

// ============================================================================
// API Functions
// ============================================================================

export async function getPlugins(): Promise<PluginMeta[]> {
  return apiGet<PluginMeta[]>("/api/plugins");
}

export async function getPluginUIManifest(
  id: string,
): Promise<PluginUIManifest> {
  return apiGet<PluginUIManifest>(`/api/plugins/${id}/ui-manifest`);
}

export async function enablePlugin(
  id: string,
): Promise<{ message: string; id: string }> {
  return apiPost(`/api/plugins/${id}/enable`, {});
}

export async function disablePlugin(
  id: string,
): Promise<{ message: string; id: string }> {
  return apiPost(`/api/plugins/${id}/disable`, {});
}

// ============================================================================
// React Query Hooks
// ============================================================================

export function usePlugins() {
  return useQuery({
    queryKey: ["plugins"],
    queryFn: getPlugins,
  });
}

export function usePluginUIManifests() {
  const { data: plugins } = usePlugins();

  return useQuery({
    queryKey: ["plugin-ui-manifests", plugins?.map((p) => p.id).join(",")],
    queryFn: async () => {
      if (!plugins) return [];

      // Fetch manifests for ALL plugins to get config sections
      const manifests = await Promise.all(
        plugins.map((p) => getPluginUIManifest(p.id).catch(() => null)),
      );

      return manifests.filter((m): m is PluginUIManifest => m !== null);
    },
    enabled: !!plugins && plugins.length > 0,
  });
}

export function useEnablePlugin() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: enablePlugin,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["plugins"] });
      queryClient.invalidateQueries({ queryKey: ["plugin-ui-manifests"] });
    },
  });
}

export function useDisablePlugin() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: disablePlugin,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["plugins"] });
      queryClient.invalidateQueries({ queryKey: ["plugin-ui-manifests"] });
    },
  });
}

/**
 * Hook to get all plugin navigation items for the sidebar
 */
export function usePluginNavItems(): PluginUINavItem[] {
  const { data: manifests } = usePluginUIManifests();

  if (!manifests) return [];

  // Flatten all nav items from all manifests
  return manifests.flatMap((manifest) => manifest.navItems);
}

/**
 * Hook to get plugins with their configuration sections
 */
export function usePluginsWithConfig() {
  const { data: plugins, isLoading: pluginsLoading } = usePlugins();
  const { data: manifests, isLoading: manifestsLoading } =
    usePluginUIManifests();

  return {
    data: plugins?.map((plugin) => {
      const manifest = manifests?.find((m) => m.id === plugin.id);
      return {
        ...plugin,
        configSection: manifest?.configSection,
      };
    }),
    isLoading: pluginsLoading || manifestsLoading,
  };
}
