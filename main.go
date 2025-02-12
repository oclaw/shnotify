package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"github.com/spf13/cobra"
)

// read config and init all needed objects
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

// support for shell track start command
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

// support for shell track end command
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
