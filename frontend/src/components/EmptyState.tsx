import { Inbox } from "lucide-react";

export function EmptyState({ title = "No data", description = "Nothing has been recorded yet." }: { title?: string; description?: string }) {
  return (
    <div className="flex min-h-44 flex-col items-center justify-center gap-2 rounded-lg border bg-card px-4 text-center">
      <Inbox className="text-muted-foreground" />
      <h3 className="text-sm font-medium">{title}</h3>
      <p className="max-w-md text-sm text-muted-foreground">{description}</p>
    </div>
  );
}
