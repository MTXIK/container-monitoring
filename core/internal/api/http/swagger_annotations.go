package http

import "github.com/nikponomarevan/container-monitoring-core/internal/domain"

type statusResponse struct {
	Status string `json:"status" example:"ok"`
}

type messageResponse struct {
	Status   string `json:"status" example:"Grafana reads metric history directly from ClickHouse"`
	TargetID string `json:"target_id,omitempty" example:"container-id"`
}

type retryRecoveryResponse struct {
	ID     string `json:"id" example:"1"`
	Status string `json:"status" example:"retry accepted"`
}

// health godoc
// @Summary Health check
// @Description Reports that the HTTP process is alive.
// @Tags health
// @Produce json
// @Success 200 {object} statusResponse
// @Router /health [get]
func swaggerHealth() {}

// ready godoc
// @Summary Readiness check
// @Description Reports that the HTTP process is ready to serve requests.
// @Tags health
// @Produce json
// @Success 200 {object} statusResponse
// @Router /ready [get]
func swaggerReady() {}

// listTargets godoc
// @Summary List targets
// @Description Returns discovered Docker container targets.
// @Tags targets
// @Produce json
// @Success 200 {array} domain.Target
// @Failure 500 {object} statusResponse
// @Router /api/v1/targets [get]
func swaggerListTargets() {}

// getTarget godoc
// @Summary Get target
// @Description Returns one discovered Docker container target.
// @Tags targets
// @Produce json
// @Param id path string true "Target ID"
// @Success 200 {object} domain.Target
// @Failure 404 {object} statusResponse
// @Router /api/v1/targets/{id} [get]
func swaggerGetTarget() {}

// createTarget godoc
// @Summary Create target
// @Description Creates or registers a monitored container target.
// @Tags targets
// @Accept json
// @Produce json
// @Param target body domain.Target true "Target"
// @Success 201 {object} domain.Target
// @Failure 400 {object} statusResponse
// @Failure 500 {object} statusResponse
// @Router /api/v1/targets [post]
func swaggerCreateTarget() {}

// updateTarget godoc
// @Summary Update target
// @Description Updates target metadata used by the admin panel.
// @Tags targets
// @Accept json
// @Produce json
// @Param id path string true "Target ID"
// @Param target body domain.Target true "Target"
// @Success 200 {object} domain.Target
// @Failure 400 {object} statusResponse
// @Failure 500 {object} statusResponse
// @Router /api/v1/targets/{id} [patch]
func swaggerUpdateTarget() {}

// deleteTarget godoc
// @Summary Delete target
// @Description Deletes a target record from the configuration store.
// @Tags targets
// @Param id path string true "Target ID"
// @Success 204
// @Failure 500 {object} statusResponse
// @Router /api/v1/targets/{id} [delete]
func swaggerDeleteTarget() {}

// latestMetrics godoc
// @Summary Latest metrics
// @Description Returns recent metric snapshots from ClickHouse.
// @Tags metrics
// @Produce json
// @Param target_id query string false "Target ID"
// @Param limit query int false "Maximum rows"
// @Success 200 {array} domain.MetricSnapshot
// @Failure 400 {object} statusResponse
// @Failure 500 {object} statusResponse
// @Router /api/v1/metrics/latest [get]
func swaggerLatestMetrics() {}

// metricHistory godoc
// @Summary Metric history
// @Description Returns metric history points from ClickHouse.
// @Tags metrics
// @Produce json
// @Param target_id query string false "Target ID"
// @Param metric_name query string false "Metric name"
// @Param from query string false "RFC3339 start time"
// @Param to query string false "RFC3339 end time"
// @Param limit query int false "Maximum rows"
// @Success 200 {array} domain.MetricPoint
// @Failure 400 {object} statusResponse
// @Failure 500 {object} statusResponse
// @Router /api/v1/metrics/history [get]
func swaggerMetricHistory() {}

// targetMetrics godoc
// @Summary Target metrics pointer
// @Description Documents target-scoped metric history access.
// @Tags metrics
// @Produce json
// @Param id path string true "Target ID"
// @Param metric_name query string false "Metric name"
// @Param limit query int false "Maximum rows"
// @Success 200 {array} domain.MetricPoint
// @Failure 400 {object} statusResponse
// @Failure 500 {object} statusResponse
// @Router /api/v1/targets/{id}/metrics [get]
func swaggerTargetMetrics() {}

// listEvents godoc
// @Summary List events
// @Description Returns recent container events from PostgreSQL.
// @Tags events
// @Produce json
// @Param limit query int false "Maximum rows"
// @Success 200 {array} domain.Event
// @Failure 400 {object} statusResponse
// @Failure 500 {object} statusResponse
// @Router /api/v1/events [get]
func swaggerListEvents() {}

