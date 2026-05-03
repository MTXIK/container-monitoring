import { Badge } from "@/components/ui/badge";
import { cn } from "@/lib/utils";

const styles: Record<string, string> = {
  open: "border-red-500/35 bg-red-500/12 text-red-300",
  acknowledged: "border-amber-500/30 bg-amber-500/12 text-amber-300",
  recovering: "border-indigo-500/35 bg-indigo-500/12 text-indigo-300",
  resolved: "border-emerald-500/30 bg-emerald-500/12 text-emerald-300",
  failed: "border-red-500/35 bg-red-500/12 text-red-300",
  pending: "border-slate-500/30 bg-slate-500/12 text-slate-300",
  running: "border-sky-500/30 bg-sky-500/12 text-sky-300",
  success: "border-emerald-500/30 bg-emerald-500/12 text-emerald-300",
  succeeded: "border-emerald-500/30 bg-emerald-500/12 text-emerald-300",
  skipped: "border-slate-500/30 bg-slate-500/12 text-slate-300",
};

export function StateBadge({ state, className }: { state: string; className?: string }) {
  return <Badge className={cn(styles[state] ?? styles.pending, className)} variant="outline">{state}</Badge>;
}
