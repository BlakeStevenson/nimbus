import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { apiGet, apiPut, apiDelete } from "../api-client";
import type {
  MediaItem,
  MediaListResponse,
  MediaFilters,
  UpdateMediaPayload,
} from "../types";

export function useMediaList(filters: MediaFilters = {}) {
  const params = new URLSearchParams();

  if (filters.kind) params.append("kind", filters.kind);
  if (filters.q) params.append("q", filters.q);
  if (filters.parentId) params.append("parent_id", String(filters.parentId));
  if (filters.limit) params.append("limit", String(filters.limit));
  if (filters.offset) params.append("offset", String(filters.offset));

  return useQuery<MediaListResponse>({
    queryKey: ["media", filters],
    queryFn: () => apiGet<MediaListResponse>("/api/media", params),
  });
}

export function useMediaItem(id: string | number) {
  return useQuery<MediaItem>({
    queryKey: ["media", id],
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
      queryClient.invalidateQueries({ queryKey: ["media"] });
      queryClient.setQueryData(["media", id], data);
    },
  });
}

export function useMediaStats() {
  const kinds = [
    "movie",
    "tv_series",
    "tv_episode",
    "music_album",
    "music_track",
    "book",
  ];

  const queries = kinds.map((kind) => ({
    queryKey: ["media", "stats", kind],
    queryFn: async () => {
      const params = new URLSearchParams();
      params.append("kind", kind);
      params.append("limit", "0");
      const data = await apiGet<MediaListResponse>("/api/media", params);
      return { kind, count: data.total };
    },
  }));

  return useQuery({
    queryKey: ["media", "stats"],
    queryFn: async () => {
      const results = await Promise.all(queries.map((q) => q.queryFn()));
      return results;
    },
  });
}

// =============================================================================
// Media Files API
// =============================================================================

export interface MediaFile {
  id: number;
  path: string;
  size: number | null;
  hash: string | null;
  created_at: string;
  updated_at: string;
}

/**
 * Fetch all files associated with a media item
 */
export function useMediaFiles(mediaId: string | number) {
  return useQuery<MediaFile[]>({
    queryKey: ["media", mediaId, "files"],
    queryFn: () => apiGet<MediaFile[]>(`/api/media/${mediaId}/files`),
    enabled: !!mediaId,
  });
}

/**
 * Delete a single media file
 */
export function useDeleteMediaFile() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({
      fileId,
      deletePhysical,
    }: {
      fileId: number;
      deletePhysical: boolean;
    }) => {
      const params = new URLSearchParams();
      if (deletePhysical) {
        params.append("delete_physical", "true");
      }
      return apiDelete(`/api/media/files/${fileId}?${params.toString()}`);
    },
    onSuccess: () => {
      // Invalidate file queries for all media items
      queryClient.invalidateQueries({ queryKey: ["media"] });
    },
  });
}

/**
 * Delete a media item with optional file deletion
 */
export function useDeleteMediaItem() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({
      mediaId,
      deleteFiles,
    }: {
      mediaId: number;
      deleteFiles: boolean;
    }) => {
      const params = new URLSearchParams();
      if (deleteFiles) {
        params.append("delete_files", "true");
      }
      return apiDelete(`/api/media/${mediaId}/with-files?${params.toString()}`);
    },
    onSuccess: () => {
      // Invalidate all media queries
      queryClient.invalidateQueries({ queryKey: ["media"] });
    },
  });
}
