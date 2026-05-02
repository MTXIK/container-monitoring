package kafka

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/nikponomarevan/container-monitoring-agent/internal/collector"
	"github.com/nikponomarevan/container-monitoring-agent/internal/config"
)

type capturedWriter struct {
	messages []message
}

func (w *capturedWriter) WriteMessages(_ context.Context, messages ...message) error {
	w.messages = append(w.messages, messages...)
	return nil
}

func (w *capturedWriter) Close() error { return nil }

func TestPublishMetricsWritesContainerMetricsPayload(t *testing.T) {
	writer := &capturedWriter{}
	publisher := NewPublisherWithWriter(config.KafkaConfig{
		MetricsTopic: "container.metrics",
		EventsTopic:  "container.events",
	}, writer)
	collectedAt := time.Date(2026, 5, 2, 12, 0, 0, 0, time.UTC)

	err := publisher.PublishMetrics(context.Background(), []collector.Metric{{
		NodeID:             "node-1",
		ContainerID:        "container-id",
		Name:               "nginx",
		CPUUsagePercent:    12.4,
		MemoryUsageBytes:   104857600,
		MemoryUsagePercent: 25.8,
		NetworkRxBytes:     123456,
		NetworkTxBytes:     78910,
		BlockReadBytes:     4567,
		BlockWriteBytes:    8910,
		CollectedAt:        collectedAt,
	}})

	if err != nil {
		t.Fatalf("PublishMetrics() error = %v", err)
	}
	if len(writer.messages) != 1 {
		t.Fatalf("len(messages) = %d, want 1", len(writer.messages))
	}
	if writer.messages[0].Topic != "container.metrics" {
		t.Fatalf("Topic = %q, want container.metrics", writer.messages[0].Topic)
	}

	var payload struct {
		NodeID        string             `json:"node_id"`
		Source        string             `json:"source"`
		TargetID      string             `json:"target_id"`
		ContainerName string             `json:"container_name"`
		Metrics       map[string]float64 `json:"metrics"`
		Timestamp     time.Time          `json:"timestamp"`
	}
	if err := json.Unmarshal(writer.messages[0].Value, &payload); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if payload.Source != "docker" || payload.TargetID != "container-id" || payload.ContainerName != "nginx" {
		t.Fatalf("payload identity = %#v", payload)
	}
	if payload.Metrics["cpu_usage_percent"] != 12.4 {
		t.Fatalf("cpu_usage_percent = %v, want 12.4", payload.Metrics["cpu_usage_percent"])
	}
	if payload.Metrics["memory_usage_percent"] != 25.8 {
		t.Fatalf("memory_usage_percent = %v, want 25.8", payload.Metrics["memory_usage_percent"])
	}
	if !payload.Timestamp.Equal(collectedAt) {
		t.Fatalf("timestamp = %s, want %s", payload.Timestamp, collectedAt)
	}
}

func TestPublishEventWritesContainerEventPayload(t *testing.T) {
	writer := &capturedWriter{}
	publisher := NewPublisherWithWriter(config.KafkaConfig{
		MetricsTopic: "container.metrics",
		EventsTopic:  "container.events",
	}, writer)
	occurredAt := time.Date(2026, 5, 2, 12, 1, 0, 0, time.UTC)

	err := publisher.PublishEvent(context.Background(), collector.Event{
		NodeID:      "node-1",
		ContainerID: "container-id",
		Name:        "nginx",
		Type:        collector.EventDie,
		Severity:    collector.SeverityCritical,
		Message:     "Container nginx died",
		Payload:     map[string]any{"exit_code": float64(137), "oom_killed": true},
		OccurredAt:  occurredAt,
	})

	if err != nil {
		t.Fatalf("PublishEvent() error = %v", err)
	}
	if len(writer.messages) != 1 {
		t.Fatalf("len(messages) = %d, want 1", len(writer.messages))
	}
	if writer.messages[0].Topic != "container.events" {
		t.Fatalf("Topic = %q, want container.events", writer.messages[0].Topic)
	}

	var payload struct {
		EventType string         `json:"event_type"`
		Severity  string         `json:"severity"`
		Message   string         `json:"message"`
		Payload   map[string]any `json:"payload"`
	}
	if err := json.Unmarshal(writer.messages[0].Value, &payload); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if payload.EventType != "container_died" {
		t.Fatalf("event_type = %q, want container_died", payload.EventType)
	}
	if payload.Severity != "critical" {
		t.Fatalf("severity = %q, want critical", payload.Severity)
	}
	if payload.Message != "Container nginx died" {
		t.Fatalf("message = %q", payload.Message)
	}
}
