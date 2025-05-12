package types

import (
	"fmt"
)

type InvocationID string

type InvocationIDGen func() (InvocationID, error)

func InvocationGenFromStringer[T fmt.Stringer](gen func() (T, error)) InvocationIDGen {
	return func() (InvocationID, error) {
		val, err := gen()
		if err != nil {
			return "", err
		}
		return InvocationID(val.String()), nil
	}
}

type InvocationRequest struct {
	InvocationID InvocationID `json:"invocation_id,omitempty"`
	MachineID    string       `json:"machine_id,omitempty"`
	ParentID     int          `json:"ppid"`
	ShellLine    string       `json:"cmd_text"`
}

type ShellInvocationRecord struct {
	InvocationID InvocationID `json:"invocation_id"`
	ParentID     int          `json:"ppid"`
	MachineID    string       `json:"machine_id"`
	ShellLine    string       `json:"cmd_text"`
	Timestamp    int64        `json:"started_at"`
}

type NotificationResult struct {
	Message string `json:"message,omitempty"`
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
