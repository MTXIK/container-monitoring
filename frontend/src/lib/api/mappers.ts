import type { AlertRule, AlertOperator } from "@/types/alert-rule";
import type { Event, EventSeverity } from "@/types/event";
import type { Incident, IncidentSeverity, IncidentStatus } from "@/types/incident";
import type { LatestMetricSnapshot } from "@/types/metric";
import type { RecoveryAction, RecoveryActionStatus, RecoveryActionType } from "@/types/recovery-action";
import type { Target, TargetStatus } from "@/types/target";

const operatorMap: Record<string, AlertOperator> = {
  gt: ">",
  lt: "<",
  gte: ">=",
  lte: "<=",
  eq: "==",
  ">": ">",
  "<": "<",
  ">=": ">=",
  "<=": "<=",
  "==": "==",
};

export const toBackendOperator = (operator: AlertOperator) =>
  ({ ">": "gt", "<": "lt", ">=": "gte", "<=": "lte", "==": "eq" })[operator];

function asRecord(value: unknown): Record<string, string> | undefined {
  if (!value || typeof value !== "object") return undefined;
  return Object.fromEntries(Object.entries(value as Record<string, unknown>).map(([key, item]) => [key, String(item)]));
}

function normalizeSeverity(value: unknown): EventSeverity {
  return value === "critical" || value === "warning" || value === "info" ? value : "info";
}

function deriveStatus(raw: Record<string, unknown>, metrics: LatestMetricSnapshot[]): TargetStatus {
  const direct = raw.status;
  if (direct === "OK" || direct === "WARNING" || direct === "CRITICAL" || direct === "UNKNOWN" || direct === "RECOVERING") {
    return direct;
  }
  const metric = metrics.find((item) => item.target_id === raw.id);
  if (!metric) return "UNKNOWN";
  if (metric.cpu_usage_percent >= 90) return "CRITICAL";
  if (metric.cpu_usage_percent >= 75) return "WARNING";
  return "OK";
}

export function mapTarget(raw: unknown, metrics: LatestMetricSnapshot[] = []): Target {
  const item = raw as Record<string, unknown>;
  return {
    id: String(item.id ?? item.ID ?? ""),
    name: String(item.name ?? item.container_name ?? item.id ?? "Unnamed target"),
    node_id: String(item.node_id ?? item.nodeID ?? ""),
    source: String(item.source ?? "docker"),
    external_id: String(item.external_id ?? item.externalID ?? ""),
    status: deriveStatus(item, metrics),
    labels: asRecord(item.labels),
    last_seen_at: String(item.last_seen_at ?? item.updated_at ?? ""),
    created_at: item.created_at ? String(item.created_at) : undefined,
    updated_at: item.updated_at ? String(item.updated_at) : undefined,
  };
}

export function mapMetricSnapshot(raw: unknown): LatestMetricSnapshot {
  const item = raw as Record<string, unknown>;
  return {
    target_id: String(item.target_id ?? ""),
    node_id: String(item.node_id ?? ""),
    container_name: String(item.container_name ?? item.target_id ?? ""),
    cpu_usage_percent: Number(item.cpu_usage_percent ?? 0),
    memory_usage_bytes: Number(item.memory_usage_bytes ?? 0),
    network_rx_bytes: Number(item.network_rx_bytes ?? 0),
    network_tx_bytes: Number(item.network_tx_bytes ?? 0),
    block_read_bytes: Number(item.block_read_bytes ?? 0),
    block_write_bytes: Number(item.block_write_bytes ?? 0),
    timestamp: String(item.timestamp ?? ""),
  };
}

export function mapEvent(raw: unknown): Event {
  const item = raw as Record<string, unknown>;
  return {
    id: String(item.id ?? crypto.randomUUID()),
    target_id: String(item.target_id ?? ""),
    node_id: String(item.node_id ?? ""),
    container_name: item.container_name ? String(item.container_name) : undefined,
    event_type: String(item.event_type ?? "unknown"),
    severity: normalizeSeverity(item.severity),
    message: String(item.message ?? ""),
    timestamp: String(item.timestamp ?? ""),
    payload: item.payload && typeof item.payload === "object" ? (item.payload as Record<string, unknown>) : undefined,
  };
}

export function mapAlertRule(raw: unknown): AlertRule {
  const item = raw as Record<string, unknown>;
  return {
    id: String(item.id ?? ""),
    name: String(item.name ?? "Unnamed rule"),
    target_id: item.target_id ? String(item.target_id) : undefined,
    metric_name: String(item.metric_name ?? ""),
    operator: operatorMap[String(item.condition_operator ?? item.operator ?? ">")] ?? ">",
    threshold: Number(item.threshold ?? 0),
    duration: String(item.duration ?? "0s"),
    severity: normalizeSeverity(item.severity),
    recovery_policy: String(item.recovery_policy ?? item.recovery_action ?? "notify_only") as AlertRule["recovery_policy"],
    enabled: Boolean(item.enabled),
  };
}

export function mapIncident(raw: unknown): Incident {
  const item = raw as Record<string, unknown>;
  return {
    id: String(item.id ?? ""),
    target_id: String(item.target_id ?? ""),
    node_id: item.node_id ? String(item.node_id) : undefined,
    rule_id: item.rule_id ? String(item.rule_id) : undefined,
    status: String(item.status ?? "open") as IncidentStatus,
    severity: String(item.severity ?? "warning") as IncidentSeverity,
    started_at: String(item.started_at ?? ""),
    resolved_at: item.resolved_at ? String(item.resolved_at) : undefined,
    last_event_at: item.last_event_at ? String(item.last_event_at) : undefined,
    description: item.description ? String(item.description) : undefined,
    value: item.value == null ? undefined : Number(item.value),
  };
}

export function mapRecoveryAction(raw: unknown): RecoveryAction {
  const item = raw as Record<string, unknown>;
  return {
    id: String(item.id ?? ""),
    incident_id: String(item.incident_id ?? ""),
    target_id: String(item.target_id ?? ""),
    action_type: String(item.action_type ?? "notify_only") as RecoveryActionType,
    status: String(item.status ?? "pending") as RecoveryActionStatus,
    started_at: String(item.started_at ?? ""),
    finished_at: item.finished_at ? String(item.finished_at) : undefined,
    result_message: item.result_message ? String(item.result_message) : undefined,
  };
}
