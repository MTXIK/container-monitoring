package docker

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/nikponomarevan/container-monitoring-agent/internal/collector"
)

type Backend struct {
	nodeID     string
	dockerHost string
	client     *http.Client
	baseURL    string
}

func NewBackend(nodeID, dockerHost string) *Backend {
	client, baseURL := newDockerHTTPClient(dockerHost)
	return &Backend{nodeID: nodeID, dockerHost: dockerHost, client: client, baseURL: baseURL}
}

func (b *Backend) CollectMetrics(ctx context.Context) ([]collector.Metric, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, b.baseURL+"/containers/json", nil)
	if err != nil {
		return nil, err
	}
	response, err := b.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	if response.StatusCode >= 300 {
		return nil, fmt.Errorf("list containers: docker status %s", response.Status)
	}

	var containers []containerSummary
	if err := json.NewDecoder(response.Body).Decode(&containers); err != nil {
		return nil, err
	}

	metrics := make([]collector.Metric, 0, len(containers))
	for _, container := range containers {
		stats, err := b.fetchStats(ctx, container.ID)
		if err != nil {
			return nil, err
		}
		if stats.ID == "" {
			stats.ID = container.ID
		}
		if stats.Name == "" && len(container.Names) > 0 {
			stats.Name = container.Names[0]
		}
		metrics = append(metrics, metricFromStats(b.nodeID, stats))
	}
	return metrics, nil
}

func (b *Backend) WatchEvents(ctx context.Context) (<-chan collector.Event, <-chan error) {
	events := make(chan collector.Event)
	errs := make(chan error, 1)

	go func() {
		defer close(events)
		defer close(errs)
		filters := url.QueryEscape(`{"type":["container"],"event":["start","stop","die","oom","restart"]}`)
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, b.baseURL+"/events?filters="+filters, nil)
		if err != nil {
			errs <- err
			return
		}
		response, err := b.client.Do(req)
		if err != nil {
			if ctx.Err() == nil {
				errs <- err
			}
			return
		}
		defer response.Body.Close()
		if response.StatusCode >= 300 {
			errs <- fmt.Errorf("watch events: docker status %s", response.Status)
			return
		}

		scanner := bufio.NewScanner(response.Body)
		for scanner.Scan() {
			var dockerEvent dockerEvent
			if err := json.Unmarshal(scanner.Bytes(), &dockerEvent); err != nil {
				errs <- err
				continue
			}
			event, ok := b.normalizeEvent(dockerEvent)
			if !ok {
				continue
			}
			select {
			case events <- event:
			case <-ctx.Done():
				return
			}
		}
		if err := scanner.Err(); err != nil && ctx.Err() == nil {
			errs <- err
		}
	}()

	return events, errs
}

func (b *Backend) fetchStats(ctx context.Context, containerID string) (containerStats, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, b.baseURL+"/containers/"+containerID+"/stats?stream=false", nil)
	if err != nil {
		return containerStats{}, err
	}
	response, err := b.client.Do(req)
	if err != nil {
		return containerStats{}, err
	}
	defer response.Body.Close()
	if response.StatusCode >= 300 {
		return containerStats{}, fmt.Errorf("container stats %s: docker status %s", containerID, response.Status)
	}
	var stats containerStats
	if err := json.NewDecoder(response.Body).Decode(&stats); err != nil {
		return containerStats{}, err
	}
	return stats, nil
}

func (b *Backend) normalizeEvent(event dockerEvent) (collector.Event, bool) {
	eventType, severity, ok := mapEvent(event.Action)
	if !ok {
		return collector.Event{}, false
	}
	name := event.Actor.Attributes["name"]
	message := fmt.Sprintf("Container %s %s", name, strings.TrimPrefix(string(eventType), "container_"))
	return collector.Event{
		NodeID:      b.nodeID,
		ContainerID: event.ID,
		Name:        name,
		Type:        eventType,
		Severity:    severity,
		Message:     message,
		Payload: map[string]any{
			"action":     event.Action,
			"exit_code":  event.Actor.Attributes["exitCode"],
			"oom_killed": event.Actor.Attributes["oomKilled"],
		},
		OccurredAt: event.time(),
	}, true
}

func mapEvent(action string) (collector.EventType, collector.Severity, bool) {
	switch action {
	case "start":
		return collector.EventStart, collector.SeverityInfo, true
	case "stop":
		return collector.EventStop, collector.SeverityWarning, true
	case "die":
		return collector.EventDie, collector.SeverityCritical, true
	case "oom":
		return collector.EventOOM, collector.SeverityCritical, true
	case "restart":
		return collector.EventRestart, collector.SeverityWarning, true
	default:
		return "", "", false
	}
}

