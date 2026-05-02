package collector

import (
	"context"
	"time"
)

type Backend interface {
	CollectMetrics(ctx context.Context) ([]Metric, error)
	WatchEvents(ctx context.Context) (<-chan Event, <-chan error)
}

type Metric struct {
	NodeID             string
	ContainerID        string
	Name               string
	CPUUsagePercent    float64
	MemoryUsageBytes   uint64
	MemoryUsagePercent float64
	NetworkRxBytes     uint64
	NetworkTxBytes     uint64
	BlockReadBytes     uint64
	BlockWriteBytes    uint64
	CollectedAt        time.Time
}

type Event struct {
	NodeID      string
	ContainerID string
	Name        string
	Type        EventType
	Severity    Severity
	Message     string
	Payload     map[string]any
	OccurredAt  time.Time
}

type EventType string

const (
	EventStart   EventType = "container_started"
	EventStop    EventType = "container_stopped"
	EventDie     EventType = "container_died"
	EventOOM     EventType = "container_oom"
	EventRestart EventType = "container_restarted"
)

type Severity string

const (
	SeverityInfo     Severity = "info"
	SeverityWarning  Severity = "warning"
	SeverityCritical Severity = "critical"
)
