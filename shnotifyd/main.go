package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"github.com/oclaw/shnotify/common"
	"github.com/oclaw/shnotify/config"
	"github.com/oclaw/shnotify/core"
	rpcserver "github.com/oclaw/shnotify/rpc/server"

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

	cfg.InitMode = config.NotifierInitOnStartup

	return cfg, nil
}



func run(ctx context.Context) error {
	cfg, err := initConfig()
	if err != nil {
		return err
	}

	shellTracker, err := core.NewInvocationTracker(cfg, &common.DefaultClock{}, core.UUIDInvocationGen)
	if err != nil {
		return err
	}

	server, err := rpcserver.NewServer(cfg, shellTracker)
	if err != nil {
		return err
	}

	return server.Serve(ctx)
}

func main() {
	// TODO monitor OS signals
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	root := cobra.Command{
		RunE: func (cmd *cobra.Command, args []string) error {
			return run(cmd.Context())
		},
	}

	if err := root.ExecuteContext(ctx); err != nil {
		os.Exit(1)
	}
}
