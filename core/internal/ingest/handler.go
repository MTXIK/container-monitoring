package ingest

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/nikponomarevan/container-monitoring-core/internal/analyzer"
	"github.com/nikponomarevan/container-monitoring-core/internal/consumer/kafka"
	"github.com/nikponomarevan/container-monitoring-core/internal/domain"
)

type Config struct {
	MetricsTopic string
	EventsTopic  string
}

type Store interface {
	SaveMetric(ctx context.Context, metric domain.MetricSample) error
	SaveEvent(ctx context.Context, event domain.Event) error
	UpsertTarget(ctx context.Context, target domain.Target) error
	EnabledRules(ctx context.Context) ([]analyzer.ThresholdRule, error)
	HasOpenIncident(ctx context.Context, ruleID, targetID string) (bool, error)
	CreateIncident(ctx context.Context, incident domain.Incident) (domain.Incident, error)
}

type State interface {
	SetLatestMetrics(ctx context.Context, metric domain.MetricSample) error
	SetTargetState(ctx context.Context, event domain.Event) error
	AlertThresholdStartedAt(ctx context.Context, ruleID, targetID string, matchedAt time.Time) (time.Time, error)
	ClearAlertThreshold(ctx context.Context, ruleID, targetID string) error
}

type Notifier interface {
	SendIncident(ctx context.Context, text string) error
}

type Recoverer interface {
	Recover(ctx context.Context, incident domain.Incident, action string) error
}

type Handler struct {
	cfg       Config
	store     Store
	state     State
	notifier  Notifier
	recoverer Recoverer
}

func NewHandler(cfg Config, store Store, state State, notifier Notifier, recoverers ...Recoverer) *Handler {
	var recoverer Recoverer
	if len(recoverers) > 0 {
		recoverer = recoverers[0]
	}
	return &Handler{cfg: cfg, store: store, state: state, notifier: notifier, recoverer: recoverer}
}

func (h *Handler) Handle(ctx context.Context, message kafka.Message) error {
	switch message.Topic {
	case h.cfg.MetricsTopic:
		return h.handleMetric(ctx, message.Value)
	case h.cfg.EventsTopic:
		return h.handleEvent(ctx, message.Value)
	default:
		return nil
	}
}

func (h *Handler) handleMetric(ctx context.Context, value []byte) error {
	var metric domain.MetricSample
	if err := json.Unmarshal(value, &metric); err != nil {
		return fmt.Errorf("decode metric: %w", err)
	}
	if metric.TargetID == "" {
		return fmt.Errorf("decode metric: target_id is required")
	}
	if err := h.store.UpsertTarget(ctx, targetFromMetric(metric)); err != nil {
		return err
	}
	if err := h.store.SaveMetric(ctx, metric); err != nil {
		return err
	}
	if h.state != nil {
		if err := h.state.SetLatestMetrics(ctx, metric); err != nil {
			return err
		}
	}

	rules, err := h.store.EnabledRules(ctx)
	if err != nil {
		return err
	}
	incidents := analyzer.Evaluate(analyzer.Metric{
		NodeID:    metric.NodeID,
		TargetID:  metric.TargetID,
		Values:    metric.Metrics,
		Timestamp: metric.Timestamp,
	}, rules)
	if h.state != nil {
		clearNonMatchingRules(ctx, h.state, metric.TargetID, rules, incidents)
	}
	for _, candidate := range incidents {
		ready, err := h.alertReady(ctx, candidate)
		if err != nil {
			return err
		}
		if !ready {
			continue
		}
		exists, err := h.store.HasOpenIncident(ctx, candidate.RuleID, candidate.TargetID)
		if err != nil {
			return err
		}
		if exists {
			continue
		}
		incident, err := h.store.CreateIncident(ctx, domain.Incident{
			RuleID:      candidate.RuleID,
			TargetID:    candidate.TargetID,
			NodeID:      candidate.NodeID,
			Status:      "open",
			Severity:    candidate.Severity,
			Description: fmt.Sprintf("%s %s %.2f", candidate.MetricName, "threshold matched", candidate.Value),
			Value:       candidate.Value,
			StartedAt:   candidate.StartedAt,
		})
		if err != nil {
			return err
		}
		if h.notifier != nil {
			if err := h.notifier.SendIncident(ctx, formatIncident(metric, incident, candidate)); err != nil {
				return err
			}
		}
		if h.recoverer != nil && candidate.RecoveryAction != "" {
			if err := h.recoverer.Recover(ctx, incident, candidate.RecoveryAction); err != nil {
				return err
			}
		}
	}
	return nil
}

