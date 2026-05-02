package recovery

import "context"

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
	Execute(ctx context.Context, request Request) error
}
