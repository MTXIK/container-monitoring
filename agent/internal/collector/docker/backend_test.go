package docker

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestMetricFromStatsCalculatesDockerResourceMetrics(t *testing.T) {
	collectedAt := time.Date(2026, 5, 2, 12, 0, 0, 0, time.UTC)
	stats := containerStats{
		ID:   "container-id",
		Name: "/nginx",
		CPUStats: cpuStats{
			CPUUsage: cpuUsage{
				TotalUsage:        600,
				PercpuUsage:       []uint64{100, 100},
				UsageInKernelmode: 30,
				UsageInUsermode:   70,
			},
			SystemCPUUsage: 2000,
		},
		PreCPUStats: cpuStats{
			CPUUsage:       cpuUsage{TotalUsage: 100},
			SystemCPUUsage: 1000,
		},
		MemoryStats: memoryStats{
			Usage: 50,
			Limit: 200,
		},
		Networks: map[string]networkStats{
			"eth0": {RxBytes: 10, TxBytes: 20},
			"eth1": {RxBytes: 30, TxBytes: 40},
		},
		BlkioStats: blkioStats{
			IOServiceBytesRecursive: []blkioEntry{
				{Op: "Read", Value: 100},
				{Op: "Write", Value: 250},
				{Op: "Sync", Value: 999},
			},
		},
		Read: collectedAt,
	}

	metric := metricFromStats("node-1", stats)

	if metric.NodeID != "node-1" || metric.ContainerID != "container-id" || metric.Name != "nginx" {
		t.Fatalf("identity = %#v", metric)
	}
	if metric.CPUUsagePercent != 100 {
		t.Fatalf("CPUUsagePercent = %v, want 100", metric.CPUUsagePercent)
	}
	if metric.MemoryUsageBytes != 50 {
		t.Fatalf("MemoryUsageBytes = %d, want 50", metric.MemoryUsageBytes)
	}
	if metric.MemoryUsagePercent != 25 {
		t.Fatalf("MemoryUsagePercent = %v, want 25", metric.MemoryUsagePercent)
	}
	if metric.NetworkRxBytes != 40 || metric.NetworkTxBytes != 60 {
		t.Fatalf("network = %d/%d, want 40/60", metric.NetworkRxBytes, metric.NetworkTxBytes)
	}
	if metric.BlockReadBytes != 100 || metric.BlockWriteBytes != 250 {
		t.Fatalf("block = %d/%d, want 100/250", metric.BlockReadBytes, metric.BlockWriteBytes)
	}
}

func TestCollectMetricsSkipsContainerWhenStatsDisappear(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/containers/json":
			_ = json.NewEncoder(w).Encode([]containerSummary{
				{ID: "missing", Names: []string{"/missing"}},
				{ID: "alive", Names: []string{"/alive"}},
			})
		case "/containers/missing/stats":
			http.Error(w, "not found", http.StatusNotFound)
		case "/containers/alive/stats":
			_ = json.NewEncoder(w).Encode(containerStats{
				ID:   "alive",
				Name: "/alive",
				Read: time.Date(2026, 5, 2, 12, 0, 0, 0, time.UTC),
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	backend := &Backend{
		nodeID:  "node-1",
		client:  server.Client(),
		baseURL: server.URL,
	}

	metrics, err := backend.CollectMetrics(context.Background())

	if err != nil {
		t.Fatalf("CollectMetrics() error = %v", err)
	}
	if len(metrics) != 1 {
		t.Fatalf("metrics = %d, want 1", len(metrics))
	}
	if metrics[0].ContainerID != "alive" {
		t.Fatalf("container = %q, want alive", metrics[0].ContainerID)
	}
}
