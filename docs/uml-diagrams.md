# UML diagrams for the diploma

This document contains Mermaid diagrams for the `container-monitoring` project.
The diagrams are written as ready-to-copy Markdown blocks.

## 1. System context diagram

```mermaid
flowchart LR
    Operator[Operator / administrator]
    Docker[Docker Engine]
    Telegram[Telegram Bot API]
    Browser[Web browser]

    subgraph Platform["Container Monitoring Platform"]
        Agent[Agent service]
        Core[Core service]
        Frontend[Frontend admin panel]
        Grafana[Grafana dashboards]
    end

    Operator --> Browser
    Browser --> Frontend
    Browser --> Grafana
    Browser --> Core
    Agent --> Docker
    Core --> Docker
    Core --> Telegram

    Agent -. telemetry .-> Core
    Core -. operational data .-> Frontend
    Core -. metrics and events .-> Grafana
```

## 2. Container-level component diagram

```mermaid
flowchart TB
    subgraph RuntimeNode["Monitored host"]
        DockerSocket["/var/run/docker.sock"]
        Target["target-nginx / monitored containers"]
        Agent["agent\nGo telemetry producer"]
    end

    subgraph MessageBus["Kafka"]
        MetricsTopic["container.metrics"]
        EventsTopic["container.events"]
    end

    subgraph CoreService["core\nGo processing service"]
        Consumer["Kafka consumer"]
        Ingest["Ingest handler"]
        Analyzer["Threshold analyzer"]
        Recovery["Recovery coordinator"]
        HTTP["Fiber HTTP API"]
        Notifier["Telegram notifier"]
    end

    subgraph Storage["Storage layer"]
        Postgres["PostgreSQL\nconfiguration, incidents, recovery"]
        ClickHouse["ClickHouse\nmetric and event history"]
        Redis["Redis\nlatest state, durations, locks"]
    end

    subgraph UI["User interfaces"]
        Frontend["React admin panel"]
        Swagger["Swagger UI"]
        Grafana["Grafana"]
    end

    Target --> DockerSocket
    Agent --> DockerSocket
    Agent --> MetricsTopic
    Agent --> EventsTopic
    MetricsTopic --> Consumer
    EventsTopic --> Consumer
    Consumer --> Ingest
    Ingest --> Analyzer
    Ingest --> Postgres
    Ingest --> ClickHouse
    Ingest --> Redis
    Ingest --> Notifier
    Ingest --> Recovery
    Recovery --> Redis
    Recovery --> Postgres
    Recovery --> DockerSocket
    HTTP --> Postgres
    HTTP --> ClickHouse
    HTTP --> Redis
    Frontend --> HTTP
    Swagger --> HTTP
    Grafana --> ClickHouse
```

## 3. Deployment diagram

```mermaid
flowchart TB
    subgraph Host["Local Docker host"]
        DockerDaemon["Docker daemon"]

        subgraph Compose["docker compose stack"]
            AgentContainer["agent container"]
            CoreContainer["core container"]
            FrontendContainer["frontend container"]
            KafkaContainer["kafka container"]
            PostgresContainer["postgres container"]
            ClickHouseContainer["clickhouse container"]
            RedisContainer["redis container"]
            GrafanaContainer["grafana container"]
            TargetContainer["target-nginx container"]
        end
    end

    User["User browser"]
    Telegram["Telegram API"]

    AgentContainer --> DockerDaemon
    CoreContainer --> DockerDaemon
    TargetContainer --> DockerDaemon
    AgentContainer --> KafkaContainer
    CoreContainer --> KafkaContainer
    CoreContainer --> PostgresContainer
    CoreContainer --> ClickHouseContainer
    CoreContainer --> RedisContainer
    CoreContainer --> Telegram
    FrontendContainer --> CoreContainer
    GrafanaContainer --> ClickHouseContainer
    User --> FrontendContainer
    User --> CoreContainer
    User --> GrafanaContainer
```

## 4. Agent internal structure

