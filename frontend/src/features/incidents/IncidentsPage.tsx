import type { ColumnDef } from "@tanstack/react-table";
import { Link } from "react-router-dom";
import { Eye } from "lucide-react";
import { ErrorState } from "@/components/ErrorState";
import { LoadingState } from "@/components/LoadingState";
import { PageHeader } from "@/components/PageHeader";
import { SeverityBadge } from "@/components/status/SeverityBadge";
import { StateBadge } from "@/components/status/StateBadge";
import { DataTable } from "@/components/tables/DataTable";
import { Button } from "@/components/ui/button";
import { formatDateTime } from "@/lib/formatters";
import { useAcknowledgeIncident, useIncidents, useResolveIncident, useTargets } from "@/lib/api/hooks";
import type { Incident } from "@/types/incident";

export function IncidentsPage() {
  const incidents = useIncidents();
  const targets = useTargets();
  const ack = useAcknowledgeIncident();
  const resolve = useResolveIncident();
  const targetName = (id: string) => targets.data?.find((item) => item.id === id)?.name ?? id;

  const columns: ColumnDef<Incident>[] = [
    { accessorKey: "id", header: "ID", cell: ({ row }) => <Link className="font-medium text-primary" to={`/incidents/${row.original.id}`}>#{row.original.id}</Link> },
    { accessorKey: "target_id", header: "Target", cell: ({ row }) => targetName(row.original.target_id) },
    { accessorKey: "rule_id", header: "Rule" },
    { accessorKey: "severity", header: "Severity", cell: ({ row }) => <SeverityBadge severity={row.original.severity} /> },
    { accessorKey: "status", header: "Status", cell: ({ row }) => <StateBadge state={row.original.status} /> },
    { accessorKey: "started_at", header: "Started At", cell: ({ row }) => formatDateTime(row.original.started_at) },
    { accessorKey: "resolved_at", header: "Resolved At", cell: ({ row }) => formatDateTime(row.original.resolved_at) },
    { accessorKey: "last_event_at", header: "Last Event At", cell: ({ row }) => formatDateTime(row.original.last_event_at) },
    {
      id: "actions",
      header: "Actions",
      cell: ({ row }) => (
        <div className="flex gap-2">
          <Button asChild variant="outline" size="sm"><Link to={`/incidents/${row.original.id}`}><Eye data-icon="inline-start" />View</Link></Button>
          <Button variant="secondary" size="sm" onClick={() => ack.mutate(row.original.id)} disabled={row.original.status !== "open"}>Acknowledge</Button>
          <Button variant="secondary" size="sm" onClick={() => resolve.mutate(row.original.id)} disabled={row.original.status === "resolved"}>Resolve</Button>
        </div>
      ),
    },
  ];

  if (incidents.isLoading || targets.isLoading) return <LoadingState label="Loading incidents" />;
  if (incidents.isError) return <ErrorState message="Unable to load incidents" />;

  return (
    <>
      <PageHeader title="Incidents" description="Operational incidents raised by alert rules and container event analysis." />
      <DataTable columns={columns} data={incidents.data ?? []} />
    </>
  );
}
