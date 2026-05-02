# container-monitoring-core

Central receiving service.

Responsibilities:

- consume container metrics and events from Kafka;
- store metric history in ClickHouse;
- store configuration, alert rules, incidents, and recovery actions in PostgreSQL;
- store latest operational state and alert locks in Redis;
- evaluate threshold rules;
- send Telegram notifications;
- execute allowed recovery actions;
- expose the HTTP API.

Infrastructure for local development is owned by this service and lives in
`deploy/`.

## Development

```bash
go test ./...
go build ./...
go run ./cmd/core
docker compose -f deploy/docker-compose.yml up -d
```
