package main

import (
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	httpapi "github.com/nikponomarevan/container-monitoring-core/internal/api/http"
	"github.com/nikponomarevan/container-monitoring-core/internal/config"
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	cfg := config.Load()

	app := httpapi.NewServer()

	go func() {
		if err := app.Listen(cfg.HTTPAddr); err != nil {
			logger.Error("http server stopped", "error", err)
			os.Exit(1)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	if err := app.Shutdown(); err != nil {
		logger.Error("http shutdown", "error", err)
		os.Exit(1)
	}
}
