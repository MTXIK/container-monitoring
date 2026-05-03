package recovery

import (
	"context"
	"testing"
	"time"

	"github.com/nikponomarevan/container-monitoring-core/internal/domain"
)

type recordingLocker struct {
	locked bool
}

func (l *recordingLocker) AcquireRecoveryLock(context.Context, string, time.Duration) (bool, error) {
	if l.locked {
		return false, nil
	}
	l.locked = true
	return true, nil
}

type recordingRecorder struct {
	actions []domain.RecoveryAction
	status  string
}

func (r *recordingRecorder) CreateRecoveryAction(_ context.Context, action domain.RecoveryAction) (domain.RecoveryAction, error) {
	action.ID = int64(len(r.actions) + 1)
	r.actions = append(r.actions, action)
	return action, nil
}

func (r *recordingRecorder) FinishRecoveryAction(_ context.Context, id int64, status, message string) error {
	r.status = status
	return nil
}

type recordingExecutor struct{}

func (e recordingExecutor) Execute(context.Context, Request) (string, error) {
	return "ok", nil
}

func TestRecoverUsesSucceededStatus(t *testing.T) {
	recorder := &recordingRecorder{}
	coordinator := NewCoordinator(&recordingLocker{}, recorder, recordingExecutor{})

	err := coordinator.Recover(context.Background(), domain.Incident{
		ID:       1,
		TargetID: "container-id",
		NodeID:   "node-1",
	}, string(ActionRestartContainer))

	if err != nil {
		t.Fatalf("Recover() error = %v", err)
	}
	if recorder.status != "succeeded" {
		t.Fatalf("status = %q, want succeeded", recorder.status)
	}
}

func TestRecoverUsesSkippedStatusWhenLockHeld(t *testing.T) {
	recorder := &recordingRecorder{}
	locker := &recordingLocker{locked: true}
	coordinator := NewCoordinator(locker, recorder, recordingExecutor{})

	err := coordinator.Recover(context.Background(), domain.Incident{
		ID:       1,
		TargetID: "container-id",
		NodeID:   "node-1",
	}, string(ActionRestartContainer))

	if err != nil {
		t.Fatalf("Recover() error = %v", err)
	}
	if recorder.status != "skipped" {
		t.Fatalf("status = %q, want skipped", recorder.status)
	}
}
