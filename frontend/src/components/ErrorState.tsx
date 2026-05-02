import { AlertTriangle } from "lucide-react";

export function ErrorState({ message = "Unable to load data" }: { message?: string }) {
  return (
    <div className="flex min-h-44 flex-col items-center justify-center gap-2 rounded-lg border border-red-500/25 bg-red-500/8 px-4 text-center">
      <AlertTriangle className="text-red-300" />
      <p className="text-sm text-red-200">{message}</p>
    </div>
  );
}
