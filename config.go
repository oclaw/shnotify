package main

import (
	"os"
	"path"

	"gopkg.in/yaml.v3"
)

type NotificationType string

const (
	NotificationCLI      NotificationType = "cli"      // trivial notification putting the text into the command line
	NotificatonOSPush                     = "os-push"  // GUI OS notification (libnotify for linux)
	NotificationTelegram                  = "telegram" // Notification published into the telegram bot
	// feel free to put here any type of supported (or proxied) notification
)

type NotificationConditions struct {
	RunLongerThanSec *int64 `yaml:"run_longer_than_sec"` // TODO use time.ParseDuration() for unmarshalling from yaml
}

type Notification struct {
	Type       NotificationType       `yaml:"type"`
	Conditions NotificationConditions `yaml:"conditions"`
}

type ShellTrackerConfig struct {
	DirPath             string         `yaml:"dir_path"`               // directory to store shell invocations
	CleanupEnabled      bool           `yaml:"cleanup_enabled"`        // if enabled service will manually delete the invocations
	TrackProcsBanList   []string       `yaml:"track_procs_ban_list"`   // do not track the binaries from the list
	TrackProcsAllowList []string       `yaml:"track_procs_allow_list"` // track only the binaries from the list
	Notifications       []Notification `yaml:"notifications"`
	// TODO notification configs
	// TODO garbage collection settings
}

func DefaultShellTrackerConfig() ShellTrackerConfig {

	const dirPath = "shnotify"

	var timeout int64 = 30 // seconds

	return ShellTrackerConfig{
		DirPath:        path.Join(os.TempDir(), dirPath),
		CleanupEnabled: true,
		Notifications: []Notification{
			{
				Type: NotificationCLI,
				Conditions: NotificationConditions{
					RunLongerThanSec: &timeout,
				},
			},
		},
	}
}

func (cfg *ShellTrackerConfig) Save(filePath string) error {
	dirPath := path.Dir(filePath)
	if err := os.MkdirAll(dirPath, os.ModePerm); err != nil {
		return err
	}

	file, err := os.OpenFile(filePath, os.O_CREATE | os.O_WRONLY, os.ModePerm)
	if err != nil {
		return err
	}
	defer file.Close()


	encoder := yaml.NewEncoder(file)
	defer encoder.Close()

	return encoder.Encode(cfg)
}

func SaveConfigToDefaultLoc(cfg *ShellTrackerConfig) error {
	dir, err := os.UserConfigDir()
	if err != nil {
		return err
	}

	return cfg.Save(path.Join(dir, "shnotify", "config.yaml"))
}
