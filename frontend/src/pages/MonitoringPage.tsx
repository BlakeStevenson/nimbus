import { useState } from "react";
import {
  useMonitoringStats,
  useMonitoringRules,
  useMissingEpisodes,
  useSchedulerJobs,
  useTriggerJob,
  useUpdateMonitoringRule,
  useDeleteMonitoringRule,
} from "../lib/api/monitoring";
import { Button } from "../components/ui/Button";
import { Card } from "../components/ui/Card";

export default function MonitoringPage() {
  const [selectedTab, setSelectedTab] = useState<
    "overview" | "rules" | "missing" | "scheduler"
  >("overview");

  const { data: stats, isLoading: statsLoading } = useMonitoringStats();
  const { data: rules, isLoading: rulesLoading } = useMonitoringRules();
  const { data: missingEpisodes, isLoading: missingLoading } = useMissingEpisodes();
  const { data: jobs, isLoading: jobsLoading } = useSchedulerJobs();

  const triggerJob = useTriggerJob();
  const updateRule = useUpdateMonitoringRule(0);
  const deleteRule = useDeleteMonitoringRule();

  const handleTriggerJob = async (jobId: number) => {
    try {
      await triggerJob.mutateAsync(jobId);
      alert("Job triggered successfully");
    } catch (error) {
      alert("Failed to trigger job");
    }
  };

  const handleToggleRule = async (ruleId: number, enabled: boolean) => {
    try {
      await updateRule.mutateAsync({ enabled: !enabled });
      alert("Rule updated successfully");
    } catch (error) {
      alert("Failed to update rule");
    }
  };

  const handleDeleteRule = async (ruleId: number) => {
    if (!confirm("Are you sure you want to delete this monitoring rule?")) {
      return;
    }

    try {
      await deleteRule.mutateAsync(ruleId);
      alert("Rule deleted successfully");
    } catch (error) {
      alert("Failed to delete rule");
    }
  };

  return (
    <div className="container mx-auto px-4 py-8">
      <div className="mb-8">
        <h1 className="text-3xl font-bold mb-2">Monitoring & Automation</h1>
        <p className="text-gray-600 dark:text-gray-400">
          Monitor and automatically search for media items
        </p>
      </div>

      {/* Tabs */}
      <div className="flex gap-2 mb-6 border-b border-gray-200 dark:border-gray-700">
        <button
          onClick={() => setSelectedTab("overview")}
          className={`px-4 py-2 font-medium transition-colors ${
            selectedTab === "overview"
              ? "text-blue-600 dark:text-blue-400 border-b-2 border-blue-600 dark:border-blue-400"
              : "text-gray-600 dark:text-gray-400 hover:text-gray-900 dark:hover:text-gray-200"
          }`}
        >
          Overview
        </button>
        <button
          onClick={() => setSelectedTab("rules")}
          className={`px-4 py-2 font-medium transition-colors ${
            selectedTab === "rules"
              ? "text-blue-600 dark:text-blue-400 border-b-2 border-blue-600 dark:border-blue-400"
              : "text-gray-600 dark:text-gray-400 hover:text-gray-900 dark:hover:text-gray-200"
          }`}
        >
          Monitoring Rules
        </button>
        <button
          onClick={() => setSelectedTab("missing")}
          className={`px-4 py-2 font-medium transition-colors ${
            selectedTab === "missing"
              ? "text-blue-600 dark:text-blue-400 border-b-2 border-blue-600 dark:border-blue-400"
              : "text-gray-600 dark:text-gray-400 hover:text-gray-900 dark:hover:text-gray-200"
          }`}
        >
          Missing / Wanted
        </button>
        <button
          onClick={() => setSelectedTab("scheduler")}
          className={`px-4 py-2 font-medium transition-colors ${
            selectedTab === "scheduler"
              ? "text-blue-600 dark:text-blue-400 border-b-2 border-blue-600 dark:border-blue-400"
              : "text-gray-600 dark:text-gray-400 hover:text-gray-900 dark:hover:text-gray-200"
          }`}
        >
          Scheduler
        </button>
      </div>

      {/* Overview Tab */}
      {selectedTab === "overview" && (
        <div className="space-y-6">
          {statsLoading ? (
            <div>Loading statistics...</div>
          ) : stats ? (
            <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
              <Card className="p-6">
                <h3 className="text-lg font-semibold mb-2">Total Monitored</h3>
                <p className="text-3xl font-bold text-blue-600 dark:text-blue-400">
                  {stats.total_monitored}
                </p>
                <p className="text-sm text-gray-600 dark:text-gray-400 mt-1">
                  {stats.total_enabled} enabled
                </p>
              </Card>

              <Card className="p-6">
                <h3 className="text-lg font-semibold mb-2">Missing Items</h3>
                <p className="text-3xl font-bold text-orange-600 dark:text-orange-400">
                  {stats.total_missing}
                </p>
                <p className="text-sm text-gray-600 dark:text-gray-400 mt-1">
                  Episodes without files
                </p>
              </Card>

              <Card className="p-6">
                <h3 className="text-lg font-semibold mb-2">Downloading</h3>
                <p className="text-3xl font-bold text-green-600 dark:text-green-400">
                  {stats.total_downloading}
                </p>
                <p className="text-sm text-gray-600 dark:text-gray-400 mt-1">
                  Currently in progress
                </p>
              </Card>

              <Card className="p-6">
                <h3 className="text-lg font-semibold mb-2">Searches (24h)</h3>
                <p className="text-3xl font-bold text-purple-600 dark:text-purple-400">
                  {stats.searches_last_24_hours}
                </p>
                <p className="text-sm text-gray-600 dark:text-gray-400 mt-1">
                  Automatic searches
                </p>
              </Card>

              <Card className="p-6">
                <h3 className="text-lg font-semibold mb-2">Grabbed (24h)</h3>
                <p className="text-3xl font-bold text-teal-600 dark:text-teal-400">
                  {stats.grabbed_last_24_hours}
                </p>
                <p className="text-sm text-gray-600 dark:text-gray-400 mt-1">
                  Downloads initiated
                </p>
              </Card>
            </div>
          ) : (
            <div>No statistics available</div>
          )}
        </div>
      )}

      {/* Monitoring Rules Tab */}
      {selectedTab === "rules" && (
        <div className="space-y-4">
          {rulesLoading ? (
            <div>Loading monitoring rules...</div>
          ) : rules && rules.length > 0 ? (
            <div className="space-y-3">
              {rules.map((rule) => (
                <Card key={rule.id} className="p-4">
                  <div className="flex items-center justify-between">
                    <div className="flex-1">
                      <div className="flex items-center gap-3">
                        <h3 className="font-semibold">Media Item #{rule.media_item_id}</h3>
                        <span
                          className={`px-2 py-1 text-xs rounded ${
                            rule.enabled
                              ? "bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-200"
                              : "bg-gray-100 text-gray-800 dark:bg-gray-800 dark:text-gray-200"
                          }`}
                        >
                          {rule.enabled ? "Enabled" : "Disabled"}
                        </span>
                        <span className="px-2 py-1 text-xs rounded bg-blue-100 text-blue-800 dark:bg-blue-900 dark:text-blue-200">
                          {rule.monitor_mode}
                        </span>
                      </div>
                      <div className="mt-2 text-sm text-gray-600 dark:text-gray-400">
                        <p>
                          Searches: {rule.search_count} | Found: {rule.items_found_count} |
                          Grabbed: {rule.items_grabbed_count}
                        </p>
                        <p>
                          {rule.automatic_search && "Auto Search"}{" "}
                          {rule.backlog_search && "• Backlog Search"}{" "}
                          {rule.prefer_season_packs && "• Prefer Season Packs"}
                        </p>
                      </div>
                    </div>
                    <div className="flex gap-2">
                      <Button
                        variant="secondary"
                        size="sm"
                        onClick={() => handleToggleRule(rule.id, rule.enabled)}
                      >
                        {rule.enabled ? "Disable" : "Enable"}
                      </Button>
                      <Button
                        variant="danger"
                        size="sm"
                        onClick={() => handleDeleteRule(rule.id)}
                      >
                        Delete
                      </Button>
                    </div>
                  </div>
                </Card>
              ))}
            </div>
          ) : (
            <Card className="p-8 text-center">
              <p className="text-gray-600 dark:text-gray-400">
                No monitoring rules configured
              </p>
            </Card>
          )}
        </div>
      )}

      {/* Missing / Wanted Tab */}
      {selectedTab === "missing" && (
        <div className="space-y-4">
          {missingLoading ? (
            <div>Loading missing episodes...</div>
          ) : missingEpisodes && missingEpisodes.length > 0 ? (
            <div className="space-y-3">
              {missingEpisodes.map((episode) => (
                <Card key={episode.id} className="p-4">
                  <div className="flex items-center justify-between">
                    <div>
                      <h3 className="font-semibold">Episode #{episode.media_item_id}</h3>
                      <p className="text-sm text-gray-600 dark:text-gray-400">
                        {episode.air_date
                          ? `Aired: ${new Date(episode.air_date).toLocaleDateString()}`
                          : "Air date unknown"}
                      </p>
                      <p className="text-sm text-gray-600 dark:text-gray-400">
                        Searches: {episode.search_count}
                      </p>
                    </div>
                    <div>
                      <span
                        className={`px-2 py-1 text-xs rounded ${
                          episode.monitored
                            ? "bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-200"
                            : "bg-gray-100 text-gray-800 dark:bg-gray-800 dark:text-gray-200"
                        }`}
                      >
                        {episode.monitored ? "Monitored" : "Not Monitored"}
                      </span>
                    </div>
                  </div>
                </Card>
              ))}
            </div>
          ) : (
            <Card className="p-8 text-center">
              <p className="text-gray-600 dark:text-gray-400">No missing episodes</p>
            </Card>
          )}
        </div>
      )}

      {/* Scheduler Tab */}
      {selectedTab === "scheduler" && (
        <div className="space-y-4">
          {jobsLoading ? (
            <div>Loading scheduler jobs...</div>
          ) : jobs && jobs.length > 0 ? (
            <div className="space-y-3">
              {jobs.map((job) => (
                <Card key={job.id} className="p-4">
                  <div className="flex items-center justify-between">
                    <div className="flex-1">
                      <div className="flex items-center gap-3">
                        <h3 className="font-semibold">{job.job_name}</h3>
                        <span
                          className={`px-2 py-1 text-xs rounded ${
                            job.enabled
                              ? "bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-200"
                              : "bg-gray-100 text-gray-800 dark:bg-gray-800 dark:text-gray-200"
                          }`}
                        >
                          {job.enabled ? "Enabled" : "Disabled"}
                        </span>
                        {job.running && (
                          <span className="px-2 py-1 text-xs rounded bg-blue-100 text-blue-800 dark:bg-blue-900 dark:text-blue-200">
                            Running
                          </span>
                        )}
                        {job.last_status && (
                          <span
                            className={`px-2 py-1 text-xs rounded ${
                              job.last_status === "success"
                                ? "bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-200"
                                : "bg-red-100 text-red-800 dark:bg-red-900 dark:text-red-200"
                            }`}
                          >
                            {job.last_status}
                          </span>
                        )}
                      </div>
                      <div className="mt-2 text-sm text-gray-600 dark:text-gray-400">
                        <p>Type: {job.job_type}</p>
                        <p>
                          Interval: {job.interval_minutes} minutes | Total runs:{" "}
                          {job.total_runs}
                        </p>
                        {job.last_run_at && (
                          <p>
                            Last run: {new Date(job.last_run_at).toLocaleString()}
                            {job.last_run_duration_ms &&
                              ` (${job.last_run_duration_ms}ms)`}
                          </p>
                        )}
                        {job.next_run_at && (
                          <p>Next run: {new Date(job.next_run_at).toLocaleString()}</p>
                        )}
                        {job.consecutive_failures > 0 && (
                          <p className="text-red-600 dark:text-red-400">
                            Consecutive failures: {job.consecutive_failures}
                          </p>
                        )}
                      </div>
                    </div>
                    <div>
                      <Button
                        variant="primary"
                        size="sm"
                        onClick={() => handleTriggerJob(job.id)}
                        disabled={job.running || !job.enabled}
                      >
                        Trigger Now
                      </Button>
                    </div>
                  </div>
                </Card>
              ))}
            </div>
          ) : (
            <Card className="p-8 text-center">
              <p className="text-gray-600 dark:text-gray-400">No scheduler jobs found</p>
            </Card>
          )}
        </div>
      )}
    </div>
  );
}
