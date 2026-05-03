package redis

import (
	"context"
	"encoding/json"
	"strings"
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

func (s *Store) AlertThresholdStartedAt(ctx context.Context, ruleID, targetID string, matchedAt time.Time) (time.Time, error) {
	key := alertThresholdKey(ruleID, targetID)
	value := matchedAt.Format(time.RFC3339Nano)
	set, err := s.client.SetNX(ctx, key, value, 24*time.Hour).Result()
	if err != nil {
		return time.Time{}, err
	}
	if set {
		return matchedAt, nil
	}
	stored, err := s.client.Get(ctx, key).Result()
	if err != nil {
		return time.Time{}, err
	}
	startedAt, err := time.Parse(time.RFC3339Nano, stored)
	if err != nil {
		_ = s.client.Set(ctx, key, value, 24*time.Hour).Err()
		return matchedAt, nil
	}
	return startedAt, nil
}

func (s *Store) ClearAlertThreshold(ctx context.Context, ruleID, targetID string) error {
	return s.client.Del(ctx, alertThresholdKey(ruleID, targetID)).Err()
}

func alertThresholdKey(ruleID, targetID string) string {
	return "alert:" + keyPart(ruleID) + ":" + keyPart(targetID) + ":started_at"
}

func keyPart(value string) string {
	return strings.NewReplacer(":", "_", " ", "_", "\n", "_", "\r", "_").Replace(value)
}