func metricFromStats(nodeID string, stats containerStats) collector.Metric {
	var rxBytes, txBytes uint64
	for _, network := range stats.Networks {
		rxBytes += network.RxBytes
		txBytes += network.TxBytes
	}

	var blockRead, blockWrite uint64
	for _, entry := range stats.BlkioStats.IOServiceBytesRecursive {
		switch strings.ToLower(entry.Op) {
		case "read":
			blockRead += entry.Value
		case "write":
			blockWrite += entry.Value
		}
	}

	memoryPercent := 0.0
	if stats.MemoryStats.Limit > 0 {
		memoryPercent = float64(stats.MemoryStats.Usage) / float64(stats.MemoryStats.Limit) * 100
	}

	return collector.Metric{
		NodeID:             nodeID,
		ContainerID:        stats.ID,
		Name:               strings.TrimPrefix(stats.Name, "/"),
		CPUUsagePercent:    calculateCPUPercent(stats),
		MemoryUsageBytes:   stats.MemoryStats.Usage,
		MemoryUsagePercent: memoryPercent,
		NetworkRxBytes:     rxBytes,
		NetworkTxBytes:     txBytes,
		BlockReadBytes:     blockRead,
		BlockWriteBytes:    blockWrite,
		CollectedAt:        stats.Read,
	}
}

func calculateCPUPercent(stats containerStats) float64 {
	cpuDelta := float64(stats.CPUStats.CPUUsage.TotalUsage - stats.PreCPUStats.CPUUsage.TotalUsage)
	systemDelta := float64(stats.CPUStats.SystemCPUUsage - stats.PreCPUStats.SystemCPUUsage)
	onlineCPUs := float64(stats.CPUStats.OnlineCPUs)
	if onlineCPUs == 0 {
		onlineCPUs = float64(len(stats.CPUStats.CPUUsage.PercpuUsage))
	}
	if cpuDelta <= 0 || systemDelta <= 0 || onlineCPUs <= 0 {
		return 0
	}
	return cpuDelta / systemDelta * onlineCPUs * 100
}

func newDockerHTTPClient(dockerHost string) (*http.Client, string) {
	if strings.HasPrefix(dockerHost, "unix://") {
		socketPath := strings.TrimPrefix(dockerHost, "unix://")
		transport := &http.Transport{
			DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
				var dialer net.Dialer
				return dialer.DialContext(ctx, "unix", socketPath)
			},
		}
		return &http.Client{Transport: transport}, "http://docker"
	}
	if dockerHost == "" {
		return http.DefaultClient, "http://localhost"
	}
	return http.DefaultClient, strings.TrimRight(dockerHost, "/")
}

type containerSummary struct {
	ID    string   `json:"Id"`
	Names []string `json:"Names"`
}

type containerStats struct {
	ID          string                  `json:"id"`
	Name        string                  `json:"name"`
	Read        time.Time               `json:"read"`
	CPUStats    cpuStats                `json:"cpu_stats"`
	PreCPUStats cpuStats                `json:"precpu_stats"`
	MemoryStats memoryStats             `json:"memory_stats"`
	Networks    map[string]networkStats `json:"networks"`
	BlkioStats  blkioStats              `json:"blkio_stats"`
}

type cpuStats struct {
	CPUUsage       cpuUsage `json:"cpu_usage"`
	SystemCPUUsage uint64   `json:"system_cpu_usage"`
	OnlineCPUs     uint32   `json:"online_cpus"`
}

type cpuUsage struct {
	TotalUsage        uint64   `json:"total_usage"`
	PercpuUsage       []uint64 `json:"percpu_usage"`
	UsageInKernelmode uint64   `json:"usage_in_kernelmode"`
	UsageInUsermode   uint64   `json:"usage_in_usermode"`
}

type memoryStats struct {
	Usage uint64 `json:"usage"`
	Limit uint64 `json:"limit"`
}

type networkStats struct {
	RxBytes uint64 `json:"rx_bytes"`
	TxBytes uint64 `json:"tx_bytes"`
}

type blkioStats struct {
	IOServiceBytesRecursive []blkioEntry `json:"io_service_bytes_recursive"`
}

type blkioEntry struct {
	Op    string `json:"op"`
	Value uint64 `json:"value"`
}

type dockerEvent struct {
	Action   string `json:"Action"`
	ID       string `json:"id"`
	Time     int64  `json:"time"`
	TimeNano int64  `json:"timeNano"`
	Actor    struct {
		Attributes map[string]string `json:"Attributes"`
	} `json:"Actor"`
}

func (e dockerEvent) time() time.Time {
	if e.TimeNano > 0 {
		return time.Unix(0, e.TimeNano).UTC()
	}
	if e.Time > 0 {
		return time.Unix(e.Time, 0).UTC()
	}
	return time.Now().UTC()
}
