package clickhouse

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/nikponomarevan/container-monitoring-core/internal/domain"
)

type Store struct {
	dsn string
}

func New(dsn string) *Store {
	return &Store{dsn: dsn}
}

func (s *Store) Ping(ctx context.Context) error {
	return s.exec(ctx, "SELECT 1", nil)
}

func (s *Store) SaveMetric(ctx context.Context, metric domain.MetricSample) error {
	row := map[string]any{
		"collected_at": metric.Timestamp.Format(time.RFC3339Nano),
		"node_id":      metric.NodeID,
		"container_id": metric.TargetID,
		"name":         metric.ContainerName,
		"cpu_percent":  metric.Metrics["cpu_usage_percent"],
		"memory_bytes": uint64(metric.Metrics["memory_usage_bytes"]),
		"rx_bytes":     uint64(metric.Metrics["network_rx_bytes"]),
		"tx_bytes":     uint64(metric.Metrics["network_tx_bytes"]),
		"block_read":   uint64(metric.Metrics["block_read_bytes"]),
		"block_write":  uint64(metric.Metrics["block_write_bytes"]),
	}
	return s.insertJSONEachRow(ctx, "INSERT INTO container_metrics FORMAT JSONEachRow", row)
}

func (s *Store) SaveEvent(ctx context.Context, event domain.Event) error {
	row := map[string]any{
		"occurred_at":  event.Timestamp.Format(time.RFC3339Nano),
		"node_id":      event.NodeID,
		"container_id": event.TargetID,
		"name":         event.ContainerName,
		"type":         event.EventType,
	}
	return s.insertJSONEachRow(ctx, "INSERT INTO container_events FORMAT JSONEachRow", row)
}

func (s *Store) insertJSONEachRow(ctx context.Context, query string, row map[string]any) error {
	value, err := json.Marshal(row)
	if err != nil {
		return err
	}
	return s.exec(ctx, query, append(value, '\n'))
}

func (s *Store) exec(ctx context.Context, query string, body []byte) error {
	endpoint := strings.TrimRight(s.dsn, "/") + "/?query=" + url.QueryEscape(query)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return fmt.Errorf("clickhouse status %s", resp.Status)
	}
	return nil
}
