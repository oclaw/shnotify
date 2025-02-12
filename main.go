package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"strconv"

	"github.com/google/uuid"
	"github.com/oclaw/shnotify/notify"
	"github.com/oclaw/shnotify/notify/cli"
	"github.com/oclaw/shnotify/types"
	"github.com/spf13/cobra"
)

type InvocationIDGen func() (string, error)

func InvocationGenFromStringer[T fmt.Stringer](gen func() (T, error)) InvocationIDGen {
	return func() (string, error) {
		val, err := gen()
		if err != nil {
			return "", err
		}
		return val.String(), nil
	}
}

var uuidInvocationGen = InvocationGenFromStringer(uuid.NewUUID)

type shellTracker struct {
	config          *ShellTrackerConfig
	clock           Clock
	storage         InvocationStorage
	invocationIDGen InvocationIDGen
	registry        *notify.Registry
}

func NewShellTracker(
	cfg *ShellTrackerConfig,
	clock Clock,
	invocationGen InvocationIDGen,
) (*shellTracker, error) {

	storage, err := NewFsInvocationStorage(cfg.DirPath)
	if err != nil {
		return nil, err
	}

	reg := notify.NewRegistry()
	reg.RegisterNotifier(types.NotificationCLI, cli.NewCliNotifier(os.Stdout))

	return &shellTracker{
		config:          cfg,
		storage:         storage,
		clock:           clock,
		invocationIDGen: invocationGen,
		registry:        reg,
	}, nil
}

func Ignore(err error, toIgnore ...error) error {
	if err == nil {
		return nil
	}
	for _, ignoring := range toIgnore {
		if errors.Is(err, ignoring) {
			return nil
		}
	}
	return err
}

type preprocessedCommand struct {
	ShellLine string // cleaned up and safe to save on filesystem shell line
	Binary    string // extracted binary name (e.g. 'ping', 'traceroute', etc)
}

func (st *shellTracker) preprocessCommand(line string) (preprocessedCommand, error) {
	return preprocessedCommand{
		ShellLine: line, // TODO add command cleanup
		Binary:    "",   // TODO add extracting of the binary
	}, nil
}

func (st *shellTracker) getExtInvocationID(rec *types.ShellInvocationRecord) (string, error) {
	if len(rec.ShellLine) == 0 {
		return "", fmt.Errorf("Empty shell line input for invocation '%s'", rec.InvocationID)
	}
	hash := sha256.New()
	hash.Write([]byte(rec.InvocationID))
	hash.Write([]byte(strconv.Itoa(rec.ParentID)))
	hash.Write([]byte(rec.ShellLine))
	ret := hash.Sum(nil)
	return hex.EncodeToString(ret), nil
}

func (st *shellTracker) saveInvocation(ctx context.Context, shellLine, invocationID string) (string, error) {
	rec := types.ShellInvocationRecord{
		InvocationID: invocationID,
		ParentID:     os.Getppid(),
		Timestamp:    st.clock.NowUnix(),
	}

	if len(rec.InvocationID) == 0 {
		var err error
		rec.InvocationID, err = st.invocationIDGen()
		if err != nil {
			return "", err
		}
	}

	command, err := st.preprocessCommand(shellLine)
	if err != nil {
		return "", err
	}

	// TODO ban & allowlist lookups

	rec.ShellLine = command.ShellLine

	rec.ExternalInvocationID, err = st.getExtInvocationID(&rec)
	if err != nil {
		return "", err
	}

	if err := st.storage.Store(ctx, &rec); err != nil {
		return "", err
	}

	return rec.ExternalInvocationID, nil
}

func (st *shellTracker) notifyInvocationFinished(ctx context.Context, extInvocationId string) error {
	now := st.clock.NowUnix()

	rec, err := st.storage.Get(ctx, extInvocationId)
	if err != nil {
		return err
	}

	execTime := now - rec.Timestamp

	// TODO abstract everything out
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

func initShellTracker(ctx context.Context) (*shellTracker, error) {
	cfg, err := ReadFromDefaultLoc()
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			fmt.Printf("config does not exist, will create default one")
			cfg = DefaultShellTrackerConfig()
			if err := SaveConfigToDefaultLoc(cfg); err != nil {
				fmt.Printf("failed to save config: %v", err)
				return nil, err
			}
		} else {
			fmt.Printf("failed to read config from default location, err: %v", err)
			return nil, err
		}
	}

	shellTracker, err := NewShellTracker(cfg, &defaultClock{}, uuidInvocationGen)
	if err != nil {
		return nil, err
	}

	if err := shellTracker.Start(ctx); err != nil {
		return nil, err
	}

	return shellTracker, nil
}

func buildStartInvocationCommand(ctx context.Context, shellTracker *shellTracker) *cobra.Command {
	var (
		shellLine, shellInvocationId string
	)
	saveInvocationCommand := &cobra.Command{
		Use:   "save-invocation",
		Short: "save invocation of the shell command into the storage and return the external id assigned to the execution",
		RunE: func(cmd *cobra.Command, args []string) error {
			ret, err := shellTracker.saveInvocation(ctx, shellLine, shellInvocationId)
			if err != nil {
				return err
			}
			cmd.OutOrStdout().Write([]byte(ret))
			return nil
		},
	}
	saveInvocationCommand.Flags().StringVar(&shellLine, "shell-line", "", "shell command line to put into the invocation")
	saveInvocationCommand.Flags().StringVar(&shellInvocationId, "invocation-id", "", "externally defined invocation id (empty by default)")
	return saveInvocationCommand
}

func buildNotifyCommand(ctx context.Context, shellTracker *shellTracker) *cobra.Command {
	var extInvocationId string
	notifyCommand := cobra.Command{
		Use:   "notify",
		Short: "trigger notification for invocation that has finished executing",
		RunE: func(cmd *cobra.Command, args []string) error {
			return shellTracker.notifyInvocationFinished(ctx, extInvocationId)
		},
	}
	notifyCommand.Flags().StringVar(&extInvocationId, "invocation-id", "", "shell command invocation id returned by save-invocation call")
	return &notifyCommand
}

func main() {
	root := cobra.Command{
		Use:   os.Args[0],
		Short: "Shell invocation tracking and notifying utility",
	}

	ctx := context.Background()
	shellTracker, err := initShellTracker(ctx)
	if err != nil {
		fmt.Printf("failed to initialize app: %v", err)
		os.Exit(1)
	}

	saveInvocationCommand := buildStartInvocationCommand(ctx, shellTracker)
	notifyCommand := buildNotifyCommand(ctx, shellTracker)

	root.AddCommand(saveInvocationCommand)
	root.AddCommand(notifyCommand)

	if err := root.Execute(); err != nil {
		os.Exit(1)
	}
}
