package kafka

import (
	"context"
	"encoding/json"
	"time"

	"github.com/nikponomarevan/container-monitoring-agent/internal/collector"
	"github.com/nikponomarevan/container-monitoring-agent/internal/config"
	kafkago "github.com/segmentio/kafka-go"
)

type Publisher struct {
	cfg config.KafkaConfig
	w   writer
}

type message = kafkago.Message

type writer interface {
	WriteMessages(ctx context.Context, messages ...message) error
	Close() error
}

func NewPublisher(cfg config.KafkaConfig) *Publisher {
	return NewPublisherWithWriter(cfg, &kafkago.Writer{
		Addr:         kafkago.TCP(cfg.Brokers...),
		RequiredAcks: kafkago.RequireOne,
		Async:        false,
	})
}

func NewPublisherWithWriter(cfg config.KafkaConfig, w writer) *Publisher {
	return &Publisher{cfg: cfg, w: w}
}

func (p *Publisher) PublishMetrics(ctx context.Context, metrics []collector.Metric) error {
	messages := make([]message, 0, len(metrics))
	for _, metric := range metrics {
		payload := metricsPayload{
			NodeID:        metric.NodeID,
			Source:        "docker",
			TargetID:      metric.ContainerID,
			ContainerName: metric.Name,
			Metrics: map[string]float64{
				"cpu_usage_percent":    metric.CPUUsagePercent,
				"memory_usage_bytes":   float64(metric.MemoryUsageBytes),
				"memory_usage_percent": metric.MemoryUsagePercent,
				"network_rx_bytes":     float64(metric.NetworkRxBytes),
				"network_tx_bytes":     float64(metric.NetworkTxBytes),
				"block_read_bytes":     float64(metric.BlockReadBytes),
				"block_write_bytes":    float64(metric.BlockWriteBytes),
			},
			Timestamp: metric.CollectedAt,
		}
		value, err := json.Marshal(payload)
		if err != nil {
			return err
		}
		messages = append(messages, message{
			Topic: p.cfg.MetricsTopic,
			Key:   []byte(metric.ContainerID),
			Value: value,
			Time:  metric.CollectedAt,
		})
	}
	if len(messages) == 0 {
		return nil
	}
	return p.w.WriteMessages(ctx, messages...)
}

func (p *Publisher) PublishEvent(ctx context.Context, event collector.Event) error {
	payload := eventPayload{
		NodeID:        event.NodeID,
		Source:        "docker",
		TargetID:      event.ContainerID,
		ContainerName: event.Name,
		EventType:     string(event.Type),
		Severity:      string(event.Severity),
		Message:       event.Message,
		Payload:       event.Payload,
		Timestamp:     event.OccurredAt,
	}
	value, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	return p.w.WriteMessages(ctx, message{
		Topic: p.cfg.EventsTopic,
		Key:   []byte(event.ContainerID),
		Value: value,
		Time:  event.OccurredAt,
	})
}

func (p *Publisher) Close() error {
	return p.w.Close()
}

type metricsPayload struct {
	NodeID        string             `json:"node_id"`
	Source        string             `json:"source"`
	TargetID      string             `json:"target_id"`
	ContainerName string             `json:"container_name"`
	Metrics       map[string]float64 `json:"metrics"`
	Timestamp     time.Time          `json:"timestamp"`
}

type eventPayload struct {
	NodeID        string         `json:"node_id"`
	Source        string         `json:"source"`
	TargetID      string         `json:"target_id"`
	ContainerName string         `json:"container_name"`
	EventType     string         `json:"event_type"`
	Severity      string         `json:"severity"`
	Message       string         `json:"message"`
	Payload       map[string]any `json:"payload"`
	Timestamp     time.Time      `json:"timestamp"`
}
