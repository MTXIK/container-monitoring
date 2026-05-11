package clickhouse

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestClickHouseTimeUsesDateTime64JSONFormat(t *testing.T) {
	value := time.Date(2026, 5, 11, 15, 36, 20, 396022177, time.FixedZone("MSK", 3*60*60))

	got := clickHouseTime(value)
	want := "2026-05-11 12:36:20.396"
	if got != want {
		t.Fatalf("clickHouseTime() = %q, want %q", got, want)
	}
}

func TestLatestMetricsSelectsLatestSnapshotPerNodeAndContainer(t *testing.T) {
	var gotQuery string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.Query().Get("query")
		w.Header().Set("Content-Type", "application/x-ndjson")
		_, _ = w.Write([]byte(`{"collected_at":"2026-05-11 12:36:20.396","node_id":"node-a","container_id":"container-a","name":"nginx","cpu_percent":42,"memory_bytes":1024,"rx_bytes":1,"tx_bytes":2,"block_read":3,"block_write":4}` + "\n"))
	}))
	defer server.Close()

	store := New(server.URL)
	metrics, err := store.LatestMetrics(context.Background(), "", 5)
	if err != nil {
		t.Fatalf("LatestMetrics() error = %v", err)
	}
	if len(metrics) != 1 {
		t.Fatalf("LatestMetrics() returned %d rows, want 1", len(metrics))
	}

	want := "ORDER BY collected_at DESC LIMIT 1 BY node_id, container_id LIMIT 5 FORMAT JSONEachRow"
	if !strings.Contains(oneLine(gotQuery), want) {
		t.Fatalf("LatestMetrics() query = %q, want to contain %q", oneLine(gotQuery), want)
	}
}

func oneLine(value string) string {
	return strings.Join(strings.Fields(value), " ")
}
