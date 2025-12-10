import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { apiGet, apiPost, apiPut, apiDelete } from "../api-client";

// Types
export interface MonitoringRule {
  id: number;
  media_item_id: number;
  enabled: boolean;
  quality_profile_id?: number;
  monitor_mode: string;
  search_on_add: boolean;
  automatic_search: boolean;
  backlog_search: boolean;
  prefer_season_packs: boolean;
  minimum_seeders: number;
  tags: string[];
  search_interval_minutes: number;
  last_search_at?: string;
  next_search_at?: string;
  search_count: number;
  items_found_count: number;
  items_grabbed_count: number;
  created_at: string;
  updated_at: string;
  created_by_user_id?: number;
}

export interface EpisodeMonitoring {
  id: number;
  media_item_id: number;
  monitored: boolean;
  has_file: boolean;
  file_id?: number;
  air_date?: string;
  air_date_utc?: string;
  search_count: number;
  last_search_at?: string;
  created_at: string;
  updated_at: string;
}

export interface SearchHistory {
  id: number;
  monitoring_rule_id?: number;
  media_item_id: number;
  search_type: string;
  trigger_source?: string;
  query?: string;
  results_found: number;
  results_approved: number;
  results_rejected: number;
  download_grabbed: boolean;
  download_id?: string;
  search_duration_ms?: number;
  status: string;
  error_message?: string;
  metadata?: Record<string, any>;
  created_at: string;
  created_by_user_id?: number;
}

export interface CalendarEvent {
  id: number;
  media_item_id: number;
  event_type: string;
  event_date: string;
  event_datetime_utc?: string;
  monitored: boolean;
  has_file: boolean;
  downloaded: boolean;
  title: string;
  parent_title?: string;
  metadata?: Record<string, any>;
  created_at: string;
  updated_at: string;
}

export interface SchedulerJob {
  id: number;
  job_name: string;
  job_type: string;
  cron_expression?: string;
  interval_minutes?: number;
  next_run_at?: string;
  last_run_at?: string;
  last_run_duration_ms?: number;
  enabled: boolean;
  running: boolean;
  total_runs: number;
  consecutive_failures: number;
  last_status?: string;
  last_error?: string;
  config?: Record<string, any>;
  created_at: string;
  updated_at: string;
}

export interface MonitoringStats {
  total_monitored: number;
  total_enabled: number;
  total_missing: number;
  total_downloading: number;
  searches_last_24_hours: number;
  grabbed_last_24_hours: number;
}

export interface CreateMonitoringRuleParams {
  media_item_id: number;
  enabled: boolean;
  quality_profile_id?: number;
  monitor_mode: string;
  search_on_add: boolean;
  automatic_search: boolean;
  backlog_search: boolean;
  prefer_season_packs: boolean;
  minimum_seeders: number;
  tags: string[];
  search_interval_minutes: number;
}

export interface UpdateMonitoringRuleParams {
  enabled?: boolean;
  quality_profile_id?: number;
  monitor_mode?: string;
  search_on_add?: boolean;
  automatic_search?: boolean;
  backlog_search?: boolean;
  prefer_season_packs?: boolean;
  minimum_seeders?: number;
  tags?: string[];
  search_interval_minutes?: number;
}

// Hooks

export function useMonitoringRules(enabledOnly: boolean = false) {
  const params = new URLSearchParams();
  if (enabledOnly) params.append("enabled", "true");

  return useQuery<MonitoringRule[]>({
    queryKey: ["monitoring-rules", enabledOnly],
    queryFn: () => apiGet<MonitoringRule[]>("/api/monitoring", params),
  });
}

export function useMonitoringRule(id: number) {
  return useQuery<MonitoringRule>({
    queryKey: ["monitoring-rule", id],
    queryFn: () => apiGet<MonitoringRule>(`/api/monitoring/${id}`),
    enabled: !!id,
  });
}

export function useMonitoringRuleByMedia(mediaId: number) {
  return useQuery<MonitoringRule>({
    queryKey: ["monitoring-rule-media", mediaId],
    queryFn: () => apiGet<MonitoringRule>(`/api/media/${mediaId}/monitoring`),
    enabled: !!mediaId,
  });
}

export function useCreateMonitoringRule() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (params: CreateMonitoringRuleParams) =>
      apiPost<MonitoringRule>("/api/monitoring", params),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["monitoring-rules"] });
      queryClient.invalidateQueries({ queryKey: ["monitoring-stats"] });
    },
  });
}

export function useUpdateMonitoringRule(id: number) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (params: UpdateMonitoringRuleParams) =>
      apiPut<MonitoringRule>(`/api/monitoring/${id}`, params),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["monitoring-rule", id] });
      queryClient.invalidateQueries({ queryKey: ["monitoring-rules"] });
      queryClient.invalidateQueries({ queryKey: ["monitoring-stats"] });
    },
  });
}

export function useDeleteMonitoringRule() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (id: number) => apiDelete(`/api/monitoring/${id}`),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["monitoring-rules"] });
      queryClient.invalidateQueries({ queryKey: ["monitoring-stats"] });
    },
  });
}

export function useMissingEpisodes(limit: number = 50) {
  const params = new URLSearchParams();
  params.append("limit", String(limit));

  return useQuery<EpisodeMonitoring[]>({
    queryKey: ["missing-episodes", limit],
    queryFn: () => apiGet<EpisodeMonitoring[]>("/api/monitoring/missing", params),
  });
}

export function useSearchHistory(mediaId: number, limit: number = 20) {
  const params = new URLSearchParams();
  params.append("limit", String(limit));

  return useQuery<SearchHistory[]>({
    queryKey: ["search-history", mediaId, limit],
    queryFn: () =>
      apiGet<SearchHistory[]>(`/api/media/${mediaId}/monitoring/history`, params),
    enabled: !!mediaId,
  });
}

export function useCalendarEvents(
  startDate: string,
  endDate: string,
  monitoredOnly: boolean = false
) {
  const params = new URLSearchParams();
  params.append("start", startDate);
  params.append("end", endDate);
  if (monitoredOnly) params.append("monitored", "true");

  return useQuery<CalendarEvent[]>({
    queryKey: ["calendar-events", startDate, endDate, monitoredOnly],
    queryFn: () => apiGet<CalendarEvent[]>("/api/calendar", params),
  });
}

export function useMonitoringStats() {
  return useQuery<MonitoringStats>({
    queryKey: ["monitoring-stats"],
    queryFn: () => apiGet<MonitoringStats>("/api/monitoring/stats"),
    refetchInterval: 30000, // Refetch every 30 seconds
  });
}

export function useSchedulerJobs() {
  return useQuery<SchedulerJob[]>({
    queryKey: ["scheduler-jobs"],
    queryFn: () => apiGet<SchedulerJob[]>("/api/scheduler/jobs"),
    refetchInterval: 10000, // Refetch every 10 seconds
  });
}

export function useTriggerJob() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (jobId: number) =>
      apiPost(`/api/scheduler/jobs/${jobId}/trigger`, {}),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["scheduler-jobs"] });
    },
  });
}
