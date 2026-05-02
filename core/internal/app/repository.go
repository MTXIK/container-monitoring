package app

import (
	"context"
	"time"

	"github.com/nikponomarevan/container-monitoring-core/internal/analyzer"
	"github.com/nikponomarevan/container-monitoring-core/internal/domain"
	"github.com/nikponomarevan/container-monitoring-core/internal/storage/clickhouse"
	"github.com/nikponomarevan/container-monitoring-core/internal/storage/postgres"
)

type Repository struct {
	Postgres   *postgres.Store
	ClickHouse *clickhouse.Store
}

func NewRepository(pg *postgres.Store, ch *clickhouse.Store) *Repository {
	return &Repository{Postgres: pg, ClickHouse: ch}
}

func (r *Repository) SaveMetric(ctx context.Context, metric domain.MetricSample) error {
	return r.ClickHouse.SaveMetric(ctx, metric)
}

func (r *Repository) LatestMetrics(ctx context.Context, targetID string, limit int) ([]domain.MetricSnapshot, error) {
	return r.ClickHouse.LatestMetrics(ctx, targetID, limit)
}

func (r *Repository) MetricHistory(ctx context.Context, targetID, metricName string, from, to time.Time, limit int) ([]domain.MetricPoint, error) {
	return r.ClickHouse.MetricHistory(ctx, targetID, metricName, from, to, limit)
}

func (r *Repository) SaveEvent(ctx context.Context, event domain.Event) error {
	if err := r.Postgres.SaveEvent(ctx, event); err != nil {
		return err
	}
	return r.ClickHouse.SaveEvent(ctx, event)
}

func (r *Repository) ListEvents(ctx context.Context, targetID string, limit int) ([]domain.Event, error) {
	return r.Postgres.ListEvents(ctx, targetID, limit)
}

func (r *Repository) UpsertTarget(ctx context.Context, target domain.Target) error {
	return r.Postgres.UpsertTarget(ctx, target)
}

func (r *Repository) EnabledRules(ctx context.Context) ([]analyzer.ThresholdRule, error) {
	return r.Postgres.EnabledRules(ctx)
}

func (r *Repository) CreateIncident(ctx context.Context, incident domain.Incident) (domain.Incident, error) {
	return r.Postgres.CreateIncident(ctx, incident)
}

func (r *Repository) ListTargets(ctx context.Context) ([]domain.Target, error) {
	return r.Postgres.ListTargets(ctx)
}

func (r *Repository) GetTarget(ctx context.Context, id string) (domain.Target, error) {
	return r.Postgres.GetTarget(ctx, id)
}

func (r *Repository) ListAlertRules(ctx context.Context) ([]domain.AlertRule, error) {
	return r.Postgres.ListAlertRules(ctx)
}

func (r *Repository) CreateAlertRule(ctx context.Context, rule domain.AlertRule) (domain.AlertRule, error) {
	return r.Postgres.CreateAlertRule(ctx, rule)
}

func (r *Repository) ListIncidents(ctx context.Context) ([]domain.Incident, error) {
	return r.Postgres.ListIncidents(ctx)
}

func (r *Repository) AcknowledgeIncident(ctx context.Context, id int64) error {
	return r.Postgres.AcknowledgeIncident(ctx, id)
}

func (r *Repository) ResolveIncident(ctx context.Context, id int64) error {
	return r.Postgres.ResolveIncident(ctx, id)
}

func (r *Repository) CreateRecoveryAction(ctx context.Context, action domain.RecoveryAction) (domain.RecoveryAction, error) {
	return r.Postgres.CreateRecoveryAction(ctx, action)
}

func (r *Repository) FinishRecoveryAction(ctx context.Context, id int64, status, message string) error {
	return r.Postgres.FinishRecoveryAction(ctx, id, status, message)
}

func (r *Repository) ListRecoveryActions(ctx context.Context) ([]domain.RecoveryAction, error) {
	return r.Postgres.ListRecoveryActions(ctx)
}
