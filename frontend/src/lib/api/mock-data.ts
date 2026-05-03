import type { AlertRule } from "@/types/alert-rule";
import type { Event } from "@/types/event";
import type { Incident } from "@/types/incident";
import type { LatestMetricSnapshot } from "@/types/metric";
import type { RecoveryAction } from "@/types/recovery-action";
import type { Target } from "@/types/target";

const now = Date.now();
const iso = (minutesAgo: number) => new Date(now - minutesAgo * 60_000).toISOString();

export const mockTargets: Target[] = [
  {
    id: "target-nginx",
    name: "target-nginx",
    node_id: "local-node",
    source: "docker",
    external_id: "demo-nginx-1",
    status: "OK",
    labels: { "container-monitoring.target": "true", image: "nginx:1.27-alpine" },
    last_seen_at: iso(1),
  },
  {
    id: "core-api",
    name: "container-monitoring-core",
    node_id: "local-node",
    source: "docker",
    external_id: "core-1",
    status: "WARNING",
    labels: { service: "core", language: "go" },
    last_seen_at: iso(3),
  },
  {
    id: "clickhouse",
    name: "clickhouse",
    node_id: "local-node",
    source: "docker",
    external_id: "clickhouse-1",
    status: "OK",
    labels: { service: "storage" },
    last_seen_at: iso(2),
  },
];

export const mockMetrics: LatestMetricSnapshot[] = [
  {
    target_id: "target-nginx",
    node_id: "local-node",
    container_name: "target-nginx",
    cpu_usage_percent: 8.2,
    memory_usage_bytes: 43_515_904,
    network_rx_bytes: 14_220_910,
    network_tx_bytes: 6_711_120,
    block_read_bytes: 1_105_920,
    block_write_bytes: 819_200,
    timestamp: iso(1),
  },
  {
    target_id: "core-api",
    node_id: "local-node",
    container_name: "container-monitoring-core",
    cpu_usage_percent: 32.7,
    memory_usage_bytes: 138_412_032,
    network_rx_bytes: 34_220_210,
    network_tx_bytes: 21_718_400,
    block_read_bytes: 5_240_832,
    block_write_bytes: 2_621_440,
    timestamp: iso(3),
  },
  {
    target_id: "clickhouse",
    node_id: "local-node",
    container_name: "clickhouse",
    cpu_usage_percent: 18.4,
    memory_usage_bytes: 274_726_912,
    network_rx_bytes: 64_220_210,
    network_tx_bytes: 19_718_400,
    block_read_bytes: 11_240_832,
    block_write_bytes: 9_621_440,
    timestamp: iso(2),
  },
];

export const mockEvents: Event[] = [
  {
    id: "evt-101",
    target_id: "target-nginx",
    node_id: "local-node",
    container_name: "target-nginx",
    event_type: "container_started",
    severity: "info",
    message: "Container target-nginx started",
    timestamp: iso(8),
  },
  {
    id: "evt-102",
    target_id: "core-api",
    node_id: "local-node",
    container_name: "container-monitoring-core",
    event_type: "container_restarted",
    severity: "warning",
    message: "Core service restarted after health check retry",
    timestamp: iso(21),
  },
  {
    id: "evt-103",
    target_id: "target-nginx",
    node_id: "local-node",
    container_name: "target-nginx",
    event_type: "container_died",
    severity: "critical",
    message: "Demo container exited unexpectedly",
    timestamp: iso(64),
  },
];

export const mockAlertRules: AlertRule[] = [
  {
    id: "rule-cpu-high",
    name: "CPU usage above 80%",
    target_id: "core-api",
    metric_name: "cpu_usage_percent",
    operator: ">",
    threshold: 80,
    duration: "2m",
    severity: "warning",
    recovery_policy: "retry_check",
    enabled: true,
  },
  {
    id: "rule-container-stopped",
    name: "Container stopped",
    target_id: "target-nginx",
    metric_name: "container_event",
    operator: "==",
    threshold: 1,
    duration: "0s",
    severity: "critical",
    recovery_policy: "restart_container",
    enabled: true,
  },
];

export const mockIncidents: Incident[] = [
  {
    id: "1",
    target_id: "target-nginx",
    node_id: "local-node",
    rule_id: "rule-container-stopped",
    status: "resolved",
    severity: "critical",
    started_at: iso(64),
    resolved_at: iso(61),
    last_event_at: iso(61),
    description: "Container target-nginx stopped and was restarted by recovery policy.",
    value: 1,
  },
  {
    id: "2",
    target_id: "core-api",
    node_id: "local-node",
    rule_id: "rule-cpu-high",
    status: "open",
    severity: "warning",
    started_at: iso(18),
    last_event_at: iso(3),
    description: "CPU usage exceeded warning threshold.",
    value: 86.2,
  },
];

export const mockRecoveryActions: RecoveryAction[] = [
  {
    id: "1",
    incident_id: "1",
    target_id: "target-nginx",
    action_type: "restart_container",
    status: "succeeded",
    started_at: iso(63),
    finished_at: iso(61),
    result_message: "Docker restart completed",
  },
  {
    id: "2",
    incident_id: "2",
    target_id: "core-api",
    action_type: "retry_check",
    status: "failed",
    started_at: iso(15),
    finished_at: iso(14),
    result_message: "Condition still above threshold",
  },
];
