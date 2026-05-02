import type { ReactNode } from "react";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { cn } from "@/lib/utils";

export function MetricCard({ label, value, tone = "neutral", icon }: { label: string; value: ReactNode; tone?: "neutral" | "ok" | "warn" | "critical" | "info"; icon?: ReactNode }) {
  return (
    <Card className="overflow-hidden">
      <CardHeader className="flex-row items-center justify-between gap-3 pb-2">
        <CardTitle className="text-xs font-medium uppercase text-muted-foreground">{label}</CardTitle>
        {icon ? <div className={cn("text-muted-foreground", tone === "critical" && "text-red-300", tone === "warn" && "text-amber-300", tone === "ok" && "text-emerald-300", tone === "info" && "text-sky-300")}>{icon}</div> : null}
      </CardHeader>
      <CardContent>
        <div className="text-3xl font-semibold tracking-normal">{value}</div>
      </CardContent>
    </Card>
  );
}
