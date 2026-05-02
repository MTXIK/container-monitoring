package config

import (
	"os"
	"strings"
	"time"
)

type Config struct {
	NodeID           string
	CollectorBackend string
	DockerHost       string
	CollectInterval  time.Duration
	Kafka            KafkaConfig
}

type KafkaConfig struct {
	Brokers      []string
	MetricsTopic string
	EventsTopic  string
}

func Load() Config {
	return Config{
		NodeID:           env("AGENT_NODE_ID", "local-node"),
		CollectorBackend: env("AGENT_COLLECTOR_BACKEND", "docker"),
		DockerHost:       env("DOCKER_HOST", "unix:///var/run/docker.sock"),
		CollectInterval:  durationEnv("COLLECT_INTERVAL", 10*time.Second),
		Kafka: KafkaConfig{
			Brokers:      splitEnv("KAFKA_BROKERS", "localhost:9092"),
			MetricsTopic: env("KAFKA_METRICS_TOPIC", "container.metrics"),
			EventsTopic:  env("KAFKA_EVENTS_TOPIC", "container.events"),
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

func durationEnv(key string, fallback time.Duration) time.Duration {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	parsed, err := time.ParseDuration(value)
	if err != nil {
		return fallback
	}
	return parsed
}
