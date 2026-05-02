import { useMemo, useState } from "react";
import type { ColumnDef } from "@tanstack/react-table";
import { Link } from "react-router-dom";
import { Eye } from "lucide-react";
import { DataTable } from "@/components/tables/DataTable";
import { ErrorState } from "@/components/ErrorState";
import { LoadingState } from "@/components/LoadingState";
import { PageHeader } from "@/components/PageHeader";
import { StatusBadge } from "@/components/status/StatusBadge";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { formatBytes, formatDateTime, formatPercent } from "@/lib/formatters";
import { useLatestMetrics, useTargets } from "@/lib/api/hooks";
import type { Target, TargetStatus } from "@/types/target";

type Row = Target & { cpu?: number; memory?: number };

export function TargetsPage() {
  const targets = useTargets();
  const metrics = useLatestMetrics();
  const [search, setSearch] = useState("");
  const [status, setStatus] = useState("all");
  const [source, setSource] = useState("all");

  const rows = useMemo<Row[]>(() => {
    const metricMap = new Map((metrics.data ?? []).map((item) => [item.target_id, item]));
    return (targets.data ?? [])
      .map((target) => ({ ...target, cpu: metricMap.get(target.id)?.cpu_usage_percent, memory: metricMap.get(target.id)?.memory_usage_bytes }))
      .filter((item) => item.name.toLowerCase().includes(search.toLowerCase()))
      .filter((item) => status === "all" || item.status === status)
      .filter((item) => source === "all" || item.source === source);
  }, [metrics.data, search, source, status, targets.data]);

  const sources = Array.from(new Set((targets.data ?? []).map((item) => item.source)));
  const columns: ColumnDef<Row>[] = [
    { accessorKey: "name", header: "Name", cell: ({ row }) => <Link className="font-medium text-primary" to={`/targets/${row.original.id}`}>{row.original.name}</Link> },
    { accessorKey: "node_id", header: "Node" },
    { accessorKey: "source", header: "Source" },
    { accessorKey: "status", header: "Status", cell: ({ row }) => <StatusBadge status={row.original.status} /> },
    { accessorKey: "cpu", header: "CPU", cell: ({ row }) => formatPercent(row.original.cpu) },
    { accessorKey: "memory", header: "Memory", cell: ({ row }) => formatBytes(row.original.memory) },
    { accessorKey: "last_seen_at", header: "Last Seen", cell: ({ row }) => formatDateTime(row.original.last_seen_at) },
    { id: "actions", header: "Actions", cell: ({ row }) => <Button asChild variant="outline" size="sm"><Link to={`/targets/${row.original.id}`}><Eye data-icon="inline-start" />View details</Link></Button> },
  ];

  if (targets.isLoading || metrics.isLoading) return <LoadingState label="Loading targets" />;
  if (targets.isError) return <ErrorState message="Unable to load targets" />;

  return (
    <>
      <PageHeader title="Targets" description="Observed containers discovered by the agent and enriched with latest runtime metrics." />
      <div className="grid gap-3 rounded-lg border bg-card p-4 md:grid-cols-[1fr_180px_180px]">
        <Input placeholder="Search by name" value={search} onChange={(event) => setSearch(event.target.value)} />
        <Select value={status} onValueChange={setStatus}>
          <SelectTrigger><SelectValue placeholder="Status" /></SelectTrigger>
          <SelectContent>
            <SelectItem value="all">All statuses</SelectItem>
            {(["OK", "WARNING", "CRITICAL", "UNKNOWN", "RECOVERING"] satisfies TargetStatus[]).map((item) => <SelectItem value={item} key={item}>{item}</SelectItem>)}
          </SelectContent>
        </Select>
        <Select value={source} onValueChange={setSource}>
          <SelectTrigger><SelectValue placeholder="Source" /></SelectTrigger>
          <SelectContent>
            <SelectItem value="all">All sources</SelectItem>
            {sources.map((item) => <SelectItem value={item} key={item}>{item}</SelectItem>)}
          </SelectContent>
        </Select>
      </div>
      <DataTable columns={columns} data={rows} />
    </>
  );
}
