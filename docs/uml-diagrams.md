# UML diagrams for chapter 2

This document contains the required figures for chapter 2 of the diploma.
Most diagrams are written in Mermaid so they can be rendered directly from Markdown.
Figures marked as screenshots define the exact screen that should be captured from the running platform.

## Рисунок 2.1. Компонентная диаграмма агентской подсистемы сбора данных

Insert in section 2.2.

```mermaid
%%{init: {"flowchart": {"defaultRenderer": "elk"}} }%%
flowchart LR
    DockerAPI["Docker Engine API"]

    subgraph Agent["Go Agent"]
        direction TB
        ConfigLoader["Config Loader"]
        Runtime["Agent Runtime"]
        subgraph Collectors["Data collection"]
            direction TB
            DockerCollector["Docker Collector"]
            EventWatcher["Docker Event Watcher"]
        end
        KafkaPublisher["Kafka Publisher"]
    end

    subgraph Kafka["Kafka"]
        direction TB
        MetricsTopic[("container.metrics")]
        EventsTopic[("container.events")]
    end

    Core["Core Service"]

    ConfigLoader --> Runtime
    Runtime --> DockerCollector
    Runtime --> EventWatcher
    DockerAPI --> DockerCollector
    DockerAPI --> EventWatcher
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
%%{init: {"flowchart": {"defaultRenderer": "elk"}} }%%
flowchart LR
    Cmd["package cmd/agent"]

    subgraph Internal["internal"]
        direction TB
        Config["package internal/config"]
        Runtime["package internal/runtime"]
        Collector["package internal/collector/docker"]
        Publisher["package internal/publisher/kafka"]
    end

    Cmd --> Config
    Cmd --> Runtime
    Runtime --> Collector
    Runtime --> Publisher
    Runtime --> Config
```

## Рисунок 2.3. Алгоритм периодического сбора метрик контейнеров

Insert in section 2.2.2.

```mermaid
%%{init: {"flowchart": {"defaultRenderer": "elk"}} }%%
flowchart TB
    Start([Start agent runtime])
    Ticker["Start collect ticker"]
    Tick["Wait for ticker tick"]
    List["Get container list"]
    ListError{"List error?"}
    Stats["Request Docker stats"]
    StatsError{"Stats error?"}
    Normalize["Normalize metric sample"]
    Batch["Append sample to batch"]
    PublishDecision{"Batch has samples?"}
    Kafka["Publish batch to container.metrics"]
    NextTick["Wait for next tick"]
    LogList["Log Docker list error"]
    LogStats["Log stats error"]

    Start --> Ticker --> Tick --> List --> ListError
    ListError -- yes --> LogList --> NextTick
    ListError -- no --> Stats
    Stats --> StatsError
    StatsError -- no --> Normalize --> Batch --> PublishDecision
    StatsError -- yes --> LogStats --> PublishDecision
    PublishDecision -- yes --> Kafka --> NextTick
    PublishDecision -- no --> NextTick
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
%%{init: {"flowchart": {"defaultRenderer": "elk"}} }%%
flowchart LR
    Stats["Docker stats"]
    Events["Docker events"]

    subgraph Agent["Go Agent"]
        direction TB
        Collector["Metric collector"]
        Watcher["Event watcher"]
        Publisher["Kafka publisher"]
    end

    subgraph Kafka["Kafka"]
        direction TB
        MetricsTopic[("container.metrics")]
        EventsTopic[("container.events")]
    end

    Core["Core Service"]

    Stats --> Collector --> Publisher --> MetricsTopic --> Core
    Events --> Watcher --> Publisher --> EventsTopic --> Core
```

## Рисунок 2.6. Жизненный цикл Go Agent

Insert in section 2.2.5.

