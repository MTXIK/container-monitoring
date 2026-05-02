# Architecture

The repository currently contains three isolated folders that are intended to
become separate Git repositories:

- `agent/` - node-side telemetry producer;
- `core/` - receiving and processing service;
- `contracts/` - protobuf contracts shared through a package boundary.

## Ownership Boundaries

The agent owns only container collection and Kafka publishing. It has no
PostgreSQL, ClickHouse, Redis, Grafana, Telegram, or HTTP API dependencies.

The core service owns the receiving side: Kafka consumption, persistence,
alerting, recovery, notifications, API, and local development infrastructure.

Contracts are versioned independently so both services can upgrade against an
explicit schema.

## Data Flow

1. Agent collects container metrics and events through a collector backend.
2. Docker is the first backend behind the collector interface.
3. Agent publishes metrics to `container.metrics` and events to
   `container.events`.
4. Core consumes Kafka messages.
5. Core writes metric history to ClickHouse.
6. Core writes configuration, incidents, and recovery actions to PostgreSQL.
7. Core stores latest container state, locks, and alert deduplication keys in
   Redis.
8. Analyzer evaluates threshold rules.
9. Core sends Telegram notifications and dispatches allowed recovery actions.
10. Grafana reads metrics from ClickHouse.

## Extensibility

Container collection is isolated behind an agent-side collector backend. Docker
is only the first implementation. Kubernetes can be added as another backend
without changing the core processing model.

Recovery execution is isolated on the core side. Docker restart support should
be implemented as one executor, not as a hard dependency of the analyzer.
