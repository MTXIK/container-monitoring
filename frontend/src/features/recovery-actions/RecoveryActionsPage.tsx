import type { ColumnDef } from "@tanstack/react-table";
import { Link } from "react-router-dom";
import { ErrorState } from "@/components/ErrorState";
import { LoadingState } from "@/components/LoadingState";
import { PageHeader } from "@/components/PageHeader";
import { StateBadge } from "@/components/status/StateBadge";
import { DataTable } from "@/components/tables/DataTable";
import { Button } from "@/components/ui/button";
import { formatDateTime } from "@/lib/formatters";
import { useRecoveryActions, useRetryRecoveryAction, useTargets } from "@/lib/api/hooks";
import type { RecoveryAction } from "@/types/recovery-action";

export function RecoveryActionsPage() {
  const actions = useRecoveryActions();
  const targets = useTargets();
  const retry = useRetryRecoveryAction();
  const targetName = (id: string) => targets.data?.find((item) => item.id === id)?.name ?? id;

  const columns: ColumnDef<RecoveryAction>[] = [
    { accessorKey: "id", header: "ID" },
    { accessorKey: "incident_id", header: "Incident ID", cell: ({ row }) => <Link className="text-primary" to={`/incidents/${row.original.incident_id}`}>#{row.original.incident_id}</Link> },
    { accessorKey: "target_id", header: "Target", cell: ({ row }) => targetName(row.original.target_id) },
    { accessorKey: "action_type", header: "Action Type" },
    { accessorKey: "status", header: "Status", cell: ({ row }) => <StateBadge state={row.original.status} /> },
    { accessorKey: "started_at", header: "Started At", cell: ({ row }) => formatDateTime(row.original.started_at) },
    { accessorKey: "finished_at", header: "Finished At", cell: ({ row }) => formatDateTime(row.original.finished_at) },
    { accessorKey: "result_message", header: "Result Message" },
    { id: "actions", header: "Actions", cell: ({ row }) => <Button size="sm" variant="outline" disabled={row.original.status !== "failed"} onClick={() => retry.mutate(row.original.id)}>Retry</Button> },
  ];

  if (actions.isLoading || targets.isLoading) return <LoadingState label="Loading recovery actions" />;
  if (actions.isError) return <ErrorState message="Unable to load recovery actions" />;

  return (
    <>
      <PageHeader title="Recovery Actions" description="Self-healing attempts and notifier actions generated for incidents." />
      <DataTable columns={columns} data={actions.data ?? []} />
    </>
  );
}
