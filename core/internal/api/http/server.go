package http

import "github.com/gofiber/fiber/v3"

func NewServer() *fiber.App {
	app := fiber.New(fiber.Config{AppName: "container-monitoring-core"})

	app.Get("/healthz", func(c fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "ok"})
	})

	app.Get("/readyz", func(c fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "ready"})
	})

	return app
}
