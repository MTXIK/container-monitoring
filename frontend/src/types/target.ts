export type TargetStatus = "OK" | "WARNING" | "CRITICAL" | "UNKNOWN" | "RECOVERING";

export interface Target {
  id: string;
  name: string;
  node_id: string;
  source: "docker" | "kubernetes" | string;
  external_id: string;
  status: TargetStatus;
  labels?: Record<string, string>;
  last_seen_at?: string;
  created_at?: string;
  updated_at?: string;
}
