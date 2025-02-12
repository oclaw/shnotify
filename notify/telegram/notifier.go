package telegram

import (
	"context"
	"fmt"
	"strings"

	"github.com/nikoksr/notify/service/telegram"
	"github.com/oclaw/shnotify/notify"
	"github.com/oclaw/shnotify/types"
)


type telegramNotifier struct {
	transport *telegram.Telegram
}

func NewTelegramNotifier(
	token string,
	chatID int64,
) (notify.Notifier, error) {

	token = strings.TrimSpace(token)
	tgTransport, err := telegram.New(token)
	if err != nil {
		return nil, err
	}
	tgTransport.SetParseMode(telegram.ModeMarkdown)
	tgTransport.AddReceivers(chatID)

	return &telegramNotifier{
		transport: tgTransport,
	}, nil
}

func (tgn *telegramNotifier) Notify(ctx context.Context, data *types.NotificationData) error {

	mdStr := fmt.Sprintf(`
Command *%s* has finished its execution
- invocation-id: *%s*
- execution time: *%d sec*
`,
	data.Invocation.ShellLine,
	data.Invocation.InvocationID,
	data.ExecTime,
)

	err := tgn.transport.Send(ctx, "shnotify update", mdStr)
	fmt.Printf("tg invocation %s end: %v", data.Invocation.InvocationID, err)
	return err
}
