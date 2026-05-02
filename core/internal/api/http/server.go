package http

import (
	"context"
	"encoding/json"
	"strconv"
	"time"

	swaggo "github.com/gofiber/contrib/v3/swaggo"
	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/cors"
	"github.com/google/uuid"
	_ "github.com/nikponomarevan/container-monitoring-core/docs"
	"github.com/nikponomarevan/container-monitoring-core/internal/domain"
)

type Repository interface {
	ListTargets(ctx context.Context) ([]domain.Target, error)
	GetTarget(ctx context.Context, id string) (domain.Target, error)
	CreateTarget(ctx context.Context, target domain.Target) (domain.Target, error)
	UpdateTarget(ctx context.Context, id string, target domain.Target) (domain.Target, error)
	DeleteTarget(ctx context.Context, id string) error
	LatestMetrics(ctx context.Context, targetID string, limit int) ([]domain.MetricSnapshot, error)
	MetricHistory(ctx context.Context, targetID, metricName string, from, to time.Time, limit int) ([]domain.MetricPoint, error)
	ListEvents(ctx context.Context, targetID string, limit int) ([]domain.Event, error)
	ListAlertRules(ctx context.Context) ([]domain.AlertRule, error)
	CreateAlertRule(ctx context.Context, rule domain.AlertRule) (domain.AlertRule, error)
	UpdateAlertRule(ctx context.Context, id string, rule domain.AlertRule) (domain.AlertRule, error)
	DeleteAlertRule(ctx context.Context, id string) error
	ListIncidents(ctx context.Context) ([]domain.Incident, error)
	GetIncident(ctx context.Context, id int64) (domain.Incident, error)
	AcknowledgeIncident(ctx context.Context, id int64) error
	ResolveIncident(ctx context.Context, id int64) error
	ListRecoveryActions(ctx context.Context) ([]domain.RecoveryAction, error)
}

