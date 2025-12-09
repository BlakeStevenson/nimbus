import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { apiGet, apiPut } from "../api-client";
import type { ConfigValue } from "../types";

export function useConfigValue(key: string) {
  return useQuery<ConfigValue>({
    queryKey: ["config", key],
    queryFn: () => apiGet<ConfigValue>(`/api/config/${key}`),
    enabled: !!key,
  });
}

export function useAllConfig() {
  return useQuery<ConfigValue[]>({
    queryKey: ["config"],
    queryFn: () => apiGet<ConfigValue[]>(`/api/config`),
  });
}

export function useUpdateConfig(key: string) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (value: any) =>
      apiPut<ConfigValue>(`/api/config/${key}`, { value }),
    onSuccess: (data) => {
      queryClient.setQueryData(["config", key], data);
      queryClient.invalidateQueries({ queryKey: ["config"] });
    },
  });
}

export function useUpdateMultipleConfigs() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (configs: Record<string, any>) => {
      const promises = Object.entries(configs).map(([key, value]) =>
        apiPut<ConfigValue>(`/api/config/${key}`, { value }),
      );
      return Promise.all(promises);
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["config"] });
    },
  });
}
