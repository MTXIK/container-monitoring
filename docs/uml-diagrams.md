# UML diagrams for chapter 2

This document contains the required figures for chapter 2 of the diploma.
Most diagrams are written in Mermaid so they can be rendered directly from Markdown.
Figures marked as screenshots define the exact screen that should be captured from the running platform.

## Рисунок 2.1. Компонентная диаграмма агентской подсистемы сбора данных

Insert in section 2.2.

```mermaid
flowchart LR
    DockerAPI["Docker Engine API"]

    subgraph Agent["Go Agent"]
        ConfigLoader["Config Loader"]
        Runtime["Agent Runtime"]
        DockerCollector["Docker Collector"]
        EventWatcher["Docker Event Watcher"]
        KafkaPublisher["Kafka Publisher"]
    end

    subgraph Kafka["Kafka"]
        MetricsTopic[("container.metrics")]
        EventsTopic[("container.events")]
    end

    Core["Core Service"]

    ConfigLoader --> Runtime
    Runtime --> DockerCollector
    Runtime --> EventWatcher
    Runtime --> KafkaPublisher
    DockerCollector --> DockerAPI
    EventWatcher --> DockerAPI
    DockerCollector --> KafkaPublisher
    EventWatcher --> KafkaPublisher
    KafkaPublisher --> MetricsTopic
    KafkaPublisher --> EventsTopic
    MetricsTopic --> Core
    EventsTopic --> Core
```

## Рисунок 2.2. UML диаграмма пакетов агентской подсистемы

Insert in section 2.2.1.

```mermaid
flowchart TB
    Cmd["package cmd/agent"]
    Config["package internal/config"]
    Runtime["package internal/runtime"]
    Collector["package internal/collector/docker"]
    Publisher["package internal/publisher/kafka"]

    Cmd --> Config
    Cmd --> Runtime
    Cmd --> Collector
    Cmd --> Publisher
    Runtime --> Config
    Runtime --> Collector
    Runtime --> Publisher
```

## Рисунок 2.3. Алгоритм периодического сбора метрик контейнеров

Insert in section 2.2.2.

```mermaid
flowchart TB
    Start([Start agent runtime])
    Ticker["Start collect ticker"]
    Tick{"Ticker tick?"}
    List["Request container list from Docker Engine API"]
    ListError{"List request failed?"}
    LogList["Log error and wait for next tick"]
    Loop{"Next container exists?"}
    Stats["Request Docker stats for container"]
    StatsError{"Stats request failed?"}
    LogStats["Log container stats error"]
    Normalize["Normalize CPU, memory, network and block IO metrics"]
    Append["Add metric sample to batch"]
    Publish{"Batch is not empty?"}
    Kafka["Publish metric batch to Kafka topic container.metrics"]
    Wait["Wait for next tick"]

    Start --> Ticker --> Tick
    Tick --> List
    List --> ListError
    ListError -- yes --> LogList --> Wait
    ListError -- no --> Loop
    Loop -- yes --> Stats --> StatsError
    StatsError -- yes --> LogStats --> Loop
    StatsError -- no --> Normalize --> Append --> Loop
    Loop -- no --> Publish
    Publish -- yes --> Kafka --> Wait
    Publish -- no --> Wait
    Wait --> Tick
```

## Рисунок 2.4. Последовательность обработки события жизненного цикла контейнера

Insert in section 2.2.3.

```mermaid
sequenceDiagram
    autonumber
    participant Docker as Docker Engine API
    participant Agent as Go Agent
    participant Watcher as Docker Event Watcher
    participant Publisher as Kafka Publisher
    participant Topic as Kafka topic container.events
    participant Core as Core Service

    Docker-->>Watcher: container lifecycle event
    Watcher-->>Agent: raw Docker event
    Agent->>Agent: normalize event fields
    Agent->>Publisher: PublishEvent(normalized event)
    Publisher->>Topic: send JSON message
    Core->>Topic: consume event message
```

## Рисунок 2.5. Потоки передачи телеметрии от агента к Kafka

Insert in section 2.2.4.

```mermaid
flowchart LR
    Stats["Docker stats"]
    Events["Docker events"]
    Agent["Go Agent"]
    MetricsTopic[("Kafka topic\ncontainer.metrics")]
    EventsTopic[("Kafka topic\ncontainer.events")]
    Core["Core Service"]

    Stats --> Agent --> MetricsTopic --> Core
    Events --> Agent --> EventsTopic --> Core
```

## Рисунок 2.6. Жизненный цикл Go Agent

Insert in section 2.2.5.

