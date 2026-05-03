package ingest

import (
	"context"
	"testing"
	"time"

	"github.com/nikponomarevan/container-monitoring-core/internal/analyzer"
	"github.com/nikponomarevan/container-monitoring-core/internal/consumer/kafka"
	"github.com/nikponomarevan/container-monitoring-core/internal/domain"
)

type recordingStore struct {
	metrics   []domain.MetricSample
	events    []domain.Event
	targets   []domain.Target
	incidents []domain.Incident
	rules     []analyzer.ThresholdRule
}

func (s *recordingStore) SaveMetric(_ context.Context, metric domain.MetricSample) error {
	s.metrics = append(s.metrics, metric)
	return nil
}

func (s *recordingStore) SaveEvent(_ context.Context, event domain.Event) error {
	s.events = append(s.events, event)
	return nil
}

func (s *recordingStore) UpsertTarget(_ context.Context, target domain.Target) error {
	s.targets = append(s.targets, target)
	return nil
}

func (s *recordingStore) EnabledRules(_ context.Context) ([]analyzer.ThresholdRule, error) {
	return s.rules, nil
}

func (s *recordingStore) HasOpenIncident(_ context.Context, ruleID, targetID string) (bool, error) {
	for _, incident := range s.incidents {
		if incident.RuleID == ruleID && incident.TargetID == targetID && incident.Status != "resolved" {
			return true, nil
		}
	}
	return false, nil
}

func (s *recordingStore) CreateIncident(_ context.Context, incident domain.Incident) (domain.Incident, error) {
	incident.ID = int64(len(s.incidents) + 1)
	s.incidents = append(s.incidents, incident)
	return incident, nil
}

type recordingState struct {
	latest      domain.MetricSample
	states      []domain.Event
	alertStarts map[string]time.Time
}

func (s *recordingState) SetLatestMetrics(_ context.Context, metric domain.MetricSample) error {
	s.latest = metric
	return nil
}

func (s *recordingState) SetTargetState(_ context.Context, event domain.Event) error {
	s.states = append(s.states, event)
	return nil
}

func (s *recordingState) AlertThresholdStartedAt(_ context.Context, ruleID, targetID string, matchedAt time.Time) (time.Time, error) {
	if s.alertStarts == nil {
		s.alertStarts = map[string]time.Time{}
	}
	key := ruleID + ":" + targetID
	if startedAt, ok := s.alertStarts[key]; ok {
		return startedAt, nil
	}
	s.alertStarts[key] = matchedAt
	return matchedAt, nil
}

func (s *recordingState) ClearAlertThreshold(_ context.Context, ruleID, targetID string) error {
	if s.alertStarts != nil {
		delete(s.alertStarts, ruleID+":"+targetID)
	}
	return nil
}

type recordingNotifier struct {
	texts []string
}

func (n *recordingNotifier) SendIncident(ctx context.Context, text string) error {
	n.texts = append(n.texts, text)
	return nil
}

func TestHandlerProcessesMetricMessageAndCreatesIncident(t *testing.T) {
	store := &recordingStore{rules: []analyzer.ThresholdRule{{
		ID:             "high-cpu",
		Metric:         analyzer.MetricCPUUsagePercent,
		Operator:       analyzer.OperatorGreaterThan,
		Threshold:      80,
		Severity:       "critical",
		RecoveryAction: "notify_only",
	}}}
	state := &recordingState{}
	notifier := &recordingNotifier{}
	handler := NewHandler(Config{
		MetricsTopic: "container.metrics",
		EventsTopic:  "container.events",
	}, store, state, notifier)

	err := handler.Handle(context.Background(), kafka.Message{
		Topic: "container.metrics",
		Value: []byte(`{
			"node_id":"node-1",
			"source":"docker",
			"target_id":"container-id",
			"container_name":"nginx",
			"metrics":{"cpu_usage_percent":91.5,"memory_usage_bytes":100},
			"timestamp":"2026-05-02T12:00:00Z"
		}`),
	})

	if err != nil {
		t.Fatalf("Handle() error = %v", err)
	}
	if len(store.metrics) != 1 {
		t.Fatalf("saved metrics = %d, want 1", len(store.metrics))
	}
	if len(store.targets) != 1 || store.targets[0].Source != "docker" || store.targets[0].Type != "container" {
		t.Fatalf("targets = %#v", store.targets)
	}
	if state.latest.TargetID != "container-id" {
		t.Fatalf("latest target = %q, want container-id", state.latest.TargetID)
	}
	if len(store.incidents) != 1 {
		t.Fatalf("incidents = %d, want 1", len(store.incidents))
	}
	if store.incidents[0].Severity != "critical" || store.incidents[0].Value != 91.5 {
		t.Fatalf("incident = %#v", store.incidents[0])
	}
	if len(notifier.texts) != 1 {
		t.Fatalf("notifications = %d, want 1", len(notifier.texts))
	}
}

