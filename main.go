package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/oclaw/shnotify/common"
	"github.com/oclaw/shnotify/config"
	"github.com/oclaw/shnotify/core"
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

	cfg.InitMode = config.NotifierInitOnDemand // TODO determine if we are running with backend or standalone

	return cfg, nil
}

// support for shell track start command
func buildStartInvocationCommand(ctx context.Context, cfg *config.ShellTrackerConfig) (*cobra.Command, error) {
	var (
		shellLine         string
		shellInvocationId string
	)

	invokeStarter, err := core.NewInvocationTracker(cfg, &common.DefaultClock{}, core.UUIDInvocationGen)
	if err != nil {
		return nil, err
	}
	saveInvocationCommand := &cobra.Command{
		Use:   "save-invocation",
		Short: "save invocation of the shell command into the storage and return the external id assigned to the execution",
		RunE: func(cmd *cobra.Command, args []string) error {
			ret, err := invokeStarter.SaveInvocation(
				ctx,
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
func buildNotifyCommand(ctx context.Context, cfg *config.ShellTrackerConfig) (*cobra.Command, error) {
	var invocationID string

	shellTracker, err := core.NewInvocationTracker(cfg, &common.DefaultClock{}, core.UUIDInvocationGen)
	if err != nil {
		return nil, err
	}
	notifyCommand := cobra.Command{
		Use:   "notify",
		Short: "trigger notification for invocation that has finished executing",
		RunE: func(cmd *cobra.Command, args []string) error {
			return shellTracker.Notify(ctx, types.InvocationID(invocationID))
		},
	}
	notifyCommand.Flags().StringVar(&invocationID, "invocation-id", "", "shell command invocation id returned by save-invocation call")
	return &notifyCommand, nil
}

func setupRootCommand(ctx context.Context, cfg *config.ShellTrackerConfig) (*cobra.Command, error) {
	root := cobra.Command{
		Use:   os.Args[0],
		Short: "Shell invocation tracking and notifying utility",
	}

	saveInvocationCommand, err := buildStartInvocationCommand(ctx, cfg)
	if err != nil {
		return nil, err
	}

	notifyCommand, err := buildNotifyCommand(ctx, cfg)
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

	ctx, cancel := context.WithTimeout(ctx, time.Second*time.Duration(cfg.DeadlineSec))
	defer cancel()

	root, err := setupRootCommand(ctx, cfg)
	if err != nil {
		return err
	}

	return root.ExecuteContext(ctx)
}

func main() {
	ctx := context.Background()

	if err := run(ctx); err != nil {
		fmt.Printf("failed to run shnotify: %v\n", err)
		os.Exit(1)
	}
}
