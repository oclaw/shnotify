package config

import (
	"os"
	"path"

	"github.com/oclaw/shnotify/types"
	"gopkg.in/yaml.v3"
)

type NotificationConditions struct {
	RunLongerThanSec *int64 `yaml:"run_longer_than_sec"` // TODO use time.ParseDuration() for unmarshalling from yaml
}

type Notification struct {
	Type       types.NotificationType `yaml:"type"`
	Conditions NotificationConditions `yaml:"conditions"`
}

type NotifierInitMode int

const (
	NotifierInitOnStartup NotifierInitMode = 1
	NotifierInitOnDemand  NotifierInitMode = 2
)

type ShellTrackerConfig struct {
	DirPath             string           `yaml:"dir_path"`               // directory to store shell invocations
	CleanupEnabled      bool             `yaml:"cleanup_enabled"`        // if enabled service will manually delete the invocations
	TrackProcsBanList   []string         `yaml:"track_procs_ban_list"`   // do not track the binaries from the list
	TrackProcsAllowList []string         `yaml:"track_procs_allow_list"` // track only the binaries from the list
	RPCSocketName       string           `yaml:"rpc_socket_name"`
	DeadlineSec         int64            `yaml:"deadline_sec"`
	Notifications       []Notification   `yaml:"notifications"`
	NotifierSettings    NotifierSettings `yaml:"notifier_settings,omitempty"`

	InitMode NotifierInitMode `yaml:"-"`

	// TODO garbage collection settings
}

type NotifierSettings struct {
	TelegramChatID int64 `yaml:"telegram_chat_id,omitempty"`
}

func DefaultShellTrackerConfig() *ShellTrackerConfig {

	const dirPath = "shnotify"

	var timeout int64 = 30 // seconds

	return &ShellTrackerConfig{
		DirPath:        path.Join(os.TempDir(), dirPath),
		DeadlineSec:    3,
		CleanupEnabled: true,
		RPCSocketName: path.Join(os.TempDir(), "shnotify-rpc.sock"),
		Notifications: []Notification{
			{
				Type: types.NotificationCLI,
				Conditions: NotificationConditions{
					RunLongerThanSec: &timeout,
				},
			},
		},
		InitMode: NotifierInitOnStartup,
	}
}

func (cfg *ShellTrackerConfig) Save(filePath string) error {
	dirPath := path.Dir(filePath)
	if err := os.MkdirAll(dirPath, os.ModePerm); err != nil {
		return err
	}

	file, err := os.OpenFile(filePath, os.O_CREATE|os.O_WRONLY, os.ModePerm)
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

func ReadFromDefaultLoc() (*ShellTrackerConfig, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return nil, err
	}

	reader, err := os.Open(path.Join(dir, "shnotify", "config.yaml"))
	if err != nil {
		return nil, err
	}
	defer reader.Close()

	var ret ShellTrackerConfig
	decoder := yaml.NewDecoder(reader)
	if err := decoder.Decode(&ret); err != nil {
		return nil, err
	}

	return &ret, nil
}
