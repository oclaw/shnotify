package cli

import (
	"context"
	"io"
	"fmt"

	"github.com/oclaw/shnotify/notify"
	"github.com/oclaw/shnotify/types"
)


type cliNotifier struct {
	out io.Writer
}

var _ notify.Notifier = (*cliNotifier)(nil)

func NewCliNotifier(stdout io.Writer) *cliNotifier {
	return &cliNotifier{
		out: stdout,
	}
}

func (cn *cliNotifier) Notify(_ context.Context, data *types.NotificationData) error {
	fmt.Fprintf(cn.out, "Command %s '%s' was executing for a really long time (%d sec)\n",
		data.Invocation.InvocationID,
		data.Invocation.ShellLine,
		data.ExecTime,
	)
	return nil
}
