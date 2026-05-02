package postgres

import (
	"context"
	"encoding/json"
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
	return nil
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
		INSERT INTO containers (id, node_id, name, image)
		VALUES ($1, $2, $3, '')
		ON CONFLICT (id) DO UPDATE SET
			node_id = EXCLUDED.node_id,
			name = EXCLUDED.name
	`, target.ID, target.NodeID, target.Name)
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
		INSERT INTO events (node_id, container_id, event_type, severity, message, payload, occurred_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, event.NodeID, event.TargetID, event.EventType, event.Severity, event.Message, payload, event.Timestamp)
	return err
}

func (s *Store) EnabledRules(ctx context.Context) ([]analyzer.ThresholdRule, error) {
	if err := s.ensureConnected(ctx); err != nil {
		return nil, err
	}
	rows, err := s.pool.Query(ctx, `
		SELECT id, metric, operator, threshold, severity, recovery_action
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
		if err := rows.Scan(&rule.ID, &rule.Metric, &rule.Operator, &rule.Threshold, &rule.Severity, &rule.RecoveryAction); err != nil {
			return nil, err
		}
		rules = append(rules, rule)
	}
	return rules, rows.Err()
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
		SELECT id, name, node_id, created_at
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
		var createdAt time.Time
		if err := rows.Scan(&target.ID, &target.Name, &target.NodeID, &createdAt); err != nil {
			return nil, err
		}
		target.Type = "container"
		target.Source = "docker"
		target.ExternalID = target.ID
		target.CreatedAt = createdAt
		targets = append(targets, target)
	}
	return targets, rows.Err()
}

func (s *Store) GetTarget(ctx context.Context, id string) (domain.Target, error) {
	if err := s.ensureConnected(ctx); err != nil {
		return domain.Target{}, err
	}
	var target domain.Target
	var createdAt time.Time
	err := s.pool.QueryRow(ctx, `
		SELECT id, name, node_id, created_at
		FROM containers
		WHERE id = $1
	`, id).Scan(&target.ID, &target.Name, &target.NodeID, &createdAt)
	target.Type = "container"
	target.Source = "docker"
	target.ExternalID = target.ID
	target.CreatedAt = createdAt
	return target, err
}

func (s *Store) ListAlertRules(ctx context.Context) ([]domain.AlertRule, error) {
	if err := s.ensureConnected(ctx); err != nil {
		return nil, err
	}
	rows, err := s.pool.Query(ctx, `
		SELECT id, name, metric, operator, threshold, severity, enabled, recovery_action
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
		if err := rows.Scan(&rule.ID, &rule.Name, &rule.MetricName, &rule.Operator, &rule.Threshold, &rule.Severity, &rule.Enabled, &rule.RecoveryAction); err != nil {
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
		INSERT INTO alert_rules (id, name, metric, operator, threshold, severity, enabled, recovery_action)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`, rule.ID, rule.Name, rule.MetricName, rule.Operator, rule.Threshold, rule.Severity, rule.Enabled, rule.RecoveryAction)
	return rule, err
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