```mermaid
flowchart TB
    Start([Process start])
    Load["Load configuration"]
    Collector["Initialize Docker collector"]
    Publisher["Initialize Kafka publisher"]
    Runtime["Start agent runtime"]
    Fork{"Runtime tasks"}
    Metrics["Collect metrics on ticker"]
    Events["Read Docker events stream"]
    PublishMetrics["Publish to container.metrics"]
    PublishEvents["Publish to container.events"]
    Signal{"SIGTERM or context canceled?"}
    Close["Close Kafka publisher"]
    Stop([Process stopped])

    Start --> Load --> Collector --> Publisher --> Runtime --> Fork
    Fork --> Metrics --> PublishMetrics --> Signal
    Fork --> Events --> PublishEvents --> Signal
    Signal -- no --> Fork
    Signal -- yes --> Close --> Stop
```

## Рисунок 2.7. Компонентная диаграмма центрального сервиса обработки данных

Insert in section 2.3.

```mermaid
flowchart TB
    subgraph Core["Core Service"]
        Consumer["Kafka Consumer"]
        MetricHandler["Metric Handler"]
        EventHandler["Event Handler"]
        Analyzer["Analyzer"]
        IncidentService["Incident Service"]
        RecoveryService["Recovery Service"]
        Notifier["Notifier"]
        HTTPAPI["HTTP API"]
        PostgresRepo["PostgreSQL Repository"]
        ClickHouseRepo["ClickHouse Repository"]
        RedisState["Redis State Storage"]
    end

    Consumer --> MetricHandler
    Consumer --> EventHandler
    MetricHandler --> Analyzer
    EventHandler --> IncidentService
    Analyzer --> IncidentService
    IncidentService --> RecoveryService
    IncidentService --> Notifier
    HTTPAPI --> IncidentService
    HTTPAPI --> RecoveryService
    HTTPAPI --> PostgresRepo
    HTTPAPI --> ClickHouseRepo
    HTTPAPI --> RedisState
    MetricHandler --> ClickHouseRepo
    MetricHandler --> RedisState
    MetricHandler --> PostgresRepo
    EventHandler --> PostgresRepo
    EventHandler --> ClickHouseRepo
    EventHandler --> RedisState
    IncidentService --> PostgresRepo
    RecoveryService --> PostgresRepo
    RecoveryService --> RedisState
```

## Рисунок 2.8. UML диаграмма пакетов центрального сервиса

Insert in section 2.3.1.

```mermaid
flowchart TB
    Cmd["package cmd/core"]
    Config["package internal/config"]
    Consumer["package internal/consumer"]
    Service["package internal/service"]
    Postgres["package internal/repository/postgres"]
    ClickHouse["package internal/repository/clickhouse"]
    Redis["package internal/state/redis"]
    Notifier["package internal/notifier"]
    Recovery["package internal/recovery"]
    HTTP["package internal/http"]

    Cmd --> Config
    Cmd --> Consumer
    Cmd --> Service
    Cmd --> HTTP
    Consumer --> Service
    HTTP --> Service
    Service --> Postgres
    Service --> ClickHouse
    Service --> Redis
    Service --> Notifier
    Service --> Recovery
    Recovery --> Redis
    Recovery --> Postgres
```

## Рисунок 2.9. Последовательность обработки сообщения телеметрии в Core Service

Insert in section 2.3.2.

```mermaid
sequenceDiagram
    autonumber
    participant Kafka as Kafka
    participant Consumer as Kafka Consumer
    participant MetricHandler as Metric Handler
    participant EventHandler as Event Handler
    participant Analyzer as Analyzer
    participant Postgres as PostgreSQL
    participant ClickHouse as ClickHouse
    participant Redis as Redis
    participant Incidents as Incident Service

    Kafka->>Consumer: telemetry message
    Consumer->>Consumer: detect topic and message type
    alt metric message
        Consumer->>MetricHandler: HandleMetric(message)
        MetricHandler->>Postgres: upsert target metadata
        MetricHandler->>ClickHouse: insert container metric
        MetricHandler->>Redis: update latest metrics
        MetricHandler->>Analyzer: evaluate metric against rules
        Analyzer-->>MetricHandler: alert candidates
        MetricHandler->>Incidents: create or update incident
        Incidents->>Postgres: persist incident
    else lifecycle event message
        Consumer->>EventHandler: HandleEvent(message)
        EventHandler->>Postgres: save event
        EventHandler->>ClickHouse: insert analytical event
        EventHandler->>Redis: update runtime state
        EventHandler->>Incidents: create incident for critical event
        Incidents->>Postgres: persist incident
    end
```

