# Demo Recovery Scenario

This scenario demonstrates the MVP flow for a Docker container failure:

1. `target-nginx` is running locally.
2. The agent watches Docker events and publishes `container_stopped` / `container_died` to Kafka.
3. Core consumes the event, stores it in PostgreSQL and ClickHouse, updates Redis state, and exposes it via HTTP API.
4. Stop, die, and OOM events create incidents, send Telegram alerts, and trigger recovery actions.
5. Grafana shows metrics and container events from ClickHouse.

## Start the Local Stack

```bash
docker compose up
```

Services:

- Core API: `http://localhost:8080`
- Swagger UI: `http://localhost:8080/swagger/`
- Grafana: `http://localhost:3000` (`admin` / `admin`)
- Demo target: `http://localhost:8081`

Optional Telegram configuration:

```bash
export TELEGRAM_BOT_TOKEN="<bot-token>"
export TELEGRAM_CHAT_ID="<chat-id>"
docker compose up
```

## Create a Metric Alert Rule

The analyzer evaluates metric thresholds. This rule opens an incident when CPU is above a very low threshold and asks the self-healer to restart the container. It is intentionally aggressive for demonstration.

```bash
curl -sS -X POST http://localhost:8080/api/v1/alert-rules \
  -H 'Content-Type: application/json' \
  -d '{
    "name": "Demo CPU recovery",
    "metric_name": "cpu_usage_percent",
    "condition_operator": "gt",
    "threshold": 0.1,
    "severity": "critical",
    "recovery_action": "restart_container"
  }'
```

Wait for the agent to publish one or two metric batches.

## Stop the Demo Container

```bash
docker compose stop target-nginx
```

Expected behavior:

- Agent receives a Docker `stop` event.
- Agent publishes the normalized event to Kafka topic `container.events`.
- Core stores the event in PostgreSQL and ClickHouse.
- Core creates an incident for `container_stopped`.
- Core sends a Telegram alert if `TELEGRAM_BOT_TOKEN` and `TELEGRAM_CHAT_ID` are configured.
- Self-healing writes a recovery action and calls Docker restart through `RECOVERY_DOCKER_HOST`.
- `/api/v1/events` returns the event.
- Grafana panel `Latest container events` shows the event.

Check via API:

```bash
curl -sS http://localhost:8080/api/v1/events | jq
curl -sS 'http://localhost:8080/api/v1/metrics/latest?limit=5' | jq
curl -sS http://localhost:8080/api/v1/incidents | jq
curl -sS http://localhost:8080/api/v1/recovery-actions | jq
```

## Restart the Demo Container

```bash
docker compose start target-nginx
```

Expected behavior:

- Agent publishes `container_started`.
- Grafana updates the event table.
- Metrics resume for `target-nginx`.

## What to Show During Defense

- Swagger UI for the available API surface.
- Grafana dashboard:
  - CPU usage by container.
  - Memory usage by container.
  - Container events over time.
  - Latest container events table.
- PostgreSQL-backed API:
  - `GET /api/v1/events`
  - `GET /api/v1/incidents`
  - `GET /api/v1/recovery-actions`
- ClickHouse-backed API:
  - `GET /api/v1/metrics/latest`
  - `GET /api/v1/metrics/history`

## Limitations of the MVP Demo

The event-to-incident path is intentionally simple: `container_stopped`, `container_died`, and `container_oom` directly open incidents and request `restart_container`. A production version should make this configurable through event-based alert rules and target selectors.