```mermaid
classDiagram
    class AgentMain {
        +loadConfig()
        +createDockerBackend()
        +createKafkaPublisher()
        +runRuntime()
    }

    class Config {
        +NodeID string
        +DockerHost string
        +CollectInterval duration
        +KafkaBrokers string
        +MetricsTopic string
        +EventsTopic string
    }

    class Runtime {
        +Run(ctx, logger, cfg, backend, publisher) error
    }

    class CollectorBackend {
        <<interface>>
        +CollectMetrics(ctx) MetricSample[]
        +WatchEvents(ctx) EventChannel
    }

    class DockerBackend {
        +CollectMetrics(ctx) MetricSample[]
        +WatchEvents(ctx) EventChannel
    }

    class Publisher {
        <<interface>>
        +PublishMetrics(ctx, metrics) error
        +PublishEvent(ctx, event) error
        +Close() error
    }

    class KafkaPublisher {
        +PublishMetrics(ctx, metrics) error
        +PublishEvent(ctx, event) error
        +Close() error
    }

    AgentMain --> Config
    AgentMain --> Runtime
    Runtime --> CollectorBackend
    Runtime --> Publisher
    DockerBackend ..|> CollectorBackend
    KafkaPublisher ..|> Publisher
```

## 5. Core internal structure

```mermaid
classDiagram
    class CoreMain {
        +loadConfig()
        +connectStorage()
        +startKafkaConsumer()
        +startHTTPServer()
    }

    class KafkaConsumer {
        +Consume(ctx, handler) error
    }

    class IngestHandler {
        +Handle(ctx, message) error
        -handleMetric(ctx, bytes) error
        -handleEvent(ctx, bytes) error
        -createIncidentForEvent(ctx, event) error
    }

    class Analyzer {
        +Evaluate(metric, rules) Incident[]
    }

    class Repository {
        <<interface>>
        +UpsertTarget(ctx, target) error
        +SaveMetric(ctx, metric) error
        +SaveEvent(ctx, event) error
        +CreateIncident(ctx, incident) Incident
    }

    class StateStore {
        <<interface>>
        +SetLatestMetrics(ctx, metric) error
        +SetTargetState(ctx, event) error
        +AcquireRecoveryLock(ctx, targetID, ttl) bool
    }

    class HTTPServer {
        +NewServer(repo, recoverer) FiberApp
    }

    class RecoveryCoordinator {
        +Recover(ctx, incident, action) error
    }

    class TelegramNotifier {
        +SendIncident(ctx, text) error
    }

    CoreMain --> KafkaConsumer
    CoreMain --> HTTPServer
    KafkaConsumer --> IngestHandler
    IngestHandler --> Repository
    IngestHandler --> StateStore
    IngestHandler --> Analyzer
    IngestHandler --> TelegramNotifier
    IngestHandler --> RecoveryCoordinator
    HTTPServer --> Repository
    HTTPServer --> RecoveryCoordinator
```

## 6. Domain model class diagram

```mermaid
classDiagram
    class Target {
        +string id
        +string name
        +string type
        +string source
        +string external_id
        +string status
        +string node_id
        +map labels
        +datetime last_seen_at
    }

    class MetricSample {
        +string node_id
        +string source
        +string target_id
        +string container_name
        +map metrics
        +datetime timestamp
    }

    class Event {
        +int id
        +string node_id
        +string source
        +string target_id
        +string container_name
        +string event_type
        +string severity
        +string message
        +map payload
        +datetime timestamp
    }

    class AlertRule {
        +string id
        +string name
        +string target_id
        +string metric_name
        +string operator
        +float threshold
        +duration duration
        +string severity
        +bool enabled
        +string recovery_policy
    }

    class Incident {
        +int id
        +string rule_id
        +string target_id
        +string node_id
        +string status
        +string severity
        +string description
        +float value
        +datetime started_at
        +datetime resolved_at
    }

    class RecoveryAction {
        +int id
        +int incident_id
        +string target_id
        +string action_type
        +string status
        +datetime started_at
        +datetime finished_at
        +string result_message
    }

    Target "1" --> "0..*" MetricSample : receives
    Target "1" --> "0..*" Event : emits
    AlertRule "1" --> "0..*" Incident : creates
    Target "1" --> "0..*" Incident : affected by
    Incident "1" --> "0..*" RecoveryAction : triggers
```

## 7. Storage model diagram