## Рисунок 2.10. ER диаграмма конфигурационных и эксплуатационных сущностей платформы

Insert in section 2.3.3.

```mermaid
erDiagram
    targets ||--o{ alert_rules : has
    targets ||--o{ events : emits
    targets ||--o{ incidents : affected_by
    alert_rules ||--o{ incidents : creates
    incidents ||--o{ recovery_actions : triggers

    targets {
        text id PK
        text name
        text type
        text source
        text external_id
        text status
        text node_id
    }

    alert_rules {
        text id PK
        text target_id FK
        text metric_name
        text operator
        float threshold
        text severity
        boolean enabled
    }

    events {
        bigint id PK
        text target_id FK
        text event_type
        text severity
        timestamp occurred_at
    }

    incidents {
        bigint id PK
        text rule_id FK
        text target_id FK
        text status
        text severity
        timestamp started_at
        timestamp resolved_at
    }

    recovery_actions {
        bigint id PK
        bigint incident_id FK
        text target_id FK
        text action_type
        text status
    }
```

## Рисунок 2.11. Распределение данных между хранилищами платформы

Insert in section 2.3.3.

```mermaid
flowchart LR
    Core["Core Service"]

    subgraph PostgreSQL["PostgreSQL"]
        Targets["targets"]
        Rules["alert_rules"]
        PgEvents["events"]
        Incidents["incidents"]
        Actions["recovery_actions"]
    end

    subgraph ClickHouse["ClickHouse"]
        Metrics["metrics"]
        AnalyticalEvents["analytical events"]
    end

    subgraph Redis["Redis"]
        Latest["latest metrics"]
        Runtime["runtime state"]
        Locks["recovery locks"]
    end

    Core --> Targets
    Core --> Rules
    Core --> PgEvents
    Core --> Incidents
    Core --> Actions
    Core --> Metrics
    Core --> AnalyticalEvents
    Core --> Latest
    Core --> Runtime
    Core --> Locks
```

## Рисунок 2.12. Алгоритм применения правила алертинга

Insert in section 2.3.4.

```mermaid
flowchart TB
    Start([Metric received])
    LoadRules["Load enabled alert rules"]
    NextRule{"Next rule exists?"}
    Applicable{"Rule applies to target and metric?"}
    Threshold{"Metric value matches threshold?"}
    Alert["Create alert event"]
    Incident["Create or reuse open incident"]
    Notify["Send notification"]
    Recovery{"Recovery policy configured?"}
    Trigger["Trigger recovery action"]
    Finish([Evaluation finished])

    Start --> LoadRules --> NextRule
    NextRule -- no --> Finish
    NextRule -- yes --> Applicable
    Applicable -- no --> NextRule
    Applicable -- yes --> Threshold
    Threshold -- no --> NextRule
    Threshold -- yes --> Alert --> Incident --> Notify --> Recovery
    Recovery -- yes --> Trigger --> NextRule
    Recovery -- no --> NextRule
```

## Рисунок 2.13. Диаграмма состояний инцидента

Insert in section 2.3.4.

```mermaid
stateDiagram-v2
    [*] --> open: alert/event creates incident
    open --> acknowledged: ack endpoint
    open --> resolved: resolve endpoint
    open --> resolved: successful recovery
    acknowledged --> resolved: resolve endpoint
    acknowledged --> resolved: successful recovery
    resolved --> [*]
```

## Рисунок 2.14. Последовательность выполнения восстановительного действия

Insert in section 2.3.5.

```mermaid
sequenceDiagram
    autonumber
    participant Incident as Incident Service
    participant Recovery as Recovery Service
    participant Redis as Redis
    participant Postgres as PostgreSQL
    participant Docker as Docker Engine
    participant Notifier as Notifier

    Incident->>Recovery: execute recovery policy
    Recovery->>Redis: acquire recovery lock
    alt lock acquired
        Recovery->>Postgres: create recovery_action running
        Recovery->>Docker: execute container operation
        Docker-->>Recovery: operation result
        Recovery->>Postgres: update recovery_action status
        Recovery->>Redis: release recovery lock
        Recovery->>Notifier: send recovery result
    else lock already exists
        Recovery->>Postgres: create recovery_action skipped
        Recovery->>Notifier: send skipped recovery notification
    end
```

## Рисунок 2.15. Схема отправки уведомления о возникновении инцидента

Insert in section 2.3.6.