func TestHandlerProcessesEventMessage(t *testing.T) {
	store := &recordingStore{}
	state := &recordingState{}
	handler := NewHandler(Config{
		MetricsTopic: "container.metrics",
		EventsTopic:  "container.events",
	}, store, state, nil)

	err := handler.Handle(context.Background(), kafka.Message{
		Topic: "container.events",
		Value: []byte(`{
			"node_id":"node-1",
			"source":"docker",
			"target_id":"container-id",
			"container_name":"nginx",
			"event_type":"container_died",
			"severity":"critical",
			"message":"Container nginx died",
			"payload":{"exit_code":137},
			"timestamp":"2026-05-02T12:00:00Z"
		}`),
	})

	if err != nil {
		t.Fatalf("Handle() error = %v", err)
	}
	if len(store.events) != 1 {
		t.Fatalf("events = %d, want 1", len(store.events))
	}
	if len(state.states) != 1 {
		t.Fatalf("states = %d, want 1", len(state.states))
	}
	if len(store.incidents) != 1 {
		t.Fatalf("incidents = %d, want 1", len(store.incidents))
	}
	if len(store.targets) != 1 || store.targets[0].Status != "CRITICAL" {
		t.Fatalf("target status = %#v, want CRITICAL", store.targets)
	}
	if store.incidents[0].RuleID != "container_died" {
		t.Fatalf("incident RuleID = %q, want container_died", store.incidents[0].RuleID)
	}
	if !store.events[0].Timestamp.Equal(time.Date(2026, 5, 2, 12, 0, 0, 0, time.UTC)) {
		t.Fatalf("timestamp = %s", store.events[0].Timestamp)
	}
}

func TestMetricTargetDoesNotOverwriteLifecycleStatus(t *testing.T) {
	metric := domain.MetricSample{
		NodeID:        "node-1",
		Source:        "docker",
		TargetID:      "container-id",
		ContainerName: "nginx",
		Timestamp:     time.Date(2026, 5, 2, 12, 0, 0, 0, time.UTC),
	}

	target := targetFromMetric(metric)

	if target.Status != "" {
		t.Fatalf("metric target status = %q, want empty status update", target.Status)
	}
}

func TestHandlerWaitsForAlertDurationBeforeCreatingIncident(t *testing.T) {
	store := &recordingStore{rules: []analyzer.ThresholdRule{{
		ID:        "sustained-cpu",
		TargetID:  "container-id",
		Metric:    analyzer.MetricCPUUsagePercent,
		Operator:  analyzer.OperatorGreaterThanOrEqual,
		Threshold: 80,
		Duration:  2 * time.Minute,
		Severity:  "critical",
	}}}
	state := &recordingState{}
	handler := NewHandler(Config{
		MetricsTopic: "container.metrics",
		EventsTopic:  "container.events",
	}, store, state, nil)

	first := []byte(`{
		"node_id":"node-1",
		"source":"docker",
		"target_id":"container-id",
		"container_name":"nginx",
		"metrics":{"cpu_usage_percent":80},
		"timestamp":"2026-05-02T12:00:00Z"
	}`)
	second := []byte(`{
		"node_id":"node-1",
		"source":"docker",
		"target_id":"container-id",
		"container_name":"nginx",
		"metrics":{"cpu_usage_percent":81},
		"timestamp":"2026-05-02T12:01:59Z"
	}`)
	third := []byte(`{
		"node_id":"node-1",
		"source":"docker",
		"target_id":"container-id",
		"container_name":"nginx",
		"metrics":{"cpu_usage_percent":82},
		"timestamp":"2026-05-02T12:02:00Z"
	}`)

	for _, value := range [][]byte{first, second} {
		if err := handler.Handle(context.Background(), kafka.Message{Topic: "container.metrics", Value: value}); err != nil {
			t.Fatalf("Handle() error = %v", err)
		}
	}
	if len(store.incidents) != 0 {
		t.Fatalf("incidents before duration = %d, want 0", len(store.incidents))
	}

	if err := handler.Handle(context.Background(), kafka.Message{Topic: "container.metrics", Value: third}); err != nil {
		t.Fatalf("Handle() error = %v", err)
	}
	if len(store.incidents) != 1 {
		t.Fatalf("incidents after duration = %d, want 1", len(store.incidents))
	}
}

func TestHandlerDoesNotCreateDuplicateIncidentForSameRuleAndTarget(t *testing.T) {
	store := &recordingStore{rules: []analyzer.ThresholdRule{{
		ID:        "high-cpu",
		TargetID:  "container-id",
		Metric:    analyzer.MetricCPUUsagePercent,
		Operator:  analyzer.OperatorGreaterThan,
		Threshold: 80,
		Severity:  "critical",
	}}}
	state := &recordingState{}
	handler := NewHandler(Config{
		MetricsTopic: "container.metrics",
		EventsTopic:  "container.events",
	}, store, state, nil)
	value := []byte(`{
		"node_id":"node-1",
		"source":"docker",
		"target_id":"container-id",
		"container_name":"nginx",
		"metrics":{"cpu_usage_percent":91.5},
		"timestamp":"2026-05-02T12:00:00Z"
	}`)

	for range 2 {
		if err := handler.Handle(context.Background(), kafka.Message{Topic: "container.metrics", Value: value}); err != nil {
			t.Fatalf("Handle() error = %v", err)
		}
	}

	if len(store.incidents) != 1 {
		t.Fatalf("incidents = %d, want 1", len(store.incidents))
	}
}
