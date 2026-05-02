package redis

import (
	"context"
	"encoding/json"
	"time"

	"github.com/nikponomarevan/container-monitoring-core/internal/domain"
	goredis "github.com/redis/go-redis/v9"
)

type Store struct {
	client *goredis.Client
}

func New(addr string) *Store {
	return &Store{client: goredis.NewClient(&goredis.Options{Addr: addr})}
}

func (s *Store) Ping(ctx context.Context) error {
	return s.client.Ping(ctx).Err()
}

func (s *Store) SetLatestMetrics(ctx context.Context, metric domain.MetricSample) error {
	value, err := json.Marshal(metric)
	if err != nil {
		return err
	}
	return s.client.Set(ctx, "target:"+metric.TargetID+":last_metrics", value, 0).Err()
}

func (s *Store) SetTargetState(ctx context.Context, event domain.Event) error {
	value, err := json.Marshal(event)
	if err != nil {
		return err
	}
	return s.client.Set(ctx, "target:"+event.TargetID+":state", value, 0).Err()
}

func (s *Store) AcquireRecoveryLock(ctx context.Context, targetID string, ttl time.Duration) (bool, error) {
	return s.client.SetNX(ctx, "recovery:"+targetID+":lock", "1", ttl).Result()
}
