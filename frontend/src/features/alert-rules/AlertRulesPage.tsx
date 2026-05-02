import type { ColumnDef } from "@tanstack/react-table";
import { Plus, Trash2 } from "lucide-react";
import { ConfirmDialog } from "@/components/ConfirmDialog";
import { ErrorState } from "@/components/ErrorState";
import { LoadingState } from "@/components/LoadingState";
import { PageHeader } from "@/components/PageHeader";
import { SeverityBadge } from "@/components/status/SeverityBadge";
import { DataTable } from "@/components/tables/DataTable";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { useAlertRules, useCreateAlertRule, useDeleteAlertRule, useTargets, useUpdateAlertRule } from "@/lib/api/hooks";
import type { AlertRule } from "@/types/alert-rule";
import { AlertRuleForm } from "./AlertRuleForm";

export function AlertRulesPage() {
  const rules = useAlertRules();
  const targets = useTargets();
  const createRule = useCreateAlertRule();
  const updateRule = useUpdateAlertRule();
  const deleteRule = useDeleteAlertRule();

  const columns: ColumnDef<AlertRule>[] = [
    { accessorKey: "name", header: "Name", cell: ({ row }) => <span className="font-medium">{row.original.name}</span> },
    { accessorKey: "metric_name", header: "Metric/Event" },
    { accessorKey: "operator", header: "Operator" },
    { accessorKey: "threshold", header: "Threshold" },
    { accessorKey: "duration", header: "Duration" },
    { accessorKey: "severity", header: "Severity", cell: ({ row }) => <SeverityBadge severity={row.original.severity} /> },
    { accessorKey: "recovery_policy", header: "Recovery Policy" },
    { accessorKey: "enabled", header: "Enabled", cell: ({ row }) => <Badge variant={row.original.enabled ? "default" : "secondary"}>{row.original.enabled ? "enabled" : "disabled"}</Badge> },
    {
      id: "actions",
      header: "Actions",
      cell: ({ row }) => (
        <div className="flex gap-2">
          <AlertRuleForm trigger={<Button variant="outline" size="sm">Edit</Button>} rule={row.original} targets={targets.data ?? []} onSubmit={(input) => updateRule.mutate({ id: row.original.id, input })} />
          <Button variant="secondary" size="sm" onClick={() => updateRule.mutate({ id: row.original.id, input: { ...row.original, enabled: !row.original.enabled } })}>
            {row.original.enabled ? "Disable" : "Enable"}
          </Button>
          <ConfirmDialog
            trigger={<Button variant="destructive" size="sm"><Trash2 data-icon="inline-start" />Delete</Button>}
            title="Delete alert rule"
            description="This removes the rule from the monitoring configuration."
            onConfirm={() => deleteRule.mutate(row.original.id)}
          />
        </div>
      ),
    },
  ];

  if (rules.isLoading || targets.isLoading) return <LoadingState label="Loading alert rules" />;
  if (rules.isError) return <ErrorState message="Unable to load alert rules" />;

  return (
    <>
      <PageHeader
        title="Alert Rules"
        description="Manage threshold rules and recovery policies. Current backend supports create; update/delete actions are gracefully handled when endpoints are unavailable."
        actions={<AlertRuleForm trigger={<Button><Plus data-icon="inline-start" />Create rule</Button>} targets={targets.data ?? []} onSubmit={(input) => createRule.mutate(input)} />}
      />
      <DataTable columns={columns} data={rules.data ?? []} />
    </>
  );
}
