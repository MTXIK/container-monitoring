# container-monitoring-core

Central receiving and control service for the Container Monitoring MVP.

Core consumes telemetry from Kafka, stores operational data, evaluates alert
rules, creates incidents, sends optional Telegram notifications, runs recovery
actions, and exposes the HTTP API used by the frontend, Swagger, e2e tests, and
manual demos.

## Responsibilities

- Consume normalized container metrics and lifecycle events from Kafka.
- Store metric history in ClickHouse.
- Store targets, alert rules, events, incidents, and recovery actions in
  PostgreSQL.
- Store latest target state, alert duration state, and recovery locks in Redis.
- Evaluate threshold alert rules:
  - target-scoped or global rules;
  - operators `gt`, `gte`, `lt`, `lte`, and `eq`;
  - duration windows;
  - deduplication for open incidents.
- Create event-driven incidents for Docker stop, die, and OOM events.
- Execute recovery policies:
  - `notify_only`;
  - `retry_check`;
  - `restart_container`.
- Send optional Telegram notifications for incidents.
- Expose HTTP endpoints for health, targets, metrics, events, alert rules,
  incidents, and recovery actions.

## Runtime Flow

1. `cmd/core/main.go` loads configuration from environment variables.
2. Core connects to PostgreSQL, ClickHouse, and Redis.
3. The repository is created over PostgreSQL and ClickHouse stores.
4. The recovery coordinator is created with Redis locks and Docker executor.
5. Kafka consumer starts reading metric and event topics.
6. HTTP API starts on `HTTP_ADDR`.
7. On `SIGINT` or `SIGTERM`, Fiber shuts down gracefully.

Metric ingest flow:

```text
Kafka metric message
  -> decode JSON
  -> upsert target in PostgreSQL
  -> write metric to ClickHouse
  -> store latest metric in Redis
  -> evaluate enabled alert rules
  -> create incident if threshold matches and duration/dedup conditions pass
  -> notify and recover if configured
```

Event ingest flow:

```text
Kafka event message
  -> decode JSON
  -> upsert target status in PostgreSQL
  -> write event to PostgreSQL and ClickHouse
  -> store latest event state in Redis
  -> create event-driven incident for stop/die/OOM
  -> notify and run recovery action
```

## Configuration

| Variable | Default | Description |
| --- | --- | --- |
| `HTTP_ADDR` | `:8080` | Fiber listen address. |
| `KAFKA_BROKERS` | `localhost:9092` | Comma-separated Kafka broker list. |
| `KAFKA_METRICS_TOPIC` | `container.metrics` | Topic consumed for metric samples. |
| `KAFKA_EVENTS_TOPIC` | `container.events` | Topic consumed for lifecycle events. |
| `KAFKA_GROUP_ID` | `container-monitoring-core` | Kafka consumer group id. |
| `POSTGRES_DSN` | local container-monitoring DSN | PostgreSQL connection string. |
| `CLICKHOUSE_DSN` | `http://localhost:8123` | ClickHouse HTTP endpoint. |
| `REDIS_ADDR` | `localhost:6379` | Redis address for latest state and locks. |
| `RECOVERY_DOCKER_HOST` | empty | Docker API endpoint used by `restart_container`. |
| `TELEGRAM_BOT_TOKEN` | empty | Optional Telegram bot token. |
| `TELEGRAM_CHAT_ID` | empty | Optional Telegram chat id. |

In root Compose, `RECOVERY_DOCKER_HOST` is set to
`unix:///var/run/docker.sock`, and the Docker socket is mounted into the core
container.

## Local Development

Create a local environment file when running core outside Compose:

```bash
cp .env.example .env
```

Then either export the variables from `.env` in your shell or pass them
directly to `go run`. The example file is configured for local infrastructure on
`localhost`.

Run tests:

```bash
go test ./...
```

Verify compilation:

```bash
go build ./...
```

Run local infrastructure owned by this service:

```bash
docker compose -f deploy/docker-compose.yml up -d
```

Run core directly against local infrastructure:

```bash
HTTP_ADDR=:8080 \
KAFKA_BROKERS=localhost:9092 \
POSTGRES_DSN='postgres://container_monitoring:container_monitoring@localhost:5432/container_monitoring?sslmode=disable' \
CLICKHOUSE_DSN=http://localhost:8123 \
REDIS_ADDR=localhost:6379 \
RECOVERY_DOCKER_HOST=unix:///var/run/docker.sock \
go run ./cmd/core
```

For the complete MVP, prefer the root stack:

```bash
cd ..
docker compose up --build
```

For the richer demo stack:

```bash
cd ..
docker compose -f docker-compose.yml -f docker-compose.demo-targets.yml up --build
```

## HTTP API

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

Metrics:

- `GET /api/v1/metrics/latest`
- `GET /api/v1/metrics/history`

Events:

- `GET /api/v1/events`

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

Recovery:

- `GET /api/v1/recovery-actions`
- `POST /api/v1/recovery-actions/:id/retry`

Swagger UI is available at:

```text
http://localhost:8080/swagger/
```

Regenerate Swagger docs after API annotation changes:

```bash
make swagger
```

## Alert Rules

Alert rules are stored in PostgreSQL and evaluated on incoming metric samples.

Example request:

```bash
curl -sS -X POST http://localhost:8080/api/v1/alert-rules \
  -H 'Content-Type: application/json' \
  -d '{
    "name": "High CPU",
    "target_id": "container-id",
    "metric_name": "cpu_usage_percent",
    "operator": ">=",
    "threshold": 80,
    "duration": "30s",
    "severity": "critical",
    "recovery_policy": "notify_only",
    "enabled": true
  }'
```

Business behavior:

- Empty `target_id` means the rule is global.
- Non-empty `target_id` scopes the rule to one container.
- API operators are normalized to backend values:
  - `>` -> `gt`;
  - `>=` -> `gte`;
  - `<` -> `lt`;
  - `<=` -> `lte`;
  - `==` -> `eq`.
- `duration` requires the threshold to remain true for the configured window.
- Existing open or acknowledged incidents deduplicate repeated matches for the
  same `rule_id + target_id`.
- Once an incident is resolved, a later matching metric can open a new incident.

## Recovery Actions

Recovery actions are coordinated through `internal/recovery`.

Statuses:

- `running` - action has been created and is executing.
- `succeeded` - executor completed successfully.
- `failed` - executor or lock operation failed.
- `skipped` - action was not executed because a recovery lock was already held.

Policies:

- `notify_only` records a successful no-op recovery action.
- `retry_check` records a successful lightweight check.
- `restart_container` calls Docker Engine API through `RECOVERY_DOCKER_HOST`.

The retry endpoint loads the failed action, loads its incident, and dispatches a
new recovery attempt through the same coordinator.

## Storage Ownership

PostgreSQL stores:

- `nodes`;
- `containers`;
- `alert_rules`;
- `events`;
- `incidents`;
- `recovery_actions`.

ClickHouse stores:

- `container_metrics`;
- `container_events`.

Redis stores:

- latest target metric snapshots;
- latest lifecycle event state;
- alert duration start timestamps;
- recovery locks.

## Testing Notes

Important packages:

- `internal/analyzer` tests threshold operators, target scoping, and matching.
- `internal/ingest` tests metric/event handling, alert duration, deduplication,
  and target status updates.
- `internal/api/http` tests frontend-facing HTTP API behavior.
- `internal/consumer/kafka` tests consumer behavior when handler errors occur.
- `internal/recovery` tests recovery status and lock behavior.

Useful commands:

```bash
go test ./internal/analyzer
go test ./internal/ingest
go test ./internal/api/http
go test ./internal/recovery
go test ./...
```

Full manual and e2e verification steps live in:

```text
../docs/test-runbook.md
```

## Operational Notes

- Kafka handler errors are logged and the message is committed so one malformed
  message does not stop the core process.
- PostgreSQL schema compatibility is partially handled by startup-time
  `ALTER TABLE ... ADD COLUMN IF NOT EXISTS` statements.
- For a clean local database after schema changes, run the root stack with
  `docker compose down -v` before starting again.
- Telegram is optional. If token or chat id is empty, notification sending is a
  no-op.