```mermaid
sequenceDiagram
    autonumber
    participant Incident as Incident Service
    participant Notifier as Notifier
    participant Telegram as Telegram API

    Incident->>Notifier: SendIncident(incident)
    alt Telegram env vars are configured
        Notifier->>Telegram: sendMessage(chat_id, text)
        Telegram-->>Notifier: delivery result
        Notifier-->>Incident: notification sent
    else Telegram env vars are missing
        Notifier-->>Incident: skip external notification
    end
```

## Рисунок 2.16. Схема пользовательского доступа к платформе мониторинга

Insert in section 2.4.

```mermaid
flowchart TB
    User["User"]
    Frontend["Frontend admin panel"]
    HTTP["Core HTTP API"]
    Swagger["Swagger UI"]
    Grafana["Grafana"]
    PostgreSQL["PostgreSQL"]
    ClickHouse["ClickHouse"]
    Redis["Redis"]

    User --> Frontend
    User --> Swagger
    User --> Grafana
    Frontend --> HTTP
    Swagger --> HTTP
    HTTP --> PostgreSQL
    HTTP --> ClickHouse
    HTTP --> Redis
    Grafana --> ClickHouse
```

## Рисунок 2.17. Последовательность обработки HTTP запроса в Core Service

Insert in section 2.4.1.

```mermaid
sequenceDiagram
    autonumber
    participant Frontend as Frontend
    participant Handler as HTTP API Handler
    participant Service as Service Layer
    participant Repository as Repository or Redis State
    participant Postgres as PostgreSQL
    participant ClickHouse as ClickHouse
    participant Redis as Redis

    Frontend->>Handler: HTTP request
    Handler->>Handler: validate path, method and payload
    Handler->>Service: call use case
    Service->>Repository: load or mutate data
    alt configuration or incidents
        Repository->>Postgres: SQL query
        Postgres-->>Repository: rows
    else historical metrics
        Repository->>ClickHouse: analytical query
        ClickHouse-->>Repository: rows
    else latest state
        Repository->>Redis: state query
        Redis-->>Repository: values
    end
    Repository-->>Service: domain result
    Service-->>Handler: response model
    Handler-->>Frontend: JSON response
```

## Рисунок 2.18. Интерфейс Swagger UI с описанием HTTP API платформы

Insert in section 2.4.2.

Screenshot requirements:

- Open Swagger UI for Core Service.
- Capture endpoint groups: health, targets, metrics, events, alert-rules, incidents, recovery-actions.
- Use this Markdown image placeholder after the screenshot is created:

```markdown
![Рисунок 2.18. Интерфейс Swagger UI с группами HTTP API endpoints платформы](images/figure-2-18-swagger-ui.png)
```

## Рисунок 2.19. Последовательность подтверждения инцидента через frontend панель

Insert in section 2.4.3.

```mermaid
sequenceDiagram
    autonumber
    participant User as User
    participant Frontend as Frontend
    participant HTTP as Core HTTP API
    participant Incident as Incident Service
    participant Postgres as PostgreSQL

    User->>Frontend: click acknowledge incident
    Frontend->>HTTP: POST /api/v1/incidents/{id}/ack
    HTTP->>Incident: AckIncident(id)
    Incident->>Postgres: update incidents set status = acknowledged
    Postgres-->>Incident: updated incident
    Incident-->>HTTP: incident model
    HTTP-->>Frontend: JSON response
    Frontend-->>User: incident status is acknowledged
```

## Рисунок 2.20. Административная панель платформы мониторинга

Insert in section 2.4.3.

Screenshot requirements:

- Open the frontend administration panel.
- Capture one representative section: Incidents, Recovery actions, Latest metrics, or Targets.
- Use this Markdown image placeholder after the screenshot is created:

```markdown
![Рисунок 2.20. Административная панель платформы мониторинга](images/figure-2-20-admin-panel.png)
```

## Рисунок 2.21. Интеграция Grafana с аналитическим хранилищем ClickHouse

Insert in section 2.4.4.

```mermaid
flowchart LR
    Core["Core Service"]
    ClickHouse["ClickHouse analytical storage"]
    Grafana["Grafana"]
    User["User"]

    Core -->|writes metrics and analytical events| ClickHouse
    Grafana -->|queries datasource| ClickHouse
    User -->|opens dashboard| Grafana
```

## Рисунок 2.22. Дашборд Container Monitoring MVP в Grafana

Insert in section 2.4.4.

Screenshot requirements:

- Open Grafana dashboard "Container Monitoring MVP".
- Capture panels: CPU usage, Memory usage, Container events over time, Critical event count, Latest container events table.
- Use this Markdown image placeholder after the screenshot is created:

