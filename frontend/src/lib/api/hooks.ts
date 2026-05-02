import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import type { AlertRuleInput } from "@/types/alert-rule";
import { api } from "./client";

export const queryKeys = {
  health: ["health"] as const,
  targets: ["targets"] as const,
  target: (id: string) => ["targets", id] as const,
  alertRules: ["alert-rules"] as const,
  incidents: ["incidents"] as const,
  incident: (id: string) => ["incidents", id] as const,
  recoveryActions: ["recovery-actions"] as const,
  events: ["events"] as const,
  latestMetrics: ["metrics", "latest"] as const,
  metricHistory: (targetId: string, metricName: string) => ["metrics", "history", targetId, metricName] as const,
};

export function useHealth() {
  return useQuery({ queryKey: queryKeys.health, queryFn: api.health, refetchInterval: 15_000, retry: 1 });
}

export function useTargets() {
  return useQuery({ queryKey: queryKeys.targets, queryFn: api.targets });
}

export function useTarget(id: string) {
  return useQuery({ queryKey: queryKeys.target(id), queryFn: () => api.target(id), enabled: Boolean(id) });
}

export function useAlertRules() {
  return useQuery({ queryKey: queryKeys.alertRules, queryFn: api.alertRules });
}

export function useCreateAlertRule() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (input: AlertRuleInput) => api.createAlertRule(input),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.alertRules });
      toast.success("Alert rule created");
    },
    onError: (error) => toast.error(error instanceof Error ? error.message : "Failed to create alert rule"),
  });
}

export function useUpdateAlertRule() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ id, input }: { id: string; input: AlertRuleInput }) => api.updateAlertRule(id, input),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.alertRules });
      toast.success("Alert rule updated");
    },
    onError: (error) => toast.error(error instanceof Error ? error.message : "Backend does not support updating rules yet"),
  });
}

export function useDeleteAlertRule() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: api.deleteAlertRule,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.alertRules });
      toast.success("Alert rule deleted");
    },
    onError: () => toast.error("Backend does not support deleting rules yet"),
  });
}

export function useIncidents() {
  return useQuery({ queryKey: queryKeys.incidents, queryFn: api.incidents });
}

export function useIncident(id: string) {
  return useQuery({ queryKey: queryKeys.incident(id), queryFn: () => api.incident(id), enabled: Boolean(id) });
}

export function useAcknowledgeIncident() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: api.acknowledgeIncident,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.incidents });
      toast.success("Incident acknowledged");
    },
  });
}

export function useResolveIncident() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: api.resolveIncident,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.incidents });
      toast.success("Incident resolved");
    },
  });
}

export function useRecoveryActions() {
  return useQuery({ queryKey: queryKeys.recoveryActions, queryFn: api.recoveryActions });
}

export function useRetryRecoveryAction() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: api.retryRecoveryAction,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.recoveryActions });
      toast.success("Recovery retry accepted");
    },
  });
}

export function useEvents() {
  return useQuery({ queryKey: queryKeys.events, queryFn: () => api.events() });
}

export function useLatestMetrics() {
  return useQuery({ queryKey: queryKeys.latestMetrics, queryFn: () => api.latestMetrics() });
}

export function useMetricHistory(targetId: string, metricName: string) {
  return useQuery({
    queryKey: queryKeys.metricHistory(targetId, metricName),
    queryFn: () => api.metricHistory(targetId, metricName),
    enabled: Boolean(targetId && metricName),
  });
}
