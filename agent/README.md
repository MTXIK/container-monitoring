# container-monitoring-agent

Node-side agent service.

Responsibilities:

- collect container metrics and lifecycle events from collector backends;
- provide Docker as the first collector backend;
- publish normalized telemetry to Kafka topics owned by the platform contract;
- avoid dependencies on PostgreSQL, ClickHouse, Redis, Grafana, Telegram, or the
  core service internals.

## Development

```bash
go test ./...
go build ./...
go run ./cmd/agent
```

## Configuration

Copy `.env.example` to `.env` for local runs.
