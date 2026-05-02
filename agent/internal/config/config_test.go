package config

import (
	"testing"
	"time"
)

func TestLoadUsesDefaults(t *testing.T) {
	t.Setenv("AGENT_NODE_ID", "")
	t.Setenv("COLLECT_INTERVAL", "")

	cfg := Load()

	if cfg.NodeID != "local-node" {
		t.Fatalf("NodeID = %q, want local-node", cfg.NodeID)
	}
	if cfg.CollectInterval != 10*time.Second {
		t.Fatalf("CollectInterval = %s, want 10s", cfg.CollectInterval)
	}
	if cfg.Kafka.MetricsTopic != "container.metrics" {
		t.Fatalf("MetricsTopic = %q, want container.metrics", cfg.Kafka.MetricsTopic)
	}
}