```mermaid
%%{init: {"flowchart": {"defaultRenderer": "elk"}} }%%
flowchart TB
    Start([Process start])

    subgraph Startup["Startup"]
        direction TB
        Load["Load configuration"]
        Collector["Initialize Docker collector"]
        Publisher["Initialize Kafka publisher"]
        Runtime["Start agent runtime"]
    end

    subgraph Work["Concurrent runtime work"]
        direction TB
        Metrics["Collect metrics on ticker"]
        Events["Read Docker events stream"]
        PublishMetrics["Publish to container.metrics"]
        PublishEvents["Publish to container.events"]
    end

    Signal["Receive SIGTERM"]
    Close["Close Kafka publisher"]
    Stop([Process stopped])

    Start --> Load --> Collector --> Publisher --> Runtime
    Runtime --> Metrics --> PublishMetrics
    Runtime --> Events --> PublishEvents
    PublishMetrics --> Signal
    PublishEvents --> Signal
    Signal --> Close --> Stop
```

## Рисунок 2.7. Компонентная диаграмма центрального сервиса обработки данных

Insert in section 2.3.

```mermaid
%%{init: {"flowchart": {"defaultRenderer": "elk"}} }%%
flowchart LR
    Kafka[("Kafka topics")]

    subgraph Core["Core Service"]
        direction LR
        Consumer["Kafka Consumer"]
        HTTPAPI["HTTP API"]

        subgraph Processing["Processing"]
            direction TB
            MetricHandler["Metric Handler"]
            EventHandler["Event Handler"]
            Analyzer["Analyzer"]
            IncidentService["Incident Service"]
            RecoveryService["Recovery Service"]
            Notifier["Notifier"]
        end

        subgraph Repositories["Repositories and state"]
            direction TB
            PostgresRepo["PostgreSQL Repository"]
            ClickHouseRepo["ClickHouse Repository"]
            RedisState["Redis State Storage"]
        end
    end

    Kafka --> Consumer

    Consumer --> MetricHandler
    Consumer --> EventHandler
    MetricHandler --> Analyzer
    EventHandler --> IncidentService
    Analyzer --> IncidentService
    IncidentService --> RecoveryService
    IncidentService --> Notifier
    HTTPAPI --> RecoveryService
    MetricHandler --> ClickHouseRepo
    MetricHandler --> RedisState
    MetricHandler --> PostgresRepo
    EventHandler --> PostgresRepo
    EventHandler --> ClickHouseRepo
    EventHandler --> RedisState
    IncidentService --> PostgresRepo
    RecoveryService --> PostgresRepo
    RecoveryService --> RedisState
    HTTPAPI --> IncidentService
    HTTPAPI --> PostgresRepo
    HTTPAPI --> ClickHouseRepo
    HTTPAPI --> RedisState
```

## Рисунок 2.8. UML диаграмма пакетов центрального сервиса

Insert in section 2.3.1.

