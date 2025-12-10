import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { apiGet, apiPost, apiPut, apiDelete } from "../api-client";
import type {
  QualityDefinition,
  QualityProfile,
  MediaQuality,
  QualityUpgradeHistory,
  DetectedQualityInfo,
  CreateQualityDefinitionParams,
  UpdateQualityDefinitionParams,
  CreateQualityProfileParams,
  UpdateQualityProfileParams,
  AssignProfileToMediaParams,
  QualityUpgradeCheckResult,
  MediaForUpgradeResponse,
} from "../types/quality";

// Quality Definitions

export function useQualityDefinitions() {
  return useQuery<QualityDefinition[]>({
    queryKey: ["quality", "definitions"],
    queryFn: () => apiGet<QualityDefinition[]>("/api/quality/definitions"),
  });
}

export function useQualityDefinition(id: number) {
  return useQuery<QualityDefinition>({
    queryKey: ["quality", "definitions", id],
    queryFn: () => apiGet<QualityDefinition>(`/api/quality/definitions/${id}`),
    enabled: !!id,
  });
}

export function useCreateQualityDefinition() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (params: CreateQualityDefinitionParams) =>
      apiPost<QualityDefinition>("/api/quality/definitions", params),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["quality", "definitions"] });
    },
  });
}

export function useUpdateQualityDefinition(id: number) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (params: UpdateQualityDefinitionParams) =>
      apiPut<QualityDefinition>(`/api/quality/definitions/${id}`, params),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["quality", "definitions"] });
      queryClient.invalidateQueries({
        queryKey: ["quality", "definitions", id],
      });
    },
  });
}

export function useDeleteQualityDefinition() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (id: number) => apiDelete(`/api/quality/definitions/${id}`),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["quality", "definitions"] });
    },
  });
}

// Quality Profiles

export function useQualityProfiles() {
  return useQuery<QualityProfile[]>({
    queryKey: ["quality", "profiles"],
    queryFn: () => apiGet<QualityProfile[]>("/api/quality/profiles"),
  });
}

export function useQualityProfile(id: number) {
  return useQuery<QualityProfile>({
    queryKey: ["quality", "profiles", id],
    queryFn: () => apiGet<QualityProfile>(`/api/quality/profiles/${id}`),
    enabled: !!id,
  });
}

export function useCreateQualityProfile() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (params: CreateQualityProfileParams) =>
      apiPost<QualityProfile>("/api/quality/profiles", params),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["quality", "profiles"] });
    },
  });
}

export function useUpdateQualityProfile(id: number) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (params: UpdateQualityProfileParams) =>
      apiPut<QualityProfile>(`/api/quality/profiles/${id}`, params),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["quality", "profiles"] });
      queryClient.invalidateQueries({ queryKey: ["quality", "profiles", id] });
    },
  });
}

export function useDeleteQualityProfile() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (id: number) => apiDelete(`/api/quality/profiles/${id}`),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["quality", "profiles"] });
    },
  });
}

// Quality Detection

export function useDetectQuality() {
  return useMutation({
    mutationFn: (releaseName: string) =>
      apiPost<DetectedQualityInfo>("/api/quality/detect", {
        release_name: releaseName,
      }),
  });
}

// Media Quality

export function useMediaQuality(mediaId: number) {
  return useQuery<MediaQuality | null>({
    queryKey: ["media", mediaId, "quality"],
    queryFn: async () => {
      try {
        return await apiGet<MediaQuality>(`/api/media/${mediaId}/quality`);
      } catch (error: any) {
        // Return null if no quality record exists (404)
        if (error?.status === 404) {
          return null;
        }
        throw error;
      }
    },
    enabled: !!mediaId && mediaId > 0,
  });
}

export function useAssignProfileToMedia(mediaId: number) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (params: AssignProfileToMediaParams) =>
      apiPost(`/api/media/${mediaId}/quality/profile`, params),
    onSuccess: async () => {
      // Invalidate and refetch queries
      await queryClient.invalidateQueries({
        queryKey: ["media", mediaId, "quality"],
      });
      await queryClient.refetchQueries({
        queryKey: ["media", mediaId, "quality"],
      });
      queryClient.invalidateQueries({ queryKey: ["media", mediaId] });
    },
  });
}

export function useCheckUpgrade(mediaId: number, qualityId?: number) {
  return useQuery<QualityUpgradeCheckResult>({
    queryKey: ["media", mediaId, "quality", "upgrade", qualityId],
    queryFn: () =>
      apiGet<QualityUpgradeCheckResult>(
        `/api/media/${mediaId}/quality/upgrade?quality_id=${qualityId}`,
      ),
    enabled: !!mediaId && !!qualityId,
  });
}

export function useQualityUpgradeHistory(mediaId: number) {
  return useQuery<QualityUpgradeHistory[]>({
    queryKey: ["media", mediaId, "quality", "history"],
    queryFn: () =>
      apiGet<QualityUpgradeHistory[]>(`/api/media/${mediaId}/quality/history`),
    enabled: !!mediaId && mediaId > 0,
  });
}

// Upgrade Management

export function useMediaForUpgrade(profileId?: number) {
  return useQuery<MediaForUpgradeResponse>({
    queryKey: ["quality", "upgrades", profileId],
    queryFn: () => {
      const params = profileId ? `?profile_id=${profileId}` : "";
      return apiGet<MediaForUpgradeResponse>(`/api/quality/upgrades${params}`);
    },
  });
}
