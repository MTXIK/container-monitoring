export type IncidentStatus = "open" | "acknowledged" | "recovering" | "resolved" | "failed";
export type IncidentSeverity = "info" | "warning" | "critical";

export interface Incident {
  id: string;
  target_id: string;
  node_id?: string;
  rule_id?: string;
  status: IncidentStatus;
  severity: IncidentSeverity;
  started_at: string;
  resolved_at?: string;
  last_event_at?: string;
  description?: string;
  value?: number;
}
