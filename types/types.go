package types

type ShellInvocationRecord struct {
	InvocationID         string `json:"invocation_id"`
	ParentID             int    `json:"ppid"`
	ShellLine            string `json:"cmd_text"`
	Timestamp            int64  `json:"started_at"`
	ExternalInvocationID string `json:"ext_invocation_id"`
}

type NotificationData struct {
	Invocation   *ShellInvocationRecord
	NowTimestamp int64
	ExecTime     int64
	// feel free to add more data that can be reused among notifiers
}

type NotificationType string

const (
	NotificationCLI      NotificationType = "cli"      // trivial notification putting the text into the command line
	NotificatonOSPush                     = "os-push"  // GUI OS notification (libnotify for linux)
	NotificationTelegram                  = "telegram" // Notification published into the telegram bot
	// feel free to put here any type of supported (or proxied) notification
)
