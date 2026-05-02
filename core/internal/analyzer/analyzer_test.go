package analyzer

import (
	"testing"
	"time"
)

func TestEvaluateCreatesIncidentForMatchingThreshold(t *testing.T) {
	collectedAt := time.Date(2026, 5, 2, 12, 0, 0, 0, time.UTC)
	metric := Metric{
		NodeID:      "node-a",
		ContainerID: "container-a",
		CPUPercent:  95.5,
		CollectedAt: collectedAt,
	}
	rules := []ThresholdRule{{
		ID:        "high-cpu",
		Metric:    MetricCPUPercent,
		Operator:  OperatorGreaterThan,
		Threshold: 90,
	}}

	incidents := Evaluate(metric, rules)

	if len(incidents) != 1 {
		t.Fatalf("len(incidents) = %d, want 1", len(incidents))
	}
	if incidents[0].RuleID != "high-cpu" {
		t.Fatalf("RuleID = %q, want high-cpu", incidents[0].RuleID)
	}
	if incidents[0].StartedAt != collectedAt {
		t.Fatalf("StartedAt = %s, want %s", incidents[0].StartedAt, collectedAt)
	}
}

func TestEvaluateIgnoresNonMatchingThreshold(t *testing.T) {
	metric := Metric{CPUPercent: 40}
	rules := []ThresholdRule{{
		ID:        "high-cpu",
		Metric:    MetricCPUPercent,
		Operator:  OperatorGreaterThan,
		Threshold: 90,
	}}

	incidents := Evaluate(metric, rules)

	if len(incidents) != 0 {
		t.Fatalf("len(incidents) = %d, want 0", len(incidents))
	}
}
