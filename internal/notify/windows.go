package notify

import (
	"context"
	"fmt"
	"time"
)

type windowsToastRequest struct {
	Title        string
	Body         string
	ClickToFocus bool
}

type windowsToastFunc func(ctx context.Context, req windowsToastRequest) error

type WindowsSender struct {
	push         windowsToastFunc
	clickToFocus bool
}

func NewWindowsSender(_ Runner, clickToFocus bool) *WindowsSender {
	return &WindowsSender{push: defaultWindowsToastPush, clickToFocus: clickToFocus}
}

func NewWindowsSenderWithPusher(push windowsToastFunc, clickToFocus bool) *WindowsSender {
	return &WindowsSender{push: push, clickToFocus: clickToFocus}
}

func (s *WindowsSender) Name() string { return "system" }

func (s *WindowsSender) Send(ctx context.Context, msg Message) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	return s.push(ctx, windowsToastRequest{
		Title:        msg.Title,
		Body:         s.formatBody(msg),
		ClickToFocus: s.clickToFocus,
	})
}

func (s *WindowsSender) formatBody(msg Message) string {
	timestamp := time.Now().Format("15:04:05")
	if msg.Workspace != "" {
		return fmt.Sprintf("%s\n%s\n%s", msg.Workspace, msg.Body, timestamp)
	}
	return fmt.Sprintf("%s\n%s", msg.Body, timestamp)
}