```mermaid
%%{init: {"flowchart": {"defaultRenderer": "elk"}} }%%
flowchart LR
    Cmd["package cmd/core"]

    subgraph Entry["entrypoints and adapters"]
        direction TB
        Config["package internal/config"]
        Consumer["package internal/consumer"]
        HTTP["package internal/http"]
    end

    Service["package internal/service"]

    subgraph Integrations["repositories and integrations"]
        direction TB
        Postgres["package internal/repository/postgres"]
        ClickHouse["package internal/repository/clickhouse"]
        Redis["package internal/state/redis"]
        Notifier["package internal/notifier"]
        Recovery["package internal/recovery"]
    end

    Cmd --> Config
    Cmd --> Consumer
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
%%{init: {"flowchart": {"defaultRenderer": "elk"}} }%%
flowchart LR
    Core["Core Service"]

    subgraph PostgreSQL["PostgreSQL"]
        direction TB
        Targets["targets"]
        Rules["alert_rules"]
        PgEvents["events"]
        Incidents["incidents"]
        Actions["recovery_actions"]
    end

    subgraph ClickHouse["ClickHouse"]
        direction TB
        Metrics["container_metrics"]
        AnalyticalEvents["analytical events"]
    end

    subgraph Redis["Redis"]
        direction TB
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
%%{init: {"flowchart": {"defaultRenderer": "elk"}} }%%
flowchart TB
    Start([Metric received])
    LoadRules["Load enabled alert rules"]
    Applicable{"Applicable?"}
    Threshold{"Threshold matched?"}
    AlertIncident["Create alert event and incident"]
    Notify["Send notification"]
    Recovery{"Recovery policy exists?"}
    Trigger["Trigger recovery action"]
    Finish([Evaluation finished])

    Start --> LoadRules --> Applicable
    Applicable -- no --> Finish
    Applicable -- yes --> Threshold
    Threshold -- no --> Finish
    Threshold -- yes --> AlertIncident --> Notify --> Recovery
    Recovery -- yes --> Trigger --> Finish
    Recovery -- no --> Finish
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
%%{init: {"flowchart": {"defaultRenderer": "elk"}} }%%
flowchart LR
    User["User"]

    subgraph UI["User interfaces"]
        direction TB
        Frontend["Frontend admin panel"]
        Swagger["Swagger UI"]
        Grafana["Grafana"]
    end

    HTTP["Core HTTP API"]

    subgraph Storage["Platform storages"]
        direction TB
        PostgreSQL["PostgreSQL"]
        ClickHouse["ClickHouse"]
        Redis["Redis"]
    end

    User --> Frontend --> HTTP
    User --> Swagger --> HTTP
    User --> Grafana --> ClickHouse
    HTTP --> PostgreSQL
    HTTP --> ClickHouse
    HTTP --> Redis
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
%%{init: {"flowchart": {"defaultRenderer": "elk"}} }%%
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
%%{init: {"flowchart": {"defaultRenderer": "elk"}} }%%
flowchart LR
    Repo["container-monitoring"]

    subgraph Apps["Applications"]
        direction TB
        Agent["agent/"]
        Core["core/"]
        Frontend["frontend/"]
    end

    subgraph Support["Shared and documentation"]
        direction TB
        Contracts["contracts/"]
        Deploy["core/deploy/"]
        Docs["docs/"]
    end

    subgraph RootFiles["Root orchestration files"]
        direction TB
        Compose["docker-compose.yml"]
        Makefile["Makefile"]
    end

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
%%{init: {"flowchart": {"defaultRenderer": "elk"}} }%%
flowchart LR
    User["User browser"]
    Docker["Docker Engine API"]

    subgraph Compose["Local Docker Compose environment"]
        direction LR
        subgraph Runtime["Runtime services"]
            direction TB
            Agent["agent"]
            Core["core"]
            Frontend["frontend"]
            TargetNginx["target-nginx"]
        end

        subgraph Infrastructure["Infrastructure services"]
            direction TB
            Kafka["kafka"]
            Postgres["postgres"]
            ClickHouse["clickhouse"]
            Redis["redis"]
            Grafana["grafana"]
        end
    end

    User --> Frontend
    User --> Grafana
    Frontend --> Core
    Agent --> Kafka
    Core --> Kafka
    Core --> Postgres
    Core --> ClickHouse
    Core --> Redis
    Grafana --> ClickHouse
    Agent --> Docker
    Core --> Docker
    TargetNginx --> Docker
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
%%{init: {"flowchart": {"defaultRenderer": "elk"}} }%%
flowchart TB
    Source["Source code"]

    subgraph BuildInputs["Build inputs"]
        direction LR
        Dockerfile["Dockerfile"]
        Compose["docker-compose.yml"]
        Migrations["Database migrations"]
    end

    Build["Build service images"]
    Infra["Start infrastructure"]
    ApplyMigrations["Apply migrations"]
    Services["Start agent and core"]
    UI["Start frontend and Grafana"]
    Ready["Container Monitoring platform is ready"]

    Source --> Dockerfile
    Source --> Compose
    Source --> Migrations
    Dockerfile --> Build
    Compose --> Infra
    Migrations --> ApplyMigrations
    Build --> Services
    Infra --> ApplyMigrations --> Services
    Services --> UI
    UI --> Ready
```
