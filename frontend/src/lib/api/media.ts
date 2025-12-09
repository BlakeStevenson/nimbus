import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { apiGet, apiPut } from '../api-client';
import type { MediaItem, MediaListResponse, MediaFilters, UpdateMediaPayload } from '../types';

export function useMediaList(filters: MediaFilters = {}) {
  const params = new URLSearchParams();

  if (filters.kind) params.append('kind', filters.kind);
  if (filters.q) params.append('q', filters.q);
  if (filters.parentId) params.append('parent_id', String(filters.parentId));
  if (filters.limit) params.append('limit', String(filters.limit));
  if (filters.offset) params.append('offset', String(filters.offset));

  return useQuery<MediaListResponse>({
    queryKey: ['media', filters],
    queryFn: () => apiGet<MediaListResponse>('/api/media', params),
  });
}

export function useMediaItem(id: string | number) {
  return useQuery<MediaItem>({
    queryKey: ['media', id],
    queryFn: () => apiGet<MediaItem>(`/api/media/${id}`),
    enabled: !!id,
  });
}

export function useUpdateMedia(id: string | number) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (data: UpdateMediaPayload) =>
      apiPut<MediaItem>(`/api/media/${id}`, data),
    onSuccess: (data) => {
      queryClient.invalidateQueries({ queryKey: ['media'] });
      queryClient.setQueryData(['media', id], data);
    },
  });
}

export function useMediaStats() {
  const kinds = ['movie', 'tv_series', 'tv_episode', 'music_album', 'music_track', 'book'];

  const queries = kinds.map(kind => ({
    queryKey: ['media', 'stats', kind],
    queryFn: async () => {
      const params = new URLSearchParams();
      params.append('kind', kind);
      params.append('limit', '0');
      const data = await apiGet<MediaListResponse>('/api/media', params);
      return { kind, count: data.total };
    },
  }));

  return useQuery({
    queryKey: ['media', 'stats'],
    queryFn: async () => {
      const results = await Promise.all(
        queries.map(q => q.queryFn())
      );
      return results;
    },
  });
}
