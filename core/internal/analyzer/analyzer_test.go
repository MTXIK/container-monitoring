package analyzer

import (
	"testing"
	"time"
)

func TestEvaluateCreatesIncidentForMatchingThreshold(t *testing.T) {
	collectedAt := time.Date(2026, 5, 2, 12, 0, 0, 0, time.UTC)
	metric := Metric{
		NodeID:    "node-a",
		TargetID:  "container-a",
		Values:    map[string]float64{"cpu_usage_percent": 95.5},
		Timestamp: collectedAt,
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
	if incidents[0].TargetID != "container-a" {
		t.Fatalf("TargetID = %q, want container-a", incidents[0].TargetID)
	}
	if incidents[0].StartedAt != collectedAt {
		t.Fatalf("StartedAt = %s, want %s", incidents[0].StartedAt, collectedAt)
	}
}

func TestEvaluateIgnoresNonMatchingThreshold(t *testing.T) {
	metric := Metric{Values: map[string]float64{"cpu_usage_percent": 40}}
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

func TestEvaluateAppliesTargetFilter(t *testing.T) {
	metric := Metric{
		TargetID: "container-a",
		Values:   map[string]float64{"cpu_usage_percent": 95},
	}
	rules := []ThresholdRule{{
		ID:        "high-cpu",
		TargetID:  "container-b",
		Metric:    MetricCPUPercent,
		Operator:  OperatorGreaterThan,
		Threshold: 90,
	}}

	incidents := Evaluate(metric, rules)

	if len(incidents) != 0 {
		t.Fatalf("len(incidents) = %d, want 0", len(incidents))
	}
}

func TestEvaluateSupportsInclusiveAndEqualityOperators(t *testing.T) {
	metric := Metric{
		TargetID: "container-a",
		Values: map[string]float64{
			"cpu_usage_percent":    90,
			"memory_usage_percent": 75,
			"network_rx_bytes":     1024,
		},
	}
	rules := []ThresholdRule{
		{ID: "gte-cpu", Metric: MetricCPUPercent, Operator: OperatorGreaterThanOrEqual, Threshold: 90},
		{ID: "lte-memory", Metric: MetricMemoryUsagePercent, Operator: OperatorLessThanOrEqual, Threshold: 75},
		{ID: "eq-rx", Metric: MetricNetworkRxBytes, Operator: OperatorEqual, Threshold: 1024},
	}

	incidents := Evaluate(metric, rules)

	if len(incidents) != 3 {
		t.Fatalf("len(incidents) = %d, want 3", len(incidents))
	}
}
