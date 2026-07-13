package grokhooks

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/hellolib/agent-notify/internal/notify"
)

// payload 描述 Grok hooks 通过 stdin 投递的事件 JSON。
// Grok 文档示例使用 camelCase 字段名，hookEventName 的值为 snake_case
//（如 pre_tool_use / session_start）。同时兼容 PascalCase 与下划线字段名。
type payload struct {
	HookEventName      string         `json:"hook_event_name"`
	HookEventNameCamel string         `json:"hookEventName"`
	SessionID          string         `json:"session_id"`
	SessionIDCamel     string         `json:"sessionId"`
	CWD                string         `json:"cwd"`
	WorkspaceRoot      string         `json:"workspaceRoot"`
	Message            string         `json:"message"`
	NotificationType   string         `json:"notification_type"`
	NotificationTypeCamel string      `json:"notificationType"`
	ToolName           string         `json:"tool_name"`
	ToolNameCamel      string         `json:"toolName"`
	ToolResponse       map[string]any `json:"tool_response"`
	ToolResponseCamel  map[string]any `json:"toolResponse"`
	ToolInput          map[string]any `json:"tool_input"`
	ToolInputCamel     map[string]any `json:"toolInput"`
	Error              string         `json:"error"`
	ErrorMessage       string         `json:"errorMessage"`
}

func (p payload) eventOf() string {
	if p.HookEventName != "" {
		return p.HookEventName
	}
	return p.HookEventNameCamel
}

func (p payload) sessionOf() string {
	if p.SessionID != "" {
		return p.SessionID
	}
	return p.SessionIDCamel
}

func (p payload) workspaceOf() string {
	if p.CWD != "" {
		return p.CWD
	}
	return p.WorkspaceRoot
}

func (p payload) toolNameOf() string {
	if p.ToolName != "" {
		return p.ToolName
	}
	return p.ToolNameCamel
}

func (p payload) toolResponseOf() map[string]any {
	if p.ToolResponse != nil {
		return p.ToolResponse
	}
	return p.ToolResponseCamel
}

func ParseMessage(data []byte) (notify.Message, error) {
	var p payload
	if err := json.Unmarshal(data, &p); err != nil {
		return notify.Message{}, err
	}

	event := normalizeEventName(p.eventOf())
	switch event {
	case "session_start":
		return notify.Message{
			Agent:     "grok",
			Event:     "session_start",
			SessionID: p.sessionOf(),
			Workspace: p.workspaceOf(),
			Title:     notify.FormatTitle("grok", "session_start"),
			Body:      notify.DefaultBody("session_start"),
		}, nil
	case "notification":
		return parseNotification(p)
	case "stop":
		return notify.Message{
			Agent:     "grok",
			Event:     "run_completed",
			SessionID: p.sessionOf(),
			Workspace: p.workspaceOf(),
			Title:     notify.FormatTitle("grok", "run_completed"),
			Body:      notify.DefaultBody("run_completed"),
		}, nil
	case "stop_failure", "post_tool_use_failure":
		errMsg := extractErrorMessage(p)
		tool := fallbackToolName(p.toolNameOf())
		body := fmt.Sprintf("错误: %s", errMsg)
		if tool != "" && tool != "未知工具" {
			body = fmt.Sprintf("工具: %s\n错误: %s", tool, errMsg)
		}
		return notify.Message{
			Agent:     "grok",
			Event:     "run_failed",
			SessionID: p.sessionOf(),
			Workspace: p.workspaceOf(),
			Title:     notify.FormatTitle("grok", "run_failed"),
			Body:      body,
		}, nil
	default:
		return notify.Message{}, fmt.Errorf("unsupported hook event: %s", p.eventOf())
	}
}

