package analyzer

import "time"

type Metric struct {
	NodeID      string
	ContainerID string
	CPUPercent  float64
	MemoryBytes uint64
	CollectedAt time.Time
}

type ThresholdRule struct {
	ID             string
	Metric         MetricName
	Operator       Operator
	Threshold      float64
	RecoveryAction string
}

type MetricName string

const (
	MetricCPUPercent  MetricName = "cpu_percent"
	MetricMemoryBytes MetricName = "memory_bytes"
)

type Operator string

const (
	OperatorGreaterThan Operator = "gt"
	OperatorLessThan    Operator = "lt"
)

type Incident struct {
	RuleID      string
	NodeID      string
	ContainerID string
	Value       float64
	StartedAt   time.Time
}

func Evaluate(metric Metric, rules []ThresholdRule) []Incident {
	incidents := make([]Incident, 0)
	for _, rule := range rules {
		value, ok := metricValue(metric, rule.Metric)
		if !ok || !matches(value, rule.Operator, rule.Threshold) {
			continue
		}
		incidents = append(incidents, Incident{
			RuleID:      rule.ID,
			NodeID:      metric.NodeID,
			ContainerID: metric.ContainerID,
			Value:       value,
			StartedAt:   metric.CollectedAt,
		})
	}
	return incidents
}

func metricValue(metric Metric, name MetricName) (float64, bool) {
	switch name {
	case MetricCPUPercent:
		return metric.CPUPercent, true
	case MetricMemoryBytes:
		return float64(metric.MemoryBytes), true
	default:
		return 0, false
	}
}

func matches(value float64, operator Operator, threshold float64) bool {
	switch operator {
	case OperatorGreaterThan:
		return value > threshold
	case OperatorLessThan:
		return value < threshold
	default:
		return false
	}
}