func NewServer(repo Repository) *fiber.App {
	app := fiber.New(fiber.Config{AppName: "container-monitoring-core"})
	app.Use(cors.New(cors.Config{
		AllowOrigins: []string{"http://localhost:5173", "http://127.0.0.1:5173"},
		AllowHeaders: []string{"Origin", "Content-Type", "Accept"},
		AllowMethods: []string{fiber.MethodGet, fiber.MethodPost, fiber.MethodPatch, fiber.MethodDelete, fiber.MethodOptions},
	}))

	app.Get("/swagger/*", swaggo.HandlerDefault)

	app.Get("/health", func(c fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "ok"})
	})
	app.Get("/healthz", func(c fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "ok"})
	})

	app.Get("/ready", func(c fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "ready"})
	})
	app.Get("/readyz", func(c fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "ready"})
	})

	api := app.Group("/api/v1")
	api.Get("/targets", func(c fiber.Ctx) error {
		targets, err := repo.ListTargets(c.Context())
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, err.Error())
		}
		return c.JSON(targets)
	})
	api.Get("/targets/:id", func(c fiber.Ctx) error {
		target, err := repo.GetTarget(c.Context(), c.Params("id"))
		if err != nil {
			return fiber.NewError(fiber.StatusNotFound, err.Error())
		}
		return c.JSON(target)
	})
	api.Post("/targets", func(c fiber.Ctx) error {
		var target domain.Target
		if err := json.Unmarshal(c.Body(), &target); err != nil {
			return fiber.NewError(fiber.StatusBadRequest, err.Error())
		}
		if target.ID == "" {
			target.ID = uuid.NewString()
		}
		normalizeTarget(&target)
		created, err := repo.CreateTarget(c.Context(), target)
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, err.Error())
		}
		return c.Status(fiber.StatusCreated).JSON(created)
	})
	api.Patch("/targets/:id", func(c fiber.Ctx) error {
		var target domain.Target
		if err := json.Unmarshal(c.Body(), &target); err != nil {
			return fiber.NewError(fiber.StatusBadRequest, err.Error())
		}
		target.ID = c.Params("id")
		normalizeTarget(&target)
		updated, err := repo.UpdateTarget(c.Context(), c.Params("id"), target)
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, err.Error())
		}
		return c.JSON(updated)
	})
	api.Delete("/targets/:id", func(c fiber.Ctx) error {
		if err := repo.DeleteTarget(c.Context(), c.Params("id")); err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, err.Error())
		}
		return c.SendStatus(fiber.StatusNoContent)
	})
	api.Get("/metrics/latest", func(c fiber.Ctx) error {
		limit, err := intQuery(c, "limit", 100)
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, err.Error())
		}
		metrics, err := repo.LatestMetrics(c.Context(), c.Query("target_id"), limit)
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, err.Error())
		}
		return c.JSON(metrics)
	})
	api.Get("/metrics/history", func(c fiber.Ctx) error {
		limit, err := intQuery(c, "limit", 500)
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, err.Error())
		}
		from, err := timeQuery(c, "from")
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, err.Error())
		}
		to, err := timeQuery(c, "to")
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, err.Error())
		}
		points, err := repo.MetricHistory(c.Context(), c.Query("target_id"), c.Query("metric_name"), from, to, limit)
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, err.Error())
		}
		return c.JSON(points)
	})
	api.Get("/targets/:id/metrics", func(c fiber.Ctx) error {
		limit, err := intQuery(c, "limit", 500)
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, err.Error())
		}
		points, err := repo.MetricHistory(c.Context(), c.Params("id"), c.Query("metric_name"), time.Time{}, time.Time{}, limit)
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, err.Error())
		}
		return c.JSON(points)
	})
	api.Get("/events", func(c fiber.Ctx) error {
		limit, err := intQuery(c, "limit", 100)
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, err.Error())
		}
		events, err := repo.ListEvents(c.Context(), c.Query("target_id"), limit)
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, err.Error())
		}
		return c.JSON(events)
	})
	api.Get("/targets/:id/events", func(c fiber.Ctx) error {
		limit, err := intQuery(c, "limit", 100)
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, err.Error())
		}
		events, err := repo.ListEvents(c.Context(), c.Params("id"), limit)
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, err.Error())
		}
		return c.JSON(events)
	})
	api.Get("/alert-rules", func(c fiber.Ctx) error {
		rules, err := repo.ListAlertRules(c.Context())
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, err.Error())
		}
		return c.JSON(rules)
	})
	api.Post("/alert-rules", func(c fiber.Ctx) error {
		rule, err := decodeAlertRule(c.Body())
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, err.Error())
		}
		if rule.ID == "" {
			rule.ID = uuid.NewString()
		}
		created, err := repo.CreateAlertRule(c.Context(), rule)
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, err.Error())
		}
		return c.Status(fiber.StatusCreated).JSON(created)
	})
	api.Patch("/alert-rules/:id", func(c fiber.Ctx) error {
		rule, err := decodeAlertRule(c.Body())
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, err.Error())
		}
		updated, err := repo.UpdateAlertRule(c.Context(), c.Params("id"), rule)
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, err.Error())
		}
		return c.JSON(updated)
	})
	api.Delete("/alert-rules/:id", func(c fiber.Ctx) error {
		if err := repo.DeleteAlertRule(c.Context(), c.Params("id")); err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, err.Error())
		}
		return c.SendStatus(fiber.StatusNoContent)
	})
	api.Get("/incidents", func(c fiber.Ctx) error {
		incidents, err := repo.ListIncidents(c.Context())
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, err.Error())
		}
		return c.JSON(incidents)
	})
	api.Get("/incidents/:id", func(c fiber.Ctx) error {
		id, err := strconv.ParseInt(c.Params("id"), 10, 64)
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, err.Error())
		}
		incident, err := repo.GetIncident(c.Context(), id)
		if err != nil {
			return fiber.NewError(fiber.StatusNotFound, err.Error())
		}
		return c.JSON(incident)
	})
	api.Post("/incidents/:id/ack", func(c fiber.Ctx) error {
		id, err := strconv.ParseInt(c.Params("id"), 10, 64)
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, err.Error())
		}
		if err := repo.AcknowledgeIncident(c.Context(), id); err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, err.Error())
		}
		return c.SendStatus(fiber.StatusNoContent)
	})
	api.Post("/incidents/:id/resolve", func(c fiber.Ctx) error {
		id, err := strconv.ParseInt(c.Params("id"), 10, 64)
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, err.Error())
		}
		if err := repo.ResolveIncident(c.Context(), id); err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, err.Error())
		}
		return c.SendStatus(fiber.StatusNoContent)
	})
	api.Get("/recovery-actions", func(c fiber.Ctx) error {
		actions, err := repo.ListRecoveryActions(c.Context())
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, err.Error())
		}
		return c.JSON(actions)
	})
	api.Post("/recovery-actions/:id/retry", func(c fiber.Ctx) error {
		return c.JSON(fiber.Map{"id": c.Params("id"), "status": "retry accepted"})
	})

	return app
}

