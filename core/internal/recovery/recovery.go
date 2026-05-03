package recovery

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/nikponomarevan/container-monitoring-core/internal/domain"
)

type Action string

const (
	ActionNotifyOnly       Action = "notify_only"
	ActionRetryCheck       Action = "retry_check"
	ActionRestartContainer Action = "restart_container"
)

type Request struct {
	NodeID      string
	ContainerID string
	Action      Action
}

type Executor interface {
	Execute(ctx context.Context, request Request) (string, error)
}

type Locker interface {
	AcquireRecoveryLock(ctx context.Context, targetID string, ttl time.Duration) (bool, error)
}

type Recorder interface {
	CreateRecoveryAction(ctx context.Context, action domain.RecoveryAction) (domain.RecoveryAction, error)
	FinishRecoveryAction(ctx context.Context, id int64, status, message string) error
}

type Coordinator struct {
	locker   Locker
	recorder Recorder
	executor Executor
	lockTTL  time.Duration
}

func NewCoordinator(locker Locker, recorder Recorder, executor Executor) *Coordinator {
	return &Coordinator{locker: locker, recorder: recorder, executor: executor, lockTTL: 5 * time.Minute}
}

func (c *Coordinator) Recover(ctx context.Context, incident domain.Incident, actionType string) error {
	action := Action(actionType)
	record, err := c.recorder.CreateRecoveryAction(ctx, domain.RecoveryAction{
		IncidentID: incident.ID,
		TargetID:   incident.TargetID,
		ActionType: string(action),
		Status:     "running",
		StartedAt:  time.Now().UTC(),
	})
	if err != nil {
		return err
	}

	if action == ActionNotifyOnly {
		return c.recorder.FinishRecoveryAction(ctx, record.ID, "succeeded", "notify_only")
	}
	locked, err := c.locker.AcquireRecoveryLock(ctx, incident.TargetID, c.lockTTL)
	if err != nil {
		_ = c.recorder.FinishRecoveryAction(ctx, record.ID, "failed", err.Error())
		return err
	}
	if !locked {
		return c.recorder.FinishRecoveryAction(ctx, record.ID, "skipped", "recovery lock is already held")
	}

	result, err := c.executor.Execute(ctx, Request{
		NodeID:      incident.NodeID,
		ContainerID: incident.TargetID,
		Action:      action,
	})
	if err != nil {
		_ = c.recorder.FinishRecoveryAction(ctx, record.ID, "failed", err.Error())
		return err
	}
	return c.recorder.FinishRecoveryAction(ctx, record.ID, "succeeded", result)
}

type DockerExecutor struct {
	client  *http.Client
	baseURL string
}

func NewDockerExecutor(dockerHost string) *DockerExecutor {
	client, baseURL := newDockerHTTPClient(dockerHost)
	return &DockerExecutor{client: client, baseURL: baseURL}
}

func (e *DockerExecutor) Execute(ctx context.Context, request Request) (string, error) {
	switch request.Action {
	case ActionRetryCheck:
		return "retry_check completed", nil
	case ActionRestartContainer:
		if e.baseURL == "" {
			return "", fmt.Errorf("RECOVERY_DOCKER_HOST is not configured")
		}
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, e.baseURL+"/containers/"+request.ContainerID+"/restart", nil)
		if err != nil {
			return "", err
		}
		resp, err := e.client.Do(req)
		if err != nil {
			return "", err
		}
		defer resp.Body.Close()
		if resp.StatusCode >= 300 {
			return "", fmt.Errorf("docker restart status %s", resp.Status)
		}
		return "container restart requested", nil
	default:
		return "no executor action required", nil
	}
}

func newDockerHTTPClient(dockerHost string) (*http.Client, string) {
	if dockerHost == "" {
		return http.DefaultClient, ""
	}
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
	return http.DefaultClient, strings.TrimRight(dockerHost, "/")
}
