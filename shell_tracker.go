package main

import (
	"context"
	"fmt"
	"os"
	"path"

	"github.com/oclaw/shnotify/notify"
	"github.com/oclaw/shnotify/notify/cli"
	"github.com/oclaw/shnotify/notify/telegram"
	"github.com/oclaw/shnotify/types"
)

type shellTracker struct {
	config          *ShellTrackerConfig
	clock           Clock
	storage         InvocationStorage
	registry        *notify.Registry
}

func NewShellTracker(
	cfg *ShellTrackerConfig,
	clock Clock,
) (*shellTracker, error) {

	storage, err := NewFsInvocationStorage(cfg.DirPath)
	if err != nil {
		return nil, err
	}

	reg := notify.NewRegistry()

	for _, notif := range cfg.Notifications {

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
			notifier, err := telegram.NewTelegramNotifier(string(token), cfg.NotifierSettings.TelegramChatID) // TODO validate presence of chat id
			if err != nil {
				return nil, err
			}
			reg.RegisterNotifier(types.NotificationTelegram, notifier)
		}
	}

	return &shellTracker{
		config:          cfg,
		storage:         storage,
		clock:           clock,
		registry:        reg,
	}, nil
}

func (st *shellTracker) NotifyInvocationFinished(ctx context.Context, extInvocationId string) error {
	now := st.clock.NowUnix()

	rec, err := st.storage.Get(ctx, extInvocationId)
	if err != nil {
		return err
	}

	execTime := now - rec.Timestamp

	// TODO abstract config condition matchers
	for _, notifConfig := range st.config.Notifications {
		var notificationNeeded bool
		longerThan := notifConfig.Conditions.RunLongerThanSec
		if longerThan != nil && *longerThan <= execTime {
			notificationNeeded = true
		}


		if notificationNeeded {
			notifier, err := st.registry.GetNotifier(ctx, notifConfig.Type)
			if err != nil {
				fmt.Printf("notification type '%s' failed: %v\n", notifConfig.Type, err)
				continue
			}
			if err := notifier.Notify(ctx, &types.NotificationData{
				Invocation:   rec,
				NowTimestamp: now,
				ExecTime:     execTime,
			}); err != nil {
				return err
			}
		}
	}

	if st.config.CleanupEnabled {
		err = st.storage.Erase(ctx, extInvocationId)
	}

	return err
}
