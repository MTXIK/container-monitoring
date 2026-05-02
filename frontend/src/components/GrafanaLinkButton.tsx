import { ExternalLink } from "lucide-react";
import { Button } from "@/components/ui/button";
import { GRAFANA_URL } from "@/lib/api/client";

export function GrafanaLinkButton({ targetId, size = "sm" }: { targetId?: string; size?: "sm" | "default" }) {
  const url = targetId ? `${GRAFANA_URL}/?var-target_id=${encodeURIComponent(targetId)}` : GRAFANA_URL;
  return (
    <Button asChild variant="outline" size={size}>
      <a href={url} target="_blank" rel="noreferrer">
        <ExternalLink data-icon="inline-start" />
        Open Grafana
      </a>
    </Button>
  );
}
