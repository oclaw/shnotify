package core

import (
	"context"
	"fmt"
	"os"
	"path"
	"strings"
	"sync"

	"github.com/google/uuid"
	"github.com/oclaw/shnotify/common"
	"github.com/oclaw/shnotify/config"
	"github.com/oclaw/shnotify/notify"
	"github.com/oclaw/shnotify/notify/cli"
	"github.com/oclaw/shnotify/notify/telegram"
	"github.com/oclaw/shnotify/types"
)

var UUIDInvocationGen = types.InvocationGenFromStringer(uuid.NewUUID)

type invocationTrackerImpl struct {
	config  *config.ShellTrackerConfig
	clock   common.Clock
	storage InvocationStorage
	gen     types.InvocationIDGen

	regInitOnce sync.Once
	registry    *notify.Registry
}

var _ InvocationTracker = (*invocationTrackerImpl)(nil)

func NewInvocationTracker(
	cfg *config.ShellTrackerConfig,
	clock common.Clock,
	gen types.InvocationIDGen,
) (*invocationTrackerImpl, error) {

	storage, err := NewFsInvocationStorage(cfg.DirPath)
	if err != nil {
		return nil, err
	}

	it := &invocationTrackerImpl{
		config:  cfg,
		storage: storage,
		gen:     gen,
		clock:   clock,
	}

	switch cfg.InitMode {
	case config.NotifierInitOnStartup:
		err = it.initNotifiers()
	}
	if err != nil {
		return nil, err
	}

	return it, nil
}

func (it *invocationTrackerImpl) initNotifiers() error {

	doInitNotifiers := func() (*notify.Registry, error) {
		reg := notify.NewRegistry()

		for _, notif := range it.config.Notifications {

			// TODO abstract to initializer map
			switch notif.Type {
			case types.NotificationCLI:
				reg.RegisterNotifier(types.NotificationCLI, cli.NewCliNotifier(os.Stdout))
			case types.NotificationTelegram:
				dir, err := os.UserConfigDir()
				if err != nil {
					return nil, err
				}
				token, err := os.ReadFile(path.Join(dir, "shnotify", ".tg.token"))
				if err != nil {
					return nil, err
				}
				notifier, err := telegram.NewTelegramNotifier(string(token), it.config.NotifierSettings.TelegramChatID) // TODO validate presence of chat id
				if err != nil {
					return nil, err
				}
				reg.RegisterNotifier(types.NotificationTelegram, notifier)
			}
		}

		return reg, nil
	}

	var err error
	it.regInitOnce.Do(func() {
		it.registry, err = doInitNotifiers()
		// fmt.Printf("notifiers initialized\n")
	})

	return err
}

type preprocessedCommand struct {
	ShellLine string // cleaned up and safe to save on filesystem shell line
	Binary    string // extracted binary name (e.g. 'ping', 'traceroute', etc)
}

func (it *invocationTrackerImpl) preprocessCommand(line string) (preprocessedCommand, error) {
	return preprocessedCommand{
		ShellLine: strings.TrimSpace(line), // TODO add command cleanup
		Binary:    "",                      // TODO add extracting of the binary
	}, nil
}

func (it *invocationTrackerImpl) SaveInvocation(
	ctx context.Context,
	req *types.InvocationRequest,
) (types.InvocationID, error) {

	rec := types.ShellInvocationRecord{
		InvocationID: req.InvocationID,
		ParentID:     req.ParentID,
		Timestamp:    it.clock.NowUnix(),
	}

	if len(rec.InvocationID) == 0 {
		var err error
		rec.InvocationID, err = it.gen()
		if err != nil {
			return "", err
		}
	}

	command, err := it.preprocessCommand(req.ShellLine)
	if err != nil {
		return "", err
	}

	// TODO ban & allowlist lookups

	rec.ShellLine = command.ShellLine

	if err := it.storage.Store(ctx, &rec); err != nil {
		return "", err
	}

	return rec.InvocationID, nil
}

func (it *invocationTrackerImpl) Notify(ctx context.Context, invocationID types.InvocationID) error {
	var err error
	switch it.config.InitMode {
	case config.NotifierInitOnDemand:
		err = it.initNotifiers()
	}
	if err != nil {
		return err
	}

	now := it.clock.NowUnix()

	rec, err := it.storage.Get(ctx, invocationID)
	if err != nil {
		return err
	}

	execTime := now - rec.Timestamp

	// TODO abstract config condition matchers
	for _, notifConfig := range it.config.Notifications {
		var notificationNeeded bool
		threshold := notifConfig.Conditions.RunLongerThan
		if threshold.LessThan(execTime) {
			notificationNeeded = true
		}

		if notificationNeeded {
			notifier, err := it.registry.GetNotifier(ctx, notifConfig.Type)
			if err != nil {
				fmt.Printf("notification type '%s' failed: %v\n", notifConfig.Type, err)
				continue
			}
			if err := it.notify(
				ctx,
				notifier,
				&types.NotificationData{
					Invocation:   rec,
					NowTimestamp: now,
					ExecTime:     execTime,
				}); err != nil {
				return err
			}
		}
	}

	if it.config.CleanupEnabled {
		err = it.storage.Erase(ctx, invocationID)
	}

	return err
}

func (it *invocationTrackerImpl) notify(
	ctx context.Context,
	notifier notify.Notifier,
	data *types.NotificationData,
) error {

	call := func(ctx context.Context) error {
		return notifier.Notify(ctx, data)
	}
	if !it.config.AsyncNotifications {
		return call(ctx)
	}

	go func() {
		ctx := context.WithoutCancel(ctx)
		if err := call(ctx); err != nil {
			fmt.Printf("Notification for invocation %s failed: %v", data.Invocation.InvocationID, err)
		}
	}()
	return nil
}