```mermaid
classDiagram
    class nodes {
        +text id
        +text name
        +timestamptz created_at
    }

    class containers {
        +text id
        +text node_id
        +text name
        +text image
        +text source
        +text external_id
        +text status
        +jsonb labels
        +timestamptz last_seen_at
    }

    class alert_rules {
        +text id
        +text name
        +text target_id
        +text metric
        +text operator
        +double threshold
        +interval duration
        +text severity
        +text recovery_action
        +boolean enabled
    }

    class incidents {
        +bigserial id
        +text rule_id
        +text node_id
        +text container_id
        +text status
        +text severity
        +double value
        +timestamptz started_at
        +timestamptz resolved_at
    }

    class events {
        +bigserial id
        +text node_id
        +text container_id
        +text event_type
        +text severity
        +jsonb payload
        +timestamptz occurred_at
    }

    class recovery_actions {
        +bigserial id
        +bigint incident_id
        +text target_id
        +text action
        +text status
        +text result_message
    }

    class container_metrics {
        +datetime64 collected_at
        +string node_id
        +string container_id
        +string name
        +float64 cpu_percent
        +uint64 memory_bytes
        +uint64 rx_bytes
        +uint64 tx_bytes
        +uint64 block_read
        +uint64 block_write
    }

    class container_events {
        +datetime64 occurred_at
        +string node_id
        +string container_id
        +string name
        +string type
    }

    nodes "1" --> "0..*" containers
    containers "1" --> "0..*" incidents
    containers "1" --> "0..*" events
    incidents "1" --> "0..*" recovery_actions
    containers "1" --> "0..*" container_metrics
    containers "1" --> "0..*" container_events
```

## 8. Metric collection sequence

```mermaid
sequenceDiagram
    autonumber
    participant Timer as Agent ticker
    participant Runtime as Agent runtime
    participant Docker as Docker Engine API
    participant Publisher as Kafka publisher
    participant Kafka as Kafka topic container.metrics

    Timer->>Runtime: collect interval elapsed
    Runtime->>Docker: GET /containers/json
    Docker-->>Runtime: running containers
    loop for each container
        Runtime->>Docker: GET /containers/{id}/stats?stream=false
        Docker-->>Runtime: CPU, memory, network, block stats
    end
    Runtime->>Runtime: normalize MetricSample
    Runtime->>Publisher: PublishMetrics(metrics)
    Publisher->>Kafka: JSON messages
```

## 9. Metric ingest and threshold alert sequence

```mermaid
sequenceDiagram
    autonumber
    participant Kafka as Kafka container.metrics
    participant Consumer as Core Kafka consumer
    participant Handler as Ingest handler
    participant Repo as Repository
    participant ClickHouse as ClickHouse
    participant Redis as Redis state
    participant Analyzer as Analyzer
    participant Telegram as Telegram notifier
    participant Recovery as Recovery coordinator

    Kafka->>Consumer: metric message
    Consumer->>Handler: Handle(message)
    Handler->>Handler: decode MetricSample
    Handler->>Repo: UpsertTarget(targetFromMetric)
    Repo->>ClickHouse: insert container_metrics
    Handler->>Redis: SetLatestMetrics(metric)
    Handler->>Repo: EnabledRules()
    Repo-->>Handler: active threshold rules
    Handler->>Analyzer: Evaluate(metric, rules)
    Analyzer-->>Handler: incident candidates
    loop for each candidate
        Handler->>Redis: check duration window
        Handler->>Repo: HasOpenIncident(rule_id, target_id)
        alt no duplicate incident
            Handler->>Repo: CreateIncident(open)
            Handler->>Telegram: SendIncident(text)
            Handler->>Recovery: Recover(incident, recovery_policy)
        else duplicate exists
            Handler-->>Consumer: skip candidate
        end
    end
```

## 10. Docker event and self-healing sequence

```mermaid
sequenceDiagram
    autonumber
    participant Docker as Docker Engine API
    participant Agent as Agent event watcher
    participant Kafka as Kafka container.events
    participant Consumer as Core Kafka consumer
    participant Handler as Ingest handler
    participant Store as PostgreSQL and ClickHouse
    participant Redis as Redis state
    participant Telegram as Telegram notifier
    participant Recovery as Recovery coordinator
    participant Executor as Docker executor

    Docker-->>Agent: stop, die, oom, restart, start event
    Agent->>Agent: normalize Event
    Agent->>Kafka: publish JSON event
    Kafka->>Consumer: event message
    Consumer->>Handler: Handle(message)
    Handler->>Handler: decode Event
    Handler->>Store: UpsertTarget(targetFromEvent)
    Handler->>Store: SaveEvent(event)
    Handler->>Redis: SetTargetState(event)
    alt event is stop, die, or oom
        Handler->>Store: HasOpenIncident(event_type, target_id)
        Handler->>Store: CreateIncident(open)
        Handler->>Telegram: SendIncident(text)
        Handler->>Recovery: Recover(incident, restart_container)
        Recovery->>Redis: AcquireRecoveryLock(target_id)
        Recovery->>Store: CreateRecoveryAction(running)
        Recovery->>Executor: POST /containers/{id}/restart
        Executor->>Docker: restart container
        Recovery->>Store: FinishRecoveryAction(succeeded or failed)
    else non-failure event
        Handler-->>Consumer: store event only
    end
```

