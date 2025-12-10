import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { apiGet, apiPut, apiDelete, apiPost } from "../api-client";
import type {
  MediaItem,
  MediaListResponse,
  MediaFilters,
  UpdateMediaPayload,
} from "../types";

export interface IndexerRelease {
  guid: string;
  title: string;
  link?: string;
  comments?: string;
  publish_date: string;
  category?: string;
  size: number;
  download_url: string;
  description?: string;
  attributes?: Record<string, string>;
  indexer_id: string;
  indexer_name: string;
}

export interface InteractiveSearchResponse {
  media_id: number;
  releases: IndexerRelease[];
  total: number;
  sources: string[];
  metadata: {
    kind: string;
    title: string;
    year?: number;
  };
}

export function useMediaList(filters: MediaFilters = {}) {
  const params = new URLSearchParams();

  if (filters.kind) params.append("kind", filters.kind);
  if (filters.q) params.append("q", filters.q);
  if (filters.parentId !== undefined)
    params.append("parent_id", String(filters.parentId));
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

/**
 * Check which TMDB IDs are already in the library
 * Returns a map of tmdb_id -> media_item_id
 */
export async function checkTMDBInLibrary(
  tmdbIds: number[],
): Promise<Record<number, number>> {
  if (tmdbIds.length === 0) return {};

  // Fetch all media items (we'll filter client-side since we need to check metadata)
  const params = new URLSearchParams();
  params.append("limit", "1000"); // Get a large batch
  const data = await apiGet<MediaListResponse>("/api/media", params);

  const result: Record<number, number> = {};

  for (const item of data.items) {
    if (item.metadata && typeof item.metadata === "object") {
      const tmdbId = (item.metadata as any).tmdb_id;
      if (tmdbId) {
        const numericId: number =
          typeof tmdbId === "string" ? parseInt(tmdbId, 10) : Number(tmdbId);
        const mediaItemId: number =
          typeof item.id === "string" ? parseInt(item.id, 10) : item.id;
        if (!isNaN(numericId) && tmdbIds.includes(numericId)) {
          result[numericId] = mediaItemId;
        }
      }
    }
  }

  return result;
}

/**
 * Hook to check which TMDB IDs are in the library
 */
export function useTMDBInLibrary(tmdbIds: number[]) {
  return useQuery({
    queryKey: ["media", "tmdb-check", tmdbIds.sort().join(",")],
    queryFn: () => checkTMDBInLibrary(tmdbIds),
    enabled: tmdbIds.length > 0,
    staleTime: 60 * 1000, // 1 minute
  });
}

// =============================================================================
// Media Files API
// =============================================================================

export interface MediaFile {
  id: number;
  media_item_id: number | null;
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

// =============================================================================
// Interactive Search API
// =============================================================================

/**
 * Perform an interactive search for a specific media item
 */
export function useInteractiveSearch(mediaId: string | number) {
  return useQuery<InteractiveSearchResponse>({
    queryKey: ["media", mediaId, "search"],
    queryFn: () =>
      apiGet<InteractiveSearchResponse>(`/api/media/${mediaId}/search`),
    enabled: false, // Manual trigger only
    staleTime: 5 * 60 * 1000, // 5 minutes
  });
}
