package clickhouse

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/nikponomarevan/container-monitoring-core/internal/domain"
)

type Store struct {
	dsn string
}

func (s *Store) LatestMetrics(ctx context.Context, targetID string, limit int) ([]domain.MetricSnapshot, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	query := `
		SELECT collected_at, node_id, container_id, name, cpu_percent, memory_bytes, rx_bytes, tx_bytes, block_read, block_write
		FROM container_metrics
	`
	if targetID != "" {
		query += " WHERE container_id = " + quote(targetID)
	}
	query += " ORDER BY collected_at DESC LIMIT 1 BY node_id, container_id LIMIT " + strconv.Itoa(limit) + " FORMAT JSONEachRow"

	rows, err := s.queryJSONEachRow(ctx, query)
	if err != nil {
		return nil, err
	}
	snapshots := make([]domain.MetricSnapshot, 0, len(rows))
	for _, row := range rows {
		snapshots = append(snapshots, snapshotFromRow(row))
	}
	return snapshots, nil
}

func (s *Store) MetricHistory(ctx context.Context, targetID, metricName string, from, to time.Time, limit int) ([]domain.MetricPoint, error) {
	if limit <= 0 || limit > 2000 {
		limit = 500
	}
	query := `
		SELECT collected_at, node_id, container_id, name, cpu_percent, memory_bytes, rx_bytes, tx_bytes, block_read, block_write
		FROM container_metrics
		WHERE 1 = 1
	`
	if targetID != "" {
		query += " AND container_id = " + quote(targetID)
	}
	if !from.IsZero() {
		query += " AND collected_at >= parseDateTime64BestEffort(" + quote(from.Format(time.RFC3339Nano)) + ")"
	}
	if !to.IsZero() {
		query += " AND collected_at <= parseDateTime64BestEffort(" + quote(to.Format(time.RFC3339Nano)) + ")"
	}
	query += " ORDER BY collected_at DESC LIMIT " + strconv.Itoa(limit) + " FORMAT JSONEachRow"

	rows, err := s.queryJSONEachRow(ctx, query)
	if err != nil {
		return nil, err
	}
	points := make([]domain.MetricPoint, 0, len(rows))
	for _, row := range rows {
		points = append(points, pointsFromRow(row, metricName)...)
	}
	return points, nil
}

func New(dsn string) *Store {
	return &Store{dsn: dsn}
}

func (s *Store) Ping(ctx context.Context) error {
	return s.exec(ctx, "SELECT 1", nil)
}

func (s *Store) SaveMetric(ctx context.Context, metric domain.MetricSample) error {
	row := map[string]any{
		"collected_at": clickHouseTime(metric.Timestamp),
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
		"occurred_at":  clickHouseTime(event.Timestamp),
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

func (s *Store) queryJSONEachRow(ctx context.Context, query string) ([]map[string]any, error) {
	endpoint := strings.TrimRight(s.dsn, "/") + "/?query=" + url.QueryEscape(query)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, nil)
	if err != nil {
		return nil, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("clickhouse status %s", resp.Status)
	}
	decoder := json.NewDecoder(resp.Body)
	var rows []map[string]any
	for {
		var row map[string]any
		if err := decoder.Decode(&row); err != nil {
			if err == io.EOF {
				break
			}
			return nil, err
		}
		rows = append(rows, row)
	}
	return rows, nil
}

func snapshotFromRow(row map[string]any) domain.MetricSnapshot {
	return domain.MetricSnapshot{
		NodeID:           stringValue(row["node_id"]),
		TargetID:         stringValue(row["container_id"]),
		ContainerName:    stringValue(row["name"]),
		CPUUsagePercent:  floatValue(row["cpu_percent"]),
		MemoryUsageBytes: uint64(floatValue(row["memory_bytes"])),
		NetworkRxBytes:   uint64(floatValue(row["rx_bytes"])),
		NetworkTxBytes:   uint64(floatValue(row["tx_bytes"])),
		BlockReadBytes:   uint64(floatValue(row["block_read"])),
		BlockWriteBytes:  uint64(floatValue(row["block_write"])),
		Timestamp:        timeValue(row["collected_at"]),
	}
}

func pointsFromRow(row map[string]any, metricName string) []domain.MetricPoint {
	all := []domain.MetricPoint{
		pointFromRow(row, "cpu_usage_percent", floatValue(row["cpu_percent"]), "percent"),
		pointFromRow(row, "memory_usage_bytes", floatValue(row["memory_bytes"]), "bytes"),
		pointFromRow(row, "network_rx_bytes", floatValue(row["rx_bytes"]), "bytes"),
		pointFromRow(row, "network_tx_bytes", floatValue(row["tx_bytes"]), "bytes"),
		pointFromRow(row, "block_read_bytes", floatValue(row["block_read"]), "bytes"),
		pointFromRow(row, "block_write_bytes", floatValue(row["block_write"]), "bytes"),
	}
	if metricName == "" {
		return all
	}
	filtered := make([]domain.MetricPoint, 0, 1)
	for _, point := range all {
		if point.MetricName == metricName {
			filtered = append(filtered, point)
		}
	}
	return filtered
}

func pointFromRow(row map[string]any, name string, value float64, unit string) domain.MetricPoint {
	return domain.MetricPoint{
		NodeID:        stringValue(row["node_id"]),
		TargetID:      stringValue(row["container_id"]),
		ContainerName: stringValue(row["name"]),
		MetricName:    name,
		Value:         value,
		Unit:          unit,
		Timestamp:     timeValue(row["collected_at"]),
	}
}

func quote(value string) string {
	escaped := strings.ReplaceAll(value, "'", "''")
	return "'" + escaped + "'"
}

func clickHouseTime(value time.Time) string {
	return value.UTC().Format("2006-01-02 15:04:05.999")
}

func stringValue(value any) string {
	if value == nil {
		return ""
	}
	return fmt.Sprint(value)
}

func floatValue(value any) float64 {
	switch typed := value.(type) {
	case float64:
		return typed
	case int:
		return float64(typed)
	case int64:
		return float64(typed)
	case json.Number:
		parsed, _ := typed.Float64()
		return parsed
	case string:
		parsed, _ := strconv.ParseFloat(typed, 64)
		return parsed
	default:
		return 0
	}
}

func timeValue(value any) time.Time {
	raw := stringValue(value)
	for _, layout := range []string{time.RFC3339Nano, "2006-01-02 15:04:05.999", "2006-01-02 15:04:05"} {
		parsed, err := time.Parse(layout, raw)
		if err == nil {
			return parsed.UTC()
		}
	}
	return time.Time{}
}
