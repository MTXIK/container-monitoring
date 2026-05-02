package http

import (
	"context"
	"encoding/json"
	"strconv"
	"time"

	swaggo "github.com/gofiber/contrib/v3/swaggo"
	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	_ "github.com/nikponomarevan/container-monitoring-core/docs"
	"github.com/nikponomarevan/container-monitoring-core/internal/domain"
)

type Repository interface {
	ListTargets(ctx context.Context) ([]domain.Target, error)
	GetTarget(ctx context.Context, id string) (domain.Target, error)
	LatestMetrics(ctx context.Context, targetID string, limit int) ([]domain.MetricSnapshot, error)
	MetricHistory(ctx context.Context, targetID, metricName string, from, to time.Time, limit int) ([]domain.MetricPoint, error)
	ListEvents(ctx context.Context, targetID string, limit int) ([]domain.Event, error)
	ListAlertRules(ctx context.Context) ([]domain.AlertRule, error)
	CreateAlertRule(ctx context.Context, rule domain.AlertRule) (domain.AlertRule, error)
	ListIncidents(ctx context.Context) ([]domain.Incident, error)
	AcknowledgeIncident(ctx context.Context, id int64) error
	ResolveIncident(ctx context.Context, id int64) error
	ListRecoveryActions(ctx context.Context) ([]domain.RecoveryAction, error)
}

func NewServer(repo Repository) *fiber.App {
	app := fiber.New(fiber.Config{AppName: "container-monitoring-core"})

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
		events, err := repo.ListEvents(c.Context(), "", limit)
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
		var rule domain.AlertRule
		if err := json.Unmarshal(c.Body(), &rule); err != nil {
			return fiber.NewError(fiber.StatusBadRequest, err.Error())
		}
		if rule.ID == "" {
			rule.ID = uuid.NewString()
		}
		if rule.Operator == "" {
			rule.Operator = "gt"
		}
		if rule.Severity == "" {
			rule.Severity = "warning"
		}
		if rule.RecoveryAction == "" {
			rule.RecoveryAction = "notify_only"
		}
		rule.Enabled = true
		created, err := repo.CreateAlertRule(c.Context(), rule)
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, err.Error())
		}
		return c.Status(fiber.StatusCreated).JSON(created)
	})
	api.Get("/incidents", func(c fiber.Ctx) error {
		incidents, err := repo.ListIncidents(c.Context())
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, err.Error())
		}
		return c.JSON(incidents)
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
