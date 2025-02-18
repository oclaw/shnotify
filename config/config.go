package config

import (
	"os"
	"path"
	"time"

	"github.com/oclaw/shnotify/common"
	"github.com/oclaw/shnotify/types"
	"gopkg.in/yaml.v3"
)

type Duration time.Duration

var (
	_ yaml.Marshaler   = common.DefaultVal[Duration]()
	_ yaml.Unmarshaler = (*Duration)(nil)
)

func (d Duration) MarshalYAML() (any, error) {
	return time.Duration(d).String(), nil
}

func (d *Duration) UnmarshalYAML(value *yaml.Node) error {
	var raw string
	if err := value.Decode(&raw); err != nil {
		return err
	}
	dd, err := time.ParseDuration(raw)
	if err != nil {
		return err
	}
	*d = Duration(dd)
	return nil
}

func (d *Duration) LessThan(sec int64) bool {
	if d == nil {
		return false
	}
	dd := time.Duration(*d)
	return sec > int64(dd.Seconds())
}

type NotificationConditions struct {
	RunLongerThan *Duration `yaml:"run_longer_than"` // 30s, 1m, 1h
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

	InitMode           NotifierInitMode `yaml:"-"` // create all notifiers at the startup of the application or at the firt invocation of the notifier
	AsyncNotifications bool             `yaml:"-"` // publish notification in a sync or async way

	// TODO garbage collection settings
}

type NotifierSettings struct {
	TelegramChatID int64 `yaml:"telegram_chat_id,omitempty"`
}

func DefaultShellTrackerConfig() *ShellTrackerConfig {

	const dirPath = "shnotify"

	timeout := Duration(time.Second * 30)

	return &ShellTrackerConfig{
		DirPath:        path.Join(os.TempDir(), dirPath),
		DeadlineSec:    3,
		CleanupEnabled: true,
		RPCSocketName:  "/tmp/shnotify-rpc.sock",
		Notifications: []Notification{
			{
				Type: types.NotificationCLI,
				Conditions: NotificationConditions{
					RunLongerThan: &timeout,
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
