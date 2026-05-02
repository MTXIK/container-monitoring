import type { AlertRule, AlertRuleInput } from "@/types/alert-rule";
import type { Event } from "@/types/event";
import type { Incident } from "@/types/incident";
import type { LatestMetricSnapshot, Metric } from "@/types/metric";
import type { RecoveryAction } from "@/types/recovery-action";
import type { Target } from "@/types/target";
import { mapAlertRule, mapEvent, mapIncident, mapMetricSnapshot, mapRecoveryAction, mapTarget, toBackendOperator } from "./mappers";
import { mockAlertRules, mockEvents, mockIncidents, mockMetrics, mockRecoveryActions, mockTargets } from "./mock-data";

const API_BASE_URL = import.meta.env.VITE_API_BASE_URL ?? "http://localhost:8080";
export const GRAFANA_URL = import.meta.env.VITE_GRAFANA_URL ?? "http://localhost:3000";
export const SWAGGER_URL = `${API_BASE_URL}/swagger/`;
const MOCK_FALLBACK = import.meta.env.VITE_ENABLE_MOCK_FALLBACK !== "false";

type RequestOptions = RequestInit & { fallback?: unknown };

class ApiError extends Error {
  status: number;

  constructor(message: string, status: number) {
    super(message);
    this.status = status;
  }
}

async function request<T>(path: string, options: RequestOptions = {}): Promise<T> {
  const { fallback, ...init } = options;
  try {
    const response = await fetch(`${API_BASE_URL}${path}`, {
      headers: { "Content-Type": "application/json", ...init.headers },
      ...init,
    });
    if (!response.ok) {
      throw new ApiError(`${response.status} ${response.statusText}`, response.status);
    }
    if (response.status === 204) return undefined as T;
    return (await response.json()) as T;
  } catch (error) {
    if (MOCK_FALLBACK && fallback !== undefined) return fallback as T;
    throw error instanceof Error ? error : new Error("API request failed");
  }
}

export const api = {
  async health() {
    return request<{ status: string }>("/health", { fallback: { status: "mock" } });
  },
  async ready() {
    return request<{ status: string }>("/ready", { fallback: { status: "mock" } });
  },
  async latestMetrics(limit = 100): Promise<LatestMetricSnapshot[]> {
    const data = await request<unknown[]>(`/api/v1/metrics/latest?limit=${limit}`, { fallback: mockMetrics });
    return data.map(mapMetricSnapshot);
  },
  async targets(): Promise<Target[]> {
    const metrics = await api.latestMetrics().catch(() => mockMetrics);
    const data = await request<unknown[]>("/api/v1/targets", { fallback: mockTargets });
    return data.map((item) => mapTarget(item, metrics));
  },
  async target(id: string): Promise<Target | undefined> {
    const metrics = await api.latestMetrics().catch(() => mockMetrics);
    const data = await request<unknown>(`/api/v1/targets/${id}`, { fallback: mockTargets.find((item) => item.id === id) });
    return data ? mapTarget(data, metrics) : undefined;
  },
  async events(limit = 100): Promise<Event[]> {
    const data = await request<unknown[]>(`/api/v1/events?limit=${limit}`, { fallback: mockEvents });
    return data.map(mapEvent);
  },
  async targetEvents(id: string, limit = 100): Promise<Event[]> {
    const data = await request<unknown[]>(`/api/v1/targets/${id}/events?limit=${limit}`, {
      fallback: mockEvents.filter((item) => item.target_id === id),
    });
    return data.map(mapEvent);
  },
  async metricHistory(targetId: string, metricName: string): Promise<Metric[]> {
    const data = await request<unknown[]>(
      `/api/v1/metrics/history?target_id=${encodeURIComponent(targetId)}&metric_name=${encodeURIComponent(metricName)}&limit=80`,
      { fallback: [] },
    );
    return data as Metric[];
  },
  async alertRules(): Promise<AlertRule[]> {
    const data = await request<unknown[]>("/api/v1/alert-rules", { fallback: mockAlertRules });
    return data.map(mapAlertRule);
  },
  async createAlertRule(input: AlertRuleInput): Promise<AlertRule> {
    const body = {
      id: input.id,
      name: input.name,
      target_id: input.target_id,
      metric_name: input.metric_name,
      condition_operator: toBackendOperator(input.operator),
      threshold: input.threshold,
      duration: input.duration,
      severity: input.severity,
      recovery_action: input.recovery_policy,
      enabled: input.enabled,
    };
    // TODO: remove local fallback when PATCH/DELETE/full rule CRUD is available in the backend.
    const data = await request<unknown>("/api/v1/alert-rules", {
      method: "POST",
      body: JSON.stringify(body),
      fallback: { ...body, id: input.id ?? crypto.randomUUID() },
    });
    return mapAlertRule(data);
  },
  async updateAlertRule(id: string, input: AlertRuleInput): Promise<AlertRule> {
    const data = await request<unknown>(`/api/v1/alert-rules/${id}`, {
      method: "PATCH",
      body: JSON.stringify(input),
      fallback: { ...input, id },
    });
    return mapAlertRule(data);
  },
  async deleteAlertRule(id: string): Promise<void> {
    await request<void>(`/api/v1/alert-rules/${id}`, { method: "DELETE", fallback: undefined });
  },
  async incidents(): Promise<Incident[]> {
    const data = await request<unknown[]>("/api/v1/incidents", { fallback: mockIncidents });
    return data.map(mapIncident);
  },
  async incident(id: string): Promise<Incident | undefined> {
    const list = await api.incidents();
    return list.find((item) => item.id === id);
  },
  async acknowledgeIncident(id: string): Promise<void> {
    await request<void>(`/api/v1/incidents/${id}/ack`, { method: "POST", fallback: undefined });
  },
  async resolveIncident(id: string): Promise<void> {
    await request<void>(`/api/v1/incidents/${id}/resolve`, { method: "POST", fallback: undefined });
  },
  async recoveryActions(): Promise<RecoveryAction[]> {
    const data = await request<unknown[]>("/api/v1/recovery-actions", { fallback: mockRecoveryActions });
    return data.map(mapRecoveryAction);
  },
  async retryRecoveryAction(id: string): Promise<void> {
    await request(`/api/v1/recovery-actions/${id}/retry`, { method: "POST", fallback: { id, status: "retry accepted" } });
  },
};
