package docker

import (
	"context"

	"github.com/nikponomarevan/container-monitoring-agent/internal/collector"
)

type Backend struct {
	nodeID     string
	dockerHost string
}

func NewBackend(nodeID, dockerHost string) *Backend {
	return &Backend{nodeID: nodeID, dockerHost: dockerHost}
}

func (b *Backend) CollectMetrics(ctx context.Context) ([]collector.Metric, error) {
	_ = ctx
	_ = b.dockerHost

	// Docker API integration is intentionally isolated in this backend.
	return []collector.Metric{}, nil
}

func (b *Backend) WatchEvents(ctx context.Context) (<-chan collector.Event, <-chan error) {
	events := make(chan collector.Event)
	errs := make(chan error, 1)

	go func() {
		defer close(events)
		defer close(errs)
		<-ctx.Done()
	}()

	return events, errs
}
