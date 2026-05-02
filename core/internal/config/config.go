package config

import (
	"os"
	"strings"
)

type Config struct {
	HTTPAddr   string
	Kafka      KafkaConfig
	Postgres   string
	ClickHouse string
	RedisAddr  string
	Telegram   TelegramConfig
}

type KafkaConfig struct {
	Brokers      []string
	MetricsTopic string
	EventsTopic  string
}

type TelegramConfig struct {
	BotToken string
	ChatID   string
}

func Load() Config {
	return Config{
		HTTPAddr: env("HTTP_ADDR", ":8080"),
		Kafka: KafkaConfig{
			Brokers:      splitEnv("KAFKA_BROKERS", "localhost:9092"),
			MetricsTopic: env("KAFKA_METRICS_TOPIC", "container.metrics"),
			EventsTopic:  env("KAFKA_EVENTS_TOPIC", "container.events"),
		},
		Postgres:   env("POSTGRES_DSN", "postgres://container_monitoring:container_monitoring@localhost:5432/container_monitoring?sslmode=disable"),
		ClickHouse: env("CLICKHOUSE_DSN", "http://localhost:8123"),
		RedisAddr:  env("REDIS_ADDR", "localhost:6379"),
		Telegram: TelegramConfig{
			BotToken: env("TELEGRAM_BOT_TOKEN", ""),
			ChatID:   env("TELEGRAM_CHAT_ID", ""),
		},
	}
}

func env(key, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	return value
}

func splitEnv(key, fallback string) []string {
	parts := strings.Split(env(key, fallback), ",")
	values := make([]string, 0, len(parts))
	for _, part := range parts {
		if value := strings.TrimSpace(part); value != "" {
			values = append(values, value)
		}
	}
	return values
}
