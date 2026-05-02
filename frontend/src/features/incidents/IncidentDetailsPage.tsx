import { useParams } from "react-router-dom";
import { ErrorState } from "@/components/ErrorState";
import { LoadingState } from "@/components/LoadingState";
import { PageHeader } from "@/components/PageHeader";
import { SeverityBadge } from "@/components/status/SeverityBadge";
import { StateBadge } from "@/components/status/StateBadge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import { formatDateTime } from "@/lib/formatters";
import { useAcknowledgeIncident, useEvents, useIncident, useRecoveryActions, useResolveIncident, useRetryRecoveryAction, useTargets } from "@/lib/api/hooks";

export function IncidentDetailsPage() {
  const { id = "" } = useParams();
  const incident = useIncident(id);
  const targets = useTargets();
  const events = useEvents();
  const recovery = useRecoveryActions();
  const ack = useAcknowledgeIncident();
  const resolve = useResolveIncident();
  const retry = useRetryRecoveryAction();

  if (incident.isLoading || targets.isLoading) return <LoadingState label="Loading incident details" />;
  if (incident.isError || !incident.data) return <ErrorState message="Incident not found" />;

  const target = targets.data?.find((item) => item.id === incident.data?.target_id);
  const relatedEvents = (events.data ?? []).filter((item) => item.target_id === incident.data?.target_id).slice(0, 8);
  const relatedRecovery = (recovery.data ?? []).filter((item) => item.incident_id === id);
  const failedRecovery = relatedRecovery.find((item) => item.status === "failed");

  return (
    <>
      <PageHeader
        title={`Incident #${incident.data.id}`}
        description={incident.data.description ?? "No description recorded."}
        actions={
          <>
            <Button variant="secondary" onClick={() => ack.mutate(id)} disabled={incident.data.status !== "open"}>Acknowledge</Button>
            <Button variant="secondary" onClick={() => resolve.mutate(id)} disabled={incident.data.status === "resolved"}>Resolve</Button>
            {failedRecovery ? <Button onClick={() => retry.mutate(failedRecovery.id)}>Retry recovery</Button> : null}
          </>
        }
      />
      <section className="grid gap-4 xl:grid-cols-2">
        <Card>
          <CardHeader><CardTitle>Main Information</CardTitle></CardHeader>
          <CardContent className="grid gap-3 text-sm">
            <Info label="Target" value={target?.name ?? incident.data.target_id} />
            <Info label="Rule" value={incident.data.rule_id ?? "—"} />
            <Info label="Started" value={formatDateTime(incident.data.started_at)} />
            <Info label="Resolved" value={formatDateTime(incident.data.resolved_at)} />
            <div className="flex justify-between"><span className="text-muted-foreground">Status</span><StateBadge state={incident.data.status} /></div>
            <div className="flex justify-between"><span className="text-muted-foreground">Severity</span><SeverityBadge severity={incident.data.severity} /></div>
          </CardContent>
        </Card>
        <Card>
          <CardHeader><CardTitle>Recovery Actions</CardTitle></CardHeader>
          <CardContent>
            <Table>
              <TableHeader><TableRow><TableHead>ID</TableHead><TableHead>Type</TableHead><TableHead>Status</TableHead><TableHead>Result</TableHead></TableRow></TableHeader>
              <TableBody>{relatedRecovery.map((item) => <TableRow key={item.id}><TableCell>{item.id}</TableCell><TableCell>{item.action_type}</TableCell><TableCell><StateBadge state={item.status} /></TableCell><TableCell>{item.result_message ?? "—"}</TableCell></TableRow>)}</TableBody>
            </Table>
          </CardContent>
        </Card>
      </section>
      <Card>
        <CardHeader><CardTitle>Timeline</CardTitle></CardHeader>
        <CardContent>
          <Table>
            <TableHeader><TableRow><TableHead>Time</TableHead><TableHead>Event Type</TableHead><TableHead>Severity</TableHead><TableHead>Message</TableHead></TableRow></TableHeader>
            <TableBody>{relatedEvents.map((item) => <TableRow key={item.id}><TableCell>{formatDateTime(item.timestamp)}</TableCell><TableCell>{item.event_type}</TableCell><TableCell><SeverityBadge severity={item.severity} /></TableCell><TableCell>{item.message}</TableCell></TableRow>)}</TableBody>
          </Table>
        </CardContent>
      </Card>
    </>
  );
}

function Info({ label, value }: { label: string; value: string }) {
  return <div className="flex justify-between gap-4"><span className="text-muted-foreground">{label}</span><span className="text-right font-mono">{value}</span></div>;
}
