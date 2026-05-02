package http

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/nikponomarevan/container-monitoring-core/internal/domain"
)

type fakeRepository struct {
	targets         map[string]domain.Target
	alertRules      map[string]domain.AlertRule
	incidents       map[int64]domain.Incident
	recoveryActions map[int64]domain.RecoveryAction
}

func newFakeRepository() *fakeRepository {
	return &fakeRepository{
		targets: map[string]domain.Target{
			"target-a": {ID: "target-a", Name: "nginx", NodeID: "node-a", Source: "docker", ExternalID: "docker-a"},
		},
		alertRules: map[string]domain.AlertRule{
			"rule-a": {ID: "rule-a", Name: "CPU high", TargetID: "target-a", MetricName: "cpu_usage_percent", Operator: "gt", Threshold: 80, Duration: 2 * time.Minute, Severity: "warning", Enabled: true, RecoveryAction: "retry_check"},
		},
		incidents: map[int64]domain.Incident{
			42: {ID: 42, RuleID: "rule-a", TargetID: "target-a", NodeID: "node-a", Status: "open", Severity: "warning", Description: "cpu high", StartedAt: time.Now()},
		},
		recoveryActions: map[int64]domain.RecoveryAction{},
	}
}

func (f *fakeRepository) ListTargets(context.Context) ([]domain.Target, error) {
	targets := make([]domain.Target, 0, len(f.targets))
	for _, target := range f.targets {
		targets = append(targets, target)
	}
	return targets, nil
}
func (f *fakeRepository) GetTarget(_ context.Context, id string) (domain.Target, error) {
	return f.targets[id], nil
}
func (f *fakeRepository) CreateTarget(_ context.Context, target domain.Target) (domain.Target, error) {
	f.targets[target.ID] = target
	return target, nil
}
func (f *fakeRepository) UpdateTarget(_ context.Context, id string, target domain.Target) (domain.Target, error) {
	target.ID = id
	f.targets[id] = target
	return target, nil
}
func (f *fakeRepository) DeleteTarget(_ context.Context, id string) error {
	delete(f.targets, id)
	return nil
}
func (f *fakeRepository) LatestMetrics(context.Context, string, int) ([]domain.MetricSnapshot, error) {
	return []domain.MetricSnapshot{}, nil
}
func (f *fakeRepository) MetricHistory(context.Context, string, string, time.Time, time.Time, int) ([]domain.MetricPoint, error) {
	return []domain.MetricPoint{}, nil
}
func (f *fakeRepository) ListEvents(context.Context, string, int) ([]domain.Event, error) {
	return []domain.Event{}, nil
}
func (f *fakeRepository) ListAlertRules(context.Context) ([]domain.AlertRule, error) {
	rules := make([]domain.AlertRule, 0, len(f.alertRules))
	for _, rule := range f.alertRules {
		rules = append(rules, rule)
	}
	return rules, nil
}
func (f *fakeRepository) CreateAlertRule(_ context.Context, rule domain.AlertRule) (domain.AlertRule, error) {
	f.alertRules[rule.ID] = rule
	return rule, nil
}
func (f *fakeRepository) UpdateAlertRule(_ context.Context, id string, rule domain.AlertRule) (domain.AlertRule, error) {
	rule.ID = id
	f.alertRules[id] = rule
	return rule, nil
}
func (f *fakeRepository) DeleteAlertRule(_ context.Context, id string) error {
	delete(f.alertRules, id)
	return nil
}
func (f *fakeRepository) ListIncidents(context.Context) ([]domain.Incident, error) {
	incidents := make([]domain.Incident, 0, len(f.incidents))
	for _, incident := range f.incidents {
		incidents = append(incidents, incident)
	}
	return incidents, nil
}
func (f *fakeRepository) GetIncident(_ context.Context, id int64) (domain.Incident, error) {
	return f.incidents[id], nil
}
func (f *fakeRepository) AcknowledgeIncident(_ context.Context, id int64) error {
	incident := f.incidents[id]
	incident.Status = "acknowledged"
	f.incidents[id] = incident
	return nil
}
func (f *fakeRepository) ResolveIncident(_ context.Context, id int64) error {
	incident := f.incidents[id]
	incident.Status = "resolved"
	f.incidents[id] = incident
	return nil
}
func (f *fakeRepository) ListRecoveryActions(context.Context) ([]domain.RecoveryAction, error) {
	return []domain.RecoveryAction{}, nil
}

func TestServerSupportsFrontendIncidentDetails(t *testing.T) {
	app := NewServer(newFakeRepository())

	resp, err := app.Test(newRequest(t, http.MethodGet, "/api/v1/incidents/42", ""))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}
	var incident domain.Incident
	if err := json.NewDecoder(resp.Body).Decode(&incident); err != nil {
		t.Fatal(err)
	}
	if incident.ID != 42 {
		t.Fatalf("incident id = %d, want 42", incident.ID)
	}
}

func TestServerSupportsFrontendAlertRuleUpdateAndDelete(t *testing.T) {
	repo := newFakeRepository()
	app := NewServer(repo)

	body := `{"name":"CPU critical","target_id":"target-a","metric_name":"cpu_usage_percent","operator":">=","threshold":95,"duration":"5m","severity":"critical","recovery_policy":"restart_container","enabled":false}`
	resp, err := app.Test(newRequest(t, http.MethodPatch, "/api/v1/alert-rules/rule-a", body))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("patch status = %d, want %d: %s", resp.StatusCode, http.StatusOK, readBody(t, resp.Body))
	}
	if repo.alertRules["rule-a"].Operator != "gte" {
		t.Fatalf("operator = %q, want gte", repo.alertRules["rule-a"].Operator)
	}
	if repo.alertRules["rule-a"].RecoveryAction != "restart_container" {
		t.Fatalf("recovery action = %q, want restart_container", repo.alertRules["rule-a"].RecoveryAction)
	}

	resp, err = app.Test(newRequest(t, http.MethodDelete, "/api/v1/alert-rules/rule-a", ""))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("delete status = %d, want %d", resp.StatusCode, http.StatusNoContent)
	}
	if _, ok := repo.alertRules["rule-a"]; ok {
		t.Fatal("rule was not deleted")
	}
}

func TestServerSupportsFrontendTargetCRUD(t *testing.T) {
	repo := newFakeRepository()
	app := NewServer(repo)

	body := `{"id":"target-b","name":"api","node_id":"node-a","source":"docker","external_id":"docker-b","status":"OK","labels":{"service":"api"}}`
	resp, err := app.Test(newRequest(t, http.MethodPost, "/api/v1/targets", body))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("create status = %d, want %d: %s", resp.StatusCode, http.StatusCreated, readBody(t, resp.Body))
	}

	body = `{"name":"api-renamed","node_id":"node-a","source":"docker","external_id":"docker-b","status":"WARNING"}`
	resp, err = app.Test(newRequest(t, http.MethodPatch, "/api/v1/targets/target-b", body))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("patch status = %d, want %d", resp.StatusCode, http.StatusOK)
	}
	if repo.targets["target-b"].Name != "api-renamed" {
		t.Fatalf("target name = %q, want api-renamed", repo.targets["target-b"].Name)
	}

	resp, err = app.Test(newRequest(t, http.MethodDelete, "/api/v1/targets/target-b", ""))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("delete status = %d, want %d", resp.StatusCode, http.StatusNoContent)
	}
}

func newRequest(t *testing.T, method, path, body string) *http.Request {
	t.Helper()
	req, err := http.NewRequest(method, path, strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	return req
}

func readBody(t *testing.T, body io.Reader) string {
	t.Helper()
	data, err := io.ReadAll(body)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}
