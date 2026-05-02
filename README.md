# container-monitoring

MVP platform for monitoring Docker containers with Kafka-based telemetry ingest,
ClickHouse metric storage, PostgreSQL configuration and incident storage, Redis
runtime state, Telegram alerts, self-healing actions, HTTP API, Swagger, and
Grafana dashboards.

Docker is implemented as the first collector backend. The core domain keeps
Docker-specific details isolated so additional backends, such as Kubernetes, can
be added later.

## Architecture

- `agent/` - node-side Go agent. It reads Docker stats and events through
  `/var/run/docker.sock`, normalizes telemetry, and publishes JSON messages to
  Kafka.
- `core/` - central Go service. It consumes Kafka telemetry, writes metrics to
  ClickHouse, writes configuration/events/incidents/recovery actions to
  PostgreSQL, updates Redis state, evaluates alert rules, sends Telegram
  notifications, runs self-healing actions, and exposes the Fiber HTTP API.
- `contracts/` - proto contracts reserved for future gRPC/internal service
  boundaries.
- `core/deploy/` - storage, Grafana provisioning, and dashboards.
- `docs/` - architecture notes and demo scenarios.

## Local Run

Start the full local MVP stack:

```bash
docker compose up
```

This starts:

- Kafka
- PostgreSQL
- ClickHouse
- Redis
- Grafana
- Core API
- Agent
- demo `target-nginx`

Local URLs:

- Core API: `http://localhost:8080`
- Swagger UI: `http://localhost:8080/swagger/`
- Grafana: `http://localhost:3000` (`admin` / `admin`)
- Demo nginx target: `http://localhost:8081`

Stop the stack:

```bash
docker compose down
```

## Configuration

Important environment variables:

- `AGENT_NODE_ID` - logical node id for the agent.
- `DOCKER_HOST` - Docker API endpoint for the agent, default
  `unix:///var/run/docker.sock`.
- `COLLECT_INTERVAL` - Docker stats collection interval.
- `KAFKA_BROKERS` - Kafka broker list.
- `POSTGRES_DSN` - PostgreSQL DSN for core.
- `CLICKHOUSE_DSN` - ClickHouse HTTP endpoint for core.
- `REDIS_ADDR` - Redis address for latest state and locks.
- `RECOVERY_DOCKER_HOST` - Docker API endpoint used by core for
  `restart_container`.
- `TELEGRAM_BOT_TOKEN` and `TELEGRAM_CHAT_ID` - optional Telegram alerts.

Example with Telegram:

```bash
TELEGRAM_BOT_TOKEN="<bot-token>" TELEGRAM_CHAT_ID="<chat-id>" docker compose up
```

## HTTP API

Health:

- `GET /health`
- `GET /ready`

Targets:

- `GET /api/v1/targets`
- `GET /api/v1/targets/:id`

Metrics from ClickHouse:

- `GET /api/v1/metrics/latest`
- `GET /api/v1/metrics/history`
- `GET /api/v1/targets/:id/metrics`

Events from PostgreSQL:

- `GET /api/v1/events`
- `GET /api/v1/targets/:id/events`

Alert rules:

- `GET /api/v1/alert-rules`
- `POST /api/v1/alert-rules`

Incidents:

- `GET /api/v1/incidents`
- `POST /api/v1/incidents/:id/ack`
- `POST /api/v1/incidents/:id/resolve`

Recovery:

- `GET /api/v1/recovery-actions`
- `POST /api/v1/recovery-actions/:id/retry`

Open Swagger UI at `http://localhost:8080/swagger/`.

Regenerate Swagger after API annotation changes:

```bash
make swagger
```

## Grafana

Grafana is provisioned with a ClickHouse datasource and the
`Container Monitoring MVP` dashboard.

Dashboard panels:

- CPU usage by container.
- Memory usage by container.
- Container events over time.
- Critical event count.
- Latest container events table.

Grafana reads metrics and events directly from ClickHouse.

## Demo Recovery Scenario

The repository includes a defense-oriented scenario:

```bash
docs/demo-recovery-scenario.md
```

Short version:

1. Start the stack with `docker compose up`.
2. Open Swagger and Grafana.
3. Stop the demo target:

   ```bash
   docker compose stop target-nginx
   ```

4. Agent catches the Docker event.
5. Core stores the event and creates an incident for `container_stopped`.
6. Telegram alert is sent if Telegram env vars are configured.
7. Self-healing writes a recovery action and calls Docker restart through
   `RECOVERY_DOCKER_HOST`.
8. API and Grafana show the event, incident, recovery action, and metrics.

Restart the target if needed:

```bash
docker compose start target-nginx
```

## Development Commands

```bash
make test
make build
make swagger
make core-up
make core-down
```

`make core-up` and `make core-down` use the older infrastructure-only compose
file under `core/deploy/`. For the full local MVP, prefer root-level
`docker compose up`.

## Data Flow

1. Agent connects to Docker Engine API.
2. Agent collects stats and watches Docker events.
3. Agent publishes:
   - metrics to `container.metrics`;
   - events to `container.events`.
4. Core consumes Kafka messages.
5. Core writes metrics/events to ClickHouse.
6. Core writes targets, events, incidents, alert rules, and recovery actions to
   PostgreSQL.
7. Core updates latest metrics/state and recovery locks in Redis.
8. Analyzer applies threshold rules.
9. Event failures such as `container_stopped`, `container_died`, and
   `container_oom` create incidents and request recovery.
10. Telegram notifier sends alerts when configured.
11. Grafana visualizes ClickHouse data.

## MVP Limitations

- Docker is the only implemented collector backend.
- Event-to-incident behavior is intentionally simple and should become
  configurable through event alert rules.
- Recovery actions are limited to `notify_only`, `retry_check`, and
  `restart_container`.
- This is not a Prometheus replacement and does not implement Kubernetes yet.
