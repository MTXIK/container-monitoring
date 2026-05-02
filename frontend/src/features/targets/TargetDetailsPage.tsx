import { Link, useParams } from "react-router-dom";
import { GrafanaLinkButton } from "@/components/GrafanaLinkButton";
import { ErrorState } from "@/components/ErrorState";
import { LoadingState } from "@/components/LoadingState";
import { PageHeader } from "@/components/PageHeader";
import { SeverityBadge } from "@/components/status/SeverityBadge";
import { StateBadge } from "@/components/status/StateBadge";
import { StatusBadge } from "@/components/status/StatusBadge";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import { formatBytes, formatDateTime, formatPercent } from "@/lib/formatters";
import { useEvents, useIncidents, useLatestMetrics, useRecoveryActions, useTarget } from "@/lib/api/hooks";

export function TargetDetailsPage() {
  const { id = "" } = useParams();
  const target = useTarget(id);
  const metrics = useLatestMetrics();
  const events = useEvents();
  const incidents = useIncidents();
  const recovery = useRecoveryActions();

  if (target.isLoading || metrics.isLoading) return <LoadingState label="Loading target details" />;
  if (target.isError || !target.data) return <ErrorState message="Target not found" />;

  const latest = (metrics.data ?? []).find((item) => item.target_id === id);
  const targetEvents = (events.data ?? []).filter((item) => item.target_id === id).slice(0, 8);
  const targetIncidents = (incidents.data ?? []).filter((item) => item.target_id === id).slice(0, 8);
  const targetRecovery = (recovery.data ?? []).filter((item) => item.target_id === id).slice(0, 8);

  return (
    <>
      <PageHeader title={target.data.name} description={`${target.data.node_id} · ${target.data.source}`} actions={<GrafanaLinkButton targetId={id} />} />
      <section className="grid gap-4 xl:grid-cols-[1fr_1.4fr]">
        <Card>
          <CardHeader><CardTitle>Target Profile</CardTitle></CardHeader>
          <CardContent className="grid gap-3 text-sm">
            {[
              ["ID", target.data.id],
              ["Node", target.data.node_id],
              ["Source", target.data.source],
              ["External ID", target.data.external_id],
              ["Last seen", formatDateTime(target.data.last_seen_at)],
            ].map(([label, value]) => <div className="flex justify-between gap-4" key={label}><span className="text-muted-foreground">{label}</span><span className="text-right font-mono">{value}</span></div>)}
            <div className="flex justify-between gap-4"><span className="text-muted-foreground">Status</span><StatusBadge status={target.data.status} /></div>
          </CardContent>
        </Card>
        <Card>
          <CardHeader><CardTitle>Latest Metrics</CardTitle></CardHeader>
          <CardContent className="grid gap-3 sm:grid-cols-2">
            <Metric label="CPU" value={formatPercent(latest?.cpu_usage_percent)} />
            <Metric label="Memory" value={formatBytes(latest?.memory_usage_bytes)} />
            <Metric label="Network RX" value={formatBytes(latest?.network_rx_bytes)} />
            <Metric label="Network TX" value={formatBytes(latest?.network_tx_bytes)} />
            <Metric label="Block read" value={formatBytes(latest?.block_read_bytes)} />
            <Metric label="Block write" value={formatBytes(latest?.block_write_bytes)} />
          </CardContent>
        </Card>
      </section>
      <Card>
        <CardHeader><CardTitle>Labels</CardTitle></CardHeader>
        <CardContent className="flex flex-wrap gap-2">
          {Object.entries(target.data.labels ?? {}).map(([key, value]) => <span className="rounded-md border bg-background px-2 py-1 font-mono text-xs" key={key}>{key}={value}</span>)}
          {Object.keys(target.data.labels ?? {}).length === 0 ? <span className="text-sm text-muted-foreground">No labels recorded.</span> : null}
        </CardContent>
      </Card>
      <section className="grid gap-4 xl:grid-cols-3">
        <RelatedTable title="Recent Events" headers={["Time", "Type", "Severity"]} rows={targetEvents.map((item) => [formatDateTime(item.timestamp), item.event_type, <SeverityBadge severity={item.severity} />])} />
        <RelatedTable title="Related Incidents" headers={["ID", "Status", "Severity"]} rows={targetIncidents.map((item) => [<Link className="text-primary" to={`/incidents/${item.id}`}>#{item.id}</Link>, <StateBadge state={item.status} />, <SeverityBadge severity={item.severity} />])} />
        <RelatedTable title="Recovery Actions" headers={["ID", "Action", "Status"]} rows={targetRecovery.map((item) => [item.id, item.action_type, <StateBadge state={item.status} />])} />
      </section>
    </>
  );
}

function Metric({ label, value }: { label: string; value: string }) {
  return <div className="rounded-md border bg-background/60 p-3"><div className="text-xs uppercase text-muted-foreground">{label}</div><div className="mt-1 font-mono text-lg">{value}</div></div>;
}

function RelatedTable({ title, headers, rows }: { title: string; headers: string[]; rows: React.ReactNode[][] }) {
  return (
    <Card>
      <CardHeader><CardTitle>{title}</CardTitle></CardHeader>
      <CardContent>
        <Table>
          <TableHeader><TableRow>{headers.map((item) => <TableHead key={item}>{item}</TableHead>)}</TableRow></TableHeader>
          <TableBody>{rows.map((row, index) => <TableRow key={index}>{row.map((cell, cellIndex) => <TableCell key={cellIndex}>{cell}</TableCell>)}</TableRow>)}</TableBody>
        </Table>
      </CardContent>
    </Card>
  );
}
