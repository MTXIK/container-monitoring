package postgres

import (
	"context"
	"encoding/json"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nikponomarevan/container-monitoring-core/internal/analyzer"
	"github.com/nikponomarevan/container-monitoring-core/internal/domain"
)

type Store struct {
	dsn  string
	pool *pgxpool.Pool
}

func New(dsn string) *Store {
	return &Store{dsn: dsn}
}

func (s *Store) Connect(ctx context.Context) error {
	pool, err := pgxpool.New(ctx, s.dsn)
	if err != nil {
		return err
	}
	s.pool = pool
	return s.ensureFrontendSchema(ctx)
}

func (s *Store) Ping(ctx context.Context) error {
	if s.pool == nil {
		if err := s.Connect(ctx); err != nil {
			return err
		}
	}
	return s.pool.Ping(ctx)
}

func (s *Store) Close() {
	if s.pool != nil {
		s.pool.Close()
	}
}

func (s *Store) UpsertTarget(ctx context.Context, target domain.Target) error {
	if err := s.ensureConnected(ctx); err != nil {
		return err
	}
	_, err := s.pool.Exec(ctx, `
		INSERT INTO nodes (id, name) VALUES ($1, $1)
		ON CONFLICT (id) DO NOTHING
	`, target.NodeID)
	if err != nil {
		return err
	}
	_, err = s.pool.Exec(ctx, `
		INSERT INTO containers (id, node_id, name, image, source, external_id, status, labels, last_seen_at, updated_at)
		VALUES ($1, $2, $3, '', $4, $5, CASE WHEN $6 = '' THEN 'OK' ELSE $6 END, $7, $8, now())
		ON CONFLICT (id) DO UPDATE SET
			node_id = EXCLUDED.node_id,
			name = EXCLUDED.name,
			source = EXCLUDED.source,
			external_id = EXCLUDED.external_id,
			status = CASE WHEN $6 = '' THEN containers.status ELSE EXCLUDED.status END,
			labels = EXCLUDED.labels,
			last_seen_at = EXCLUDED.last_seen_at,
			updated_at = now()
	`, target.ID, target.NodeID, target.Name, defaultString(target.Source, "docker"), defaultString(target.ExternalID, target.ID), target.Status, jsonMap(target.Labels), defaultTime(target.LastSeenAt))
	return err
}

func (s *Store) SaveEvent(ctx context.Context, event domain.Event) error {
	if err := s.ensureConnected(ctx); err != nil {
		return err
	}
	payload, err := json.Marshal(event.Payload)
	if err != nil {
		return err
	}
	_, err = s.pool.Exec(ctx, `
		INSERT INTO events (node_id, container_id, name, event_type, severity, message, payload, occurred_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`, event.NodeID, event.TargetID, event.ContainerName, event.EventType, event.Severity, event.Message, payload, event.Timestamp)
	return err
}

func (s *Store) ListEvents(ctx context.Context, targetID string, limit int) ([]domain.Event, error) {
	if err := s.ensureConnected(ctx); err != nil {
		return nil, err
	}
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	query := `
		SELECT id, node_id, container_id, name, event_type, severity, message, payload, occurred_at
		FROM events
	`
	args := []any{}
	if targetID != "" {
		query += " WHERE container_id = $1"
		args = append(args, targetID)
	}
	query += " ORDER BY occurred_at DESC LIMIT $" + strconv.Itoa(len(args)+1)
	args = append(args, limit)

	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var events []domain.Event
	for rows.Next() {
		var event domain.Event
		var payload []byte
		if err := rows.Scan(&event.ID, &event.NodeID, &event.TargetID, &event.ContainerName, &event.EventType, &event.Severity, &event.Message, &payload, &event.Timestamp); err != nil {
			return nil, err
		}
		event.Source = "docker"
		if err := json.Unmarshal(payload, &event.Payload); err != nil {
			return nil, err
		}
		events = append(events, event)
	}
	return events, rows.Err()
}

func (s *Store) EnabledRules(ctx context.Context) ([]analyzer.ThresholdRule, error) {
	if err := s.ensureConnected(ctx); err != nil {
		return nil, err
	}
	rows, err := s.pool.Query(ctx, `
		SELECT id, COALESCE(target_id, ''), metric, operator, threshold, duration, severity, recovery_action
		FROM alert_rules
		WHERE enabled = true
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var rules []analyzer.ThresholdRule
	for rows.Next() {
		var rule analyzer.ThresholdRule
		if err := rows.Scan(&rule.ID, &rule.TargetID, &rule.Metric, &rule.Operator, &rule.Threshold, &rule.Duration, &rule.Severity, &rule.RecoveryAction); err != nil {
			return nil, err
		}
		rules = append(rules, rule)
	}
	return rules, rows.Err()
}

func (s *Store) HasOpenIncident(ctx context.Context, ruleID, targetID string) (bool, error) {
	if err := s.ensureConnected(ctx); err != nil {
		return false, err
	}
	var exists bool
	err := s.pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM incidents
			WHERE rule_id = $1
				AND container_id = $2
				AND status <> 'resolved'
		)
	`, ruleID, targetID).Scan(&exists)
	return exists, err
}

