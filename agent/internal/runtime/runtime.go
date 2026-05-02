package runtime

import (
	"context"
	"log/slog"
	"time"

	"github.com/nikponomarevan/container-monitoring-agent/internal/collector"
	"github.com/nikponomarevan/container-monitoring-agent/internal/config"
	"github.com/nikponomarevan/container-monitoring-agent/internal/publisher"
)

func Run(ctx context.Context, logger *slog.Logger, cfg config.Config, backend collector.Backend, pub publisher.Publisher) error {
	defer pub.Close()

	ticker := time.NewTicker(cfg.CollectInterval)
	defer ticker.Stop()

	events, eventErrs := backend.WatchEvents(ctx)

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			metrics, err := backend.CollectMetrics(ctx)
			if err != nil {
				logger.Error("collect metrics", "error", err)
				continue
			}
			if err := pub.PublishMetrics(ctx, metrics); err != nil {
				logger.Error("publish metrics", "error", err)
			}
		case event, ok := <-events:
			if !ok {
				events = nil
				continue
			}
			if err := pub.PublishEvent(ctx, event); err != nil {
				logger.Error("publish event", "error", err)
			}
		case err, ok := <-eventErrs:
			if ok && err != nil {
				logger.Error("watch docker events", "error", err)
			}
		}
	}
}
