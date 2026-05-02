package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	httpapi "github.com/nikponomarevan/container-monitoring-core/internal/api/http"
	coreapp "github.com/nikponomarevan/container-monitoring-core/internal/app"
	"github.com/nikponomarevan/container-monitoring-core/internal/config"
	kafkaconsumer "github.com/nikponomarevan/container-monitoring-core/internal/consumer/kafka"
	"github.com/nikponomarevan/container-monitoring-core/internal/ingest"
	"github.com/nikponomarevan/container-monitoring-core/internal/notifier/telegram"
	"github.com/nikponomarevan/container-monitoring-core/internal/recovery"
	redisstate "github.com/nikponomarevan/container-monitoring-core/internal/state/redis"
	"github.com/nikponomarevan/container-monitoring-core/internal/storage/clickhouse"
	"github.com/nikponomarevan/container-monitoring-core/internal/storage/postgres"
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	cfg := config.Load()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	pg := postgres.New(cfg.Postgres)
	defer pg.Close()
	if err := pg.Ping(ctx); err != nil {
		logger.Error("postgres ping", "error", err)
		os.Exit(1)
	}
	ch := clickhouse.New(cfg.ClickHouse)
	if err := ch.Ping(ctx); err != nil {
		logger.Error("clickhouse ping", "error", err)
		os.Exit(1)
	}
	state := redisstate.New(cfg.RedisAddr)
	if err := state.Ping(ctx); err != nil {
		logger.Error("redis ping", "error", err)
		os.Exit(1)
	}

	repo := coreapp.NewRepository(pg, ch)
	notifier := telegram.New(cfg.Telegram.BotToken, cfg.Telegram.ChatID)
	recoverer := recovery.NewCoordinator(state, repo, recovery.NewDockerExecutor(cfg.DockerHost))
	handler := ingest.NewHandler(ingest.Config{
		MetricsTopic: cfg.Kafka.MetricsTopic,
		EventsTopic:  cfg.Kafka.EventsTopic,
	}, repo, state, notifier, recoverer)
	consumer := kafkaconsumer.NewConsumer(cfg.Kafka.Brokers, []string{cfg.Kafka.MetricsTopic, cfg.Kafka.EventsTopic}, cfg.Kafka.GroupID)

	go func() {
		if err := consumer.Run(ctx, handler); err != nil {
			logger.Error("kafka consumer stopped", "error", err)
			stop()
		}
	}()

	app := httpapi.NewServer(repo)

	go func() {
		if err := app.Listen(cfg.HTTPAddr); err != nil {
			logger.Error("http server stopped", "error", err)
			stop()
		}
	}()

	<-ctx.Done()

	if err := app.Shutdown(); err != nil {
		logger.Error("http shutdown", "error", err)
		os.Exit(1)
	}
}
