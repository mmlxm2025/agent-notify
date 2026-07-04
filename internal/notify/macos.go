package notify

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

type Runner func(ctx context.Context, name string, args ...string) error

// PathResolver 返回 terminal-notifier 可执行文件路径，找不到返回空串。
type PathResolver func() string

type MacOSSender struct {
	run          Runner
	clickToFocus bool
	notifierPath PathResolver
}

// NewMacOSSender 构造 macOS 系统通知发送器。clickToFocus 为 true 时，点击通知会激活宿主应用。
func NewMacOSSender(run Runner, clickToFocus bool) *MacOSSender {
	return &MacOSSender{run: run, clickToFocus: clickToFocus, notifierPath: defaultTerminalNotifierPath}
}

// NewMacOSSenderWithResolver 供测试注入 notifierPath 解析。
func NewMacOSSenderWithResolver(run Runner, clickToFocus bool, resolver PathResolver) *MacOSSender {
	return &MacOSSender{run: run, clickToFocus: clickToFocus, notifierPath: resolver}
}

func DefaultRunner(ctx context.Context, name string, args ...string) error {
	return exec.CommandContext(ctx, name, args...).Run()
}

func (s *MacOSSender) Name() string { return "system" }

func (s *MacOSSender) Send(ctx context.Context, msg Message) error {
	// Use terminal-notifier if available for better notifications with icon support
	if s.tryTerminalNotifier(ctx, msg) {
		return nil
	}

	// Fallback to osascript with improved content
	formattedBody := s.formatBody(msg)
	script := fmt.Sprintf(`display notification %q with title %q sound name "Submarine"`, formattedBody, msg.Title)
	return s.run(ctx, "osascript", "-e", script)
}

// defaultTerminalNotifierPath 返回 terminal-notifier 可执行文件路径：
// 优先用随 npx 解压到 ~/.agent-notify/terminal-notifier.app 的本地预置 bundle，
// 其次回退到系统 PATH 上的 terminal-notifier。找不到返回空串。
func defaultTerminalNotifierPath() string {
	if home, err := os.UserHomeDir(); err == nil {
		localExe := filepath.Join(home, ".agent-notify", "terminal-notifier.app", "Contents", "MacOS", "terminal-notifier")
		if info, err := os.Stat(localExe); err == nil && !info.IsDir() {
			return localExe
		}
	}
	if p, err := exec.LookPath("terminal-notifier"); err == nil {
		return p
	}
	return ""
}

// agentIconPath 按 agent 选择应用图标；图标文件存在才返回，否则返回空串。
func agentIconPath(agent string) string {
	var app string
	switch agent {
	case "codex":
		app = "Codex.app"
	case "zcode":
		app = "ZCode.app"
	default:
		app = "Claude.app"
	}
	icon := filepath.Join("/Applications", app, "Contents", "Resources", "AppIcon.icns")
	if info, err := os.Stat(icon); err == nil && !info.IsDir() {
		return icon
	}
	return ""
}

// tryTerminalNotifier attempts to use terminal-notifier for richer notifications
func (s *MacOSSender) tryTerminalNotifier(ctx context.Context, msg Message) bool {
	exe := s.notifierPath()
	if exe == "" {
		return false
	}

	args := []string{
		"-title", msg.Title,
		"-subtitle", time.Now().Format("15:04:05"),
		"-message", s.formatBody(msg),
		"-sound", "Submarine",
		"-group", fmt.Sprintf("com.agent-notify.%s", msg.Agent),
	}

	// 点击通知时激活触发事件的宿主应用（终端 / IDE）
	if s.clickToFocus && msg.SourceApp.BundleID != "" {
		args = append(args, "-activate", msg.SourceApp.BundleID)
	}

	// 图标按 agent 选择，存在才追加
	if icon := agentIconPath(msg.Agent); icon != "" {
		args = append(args, "-appIcon", icon)
	}

	// terminal-notifier returns 0 on success, non-zero on failure
	return s.run(ctx, exe, args...) == nil
}

// formatBody formats the notification body. 时间戳已移至 subtitle，
// 正文只留工作区（缩短为末尾项目名，避免长路径换行）与消息。
func (s *MacOSSender) formatBody(msg Message) string {
	if msg.Workspace != "" {
		return fmt.Sprintf("📁 %s\n%s", shortenWorkspace(msg.Workspace), msg.Body)
	}
	return msg.Body
}

// shortenWorkspace 将长路径缩短为末尾项目名，避免通知正文因路径过长而换行。
// 例如 /Users/foo/workspace/github/hellolib/agent-notify → hellolib/agent-notify
func shortenWorkspace(ws string) string {
	parts := strings.Split(filepath.ToSlash(ws), "/")
	// 过滤空段（首尾斜杠）
	var segs []string
	for _, p := range parts {
		if p != "" {
			segs = append(segs, p)
		}
	}
	if len(segs) <= 2 {
		return ws
	}
	// 取末尾两段，保留父级以增强可辨识度
	return strings.Join(segs[len(segs)-2:], "/")
}
