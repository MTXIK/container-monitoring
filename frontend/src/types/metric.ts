export interface Metric {
  target_id: string;
  node_id: string;
  metric_name: string;
  value: number;
  unit: string;
  timestamp: string;
  labels?: Record<string, string>;
}

export interface LatestMetricSnapshot {
  target_id: string;
  node_id: string;
  container_name: string;
  cpu_usage_percent: number;
  memory_usage_bytes: number;
  network_rx_bytes: number;
  network_tx_bytes: number;
  block_read_bytes: number;
  block_write_bytes: number;
  timestamp: string;
}
