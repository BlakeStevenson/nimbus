import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { apiGet, apiPut } from '../api-client';
import type { ConfigValue } from '../types';

export function useConfigValue(key: string) {
  return useQuery<ConfigValue>({
    queryKey: ['config', key],
    queryFn: () => apiGet<ConfigValue>(`/api/config/${key}`),
    enabled: !!key,
  });
}

export function useUpdateConfig(key: string) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (value: any) =>
      apiPut<ConfigValue>(`/api/config/${key}`, { value }),
    onSuccess: (data) => {
      queryClient.setQueryData(['config', key], data);
      queryClient.invalidateQueries({ queryKey: ['config'] });
    },
  });
}
