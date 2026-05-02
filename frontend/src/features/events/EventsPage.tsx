import { useMemo, useState } from "react";
import type { ColumnDef } from "@tanstack/react-table";
import { ErrorState } from "@/components/ErrorState";
import { LoadingState } from "@/components/LoadingState";
import { PageHeader } from "@/components/PageHeader";
import { SeverityBadge } from "@/components/status/SeverityBadge";
import { DataTable } from "@/components/tables/DataTable";
import { Input } from "@/components/ui/input";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { formatDateTime } from "@/lib/formatters";
import { useEvents, useTargets } from "@/lib/api/hooks";
import type { Event } from "@/types/event";

export function EventsPage() {
  const events = useEvents();
  const targets = useTargets();
  const [type, setType] = useState("all");
  const [severity, setSeverity] = useState("all");
  const [target, setTarget] = useState("all");
  const [range, setRange] = useState("all");

  const rows = useMemo(() => {
    const now = Date.now();
    return (events.data ?? [])
      .filter((item) => type === "all" || item.event_type === type)
      .filter((item) => severity === "all" || item.severity === severity)
      .filter((item) => target === "all" || item.target_id === target)
      .filter((item) => {
        if (range === "all") return true;
        const minutes = range === "1h" ? 60 : range === "24h" ? 1440 : 10_080;
        return now - new Date(item.timestamp).getTime() <= minutes * 60_000;
      });
  }, [events.data, range, severity, target, type]);

  const types = Array.from(new Set((events.data ?? []).map((item) => item.event_type)));
  const targetName = (id: string) => targets.data?.find((item) => item.id === id)?.name ?? id;
  const columns: ColumnDef<Event>[] = [
    { accessorKey: "timestamp", header: "Timestamp", cell: ({ row }) => formatDateTime(row.original.timestamp) },
    { accessorKey: "node_id", header: "Node" },
    { accessorKey: "target_id", header: "Target", cell: ({ row }) => targetName(row.original.target_id) },
    { accessorKey: "event_type", header: "Event Type" },
    { accessorKey: "severity", header: "Severity", cell: ({ row }) => <SeverityBadge severity={row.original.severity} /> },
    { accessorKey: "message", header: "Message" },
  ];

  if (events.isLoading || targets.isLoading) return <LoadingState label="Loading events" />;
  if (events.isError) return <ErrorState message="Unable to load events" />;

  return (
    <>
      <PageHeader title="Events" description="Docker event journal collected from monitored container runtimes." />
      <div className="grid gap-3 rounded-lg border bg-card p-4 md:grid-cols-4">
        <Select value={type} onValueChange={setType}>
          <SelectTrigger><SelectValue /></SelectTrigger>
          <SelectContent><SelectItem value="all">All event types</SelectItem>{types.map((item) => <SelectItem value={item} key={item}>{item}</SelectItem>)}</SelectContent>
        </Select>
        <Select value={severity} onValueChange={setSeverity}>
          <SelectTrigger><SelectValue /></SelectTrigger>
          <SelectContent>{["all", "info", "warning", "critical"].map((item) => <SelectItem value={item} key={item}>{item}</SelectItem>)}</SelectContent>
        </Select>
        <Select value={target} onValueChange={setTarget}>
          <SelectTrigger><SelectValue /></SelectTrigger>
          <SelectContent><SelectItem value="all">All targets</SelectItem>{(targets.data ?? []).map((item) => <SelectItem value={item.id} key={item.id}>{item.name}</SelectItem>)}</SelectContent>
        </Select>
        <Select value={range} onValueChange={setRange}>
          <SelectTrigger><SelectValue /></SelectTrigger>
          <SelectContent>{["all", "1h", "24h", "7d"].map((item) => <SelectItem value={item} key={item}>{item}</SelectItem>)}</SelectContent>
        </Select>
        <Input className="hidden" aria-hidden />
      </div>
      <DataTable columns={columns} data={rows} />
    </>
  );
}
