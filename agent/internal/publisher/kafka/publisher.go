package kafka

import (
	"context"

	"github.com/nikponomarevan/container-monitoring-agent/internal/collector"
	"github.com/nikponomarevan/container-monitoring-agent/internal/config"
)

type Publisher struct {
	cfg config.KafkaConfig
}

func NewPublisher(cfg config.KafkaConfig) *Publisher {
	return &Publisher{cfg: cfg}
}

func (p *Publisher) PublishMetrics(ctx context.Context, metrics []collector.Metric) error {
	_ = ctx
	_ = metrics
	_ = p.cfg.MetricsTopic
	return nil
}

func (p *Publisher) PublishEvent(ctx context.Context, event collector.Event) error {
	_ = ctx
	_ = event
	_ = p.cfg.EventsTopic
	return nil
}

func (p *Publisher) Close() error {
	return nil
}
