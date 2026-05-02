package domain

import "time"

type Target struct {
	ID         string         `json:"id"`
	Name       string         `json:"name"`
	Type       string         `json:"type"`
	Source     string         `json:"source"`
	ExternalID string         `json:"external_id"`
	NodeID     string         `json:"node_id"`
	Labels     map[string]any `json:"labels,omitempty"`
	CreatedAt  time.Time      `json:"created_at,omitempty"`
	UpdatedAt  time.Time      `json:"updated_at,omitempty"`
}

type MetricSample struct {
	NodeID        string             `json:"node_id"`
	Source        string             `json:"source"`
	TargetID      string             `json:"target_id"`
	ContainerName string             `json:"container_name"`
	Metrics       map[string]float64 `json:"metrics"`
	Timestamp     time.Time          `json:"timestamp"`
}

type Event struct {
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

type AlertRule struct {
	ID             string        `json:"id"`
	Name           string        `json:"name"`
	MetricName     string        `json:"metric_name"`
	Operator       string        `json:"condition_operator"`
	Threshold      float64       `json:"threshold"`
	Duration       time.Duration `json:"duration"`
	Severity       string        `json:"severity"`
	Enabled        bool          `json:"enabled"`
	RecoveryAction string        `json:"recovery_action"`
}

type Incident struct {
	ID          int64      `json:"id"`
	RuleID      string     `json:"rule_id"`
	TargetID    string     `json:"target_id"`
	NodeID      string     `json:"node_id"`
	Status      string     `json:"status"`
	Severity    string     `json:"severity"`
	Description string     `json:"description"`
	Value       float64    `json:"value"`
	StartedAt   time.Time  `json:"started_at"`
	ResolvedAt  *time.Time `json:"resolved_at,omitempty"`
}

type RecoveryAction struct {
	ID            int64      `json:"id"`
	IncidentID    int64      `json:"incident_id"`
	TargetID      string     `json:"target_id"`
	ActionType    string     `json:"action_type"`
	Status        string     `json:"status"`
	StartedAt     time.Time  `json:"started_at"`
	FinishedAt    *time.Time `json:"finished_at,omitempty"`
	ResultMessage string     `json:"result_message"`
}
