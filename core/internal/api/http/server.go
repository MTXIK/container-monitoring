package http

import (
	"context"
	"encoding/json"
	"strconv"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/nikponomarevan/container-monitoring-core/internal/domain"
)

type Repository interface {
	ListTargets(ctx context.Context) ([]domain.Target, error)
	GetTarget(ctx context.Context, id string) (domain.Target, error)
	ListAlertRules(ctx context.Context) ([]domain.AlertRule, error)
	CreateAlertRule(ctx context.Context, rule domain.AlertRule) (domain.AlertRule, error)
	ListIncidents(ctx context.Context) ([]domain.Incident, error)
	AcknowledgeIncident(ctx context.Context, id int64) error
	ResolveIncident(ctx context.Context, id int64) error
	ListRecoveryActions(ctx context.Context) ([]domain.RecoveryAction, error)
}

func NewServer(repo Repository) *fiber.App {
	app := fiber.New(fiber.Config{AppName: "container-monitoring-core"})

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
		return c.JSON(fiber.Map{"status": "latest metrics are stored in Redis under target:{id}:last_metrics"})
	})
	api.Get("/metrics/history", func(c fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "Grafana reads metric history directly from ClickHouse"})
	})
	api.Get("/targets/:id/metrics", func(c fiber.Ctx) error {
		return c.JSON(fiber.Map{"target_id": c.Params("id"), "status": "Grafana reads metric history directly from ClickHouse"})
	})
	api.Get("/events", func(c fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "events are persisted in PostgreSQL and ClickHouse"})
	})
	api.Get("/targets/:id/events", func(c fiber.Ctx) error {
		return c.JSON(fiber.Map{"target_id": c.Params("id"), "status": "events are persisted in PostgreSQL and ClickHouse"})
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
