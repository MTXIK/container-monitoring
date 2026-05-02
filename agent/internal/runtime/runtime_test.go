package runtime

import (
	"context"
	"io"
	"log/slog"
	"sync/atomic"
	"syscall"
	"testing"
	"time"

	"github.com/nikponomarevan/container-monitoring-agent/internal/collector"
	"github.com/nikponomarevan/container-monitoring-agent/internal/config"
)

func TestRunDisablesClosedEventErrorChannel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	backend := &stubBackend{
		events: make(chan collector.Event),
		errs:   make(chan error),
	}
	close(backend.events)
	close(backend.errs)

	pub := &stubPublisher{}
	done := make(chan error, 1)
	go func() {
		done <- Run(ctx, slog.New(slog.NewTextHandler(io.Discard, nil)), config.Config{
			CollectInterval: time.Hour,
		}, backend, pub)
	}()

	startCPU := processCPUTime(t)
	time.Sleep(100 * time.Millisecond)
	if count := atomic.LoadInt64(&backend.collectCalls); count > 0 {
		t.Fatalf("CollectMetrics() calls before ticker = %d, want 0", count)
	}
	usedCPU := processCPUTime(t) - startCPU
	if usedCPU > 50*time.Millisecond {
		t.Fatalf("Run() used %s CPU while only closed event channels were ready; want no busy loop", usedCPU)
	}

	cancel()
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("Run() error = %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("Run() did not stop after context cancellation")
	}
}

func processCPUTime(t *testing.T) time.Duration {
	t.Helper()

	var usage syscall.Rusage
	if err := syscall.Getrusage(syscall.RUSAGE_SELF, &usage); err != nil {
		t.Fatalf("Getrusage() error = %v", err)
	}
	return timevalDuration(usage.Utime) + timevalDuration(usage.Stime)
}

func timevalDuration(value syscall.Timeval) time.Duration {
	return time.Duration(value.Sec)*time.Second + time.Duration(value.Usec)*time.Microsecond
}

type stubBackend struct {
	events       chan collector.Event
	errs         chan error
	collectCalls int64
}

func (b *stubBackend) CollectMetrics(context.Context) ([]collector.Metric, error) {
	atomic.AddInt64(&b.collectCalls, 1)
	return nil, nil
}

func (b *stubBackend) WatchEvents(context.Context) (<-chan collector.Event, <-chan error) {
	return b.events, b.errs
}

type stubPublisher struct{}

func (p *stubPublisher) PublishMetrics(context.Context, []collector.Metric) error {
	return nil
}

func (p *stubPublisher) PublishEvent(context.Context, collector.Event) error {
	return nil
}

func (p *stubPublisher) Close() error {
	return nil
}
