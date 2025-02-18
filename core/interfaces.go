package core

import (
	"context"

	"github.com/oclaw/shnotify/types"
)

type InvocationTracker interface {
	SaveInvocation(ctx context.Context, req *types.InvocationRequest) (types.InvocationID, error)
	Notify(ctx context.Context, id types.InvocationID) error
}

type InvocationStorage interface {
	Store(ctx context.Context, rec *types.ShellInvocationRecord) error
	Get(ctx context.Context, id types.InvocationID) (*types.ShellInvocationRecord, error)
	Erase(ctx context.Context, id types.InvocationID) error
}
