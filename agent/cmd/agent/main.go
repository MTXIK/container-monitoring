package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/nikponomarevan/container-monitoring-agent/internal/collector/docker"
	"github.com/nikponomarevan/container-monitoring-agent/internal/config"
	"github.com/nikponomarevan/container-monitoring-agent/internal/publisher/kafka"
	"github.com/nikponomarevan/container-monitoring-agent/internal/runtime"
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	cfg := config.Load()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	collector := docker.NewBackend(cfg.NodeID, cfg.DockerHost)
	publisher := kafka.NewPublisher(cfg.Kafka)

	if err := runtime.Run(ctx, logger, cfg, collector, publisher); err != nil {
		logger.Error("agent stopped with error", "error", err)
		os.Exit(1)
	}
}