func (h *Handler) handleEvent(ctx context.Context, value []byte) error {
	var event domain.Event
	if err := json.Unmarshal(value, &event); err != nil {
		return fmt.Errorf("decode event: %w", err)
	}
	if event.TargetID == "" {
		return fmt.Errorf("decode event: target_id is required")
	}
	if err := h.store.UpsertTarget(ctx, targetFromEvent(event)); err != nil {
		return err
	}
	if err := h.store.SaveEvent(ctx, event); err != nil {
		return err
	}
	if h.state != nil {
		if err := h.state.SetTargetState(ctx, event); err != nil {
			return err
		}
	}
	return h.createIncidentForEvent(ctx, event)
}

func targetFromMetric(metric domain.MetricSample) domain.Target {
	return domain.Target{
		ID:         metric.TargetID,
		Name:       metric.ContainerName,
		Type:       "container",
		Source:     metric.Source,
		ExternalID: metric.TargetID,
		NodeID:     metric.NodeID,
		Status:     "",
		LastSeenAt: metric.Timestamp,
	}
}

func targetFromEvent(event domain.Event) domain.Target {
	return domain.Target{
		ID:         event.TargetID,
		Name:       event.ContainerName,
		Type:       "container",
		Source:     event.Source,
		ExternalID: event.TargetID,
		NodeID:     event.NodeID,
		Status:     targetStatusForEvent(event),
		LastSeenAt: event.Timestamp,
	}
}

func targetStatusForEvent(event domain.Event) string {
	switch event.EventType {
	case "container_started", "container_restarted":
		return "OK"
	case "container_stopped", "container_died", "container_oom":
		return "CRITICAL"
	}
	switch event.Severity {
	case "critical":
		return "CRITICAL"
	case "warning":
		return "WARNING"
	default:
		return "UNKNOWN"
	}
}

func formatIncident(metric domain.MetricSample, incident domain.Incident, candidate analyzer.Incident) string {
	return fmt.Sprintf("%s: container %s on %s\nReason: %s threshold matched\nCurrent value: %.2f\nIncident: %d\nRecovery: %s",
		incident.Severity,
		metric.ContainerName,
		metric.NodeID,
		candidate.MetricName,
		candidate.Value,
		incident.ID,
		candidate.RecoveryAction,
	)
}

func (h *Handler) createIncidentForEvent(ctx context.Context, event domain.Event) error {
	action := recoveryActionForEvent(event.EventType)
	if action == "" {
		return nil
	}
	exists, err := h.store.HasOpenIncident(ctx, event.EventType, event.TargetID)
	if err != nil {
		return err
	}
	if exists {
		return nil
	}
	incident, err := h.store.CreateIncident(ctx, domain.Incident{
		RuleID:      event.EventType,
		TargetID:    event.TargetID,
		NodeID:      event.NodeID,
		Status:      "open",
		Severity:    event.Severity,
		Description: event.Message,
		Value:       1,
		StartedAt:   event.Timestamp,
	})
	if err != nil {
		return err
	}
	if h.notifier != nil {
		if err := h.notifier.SendIncident(ctx, formatEventIncident(event, incident, action)); err != nil {
			return err
		}
	}
	if h.recoverer != nil {
		return h.recoverer.Recover(ctx, incident, action)
	}
	return nil
}

func (h *Handler) alertReady(ctx context.Context, candidate analyzer.Incident) (bool, error) {
	if candidate.Duration <= 0 || h.state == nil {
		return true, nil
	}
	startedAt, err := h.state.AlertThresholdStartedAt(ctx, candidate.RuleID, candidate.TargetID, candidate.StartedAt)
	if err != nil {
		return false, err
	}
	return !candidate.StartedAt.Before(startedAt.Add(candidate.Duration)), nil
}

func clearNonMatchingRules(ctx context.Context, state State, targetID string, rules []analyzer.ThresholdRule, incidents []analyzer.Incident) {
	matched := make(map[string]struct{}, len(incidents))
	for _, incident := range incidents {
		matched[incident.RuleID] = struct{}{}
	}
	for _, rule := range rules {
		if rule.Duration <= 0 {
			continue
		}
		if rule.TargetID != "" && rule.TargetID != targetID {
			continue
		}
		if _, ok := matched[rule.ID]; ok {
			continue
		}
		_ = state.ClearAlertThreshold(ctx, rule.ID, targetID)
	}
}

func recoveryActionForEvent(eventType string) string {
	switch eventType {
	case "container_stopped", "container_died", "container_oom":
		return "restart_container"
	default:
		return ""
	}
}

func formatEventIncident(event domain.Event, incident domain.Incident, action string) string {
	return fmt.Sprintf("%s: container %s on %s\nReason: %s\nIncident: %d\nRecovery: %s",
		incident.Severity,
		event.ContainerName,
		event.NodeID,
		event.EventType,
		incident.ID,
		action,
	)
}
