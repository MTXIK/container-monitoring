import { Navigate, createBrowserRouter } from "react-router-dom";
import { AppShell } from "@/components/layout/AppShell";
import { AlertRulesPage } from "@/features/alert-rules/AlertRulesPage";
import { DashboardPage } from "@/features/dashboard/DashboardPage";
import { EventsPage } from "@/features/events/EventsPage";
import { IncidentDetailsPage } from "@/features/incidents/IncidentDetailsPage";
import { IncidentsPage } from "@/features/incidents/IncidentsPage";
import { MetricsPage } from "@/features/metrics/MetricsPage";
import { RecoveryActionsPage } from "@/features/recovery-actions/RecoveryActionsPage";
import { TargetDetailsPage } from "@/features/targets/TargetDetailsPage";
import { TargetsPage } from "@/features/targets/TargetsPage";

export const router = createBrowserRouter([
  {
    path: "/",
    element: <AppShell />,
    children: [
      { index: true, element: <Navigate to="/dashboard" replace /> },
      { path: "dashboard", element: <DashboardPage /> },
      { path: "targets", element: <TargetsPage /> },
      { path: "targets/:id", element: <TargetDetailsPage /> },
      { path: "alert-rules", element: <AlertRulesPage /> },
      { path: "incidents", element: <IncidentsPage /> },
      { path: "incidents/:id", element: <IncidentDetailsPage /> },
      { path: "recovery-actions", element: <RecoveryActionsPage /> },
      { path: "events", element: <EventsPage /> },
      { path: "metrics", element: <MetricsPage /> },
    ],
  },
]);
