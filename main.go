package main

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/oclaw/shnotify/types"
	"github.com/spf13/cobra"
)

func initConfig() (*ShellTrackerConfig, error) {
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

	return cfg, nil
}

// support for shell track start command
func buildStartInvocationCommand(ctx context.Context, cfg *ShellTrackerConfig) (*cobra.Command, error) {
	var (
		shellLine string
		shellInvocationId string
	)

	invokeStarter, err := NewInvocationStarter(cfg, &defaultClock{}, uuidInvocationGen)
	if err != nil {
		return nil, err
	}
	saveInvocationCommand := &cobra.Command{
		Use:   "save-invocation",
		Short: "save invocation of the shell command into the storage and return the external id assigned to the execution",
		RunE: func(cmd *cobra.Command, args []string) error {
			ret, err := invokeStarter.SaveInvocation(ctx, shellLine, types.InvocationID(shellInvocationId))
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
func buildNotifyCommand(ctx context.Context, cfg *ShellTrackerConfig) (*cobra.Command, error) {
	var extInvocationId string

	shellTracker, err := NewShellTracker(cfg, &defaultClock{})
	if err != nil {
		return nil, err
	}
	notifyCommand := cobra.Command{
		Use:   "notify",
		Short: "trigger notification for invocation that has finished executing",
		RunE: func(cmd *cobra.Command, args []string) error {
			return shellTracker.NotifyInvocationFinished(ctx, extInvocationId)
		},
	}
	notifyCommand.Flags().StringVar(&extInvocationId, "invocation-id", "", "shell command invocation id returned by save-invocation call")
	return &notifyCommand, nil
}

func setupRootCommand(ctx context.Context, cfg *ShellTrackerConfig) (*cobra.Command, error) {
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