func parseNotification(p payload) (notify.Message, error) {
	msg := p.Message
	notifType := p.NotificationType
	if notifType == "" {
		notifType = p.NotificationTypeCamel
	}
	combined := strings.ToLower(strings.TrimSpace(msg + " " + notifType))

	if isPermissionNotification(combined) {
		return notify.Message{
			Agent:     "grok",
			Event:     "permission_required",
			SessionID: p.sessionOf(),
			Workspace: p.workspaceOf(),
			Title:     notify.FormatTitle("grok", "permission_required"),
			Body:      permissionBody(msg, p.toolNameOf()),
		}, nil
	}

	if isInputRequiredNotification(combined) || msg != "" || notifType != "" {
		hint := extractInputHint(msg)
		if hint == "" {
			hint = notifType
		}
		if hint == "" {
			hint = "等待您的操作"
		}
		return notify.Message{
			Agent:     "grok",
			Event:     "input_required",
			SessionID: p.sessionOf(),
			Workspace: p.workspaceOf(),
			Title:     notify.FormatTitle("grok", "input_required"),
			Body:      fmt.Sprintf("提示: %s", hint),
		}, nil
	}

	return notify.Message{}, fmt.Errorf("unsupported notification: message=%q type=%q", msg, notifType)
}

func permissionBody(msg, toolName string) string {
	tool := fallbackToolName(toolName)
	if strings.TrimSpace(msg) != "" {
		if tool != "" && tool != "未知工具" {
			return fmt.Sprintf("工具: %s\n%s", tool, strings.TrimSpace(msg))
		}
		return strings.TrimSpace(msg)
	}
	if tool != "" && tool != "未知工具" {
		return fmt.Sprintf("工具: %s\n操作需要您的授权许可", tool)
	}
	return "操作需要您的授权许可"
}

// normalizeEventName 将 PascalCase / snake_case 事件名统一为 snake_case。
func normalizeEventName(name string) string {
	name = strings.TrimSpace(name)
	switch name {
	case "SessionStart", "session_start":
		return "session_start"
	case "Notification", "notification":
		return "notification"
	case "Stop", "stop":
		return "stop"
	case "StopFailure", "stop_failure":
		return "stop_failure"
	case "PostToolUseFailure", "post_tool_use_failure":
		return "post_tool_use_failure"
	default:
		// 已是 snake_case 或未知
		return strings.ToLower(name)
	}
}

func isPermissionNotification(msg string) bool {
	return strings.Contains(msg, "permission") ||
		strings.Contains(msg, "approval") ||
		strings.Contains(msg, "authorize") ||
		strings.Contains(msg, "授权") ||
		strings.Contains(msg, "许可")
}

func isInputRequiredNotification(msg string) bool {
	return strings.Contains(msg, "waiting for your input") ||
		strings.Contains(msg, "waiting for input") ||
		strings.Contains(msg, "needs input") ||
		strings.Contains(msg, "idle_prompt") ||
		strings.Contains(msg, "等待输入") ||
		strings.Contains(msg, "等待您的")
}

func extractInputHint(msg string) string {
	msg = strings.TrimSpace(msg)
	prefixes := []string{
		"grok is waiting for your input",
		"claude is waiting for your input",
		"waiting for your input: ",
		"waiting for input: ",
		"needs input: ",
	}
	for _, prefix := range prefixes {
		if strings.HasPrefix(strings.ToLower(msg), prefix) {
			return strings.TrimSpace(msg[len(prefix):])
		}
	}
	if len(msg) > 100 {
		return msg[:97] + "..."
	}
	return msg
}

func extractErrorMessage(p payload) string {
	if p.ErrorMessage != "" {
		return truncate(p.ErrorMessage, 200)
	}
	if p.Error != "" {
		return truncate(p.Error, 200)
	}
	response := p.toolResponseOf()
	if response == nil {
		return "未知错误"
	}
	if err, ok := response["error"]; ok {
		if errStr, ok := err.(string); ok && errStr != "" {
			return truncate(errStr, 200)
		}
	}
	if err, ok := response["message"]; ok {
		if errStr, ok := err.(string); ok && errStr != "" {
			return truncate(errStr, 200)
		}
	}
	return "操作失败"
}

func fallbackToolName(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return "未知工具"
	}
	return name
}

func truncate(s string, max int) string {
	if len(s) > max {
		return s[:max-3] + "..."
	}
	return s
}
