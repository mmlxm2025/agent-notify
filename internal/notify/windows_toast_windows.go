//go:build windows

package notify

import (
	"context"
	"os"

	"github.com/hellolib/toast"
)

func defaultWindowsToastPush(ctx context.Context, req windowsToastRequest) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	opts := []toast.NotificationOption{
		toast.WithAppID("agent-notify"),
		toast.WithTitle(req.Title),
		toast.WithMessage(req.Body),
		toast.WithAudio(toast.Default),
		toast.WithLongDuration(),
	}
	if req.ClickToFocus {
		if focus, err := toast.PrepareFocusActivation(os.Getppid()); err == nil {
			opts = append(opts,
				toast.WithActivationType("protocol"),
				toast.WithActivationArguments(focus.Arguments),
			)
		}
	}

	return toast.Push(req.Body, opts...)
}
