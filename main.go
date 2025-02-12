package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"

	"github.com/google/uuid"
	"github.com/spf13/cobra"
)

type ShellInvocationRecord struct {
	InvocationID         string `json:"invocation_id"`
	ParentID             int    `json:"ppid"`
	ShellLine            string `json:"cmd_text"`
	Timestamp            int64  `json:"started_at"`
	ExternalInvocationID string `json:"ext_invocation_id"`
}

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

type NotificationData struct {
	Invocation *ShellInvocationRecord
	NowTimestamp int64
	ExecTime int64
	// feel free to add more data that can be reused among notifiers
}

type notifier interface {
	Notify(context.Context, *NotificationData) error
}

type cliNotifier struct {
	out io.Writer
}

var _ notifier = (*cliNotifier)(nil)

func NewCliNotifier(stdout io.Writer) *cliNotifier {
	return &cliNotifier{
		out: stdout,
	}
}

func (cn *cliNotifier) Notify(_ context.Context, data *NotificationData) error {
	fmt.Fprintf(cn.out, "Command %s '%s' was executing for a really long time (%d sec)\n",
		data.Invocation.InvocationID,
		data.Invocation.ShellLine,
		data.ExecTime,
	)
	return nil
}

type shellTracker struct {
	config          *ShellTrackerConfig
	clock           Clock
	storage         InvocationStorage
	invocationIDGen InvocationIDGen

	notifiers map[NotificationType]notifier
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

	return &shellTracker{
		config:          cfg,
		storage:         storage,
		clock:           clock,
		invocationIDGen: invocationGen,
		notifiers: map[NotificationType]notifier {
			NotificationCLI: NewCliNotifier(os.Stdout),
		},
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

func (st *shellTracker) getExtInvocationID(rec *ShellInvocationRecord) (string, error) {
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
	rec := ShellInvocationRecord{
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
			notifier, ok := st.notifiers[notifConfig.Type]
			if !ok {
				fmt.Printf("notification type '%s' is not suppported yet\n", notifConfig.Type)
				continue
			}
			if err := notifier.Notify(ctx, &NotificationData{
				Invocation: rec,
				NowTimestamp: now,
				ExecTime: execTime,
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

func main() {

	root := cobra.Command{
		Use:   os.Args[0],
		Short: "Shell invocation tracking and notifying utility",
	}

	cfg, err := ReadFromDefaultLoc()
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			fmt.Printf("config does not exist, will create default one")
			cfg = DefaultShellTrackerConfig()
			if err := SaveConfigToDefaultLoc(cfg); err != nil {
				fmt.Printf("failed to save config: %v", err)
				os.Exit(1) // TODO
			}
		} else {
			fmt.Printf("failed to read config from default location, err: %v", err)
			os.Exit(1) // TODO
		}
	}

	shellTracker, err := NewShellTracker(cfg, &defaultClock{}, uuidInvocationGen)
	if err != nil {
		os.Exit(1) // TODO put all stuff inside the root command
	}

	ctx := context.Background() // TODO

	if err := shellTracker.Start(ctx); err != nil {
		os.Exit(1)
	}

	var (
		shellLine, shellInvocationId string
	)
	saveInvocationCommand := cobra.Command{
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

	var extInvocationId string
	notifyCommand := cobra.Command{
		Use:   "notify",
		Short: "trigger notification for invocation that has finished executing",
		RunE: func(cmd *cobra.Command, args []string) error {
			return shellTracker.notifyInvocationFinished(ctx, extInvocationId)
		},
	}
	notifyCommand.Flags().StringVar(&extInvocationId, "invocation-id", "", "shell command invocation id returned by save-invocation call")

	root.AddCommand(&saveInvocationCommand)
	root.AddCommand(&notifyCommand)

	if err := root.Execute(); err != nil {
		os.Exit(1)
	}
}
