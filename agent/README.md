# container-monitoring-agent

Node-side telemetry producer for the Container Monitoring MVP.

The agent runs close to the container runtime, reads Docker metrics and lifecycle
events, normalizes them into platform telemetry, and publishes JSON messages to
Kafka. It deliberately does not know about PostgreSQL, ClickHouse, Redis,
Grafana, Telegram, or HTTP API internals. Those responsibilities belong to
`core/`.

## Responsibilities

- Connect to Docker Engine API through `DOCKER_HOST`.
- Collect periodic container stats:
  - CPU usage percent;
  - memory usage bytes and percent;
  - network RX/TX bytes;
  - block read/write bytes.
- Watch Docker lifecycle events:
  - `start`;
  - `stop`;
  - `die`;
  - `oom`;
  - `restart`.
- Normalize Docker data into platform fields such as `node_id`, `source`,
  `target_id`, `container_name`, `metrics`, and `timestamp`.
- Publish metrics to Kafka topic `container.metrics` by default.
- Publish events to Kafka topic `container.events` by default.
- Continue collecting metrics when stats for one listed container disappear
  during a collection cycle.

## Runtime Flow

1. `cmd/agent/main.go` loads configuration from environment variables.
2. The Docker collector backend is created.
3. The Kafka publisher is created.
4. Runtime starts:
   - a ticker for periodic metric collection;
   - a Docker event watcher stream.
5. Metrics and events are serialized as JSON and published to Kafka.
6. On `SIGINT` or `SIGTERM`, the runtime exits and closes the publisher.

## Kafka Message Shape

Metric payload example:

```json
{
  "node_id": "local-node",
  "source": "docker",
  "target_id": "container-id",
  "container_name": "nginx",
  "metrics": {
    "cpu_usage_percent": 12.3,
    "memory_usage_bytes": 104857600,
    "memory_usage_percent": 25.1,
    "network_rx_bytes": 1234,
    "network_tx_bytes": 5678,
    "block_read_bytes": 0,
    "block_write_bytes": 4096
  },
  "timestamp": "2026-05-04T12:00:00Z"
}
```

Event payload example:

```json
{
  "node_id": "local-node",
  "source": "docker",
  "target_id": "container-id",
  "container_name": "nginx",
  "event_type": "container_stopped",
  "severity": "warning",
  "message": "Container nginx stopped",
  "payload": {
    "action": "stop",
    "exit_code": "",
    "oom_killed": ""
  },
  "timestamp": "2026-05-04T12:00:00Z"
}
```

## Configuration

| Variable | Default | Description |
| --- | --- | --- |
| `AGENT_NODE_ID` | `local-node` | Logical node id attached to every metric and event. |
| `AGENT_COLLECTOR_BACKEND` | `docker` | Reserved backend selector. The current entrypoint always uses Docker. |
| `DOCKER_HOST` | `unix:///var/run/docker.sock` | Docker API endpoint. Unix socket is used in local Compose. |
| `COLLECT_INTERVAL` | `10s` | Periodic stats collection interval. Uses Go duration syntax. |
| `KAFKA_BROKERS` | `localhost:9092` | Comma-separated Kafka broker list. |
| `KAFKA_METRICS_TOPIC` | `container.metrics` | Topic for normalized metric messages. |
| `KAFKA_EVENTS_TOPIC` | `container.events` | Topic for normalized lifecycle event messages. |

Invalid `COLLECT_INTERVAL` values fall back to `10s`.

## Local Development

Create a local environment file when running the agent outside Compose:

```bash
cp .env.example .env
```

Then either export the variables from `.env` in your shell or pass them
directly to `go run`.

Run tests:

```bash
go test ./...
```

Verify compilation:

```bash
go build ./...
```

Run the agent directly against local Docker and Kafka:

```bash
AGENT_NODE_ID=local-node \
DOCKER_HOST=unix:///var/run/docker.sock \
KAFKA_BROKERS=localhost:9092 \
go run ./cmd/agent
```

For a complete local platform, prefer the root Compose stack:

```bash
cd ..
docker compose up --build
```

For demonstration targets:

```bash
cd ..
docker compose -f docker-compose.yml -f docker-compose.demo-targets.yml up --build
```

## Docker Notes

The Docker backend supports Unix socket endpoints:

```bash
DOCKER_HOST=unix:///var/run/docker.sock
```

For Compose, the agent mounts the host socket read-only:

```yaml
volumes:
  - /var/run/docker.sock:/var/run/docker.sock:ro
```

The agent lists running containers through `/containers/json`, then requests
non-streaming stats through `/containers/{id}/stats?stream=false`. If a
container disappears between listing and stats collection, that container is
skipped and the rest of the batch is still published.

## Testing Notes

Important packages:

- `internal/collector/docker` tests Docker metric normalization and collection
  behavior.
- `internal/config` tests environment parsing.
- `internal/publisher/kafka` tests Kafka payload serialization.
- `internal/runtime` tests runtime orchestration.

Useful commands:

```bash
go test ./internal/collector/docker
go test ./internal/publisher/kafka
go test ./...
```

## Current Limitations

- Docker is the only implemented collector backend.
- Docker event watching starts once at runtime startup. If the event stream is
  broken by the daemon or network layer, the current implementation logs the
  watcher error but does not reconnect the stream.
- The agent publishes JSON directly. Protobuf contracts in `contracts/` are
  reserved for future service boundaries.
