import { Loader2 } from "lucide-react";

export function LoadingState({ label = "Loading data" }: { label?: string }) {
  return (
    <div className="flex min-h-44 items-center justify-center rounded-lg border bg-card text-sm text-muted-foreground">
      <Loader2 className="mr-2 animate-spin" />
      {label}
    </div>
  );
}
