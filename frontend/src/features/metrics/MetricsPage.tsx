import { useMemo, useState } from "react";
import type { ColumnDef } from "@tanstack/react-table";
import { GrafanaLinkButton } from "@/components/GrafanaLinkButton";
import { ErrorState } from "@/components/ErrorState";
import { LoadingState } from "@/components/LoadingState";
import { PageHeader } from "@/components/PageHeader";
import { DataTable } from "@/components/tables/DataTable";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { formatBytes, formatDateTime, formatPercent } from "@/lib/formatters";
import { useLatestMetrics, useMetricHistory, useTargets } from "@/lib/api/hooks";
import type { LatestMetricSnapshot } from "@/types/metric";

export function MetricsPage() {
  const metrics = useLatestMetrics();
  const targets = useTargets();
  const [target, setTarget] = useState("all");
  const selectedTarget = target === "all" ? metrics.data?.[0]?.target_id ?? "" : target;
  const history = useMetricHistory(selectedTarget, "cpu_usage_percent");

  const rows = useMemo(() => (metrics.data ?? []).filter((item) => target === "all" || item.target_id === target), [metrics.data, target]);
  const targetName = (id: string) => targets.data?.find((item) => item.id === id)?.name ?? id;

  const columns: ColumnDef<LatestMetricSnapshot>[] = [
    { accessorKey: "target_id", header: "Target", cell: ({ row }) => targetName(row.original.target_id) },
    { accessorKey: "node_id", header: "Node" },
    { accessorKey: "cpu_usage_percent", header: "CPU", cell: ({ row }) => formatPercent(row.original.cpu_usage_percent) },
    { accessorKey: "memory_usage_bytes", header: "Memory", cell: ({ row }) => formatBytes(row.original.memory_usage_bytes) },
    { accessorKey: "network_rx_bytes", header: "Network RX", cell: ({ row }) => formatBytes(row.original.network_rx_bytes) },
    { accessorKey: "network_tx_bytes", header: "Network TX", cell: ({ row }) => formatBytes(row.original.network_tx_bytes) },
    { accessorKey: "block_read_bytes", header: "Block Read", cell: ({ row }) => formatBytes(row.original.block_read_bytes) },
    { accessorKey: "block_write_bytes", header: "Block Write", cell: ({ row }) => formatBytes(row.original.block_write_bytes) },
    { accessorKey: "timestamp", header: "Timestamp", cell: ({ row }) => formatDateTime(row.original.timestamp) },
  ];

  if (metrics.isLoading || targets.isLoading) return <LoadingState label="Loading metrics" />;
  if (metrics.isError) return <ErrorState message="Unable to load latest metrics" />;

  return (
    <>
      <PageHeader title="Metrics" description="Latest container metrics for quick inspection. Grafana remains the primary time-series view." actions={<GrafanaLinkButton targetId={selectedTarget} />} />
      <div className="w-full max-w-sm rounded-lg border bg-card p-4">
        <Select value={target} onValueChange={setTarget}>
          <SelectTrigger><SelectValue /></SelectTrigger>
          <SelectContent>
            <SelectItem value="all">All targets</SelectItem>
            {(targets.data ?? []).map((item) => <SelectItem value={item.id} key={item.id}>{item.name}</SelectItem>)}
          </SelectContent>
        </Select>
      </div>
      <Card>
        <CardHeader><CardTitle>CPU History Preview</CardTitle></CardHeader>
        <CardContent>
          <div className="flex h-28 items-end gap-1 rounded-md border bg-background/60 p-3">
            {(history.data ?? []).slice(-48).map((point, index) => (
              <div className="min-w-1 flex-1 rounded-t bg-primary/70" style={{ height: `${Math.max(4, Math.min(100, point.value))}%` }} title={`${point.value}`} key={`${point.timestamp}-${index}`} />
            ))}
            {(history.data ?? []).length === 0 ? <div className="text-sm text-muted-foreground">No history endpoint data yet. Use Grafana for full time-series charts.</div> : null}
          </div>
        </CardContent>
      </Card>
      <DataTable columns={columns} data={rows} />
    </>
  );
}
