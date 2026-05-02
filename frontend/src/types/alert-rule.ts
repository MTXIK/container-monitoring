export type AlertOperator = ">" | "<" | ">=" | "<=" | "==";
export type AlertSeverity = "info" | "warning" | "critical";
export type RecoveryPolicy = "notify_only" | "retry_check" | "restart_container";

export interface AlertRule {
  id: string;
  name: string;
  target_id?: string;
  metric_name: string;
  operator: AlertOperator;
  threshold: number;
  duration: string;
  severity: AlertSeverity;
  recovery_policy: RecoveryPolicy;
  enabled: boolean;
}

export type AlertRuleInput = Omit<AlertRule, "id"> & { id?: string };
