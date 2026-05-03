export type RecoveryActionType = "notify_only" | "retry_check" | "restart_container";
export type RecoveryActionStatus = "pending" | "running" | "succeeded" | "failed" | "skipped";

export interface RecoveryAction {
  id: string;
  incident_id: string;
  target_id: string;
  action_type: RecoveryActionType;
  status: RecoveryActionStatus;
  started_at: string;
  finished_at?: string;
  result_message?: string;
}
