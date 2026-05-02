export type EventSeverity = "info" | "warning" | "critical";

export interface Event {
  id: string;
  target_id: string;
  node_id: string;
  container_name?: string;
  event_type: string;
  severity: EventSeverity;
  message: string;
  timestamp: string;
  payload?: Record<string, unknown>;
}
