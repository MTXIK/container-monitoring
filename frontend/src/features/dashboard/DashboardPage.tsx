import { AlertTriangle, CheckCircle2, LifeBuoy, Server, ShieldAlert } from "lucide-react";
import { Link } from "react-router-dom";
import { ErrorState } from "@/components/ErrorState";
import { GrafanaLinkButton } from "@/components/GrafanaLinkButton";
import { LoadingState } from "@/components/LoadingState";
import { MetricCard } from "@/components/MetricCard";
import { PageHeader } from "@/components/PageHeader";
import { SeverityBadge } from "@/components/status/SeverityBadge";
import { StateBadge } from "@/components/status/StateBadge";
import { StatusBadge } from "@/components/status/StatusBadge";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import { formatBytes, formatDateTime, formatPercent } from "@/lib/formatters";
import { useEvents, useIncidents, useLatestMetrics, useRecoveryActions, useTargets } from "@/lib/api/hooks";

export function DashboardPage() {
  const targets = useTargets();
  const incidents = useIncidents();
  const events = useEvents();
  const recovery = useRecoveryActions();
  const metrics = useLatestMetrics();

  const isLoading = [targets, incidents, events, recovery, metrics].some((query) => query.isLoading);
  const isError = [targets, incidents, events, recovery, metrics].some((query) => query.isError);

  if (isLoading) return <LoadingState label="Loading platform summary" />;
  if (isError) return <ErrorState message="Unable to load dashboard data" />;

  const targetList = targets.data ?? [];
  const incidentList = incidents.data ?? [];
  const recoveryList = recovery.data ?? [];
  const metricList = metrics.data ?? [];
  const today = new Date().toDateString();
  const openIncidents = incidentList.filter((item) => ["open", "acknowledged", "recovering", "failed"].includes(item.status));

  return (
    <>
      <PageHeader
        title="Dashboard"
        description="Operational overview for monitored containers, active incidents, latest events, and recovery activity."
        actions={<GrafanaLinkButton />}
      />
      <section className="grid gap-4 sm:grid-cols-2 xl:grid-cols-6">
        <MetricCard label="Total Targets" value={targetList.length} icon={<Server />} />
        <MetricCard label="Healthy" value={targetList.filter((item) => item.status === "OK").length} tone="ok" icon={<CheckCircle2 />} />
        <MetricCard label="Warning" value={targetList.filter((item) => item.status === "WARNING").length} tone="warn" icon={<AlertTriangle />} />
        <MetricCard label="Critical" value={targetList.filter((item) => item.status === "CRITICAL").length} tone="critical" icon={<ShieldAlert />} />
        <MetricCard label="Open Incidents" value={openIncidents.length} tone={openIncidents.length ? "critical" : "neutral"} icon={<ShieldAlert />} />
        <MetricCard label="Recovery Today" value={recoveryList.filter((item) => new Date(item.started_at).toDateString() === today).length} tone="info" icon={<LifeBuoy />} />
      </section>
      <section className="grid gap-4 xl:grid-cols-2">
        <Card>
          <CardHeader><CardTitle>Recent Incidents</CardTitle></CardHeader>
          <CardContent>
            <Table>
              <TableHeader><TableRow><TableHead>ID</TableHead><TableHead>Severity</TableHead><TableHead>Status</TableHead><TableHead>Started</TableHead></TableRow></TableHeader>
              <TableBody>
                {incidentList.slice(0, 6).map((item) => (
                  <TableRow key={item.id}>
                    <TableCell><Link className="font-medium text-primary" to={`/incidents/${item.id}`}>#{item.id}</Link></TableCell>
                    <TableCell><SeverityBadge severity={item.severity} /></TableCell>
                    <TableCell><StateBadge state={item.status} /></TableCell>
                    <TableCell>{formatDateTime(item.started_at)}</TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </CardContent>
        </Card>
        <Card>
          <CardHeader><CardTitle>Recent Events</CardTitle></CardHeader>
          <CardContent>
            <Table>
              <TableHeader><TableRow><TableHead>Time</TableHead><TableHead>Target</TableHead><TableHead>Type</TableHead><TableHead>Severity</TableHead></TableRow></TableHeader>
              <TableBody>
                {(events.data ?? []).slice(0, 6).map((item) => (
                  <TableRow key={item.id}>
                    <TableCell>{formatDateTime(item.timestamp)}</TableCell>
                    <TableCell>{item.container_name ?? item.target_id}</TableCell>
                    <TableCell>{item.event_type}</TableCell>
                    <TableCell><SeverityBadge severity={item.severity} /></TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </CardContent>
        </Card>
      </section>
      <section className="grid gap-4 xl:grid-cols-2">
        <Card>
          <CardHeader><CardTitle>Top CPU Containers</CardTitle></CardHeader>
          <CardContent>
            <div className="flex flex-col gap-3">
              {[...metricList].sort((a, b) => b.cpu_usage_percent - a.cpu_usage_percent).slice(0, 5).map((item) => (
                <div className="flex items-center justify-between rounded-md border bg-background/60 px-3 py-2" key={item.target_id}>
                  <div className="flex flex-col"><span className="text-sm font-medium">{item.container_name}</span><span className="text-xs text-muted-foreground">{item.node_id}</span></div>
                  <span className="font-mono text-sm">{formatPercent(item.cpu_usage_percent)}</span>
                </div>
              ))}
            </div>
          </CardContent>
        </Card>
        <Card>
          <CardHeader><CardTitle>Top Memory Containers</CardTitle></CardHeader>
          <CardContent>
            <div className="flex flex-col gap-3">
              {[...metricList].sort((a, b) => b.memory_usage_bytes - a.memory_usage_bytes).slice(0, 5).map((item) => {
                const target = targetList.find((candidate) => candidate.id === item.target_id);
                return (
                  <div className="flex items-center justify-between rounded-md border bg-background/60 px-3 py-2" key={item.target_id}>
                    <div className="flex items-center gap-3"><StatusBadge status={target?.status ?? "UNKNOWN"} /><span className="text-sm font-medium">{item.container_name}</span></div>
                    <span className="font-mono text-sm">{formatBytes(item.memory_usage_bytes)}</span>
                  </div>
                );
              })}
            </div>
          </CardContent>
        </Card>
      </section>
    </>
  );
}
