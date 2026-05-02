import { Badge } from "@/components/ui/badge";
import { cn } from "@/lib/utils";

const styles = {
  info: "border-sky-500/30 bg-sky-500/12 text-sky-300",
  warning: "border-amber-500/30 bg-amber-500/12 text-amber-300",
  critical: "border-red-500/35 bg-red-500/12 text-red-300",
};

export function SeverityBadge({ severity, className }: { severity: "info" | "warning" | "critical"; className?: string }) {
  return <Badge className={cn(styles[severity], className)} variant="outline">{severity}</Badge>;
}
