# Frontend Admin Panel

`frontend/` contains the React admin panel for Container Monitoring Platform.
It is not a Grafana replacement. Grafana remains the detailed time-series view,
while the frontend is the operational control surface for targets, alert rules,
incidents, recovery actions, events, and latest metrics.

## Stack

- React + Vite + TypeScript
- Tailwind CSS
- shadcn/ui-style local components
- TanStack Query for API state
- TanStack Table for tabular views
- React Router for routes

The app is a client-side SPA. The backend API base URL is configured through
`VITE_API_BASE_URL`, defaulting to `http://localhost:8080`.

## Local Run

Run frontend only:

```bash
cd frontend
npm install
npm run dev
```

Run the full MVP stack:

```bash
docker compose up
```

Local URLs:

- Frontend: `http://localhost:5173`
- Core API: `http://localhost:8080`
- Swagger: `http://localhost:8080/swagger/`
- Grafana: `http://localhost:3000` (`admin` / `admin`)

## Configuration

Create `frontend/.env` for local overrides:

```bash
VITE_API_BASE_URL=http://localhost:8080
VITE_GRAFANA_URL=http://localhost:3000
VITE_ENABLE_MOCK_FALLBACK=true
```

`VITE_ENABLE_MOCK_FALLBACK` is disabled by default. Set it to `true` only for
local demos where mock data is intentional. Mock data lives in
`frontend/src/lib/api/mock-data.ts` and is kept separate from the API client.

## Routes

| Route | Purpose |
| --- | --- |
| `/dashboard` | Summary cards, recent incidents/events, top CPU/memory containers, Grafana link. |
| `/targets` | Filterable target table with status, source, CPU, memory, and details action. |
| `/targets/:id` | Target profile, labels, latest metrics, related events/incidents/recovery actions. |
| `/alert-rules` | Alert rule create/edit/enable/disable/delete UI. |
| `/incidents` | Incident table with acknowledge and resolve actions. |
| `/incidents/:id` | Incident profile, timeline, recovery actions, retry failed recovery. |
| `/recovery-actions` | Recovery action attempts and retry action for failed records. |
| `/events` | Docker event journal with type, severity, target, and time-range filters. |
| `/metrics` | Latest metrics table and lightweight CPU history preview. |

External sidebar links:

- Swagger opens `VITE_API_BASE_URL + /swagger/`
- Grafana opens `VITE_GRAFANA_URL`

## Frontend Architecture

Key folders:

```text
frontend/src/
  app/                 router and providers
  components/          shared layout, states, UI primitives, status badges, tables
  features/            route-level feature pages
  lib/api/             API client, mappers, query hooks, mock fallback data
  lib/formatters/      date, byte, percent, and ID formatting
  lib/utils/           shared utility helpers
  types/               frontend domain types
```

Data access is centralized in `frontend/src/lib/api/client.ts`.
React Query hooks are defined in `frontend/src/lib/api/hooks.ts`.
Backend response normalization is handled by mapper functions in
`frontend/src/lib/api/mappers.ts`, so feature pages do not depend on backend
storage naming details.

## Backend API Used by the Frontend

Health:

- `GET /health`
- `GET /ready`

Targets:

- `GET /api/v1/targets`
- `GET /api/v1/targets/:id`
- `POST /api/v1/targets`
- `PATCH /api/v1/targets/:id`
- `DELETE /api/v1/targets/:id`
- `GET /api/v1/targets/:id/events`
- `GET /api/v1/targets/:id/metrics`

Alert rules:

- `GET /api/v1/alert-rules`
- `POST /api/v1/alert-rules`
- `PATCH /api/v1/alert-rules/:id`
- `DELETE /api/v1/alert-rules/:id`

Incidents:

- `GET /api/v1/incidents`
- `GET /api/v1/incidents/:id`
- `POST /api/v1/incidents/:id/ack`
- `POST /api/v1/incidents/:id/resolve`

Recovery actions:

- `GET /api/v1/recovery-actions`
- `POST /api/v1/recovery-actions/:id/retry`

Events and metrics:

- `GET /api/v1/events`
- `GET /api/v1/metrics/latest`
- `GET /api/v1/metrics/history`

## UI States

Each route should handle:

- loading state with `LoadingState`
- error state with `ErrorState`
- empty state with `EmptyState`
- successful data state

Mutations report results through toast notifications from `sonner`.

## Status and Severity Mapping

Target statuses:

- `OK` - green
- `WARNING` - amber
- `CRITICAL` - red
- `UNKNOWN` - gray
- `RECOVERING` - indigo

Incident and recovery states are rendered with `StateBadge`.
Severity values are rendered with `SeverityBadge`.

Backend ingest updates target status from Docker events:

- `container_started`, `container_restarted` -> `OK`
- `container_stopped`, `container_died`, `container_oom` -> `CRITICAL`

## Demo Scenario

1. Start the stack:

   ```bash
   docker compose up
   ```

2. Open `http://localhost:5173/dashboard`.
3. Open Targets and confirm `target-nginx` appears.
4. Open Alert Rules and create or inspect rules.
5. Stop the demo target:

   ```bash
   docker compose stop target-nginx
   ```

6. Open Incidents and confirm a new incident appears.
7. Open Recovery Actions and confirm the recovery attempt is visible.
8. Open Grafana from the frontend for detailed ClickHouse-backed charts.

## Development Notes

- Keep Grafana as the detailed charting surface. The frontend should only show
  latest metrics and lightweight previews.
- Keep API calls inside `lib/api`; feature pages should use query hooks.
- Add mapper functions when backend JSON differs from frontend types.
- Keep mock data explicit and isolated from production API logic.
- Run before handoff:

  ```bash
  cd frontend
  npm run lint
  npm run build
  ```