// targetEvents godoc
// @Summary Target events pointer
// @Description Documents target-scoped event access.
// @Tags events
// @Produce json
// @Param id path string true "Target ID"
// @Param limit query int false "Maximum rows"
// @Success 200 {array} domain.Event
// @Failure 400 {object} statusResponse
// @Failure 500 {object} statusResponse
// @Router /api/v1/targets/{id}/events [get]
func swaggerTargetEvents() {}

// listAlertRules godoc
// @Summary List alert rules
// @Description Returns enabled and disabled threshold alert rules.
// @Tags alert-rules
// @Produce json
// @Success 200 {array} domain.AlertRule
// @Failure 500 {object} statusResponse
// @Router /api/v1/alert-rules [get]
func swaggerListAlertRules() {}

// createAlertRule godoc
// @Summary Create alert rule
// @Description Creates a threshold alert rule. Missing id, operator, duration, severity, recovery_policy, and enabled fields are defaulted by the API.
// @Tags alert-rules
// @Accept json
// @Produce json
// @Param rule body domain.AlertRule true "Alert rule"
// @Success 201 {object} domain.AlertRule
// @Failure 400 {object} statusResponse
// @Failure 500 {object} statusResponse
// @Router /api/v1/alert-rules [post]
func swaggerCreateAlertRule() {}

// updateAlertRule godoc
// @Summary Update alert rule
// @Description Updates a threshold alert rule.
// @Tags alert-rules
// @Accept json
// @Produce json
// @Param id path string true "Alert rule ID"
// @Param rule body domain.AlertRule true "Alert rule"
// @Success 200 {object} domain.AlertRule
// @Failure 400 {object} statusResponse
// @Failure 500 {object} statusResponse
// @Router /api/v1/alert-rules/{id} [patch]
func swaggerUpdateAlertRule() {}

// deleteAlertRule godoc
// @Summary Delete alert rule
// @Description Deletes a threshold alert rule.
// @Tags alert-rules
// @Param id path string true "Alert rule ID"
// @Success 204
// @Failure 500 {object} statusResponse
// @Router /api/v1/alert-rules/{id} [delete]
func swaggerDeleteAlertRule() {}

// listIncidents godoc
// @Summary List incidents
// @Description Returns recent incidents ordered by start time.
// @Tags incidents
// @Produce json
// @Success 200 {array} domain.Incident
// @Failure 500 {object} statusResponse
// @Router /api/v1/incidents [get]
func swaggerListIncidents() {}

// getIncident godoc
// @Summary Get incident
// @Description Returns one incident by ID.
// @Tags incidents
// @Produce json
// @Param id path int true "Incident ID"
// @Success 200 {object} domain.Incident
// @Failure 400 {object} statusResponse
// @Failure 404 {object} statusResponse
// @Router /api/v1/incidents/{id} [get]
func swaggerGetIncident() {}

// acknowledgeIncident godoc
// @Summary Acknowledge incident
// @Description Marks an incident as acknowledged.
// @Tags incidents
// @Param id path int true "Incident ID"
// @Success 204
// @Failure 400 {object} statusResponse
// @Failure 500 {object} statusResponse
// @Router /api/v1/incidents/{id}/ack [post]
func swaggerAcknowledgeIncident() {}

// resolveIncident godoc
// @Summary Resolve incident
// @Description Marks an incident as resolved and sets resolved_at.
// @Tags incidents
// @Param id path int true "Incident ID"
// @Success 204
// @Failure 400 {object} statusResponse
// @Failure 500 {object} statusResponse
// @Router /api/v1/incidents/{id}/resolve [post]
func swaggerResolveIncident() {}

// listRecoveryActions godoc
// @Summary List recovery actions
// @Description Returns recent recovery action attempts and results.
// @Tags recovery
// @Produce json
// @Success 200 {array} domain.RecoveryAction
// @Failure 500 {object} statusResponse
// @Router /api/v1/recovery-actions [get]
func swaggerListRecoveryActions() {}

// retryRecoveryAction godoc
// @Summary Retry recovery action
// @Description Accepts a retry request for a recovery action placeholder endpoint.
// @Tags recovery
// @Produce json
// @Param id path int true "Recovery action ID"
// @Success 200 {object} retryRecoveryResponse
// @Router /api/v1/recovery-actions/{id}/retry [post]
func swaggerRetryRecoveryAction() {}

var (
	_ domain.Target
	_ domain.MetricSnapshot
	_ domain.MetricPoint
	_ domain.Event
	_ domain.AlertRule
	_ domain.Incident
	_ domain.RecoveryAction
)
