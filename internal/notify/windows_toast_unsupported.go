//go:build !windows

package notify

import (
	"context"
	"errors"
)

func defaultWindowsToastPush(context.Context, windowsToastRequest) error {
	return errors.New("windows toast notifications are only supported on windows")
}
