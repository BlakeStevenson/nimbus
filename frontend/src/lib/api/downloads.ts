import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

export interface Download {
  id: string;
  plugin_id: string;
  name: string;
  status:
    | "queued"
    | "downloading"
    | "processing"
    | "paused"
    | "completed"
    | "failed"
    | "cancelled";
  progress: number; // Progress as percentage (0-100), can be float
  total_bytes?: number;
  downloaded_bytes: number;
  speed?: number; // Download speed in bytes per second
  url?: string;
  file_name?: string;
  destination_path?: string;
  error_message?: string;
  queue_position?: number;
  priority: number;
  created_at?: string;
  added_at?: string;
  started_at?: string;
  completed_at?: string;
  metadata?: Record<string, any>;
}

export interface DownloaderInfo {
  id: string;
  name: string;
  version: string;
  description: string;
}

export interface CreateDownloadRequest {
  plugin_id: string;
  name: string;
  url?: string;
  file_content?: string; // Base64 encoded file content
  file_name?: string;
  priority?: number;
  metadata?: Record<string, any>;
}

// Fetch available downloaders
export function useDownloaders() {
  return useQuery<{ downloaders: DownloaderInfo[]; count: number }>({
    queryKey: ["downloaders"],
    queryFn: async () => {
      const response = await fetch("/api/downloaders", {
        credentials: "include",
      });

      if (!response.ok) {
        throw new Error("Failed to fetch downloaders");
      }

      return response.json();
    },
  });
}

// Fetch downloads list
export function useDownloads(pluginId?: string, status?: string) {
  return useQuery<{ downloads: Download[]; total: number }>({
    queryKey: ["downloads", pluginId, status],
    queryFn: async () => {
      const params = new URLSearchParams();
      if (pluginId) params.append("plugin_id", pluginId);
      if (status) params.append("status", status);

      const response = await fetch(`/api/downloads?${params}`, {
        credentials: "include",
      });

      if (!response.ok) {
        throw new Error("Failed to fetch downloads");
      }

      return response.json();
    },
    refetchInterval: 2000, // Poll every 2 seconds for updates
  });
}

// Create a new download
export function useCreateDownload() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (request: CreateDownloadRequest) => {
      const response = await fetch("/api/downloads", {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
        },
        credentials: "include",
        body: JSON.stringify(request),
      });

      if (!response.ok) {
        const errorData = await response.json().catch(() => ({}));
        throw new Error(errorData.error || "Failed to create download");
      }

      return response.json() as Promise<Download>;
    },
    onSuccess: () => {
      // Invalidate downloads query to refresh the list
      queryClient.invalidateQueries({ queryKey: ["downloads"] });
    },
  });
}

// Pause a download
export function usePauseDownload() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async ({
      pluginId,
      downloadId,
    }: {
      pluginId: string;
      downloadId: string;
    }) => {
      const response = await fetch(
        `/api/downloads/${pluginId}/${downloadId}/pause`,
        {
          method: "POST",
          credentials: "include",
        },
      );

      if (!response.ok) {
        throw new Error("Failed to pause download");
      }

      return response.json();
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["downloads"] });
    },
  });
}

// Resume a download
export function useResumeDownload() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async ({
      pluginId,
      downloadId,
    }: {
      pluginId: string;
      downloadId: string;
    }) => {
      const response = await fetch(
        `/api/downloads/${pluginId}/${downloadId}/resume`,
        {
          method: "POST",
          credentials: "include",
        },
      );

      if (!response.ok) {
        throw new Error("Failed to resume download");
      }

      return response.json();
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["downloads"] });
    },
  });
}

// Retry a download
export function useRetryDownload() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async ({
      pluginId,
      downloadId,
    }: {
      pluginId: string;
      downloadId: string;
    }) => {
      const response = await fetch(
        `/api/downloads/${pluginId}/${downloadId}/retry`,
        {
          method: "POST",
          credentials: "include",
        },
      );

      if (!response.ok) {
        throw new Error("Failed to retry download");
      }

      return response.json();
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["downloads"] });
    },
  });
}

// Cancel a download
export function useCancelDownload() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async ({
      pluginId,
      downloadId,
    }: {
      pluginId: string;
      downloadId: string;
    }) => {
      const response = await fetch(
        `/api/downloads/${pluginId}/${downloadId}/cancel`,
        {
          method: "POST",
          credentials: "include",
        },
      );

      if (!response.ok) {
        throw new Error("Failed to cancel download");
      }

      return response.json();
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["downloads"] });
    },
  });
}

// Delete a download
export function useDeleteDownload() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async ({
      pluginId,
      downloadId,
    }: {
      pluginId: string;
      downloadId: string;
    }) => {
      const response = await fetch(`/api/downloads/${pluginId}/${downloadId}`, {
        method: "DELETE",
        credentials: "include",
      });

      if (!response.ok) {
        throw new Error("Failed to delete download");
      }

      return response.json();
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["downloads"] });
    },
  });
}
