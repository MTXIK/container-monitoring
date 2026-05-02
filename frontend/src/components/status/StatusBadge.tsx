import { Badge } from "@/components/ui/badge";
import { cn } from "@/lib/utils";
import type { TargetStatus } from "@/types/target";

const styles: Record<TargetStatus, string> = {
  OK: "border-emerald-500/30 bg-emerald-500/12 text-emerald-300",
  WARNING: "border-amber-500/30 bg-amber-500/12 text-amber-300",
  CRITICAL: "border-red-500/35 bg-red-500/12 text-red-300",
  UNKNOWN: "border-slate-500/30 bg-slate-500/12 text-slate-300",
  RECOVERING: "border-indigo-500/35 bg-indigo-500/12 text-indigo-300",
};

export function StatusBadge({ status, className }: { status: TargetStatus; className?: string }) {
  return <Badge className={cn(styles[status], className)} variant="outline">{status}</Badge>;
}
