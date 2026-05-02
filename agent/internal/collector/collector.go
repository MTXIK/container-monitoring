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
	NodeID      string
	ContainerID string
	Name        string
	CPUPercent  float64
	MemoryBytes uint64
	RxBytes     uint64
	TxBytes     uint64
	BlockRead   uint64
	BlockWrite  uint64
	CollectedAt time.Time
}

type Event struct {
	NodeID      string
	ContainerID string
	Name        string
	Type        EventType
	OccurredAt  time.Time
}

type EventType string

const (
	EventStart EventType = "start"
	EventStop  EventType = "stop"
	EventDie   EventType = "die"
	EventOOM   EventType = "oom"
)
