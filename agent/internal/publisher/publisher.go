package publisher

import (
	"context"

	"github.com/nikponomarevan/container-monitoring-agent/internal/collector"
)

type Publisher interface {
	PublishMetrics(ctx context.Context, metrics []collector.Metric) error
	PublishEvent(ctx context.Context, event collector.Event) error
	Close() error
}
