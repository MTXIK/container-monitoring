# Container Monitoring Platform Frontend

React/Vite admin panel for the container monitoring MVP.

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

Mock fallback is enabled by default so the UI stays usable when an MVP endpoint is empty or not implemented yet.

## Commands

```bash
npm run dev
npm run build
npm run preview
```

## Notes

The backend now exposes the admin-panel endpoints used by the frontend, including target CRUD, alert rule CRUD, incident details, incident acknowledge/resolve, recovery retries, latest metrics, metric history, and events.
