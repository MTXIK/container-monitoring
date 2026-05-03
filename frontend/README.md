# Container Monitoring Platform Frontend

React/Vite admin panel for the container monitoring MVP.

This frontend is an administrative panel, not a Grafana replacement. Grafana is
still used for detailed ClickHouse-backed time-series dashboards. The frontend
is used for operational workflows: targets, alert rules, incidents, recovery
actions, events, latest metrics, and links into Grafana.

Full documentation: [`../docs/frontend.md`](../docs/frontend.md)

## Local Development

```bash
cd frontend
npm install
npm run dev
```

Default URLs:

- Frontend: `http://localhost:5173`
- Core API: `http://localhost:8080`
- Grafana: `http://localhost:3000`

## Configuration

Create a local `.env` if you need non-default endpoints:

```bash
VITE_API_BASE_URL=http://localhost:8080
VITE_GRAFANA_URL=http://localhost:3000
VITE_ENABLE_MOCK_FALLBACK=true
```

Mock fallback is disabled by default. Set `VITE_ENABLE_MOCK_FALLBACK=true` only for local demos where mock data is intentional.

## Routes

- `/dashboard` - platform summary, recent incidents/events, top resource users.
- `/targets` - target table with status/source/search filters.
- `/targets/:id` - target profile, labels, latest metrics, related activity.
- `/alert-rules` - alert rule CRUD.
- `/incidents` - incident table with acknowledge/resolve actions.
- `/incidents/:id` - incident details, timeline, recovery actions.
- `/recovery-actions` - self-healing action log and retry.
- `/events` - Docker event journal.
- `/metrics` - latest metrics table and Grafana link.

## Commands

```bash
npm run dev
npm run build
npm run preview
```

## Notes

The backend now exposes the admin-panel endpoints used by the frontend, including target CRUD, alert rule CRUD, incident details, incident acknowledge/resolve, recovery retries, latest metrics, metric history, and events.
