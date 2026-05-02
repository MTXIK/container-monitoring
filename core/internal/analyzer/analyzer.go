package analyzer

import "time"

type Metric struct {
	NodeID    string
	TargetID  string
	Values    map[string]float64
	Timestamp time.Time
}

type ThresholdRule struct {
	ID             string
	Metric         MetricName
	Operator       Operator
	Threshold      float64
	Severity       string
	RecoveryAction string
}

type MetricName string

const (
	MetricCPUUsagePercent    MetricName = "cpu_usage_percent"
	MetricMemoryUsageBytes   MetricName = "memory_usage_bytes"
	MetricMemoryUsagePercent MetricName = "memory_usage_percent"
	MetricNetworkRxBytes     MetricName = "network_rx_bytes"
	MetricNetworkTxBytes     MetricName = "network_tx_bytes"
	MetricBlockReadBytes     MetricName = "block_read_bytes"
	MetricBlockWriteBytes    MetricName = "block_write_bytes"

	MetricCPUPercent  MetricName = MetricCPUUsagePercent
	MetricMemoryBytes MetricName = MetricMemoryUsageBytes
)

type Operator string

const (
	OperatorGreaterThan Operator = "gt"
	OperatorLessThan    Operator = "lt"
)

type Incident struct {
	RuleID         string
	NodeID         string
	TargetID       string
	MetricName     string
	Value          float64
	Severity       string
	RecoveryAction string
	StartedAt      time.Time
}

func Evaluate(metric Metric, rules []ThresholdRule) []Incident {
	incidents := make([]Incident, 0)
	for _, rule := range rules {
		value, ok := metricValue(metric, rule.Metric)
		if !ok || !matches(value, rule.Operator, rule.Threshold) {
			continue
		}
		incidents = append(incidents, Incident{
			RuleID:         rule.ID,
			NodeID:         metric.NodeID,
			TargetID:       metric.TargetID,
			MetricName:     string(rule.Metric),
			Value:          value,
			Severity:       rule.Severity,
			RecoveryAction: rule.RecoveryAction,
			StartedAt:      metric.Timestamp,
		})
	}
	return incidents
}

func metricValue(metric Metric, name MetricName) (float64, bool) {
	value, ok := metric.Values[string(name)]
	return value, ok
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