## 11. Frontend API interaction sequence

```mermaid
sequenceDiagram
    autonumber
    participant User as Administrator
    participant UI as React frontend
    participant Query as TanStack Query hooks
    participant API as Core HTTP API
    participant Repo as Repository
    participant Postgres as PostgreSQL
    participant ClickHouse as ClickHouse

    User->>UI: open dashboard, targets, incidents, metrics
    UI->>Query: useQuery(route data)
    Query->>API: GET /api/v1/*
    API->>Repo: load operational data
    alt configuration or incidents
        Repo->>Postgres: select targets, rules, events, incidents, recovery
        Postgres-->>Repo: rows
    else metric history
        Repo->>ClickHouse: select container_metrics
        ClickHouse-->>Repo: points
    end
    Repo-->>API: domain objects
    API-->>Query: JSON response
    Query-->>UI: cached data and status
    UI-->>User: tables, badges, details, actions
```

## 12. Alert rule management sequence

```mermaid
sequenceDiagram
    autonumber
    participant Admin as Administrator
    participant UI as AlertRulesPage
    participant API as Core HTTP API
    participant Repo as Repository
    participant Postgres as PostgreSQL
    participant Handler as Ingest handler
    participant Analyzer as Analyzer

    Admin->>UI: create or edit threshold rule
    UI->>API: POST/PATCH /api/v1/alert-rules
    API->>API: decode and normalize operator
    API->>Repo: CreateAlertRule or UpdateAlertRule
    Repo->>Postgres: upsert alert_rules
    Postgres-->>Repo: saved rule
    Repo-->>API: AlertRule
    API-->>UI: JSON rule
    UI-->>Admin: updated table

    Note over Handler,Analyzer: Later, every incoming metric is evaluated against enabled rules.
    Handler->>Repo: EnabledRules()
    Repo->>Postgres: select enabled alert_rules
    Repo-->>Handler: rules
    Handler->>Analyzer: Evaluate(metric, rules)
```

## 13. Target status state diagram

```mermaid
stateDiagram-v2
    [*] --> UNKNOWN
    UNKNOWN --> OK: container_started / container_restarted
    UNKNOWN --> WARNING: warning event
    UNKNOWN --> CRITICAL: stopped / died / oom
    OK --> WARNING: warning event
    OK --> CRITICAL: stopped / died / oom
    WARNING --> OK: container_started / container_restarted
    WARNING --> CRITICAL: stopped / died / oom
    CRITICAL --> OK: container_started / container_restarted
    CRITICAL --> RECOVERING: recovery action running
    RECOVERING --> OK: restart succeeds and metrics resume
    RECOVERING --> CRITICAL: recovery failed or lock skipped
```

## 14. Incident lifecycle state diagram

```mermaid
stateDiagram-v2
    [*] --> open: threshold matched or failure event received
    open --> acknowledged: POST /api/v1/incidents/{id}/ack
    open --> resolved: POST /api/v1/incidents/{id}/resolve
    acknowledged --> resolved: POST /api/v1/incidents/{id}/resolve
    resolved --> [*]

    note right of open
      Open and acknowledged incidents
      deduplicate new matches for the
      same rule_id and target_id.
    end note
```

## 15. Recovery action lifecycle state diagram

```mermaid
stateDiagram-v2
    [*] --> running: CreateRecoveryAction
    running --> succeeded: notify_only, retry_check, or restart succeeds
    running --> failed: executor or lock error
    running --> skipped: recovery lock is already held
    failed --> running: POST /api/v1/recovery-actions/{id}/retry
    succeeded --> [*]
    skipped --> [*]

    note right of running
      restart_container uses Docker Engine API
      through RECOVERY_DOCKER_HOST.
    end note
```