type alertRuleRequest struct {
	ID             string  `json:"id"`
	Name           string  `json:"name"`
	TargetID       string  `json:"target_id"`
	MetricName     string  `json:"metric_name"`
	Operator       string  `json:"operator"`
	LegacyOperator string  `json:"condition_operator"`
	Threshold      float64 `json:"threshold"`
	Duration       string  `json:"duration"`
	Severity       string  `json:"severity"`
	RecoveryPolicy string  `json:"recovery_policy"`
	RecoveryAction string  `json:"recovery_action"`
	Enabled        *bool   `json:"enabled"`
}

func decodeAlertRule(body []byte) (domain.AlertRule, error) {
	var req alertRuleRequest
	if err := json.Unmarshal(body, &req); err != nil {
		return domain.AlertRule{}, err
	}
	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}
	duration, err := time.ParseDuration(defaultString(req.Duration, "0s"))
	if err != nil {
		return domain.AlertRule{}, err
	}
	operator := req.Operator
	if operator == "" {
		operator = req.LegacyOperator
	}
	recovery := req.RecoveryPolicy
	if recovery == "" {
		recovery = req.RecoveryAction
	}
	return domain.AlertRule{
		ID:             req.ID,
		Name:           req.Name,
		TargetID:       req.TargetID,
		MetricName:     req.MetricName,
		Operator:       normalizeOperator(operator),
		Threshold:      req.Threshold,
		Duration:       duration,
		Severity:       defaultString(req.Severity, "warning"),
		Enabled:        enabled,
		RecoveryAction: defaultString(recovery, "notify_only"),
	}, nil
}

func normalizeOperator(operator string) string {
	switch operator {
	case ">", "gt":
		return "gt"
	case "<", "lt":
		return "lt"
	case ">=", "gte":
		return "gte"
	case "<=", "lte":
		return "lte"
	case "==", "eq":
		return "eq"
	default:
		return "gt"
	}
}

func normalizeTarget(target *domain.Target) {
	target.Type = defaultString(target.Type, "container")
	target.Source = defaultString(target.Source, "docker")
	target.ExternalID = defaultString(target.ExternalID, target.ID)
	target.Status = defaultString(target.Status, "UNKNOWN")
	if target.LastSeenAt.IsZero() {
		target.LastSeenAt = time.Now().UTC()
	}
}

func defaultString(value, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}

func intQuery(c fiber.Ctx, name string, fallback int) (int, error) {
	raw := c.Query(name)
	if raw == "" {
		return fallback, nil
	}
	value, err := strconv.Atoi(raw)
	if err != nil {
		return 0, err
	}
	return value, nil
}

func timeQuery(c fiber.Ctx, name string) (time.Time, error) {
	raw := c.Query(name)
	if raw == "" {
		return time.Time{}, nil
	}
	return time.Parse(time.RFC3339Nano, raw)
}
