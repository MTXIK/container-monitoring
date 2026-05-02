import { Activity, Bell, Gauge, LayoutDashboard, LifeBuoy, ListTree, Settings2, TerminalSquare } from "lucide-react";
import { NavLink, Outlet, useLocation } from "react-router-dom";
import { Toaster } from "sonner";
import { Button } from "@/components/ui/button";
import { GRAFANA_URL, SWAGGER_URL } from "@/lib/api/client";
import { useHealth } from "@/lib/api/hooks";
import { cn } from "@/lib/utils";

const navItems = [
  { label: "Dashboard", href: "/dashboard", icon: LayoutDashboard },
  { label: "Targets", href: "/targets", icon: TerminalSquare },
  { label: "Alert Rules", href: "/alert-rules", icon: Settings2 },
  { label: "Incidents", href: "/incidents", icon: Bell },
  { label: "Recovery Actions", href: "/recovery-actions", icon: LifeBuoy },
  { label: "Events", href: "/events", icon: ListTree },
  { label: "Metrics", href: "/metrics", icon: Activity },
];

const titles: Record<string, string> = {
  "/dashboard": "Dashboard",
  "/targets": "Targets",
  "/alert-rules": "Alert Rules",
  "/incidents": "Incidents",
  "/recovery-actions": "Recovery Actions",
  "/events": "Events",
  "/metrics": "Metrics",
};

export function AppShell() {
  const location = useLocation();
  const health = useHealth();
  const title = titles[`/${location.pathname.split("/")[1]}`] ?? "Container Monitoring Platform";
  const healthy = health.data?.status === "ok" || health.data?.status === "mock";

  return (
    <div className="min-h-screen">
      <aside className="fixed inset-y-0 left-0 hidden w-64 border-r bg-card/80 backdrop-blur xl:block">
        <div className="flex h-16 items-center gap-3 border-b px-5">
          <div className="flex size-9 items-center justify-center rounded-md bg-primary text-sm font-bold text-primary-foreground">CMP</div>
          <div>
            <div className="text-sm font-semibold">Container Monitoring</div>
            <div className="text-xs text-muted-foreground">Platform admin</div>
          </div>
        </div>
        <nav className="flex flex-col gap-1 p-3">
          {navItems.map((item) => (
            <NavLink
              key={item.href}
              to={item.href}
              className={({ isActive }) =>
                cn(
                  "flex items-center gap-3 rounded-md px-3 py-2 text-sm text-muted-foreground transition-colors hover:bg-accent hover:text-foreground",
                  isActive && "bg-accent text-foreground",
                )
              }
            >
              <item.icon />
              {item.label}
            </NavLink>
          ))}
        </nav>
        <div className="absolute bottom-0 left-0 right-0 border-t p-3">
          <div className="grid grid-cols-2 gap-2">
            <Button asChild variant="outline" size="sm">
              <a href={SWAGGER_URL} target="_blank" rel="noreferrer">Swagger</a>
            </Button>
            <Button asChild size="sm">
              <a href={GRAFANA_URL} target="_blank" rel="noreferrer">Grafana</a>
            </Button>
          </div>
        </div>
      </aside>
      <div className="xl:pl-64">
        <header className="sticky top-0 z-10 flex h-16 items-center justify-between border-b bg-background/88 px-4 backdrop-blur md:px-6">
          <div className="flex min-w-0 items-center gap-3">
            <Gauge className="text-primary xl:hidden" />
            <div>
              <div className="truncate text-sm font-medium text-muted-foreground">Container Monitoring Platform</div>
              <div className="truncate text-lg font-semibold">{title}</div>
            </div>
          </div>
          <div className="flex items-center gap-3">
            <div className="hidden items-center gap-2 rounded-md border bg-card px-3 py-2 text-xs text-muted-foreground sm:flex">
              <span className={cn("size-2 rounded-full", healthy ? "bg-emerald-400" : "bg-red-400")} />
              backend {healthy ? "healthy" : "unavailable"}
            </div>
            <Button asChild variant="outline" size="sm">
              <a href={GRAFANA_URL} target="_blank" rel="noreferrer">Open Grafana</a>
            </Button>
          </div>
        </header>
        <main className="mx-auto flex w-full max-w-[1440px] flex-col gap-6 p-4 md:p-6">
          <Outlet />
        </main>
      </div>
      <Toaster theme="dark" richColors />
    </div>
  );
}
