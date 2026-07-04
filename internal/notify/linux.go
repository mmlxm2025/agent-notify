package notify

import (
	"context"
	"fmt"
	"time"

	"github.com/hellolib/agent-notify/internal/linuxfocus"
)

type LinuxSender struct {
	run          Runner
	clickToFocus bool
	startFocus   linuxFocusStarter
}

type linuxFocusStarter func(ctx context.Context, title, body string) error

func NewLinuxSender(run Runner, clickToFocus bool) *LinuxSender {
	return &LinuxSender{run: run, clickToFocus: clickToFocus, startFocus: defaultLinuxFocusStarter}
}

func NewLinuxSenderWithFocusStarter(run Runner, clickToFocus bool, starter linuxFocusStarter) *LinuxSender {
	return &LinuxSender{run: run, clickToFocus: clickToFocus, startFocus: starter}
}

func (s *LinuxSender) Name() string { return "system" }

func (s *LinuxSender) Send(ctx context.Context, msg Message) error {
	// Use notify-send for Linux notifications
	// Format: notify-send "Title" "Body" [options]

	formattedBody := s.formatBody(msg)
	if s.clickToFocus && s.startFocus != nil {
		if err := s.startFocus(ctx, msg.Title, formattedBody); err == nil {
			return nil
		}
	}

	// notify-send arguments:
	// -a "Claude Code" sets app name
	// -u normal sets urgency
	// -t 5000 sets timeout in milliseconds (5 seconds)
	if err := linuxfocus.SendNotification(ctx, msg.Title, formattedBody); err == nil {
		return nil
	}
	return s.run(ctx, linuxfocus.CommandPath("notify-send"),
		"-a", "Claude Code",
		"-u", "normal",
		"-t", "5000",
		msg.Title,
		formattedBody,
	)
}

func defaultLinuxFocusStarter(ctx context.Context, title, body string) error {
	windowID, err := linuxfocus.ResolveWindowID(ctx, 0)
	if err != nil {
		return err
	}
	return linuxfocus.StartDetached(ctx, linuxfocus.Request{
		Title:    title,
		Body:     body,
		WindowID: windowID,
	})
}

func (s *LinuxSender) formatBody(msg Message) string {
	timestamp := time.Now().Format("15:04:05")
	if msg.Workspace != "" {
		return fmt.Sprintf("%s\n%s\n%s", msg.Workspace, msg.Body, timestamp)
	}
	return fmt.Sprintf("%s\n%s", msg.Body, timestamp)
}