func (s *Store) CreateIncident(ctx context.Context, incident domain.Incident) (domain.Incident, error) {
	if err := s.ensureConnected(ctx); err != nil {
		return domain.Incident{}, err
	}
	err := s.pool.QueryRow(ctx, `
		INSERT INTO incidents (rule_id, node_id, container_id, status, severity, description, value, started_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id
	`, incident.RuleID, incident.NodeID, incident.TargetID, incident.Status, incident.Severity, incident.Description, incident.Value, incident.StartedAt).Scan(&incident.ID)
	return incident, err
}

func (s *Store) ListTargets(ctx context.Context) ([]domain.Target, error) {
	if err := s.ensureConnected(ctx); err != nil {
		return nil, err
	}
	rows, err := s.pool.Query(ctx, `
		SELECT id, name, node_id, source, external_id, status, labels, last_seen_at, created_at, updated_at
		FROM containers
		ORDER BY name
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var targets []domain.Target
	for rows.Next() {
		var target domain.Target
		var labels []byte
		if err := rows.Scan(&target.ID, &target.Name, &target.NodeID, &target.Source, &target.ExternalID, &target.Status, &labels, &target.LastSeenAt, &target.CreatedAt, &target.UpdatedAt); err != nil {
			return nil, err
		}
		target.Type = "container"
		if err := json.Unmarshal(labels, &target.Labels); err != nil {
			return nil, err
		}
		targets = append(targets, target)
	}
	return targets, rows.Err()
}

func (s *Store) GetTarget(ctx context.Context, id string) (domain.Target, error) {
	if err := s.ensureConnected(ctx); err != nil {
		return domain.Target{}, err
	}
	var target domain.Target
	var labels []byte
	err := s.pool.QueryRow(ctx, `
		SELECT id, name, node_id, source, external_id, status, labels, last_seen_at, created_at, updated_at
		FROM containers
		WHERE id = $1
	`, id).Scan(&target.ID, &target.Name, &target.NodeID, &target.Source, &target.ExternalID, &target.Status, &labels, &target.LastSeenAt, &target.CreatedAt, &target.UpdatedAt)
	target.Type = "container"
	if err == nil {
		err = json.Unmarshal(labels, &target.Labels)
	}
	return target, err
}

func (s *Store) CreateTarget(ctx context.Context, target domain.Target) (domain.Target, error) {
	if err := s.UpsertTarget(ctx, target); err != nil {
		return domain.Target{}, err
	}
	return s.GetTarget(ctx, target.ID)
}

func (s *Store) UpdateTarget(ctx context.Context, id string, target domain.Target) (domain.Target, error) {
	if err := s.ensureConnected(ctx); err != nil {
		return domain.Target{}, err
	}
	if target.ID == "" {
		target.ID = id
	}
	if target.NodeID == "" {
		existing, err := s.GetTarget(ctx, id)
		if err != nil {
			return domain.Target{}, err
		}
		target.NodeID = existing.NodeID
	}
	if _, err := s.pool.Exec(ctx, `
		INSERT INTO nodes (id, name) VALUES ($1, $1)
		ON CONFLICT (id) DO NOTHING
	`, target.NodeID); err != nil {
		return domain.Target{}, err
	}
	_, err := s.pool.Exec(ctx, `
		UPDATE containers
		SET node_id = $1, name = $2, source = $3, external_id = $4, status = $5, labels = $6, last_seen_at = $7, updated_at = now()
		WHERE id = $8
	`, target.NodeID, target.Name, defaultString(target.Source, "docker"), defaultString(target.ExternalID, id), defaultString(target.Status, "UNKNOWN"), jsonMap(target.Labels), defaultTime(target.LastSeenAt), id)
	if err != nil {
		return domain.Target{}, err
	}
	return s.GetTarget(ctx, id)
}

func (s *Store) DeleteTarget(ctx context.Context, id string) error {
	if err := s.ensureConnected(ctx); err != nil {
		return err
	}
	_, err := s.pool.Exec(ctx, `DELETE FROM containers WHERE id = $1`, id)
	return err
}

func (s *Store) ListAlertRules(ctx context.Context) ([]domain.AlertRule, error) {
	if err := s.ensureConnected(ctx); err != nil {
		return nil, err
	}
	rows, err := s.pool.Query(ctx, `
		SELECT id, name, target_id, metric, operator, threshold, duration, severity, enabled, recovery_action
		FROM alert_rules
		ORDER BY name
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var rules []domain.AlertRule
	for rows.Next() {
		var rule domain.AlertRule
		if err := rows.Scan(&rule.ID, &rule.Name, &rule.TargetID, &rule.MetricName, &rule.Operator, &rule.Threshold, &rule.Duration, &rule.Severity, &rule.Enabled, &rule.RecoveryAction); err != nil {
			return nil, err
		}
		rules = append(rules, rule)
	}
	return rules, rows.Err()
}

func (s *Store) CreateAlertRule(ctx context.Context, rule domain.AlertRule) (domain.AlertRule, error) {
	if err := s.ensureConnected(ctx); err != nil {
		return domain.AlertRule{}, err
	}
	_, err := s.pool.Exec(ctx, `
		INSERT INTO alert_rules (id, name, target_id, metric, operator, threshold, duration, severity, enabled, recovery_action, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, now())
	`, rule.ID, rule.Name, nullEmpty(rule.TargetID), rule.MetricName, rule.Operator, rule.Threshold, rule.Duration, rule.Severity, rule.Enabled, rule.RecoveryAction)
	if err != nil {
		return domain.AlertRule{}, err
	}
	return rule, nil
}

func (s *Store) UpdateAlertRule(ctx context.Context, id string, rule domain.AlertRule) (domain.AlertRule, error) {
	if err := s.ensureConnected(ctx); err != nil {
		return domain.AlertRule{}, err
	}
	rule.ID = id
	_, err := s.pool.Exec(ctx, `
		UPDATE alert_rules
		SET name = $1, target_id = $2, metric = $3, operator = $4, threshold = $5, duration = $6,
			severity = $7, enabled = $8, recovery_action = $9, updated_at = now()
		WHERE id = $10
	`, rule.Name, nullEmpty(rule.TargetID), rule.MetricName, rule.Operator, rule.Threshold, rule.Duration, rule.Severity, rule.Enabled, rule.RecoveryAction, id)
	if err != nil {
		return domain.AlertRule{}, err
	}
	return rule, nil
}

func (s *Store) DeleteAlertRule(ctx context.Context, id string) error {
	if err := s.ensureConnected(ctx); err != nil {
		return err
	}
	_, err := s.pool.Exec(ctx, `DELETE FROM alert_rules WHERE id = $1`, id)
	return err
}

func (s *Store) ListIncidents(ctx context.Context) ([]domain.Incident, error) {
	if err := s.ensureConnected(ctx); err != nil {
		return nil, err
	}
	rows, err := s.pool.Query(ctx, `
		SELECT id, rule_id, container_id, node_id, status, severity, description, value, started_at, resolved_at
		FROM incidents
		ORDER BY started_at DESC
		LIMIT 200
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var incidents []domain.Incident
	for rows.Next() {
		var incident domain.Incident
		if err := rows.Scan(&incident.ID, &incident.RuleID, &incident.TargetID, &incident.NodeID, &incident.Status, &incident.Severity, &incident.Description, &incident.Value, &incident.StartedAt, &incident.ResolvedAt); err != nil {
			return nil, err
		}
		incidents = append(incidents, incident)
	}
	return incidents, rows.Err()
}

func (s *Store) GetIncident(ctx context.Context, id int64) (domain.Incident, error) {
	if err := s.ensureConnected(ctx); err != nil {
		return domain.Incident{}, err
	}
	var incident domain.Incident
	err := s.pool.QueryRow(ctx, `
		SELECT id, rule_id, container_id, node_id, status, severity, description, value, started_at, resolved_at
		FROM incidents
		WHERE id = $1
	`, id).Scan(&incident.ID, &incident.RuleID, &incident.TargetID, &incident.NodeID, &incident.Status, &incident.Severity, &incident.Description, &incident.Value, &incident.StartedAt, &incident.ResolvedAt)
	return incident, err
}

func (s *Store) AcknowledgeIncident(ctx context.Context, id int64) error {
	return s.setIncidentStatus(ctx, id, "acknowledged", false)
}

func (s *Store) ResolveIncident(ctx context.Context, id int64) error {
	return s.setIncidentStatus(ctx, id, "resolved", true)
}

func (s *Store) CreateRecoveryAction(ctx context.Context, action domain.RecoveryAction) (domain.RecoveryAction, error) {
	if err := s.ensureConnected(ctx); err != nil {
		return domain.RecoveryAction{}, err
	}
	err := s.pool.QueryRow(ctx, `
		INSERT INTO recovery_actions (incident_id, target_id, action, status, result_message, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id
	`, action.IncidentID, action.TargetID, action.ActionType, action.Status, action.ResultMessage, action.StartedAt).Scan(&action.ID)
	return action, err
}

func (s *Store) FinishRecoveryAction(ctx context.Context, id int64, status, message string) error {
	if err := s.ensureConnected(ctx); err != nil {
		return err
	}
	_, err := s.pool.Exec(ctx, `
		UPDATE recovery_actions
		SET status = $1, result_message = $2, finished_at = now()
		WHERE id = $3
	`, status, message, id)
	return err
}

func (s *Store) ListRecoveryActions(ctx context.Context) ([]domain.RecoveryAction, error) {
	if err := s.ensureConnected(ctx); err != nil {
		return nil, err
	}
	rows, err := s.pool.Query(ctx, `
		SELECT id, incident_id, target_id, action, status, created_at, finished_at, result_message
		FROM recovery_actions
		ORDER BY created_at DESC
		LIMIT 200
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var actions []domain.RecoveryAction
	for rows.Next() {
		var action domain.RecoveryAction
		if err := rows.Scan(&action.ID, &action.IncidentID, &action.TargetID, &action.ActionType, &action.Status, &action.StartedAt, &action.FinishedAt, &action.ResultMessage); err != nil {
			return nil, err
		}
		actions = append(actions, action)
	}
	return actions, rows.Err()
}

func (s *Store) GetRecoveryAction(ctx context.Context, id int64) (domain.RecoveryAction, error) {
	if err := s.ensureConnected(ctx); err != nil {
		return domain.RecoveryAction{}, err
	}
	var action domain.RecoveryAction
	err := s.pool.QueryRow(ctx, `
		SELECT id, incident_id, target_id, action, status, created_at, finished_at, result_message
		FROM recovery_actions
		WHERE id = $1
	`, id).Scan(&action.ID, &action.IncidentID, &action.TargetID, &action.ActionType, &action.Status, &action.StartedAt, &action.FinishedAt, &action.ResultMessage)
	return action, err
}

func (s *Store) setIncidentStatus(ctx context.Context, id int64, status string, resolve bool) error {
	if err := s.ensureConnected(ctx); err != nil {
		return err
	}
	if resolve {
		_, err := s.pool.Exec(ctx, `UPDATE incidents SET status = $1, resolved_at = now() WHERE id = $2`, status, id)
		return err
	}
	_, err := s.pool.Exec(ctx, `UPDATE incidents SET status = $1 WHERE id = $2`, status, id)
	return err
}

func (s *Store) ensureConnected(ctx context.Context) error {
	if s.pool != nil {
		return nil
	}
	return s.Connect(ctx)
}

func defaultString(value, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}

func defaultTime(value time.Time) time.Time {
	if value.IsZero() {
		return time.Now().UTC()
	}
	return value
}

func jsonMap(value map[string]any) []byte {
	if value == nil {
		value = map[string]any{}
	}
	data, _ := json.Marshal(value)
	return data
}

func nullEmpty(value string) any {
	if value == "" {
		return nil
	}
	return value
}

func (s *Store) ensureFrontendSchema(ctx context.Context) error {
	_, err := s.pool.Exec(ctx, `
		ALTER TABLE containers
			ADD COLUMN IF NOT EXISTS source TEXT NOT NULL DEFAULT 'docker',
			ADD COLUMN IF NOT EXISTS external_id TEXT NOT NULL DEFAULT '',
			ADD COLUMN IF NOT EXISTS status TEXT NOT NULL DEFAULT 'UNKNOWN',
			ADD COLUMN IF NOT EXISTS labels JSONB NOT NULL DEFAULT '{}'::jsonb,
			ADD COLUMN IF NOT EXISTS last_seen_at TIMESTAMPTZ NOT NULL DEFAULT now(),
			ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ NOT NULL DEFAULT now();

		UPDATE containers SET external_id = id WHERE external_id = '';

		ALTER TABLE alert_rules
			ADD COLUMN IF NOT EXISTS target_id TEXT,
			ADD COLUMN IF NOT EXISTS duration INTERVAL NOT NULL DEFAULT '0 seconds',
			ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ NOT NULL DEFAULT now();
	`)
	return err
}
