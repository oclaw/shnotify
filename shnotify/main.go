package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/oclaw/shnotify/config"
	"github.com/oclaw/shnotify/core"
	"github.com/oclaw/shnotify/rpc"
	"github.com/oclaw/shnotify/types"

	"github.com/spf13/cobra"
)

func initConfig() (*config.ShellTrackerConfig, error) {
	cfg, err := config.ReadFromDefaultLoc()
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			fmt.Printf("config does not exist, will create default one")
			cfg = config.DefaultShellTrackerConfig()
			if err := config.SaveConfigToDefaultLoc(cfg); err != nil {
				fmt.Printf("failed to save config: %v", err)
				return nil, err
			}
		} else {
			fmt.Printf("failed to read config from default location, err: %v", err)
			return nil, err
		}
	}

	cfg.InitMode = config.NotifierInitOnDemand

	return cfg, nil
}

// support for shell track start command
func buildStartInvocationCommand(tracker core.InvocationTracker) (*cobra.Command, error) {
	var (
		shellLine         string
		shellInvocationId string
	)

	saveInvocationCommand := &cobra.Command{
		Use:   "save-invocation",
		Short: "save invocation of the shell command into the storage and return the external id assigned to the execution",
		RunE: func(cmd *cobra.Command, args []string) error {
			ret, err := tracker.SaveInvocation(
				cmd.Context(),
				&types.InvocationRequest{
					InvocationID: types.InvocationID(shellInvocationId),
					ShellLine:    shellLine,
					ParentID:     os.Getppid(),
				},
			)
			if err != nil {
				return err
			}
			cmd.OutOrStdout().Write([]byte(ret))
			return nil
		},
	}
	saveInvocationCommand.Flags().StringVar(&shellLine, "shell-line", "", "shell command line to put into the invocation")
	saveInvocationCommand.Flags().StringVar(&shellInvocationId, "invocation-id", "", "externally defined invocation id (empty by default)")
	return saveInvocationCommand, nil
}

// support for shell track end command
func buildNotifyCommand(tracker core.InvocationTracker) (*cobra.Command, error) {
	var invocationID string

	notifyCommand := cobra.Command{
		Use:   "notify",
		Short: "trigger notification for invocation that has finished executing",
		RunE: func(cmd *cobra.Command, args []string) error {
			return tracker.Notify(cmd.Context(), types.InvocationID(invocationID))
		},
	}
	notifyCommand.Flags().StringVar(&invocationID, "invocation-id", "", "shell command invocation id returned by save-invocation call")
	return &notifyCommand, nil
}

func setupRootCommand(tracker core.InvocationTracker) (*cobra.Command, error) {
	root := cobra.Command{
		Use:   os.Args[0],
		Short: "Shell invocation tracking and notifying utility",
	}

	saveInvocationCommand, err := buildStartInvocationCommand(tracker)
	if err != nil {
		return nil, err
	}

	notifyCommand, err := buildNotifyCommand(tracker)
	if err != nil {
		return nil, err
	}

	root.AddCommand(
		saveInvocationCommand,
		notifyCommand,
	)
	return &root, nil
}

func run(ctx context.Context) error {
	cfg, err := initConfig()
	if err != nil {
		return err
	}

	client, err := rpc.NewClient(cfg.RPCSocketName)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(ctx, time.Second*time.Duration(cfg.DeadlineSec))
	defer cancel()

	root, err := setupRootCommand(client)
	if err != nil {
		return err
	}

	return root.ExecuteContext(ctx)
}

func main() {
	// TODO monitor OS signals
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// TODO determine and ignore errors caused by absense of the daemon
	if err := run(ctx); err != nil {
		fmt.Printf("failed to run shnotify: %v\n", err)
		os.Exit(1)
	}
}
