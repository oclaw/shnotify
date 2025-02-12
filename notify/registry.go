package notify

import (
	"fmt"
	"context"
	"github.com/oclaw/shnotify/types"
)

type Notifier interface {
	Notify(context.Context, *types.NotificationData) error
}

type Registry struct {
	notifiers map[types.NotificationType]Notifier
}

func NewRegistry() *Registry {
	return &Registry{
		notifiers: make(map[types.NotificationType]Notifier),
	}
}

func (rg *Registry) RegisterNotifier(nType types.NotificationType, impl Notifier) {
	if _, exists := rg.notifiers[nType]; exists {
		panic(fmt.Errorf("duplicate registration for %s\n", nType))
	}
	rg.notifiers[nType] = impl
}

func (rg *Registry) GetNotifier(ctx context.Context, nType types.NotificationType) (Notifier, error) {
	n, ok := rg.notifiers[nType]
	if !ok {
		return nil, fmt.Errorf("notifier %s is not supported", nType)
	}
	return n, nil
}
