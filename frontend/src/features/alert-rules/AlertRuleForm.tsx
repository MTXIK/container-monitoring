import { useEffect, useState } from "react";
import { Dialog, DialogContent, DialogDescription, DialogHeader, DialogTitle, DialogTrigger } from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import type { AlertRule, AlertRuleInput } from "@/types/alert-rule";
import type { Target } from "@/types/target";

const defaultForm: AlertRuleInput = {
  name: "",
  target_id: undefined,
  metric_name: "cpu_usage_percent",
  operator: ">",
  threshold: 80,
  duration: "2m",
  severity: "warning",
  recovery_policy: "notify_only",
  enabled: true,
};

export function AlertRuleForm({ trigger, rule, targets, onSubmit }: { trigger: React.ReactNode; rule?: AlertRule; targets: Target[]; onSubmit: (input: AlertRuleInput) => void }) {
  const [open, setOpen] = useState(false);
  const [form, setForm] = useState<AlertRuleInput>(rule ?? defaultForm);

  useEffect(() => setForm(rule ?? defaultForm), [rule]);

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogTrigger asChild>{trigger}</DialogTrigger>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>{rule ? "Edit alert rule" : "Create alert rule"}</DialogTitle>
          <DialogDescription>Define a threshold condition and the recovery policy to apply when it fires.</DialogDescription>
        </DialogHeader>
        <form
          className="mt-4 grid gap-4"
          onSubmit={(event) => {
            event.preventDefault();
            onSubmit(form);
            setOpen(false);
          }}
        >
          <Field label="Name"><Input value={form.name} onChange={(event) => setForm({ ...form, name: event.target.value })} required /></Field>
          <Field label="Target">
            <Select value={form.target_id ?? "all"} onValueChange={(value) => setForm({ ...form, target_id: value === "all" ? undefined : value })}>
              <SelectTrigger><SelectValue placeholder="Target" /></SelectTrigger>
              <SelectContent>
                <SelectItem value="all">Any target</SelectItem>
                {targets.map((target) => <SelectItem key={target.id} value={target.id}>{target.name}</SelectItem>)}
              </SelectContent>
            </Select>
          </Field>
          <div className="grid gap-4 md:grid-cols-3">
            <Field label="Metric/Event"><Input value={form.metric_name} onChange={(event) => setForm({ ...form, metric_name: event.target.value })} required /></Field>
            <Field label="Operator">
              <Select value={form.operator} onValueChange={(value) => setForm({ ...form, operator: value as AlertRuleInput["operator"] })}>
                <SelectTrigger><SelectValue /></SelectTrigger>
                <SelectContent>{[">", "<", ">=", "<=", "=="].map((item) => <SelectItem key={item} value={item}>{item}</SelectItem>)}</SelectContent>
              </Select>
            </Field>
            <Field label="Threshold"><Input type="number" value={form.threshold} onChange={(event) => setForm({ ...form, threshold: Number(event.target.value) })} /></Field>
          </div>
          <div className="grid gap-4 md:grid-cols-3">
            <Field label="Duration"><Input value={form.duration} onChange={(event) => setForm({ ...form, duration: event.target.value })} /></Field>
            <Field label="Severity">
              <Select value={form.severity} onValueChange={(value) => setForm({ ...form, severity: value as AlertRuleInput["severity"] })}>
                <SelectTrigger><SelectValue /></SelectTrigger>
                <SelectContent>{["info", "warning", "critical"].map((item) => <SelectItem key={item} value={item}>{item}</SelectItem>)}</SelectContent>
              </Select>
            </Field>
            <Field label="Recovery Policy">
              <Select value={form.recovery_policy} onValueChange={(value) => setForm({ ...form, recovery_policy: value as AlertRuleInput["recovery_policy"] })}>
                <SelectTrigger><SelectValue /></SelectTrigger>
                <SelectContent>{["notify_only", "retry_check", "restart_container"].map((item) => <SelectItem key={item} value={item}>{item}</SelectItem>)}</SelectContent>
              </Select>
            </Field>
          </div>
          <label className="flex items-center gap-2 text-sm">
            <input className="size-4 accent-blue-500" type="checkbox" checked={form.enabled} onChange={(event) => setForm({ ...form, enabled: event.target.checked })} />
            Enabled
          </label>
          <div className="flex justify-end gap-2">
            <Button type="button" variant="outline" onClick={() => setOpen(false)}>Cancel</Button>
            <Button type="submit">{rule ? "Save changes" : "Create rule"}</Button>
          </div>
        </form>
      </DialogContent>
    </Dialog>
  );
}

function Field({ label, children }: { label: string; children: React.ReactNode }) {
  return <div className="flex flex-col gap-2"><Label>{label}</Label>{children}</div>;
}