```markdown
![Рисунок 2.22. Дашборд Container Monitoring MVP в Grafana](images/figure-2-22-grafana-dashboard.png)
```

## Рисунок 2.23. Структура репозитория платформы мониторинга

Insert in section 2.5.1.

```mermaid
flowchart TB
    Repo["container-monitoring"]
    Agent["agent/"]
    Core["core/"]
    Frontend["frontend/"]
    Contracts["contracts/"]
    Deploy["core/deploy/"]
    Docs["docs/"]
    Compose["docker-compose.yml"]
    Makefile["Makefile"]

    Repo --> Agent
    Repo --> Core
    Repo --> Frontend
    Repo --> Contracts
    Core --> Deploy
    Repo --> Docs
    Repo --> Compose
    Repo --> Makefile
```

## Рисунок 2.24. Схема локального Docker Compose окружения платформы

Insert in section 2.5.2.

```mermaid
flowchart TB
    subgraph Compose["Local Docker Compose environment"]
        Agent["agent"]
        Core["core"]
        Kafka["kafka"]
        Postgres["postgres"]
        ClickHouse["clickhouse"]
        Redis["redis"]
        Grafana["grafana"]
        Frontend["frontend"]
        TargetNginx["target-nginx"]
    end

    User["User browser"]
    Docker["Docker Engine API"]

    Agent --> Docker
    Core --> Docker
    TargetNginx --> Docker
    Agent --> Kafka
    Core --> Kafka
    Core --> Postgres
    Core --> ClickHouse
    Core --> Redis
    Grafana --> ClickHouse
    Frontend --> Core
    User --> Frontend
    User --> Grafana
    User --> Core
```

## Рисунок 2.25. ER диаграмма основных сущностей PostgreSQL

Insert in section 2.5.3.

```mermaid
erDiagram
    targets ||--o{ alert_rules : configured_for
    targets ||--o{ events : emits
    targets ||--o{ incidents : affected_by
    alert_rules ||--o{ incidents : opens
    incidents ||--o{ recovery_actions : has

    targets {
        text id PK
        text name
        text type
        text source
        text external_id
        text status
        text node_id
        jsonb labels
        timestamptz last_seen_at
        timestamptz created_at
        timestamptz updated_at
    }

    alert_rules {
        text id PK
        text target_id FK
        text metric_name
        text operator
        float threshold
        interval duration
        text severity
        text recovery_policy
        boolean enabled
        timestamptz created_at
        timestamptz updated_at
    }

    events {
        bigserial id PK
        text node_id
        text target_id FK
        text container_name
        text event_type
        text severity
        text message
        jsonb payload
        timestamptz occurred_at
    }

    incidents {
        bigserial id PK
        text rule_id FK
        text target_id FK
        text node_id
        text status
        text severity
        text description
        float value
        timestamptz started_at
        timestamptz acknowledged_at
        timestamptz resolved_at
    }

    recovery_actions {
        bigserial id PK
        bigint incident_id FK
        text target_id FK
        text action_type
        text status
        timestamptz started_at
        timestamptz finished_at
        text result_message
    }
```

## Рисунок 2.26. Логическая схема хранения метрик и событий в ClickHouse

Insert in section 2.5.3.

```mermaid
classDiagram
    class container_metrics {
        +DateTime64 timestamp
        +String node_id
        +String target_id
        +String container_name
        +Float64 cpu_percent
        +UInt64 memory_usage_bytes
        +UInt64 memory_limit_bytes
        +UInt64 network_rx_bytes
        +UInt64 network_tx_bytes
        +UInt64 block_read_bytes
        +UInt64 block_write_bytes
    }

    class container_events {
        +DateTime64 timestamp
        +String node_id
        +String target_id
        +String container_name
        +String event_type
        +String severity
        +String message
        +String payload_json
    }

    container_metrics ..> container_events : correlated by timestamp, node_id and target_id
```

## Рисунок 2.27. Итоговая схема сборки и запуска платформы мониторинга

Insert in section 2.5.5.

```mermaid
flowchart LR
    Source["Source code"]
    Dockerfile["Dockerfile"]
    Compose["docker-compose.yml"]
    Migrations["Database migrations"]
    Infra["Start infrastructure\nKafka, PostgreSQL, ClickHouse, Redis"]
    Services["Start agent and core"]
    UI["Start frontend and Grafana"]
    Ready["Container Monitoring platform is ready"]

    Source --> Dockerfile
    Dockerfile --> Compose
    Source --> Migrations
    Compose --> Infra
    Migrations --> Infra
    Infra --> Services
    Services --> UI
    UI --> Ready
```
