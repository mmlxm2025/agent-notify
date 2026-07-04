package notify

import "runtime"

// NewSystemSender returns the appropriate system notification sender for the current platform.
// - darwin: uses macOS notifications (osascript/terminal-notifier)
// - linux: uses notify-send
// - windows: uses Windows Runtime toast notifications
// - other: returns an explicit unsupported sender
//
// clickToFocus 控制点击通知是否激活宿主应用（macOS/Windows 生效）。
func NewSystemSender(run Runner, clickToFocus bool) Sender {
	switch runtime.GOOS {
	case "darwin":
		return NewMacOSSender(run, clickToFocus)
	case "linux":
		return NewLinuxSender(run)
	case "windows":
		return NewWindowsSender(run, clickToFocus)
	default:
		return NewUnsupportedSender(runtime.GOOS)
	}
}
