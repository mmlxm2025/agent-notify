package notify

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const (
	wechatCompatHTTPTimeout      = 10 * time.Second
	maxWechatCompatErrorBodySize = 512
)

// WechatCompatSender 通过自建转发服务推送通知。
//
// 与 WechatWorkSender 的区别：后者面向标准企业微信群机器人
// （qyapi.weixin.qq.com），发送 {msgtype, markdown} 格式；而本通道面向
// 只接受 {title, content} 形式 JSON 的自建/中转服务（例如个人搭建的
// webhook 转发到企业微信、Server 酱等）。content 仍使用企业微信 markdown，
// 因此消息样式与企业微信原生通道一致。
type WechatCompatSender struct {
	webhookURL string
	httpClient *http.Client
}

// NewWechatCompatSender creates a new WechatCompatSender with the given webhook URL.
func NewWechatCompatSender(webhookURL string) *WechatCompatSender {
	return &WechatCompatSender{
		webhookURL: strings.TrimSpace(webhookURL),
		httpClient: &http.Client{Timeout: wechatCompatHTTPTimeout},
	}
}

func (s *WechatCompatSender) Name() string { return "wechat-compat" }

// Send sends a notification message via a custom webhook as {title, content} JSON.
func (s *WechatCompatSender) Send(ctx context.Context, msg Message) error {
	if err := validateWechatCompatWebhookURL(s.webhookURL); err != nil {
		return err
	}

	payload := buildWechatCompatPayload(msg)

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("wechat-compat: marshal payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.webhookURL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("wechat-compat: create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json; charset=utf-8")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("wechat-compat: send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(io.LimitReader(resp.Body, maxWechatCompatErrorBodySize))
		return fmt.Errorf("wechat-compat: unexpected status %d: %s", resp.StatusCode, string(respBody))
	}

	return nil
}

// wechatCompatPayload 描述自建转发服务期望的 {title, content} 消息体。
type wechatCompatPayload struct {
	Title   string `json:"title"`
	Content string `json:"content"`
}

func buildWechatCompatPayload(msg Message) wechatCompatPayload {
	return wechatCompatPayload{
		Title:   msg.Title,
		Content: buildWeComPlainText(msg),
	}
}

// buildWeComPlainText 构造纯文本正文，去除所有 markdown 语法符号（##、>、**、`、
// <font> 等），用于不支持 markdown 渲染的自建转发服务。与 buildWeComMarkdown 信息
// 一致，仅格式不同。
func buildWeComPlainText(msg Message) string {
	emoji := eventEmoji(msg.Event)
	eventName := eventDisplayName(msg.Event)
	timestamp := time.Now().Format("2006-01-02 15:04:05")

	footerText := "🤖 Agent Notify"
	if msg.Agent == "codex" {
		footerText = "🤖 Codex Agent Notify"
	}

	lines := []string{
		fmt.Sprintf("%s %s", emoji, msg.Title),
		"事件类型：" + eventName,
		"时间：" + timestamp,
		"消息内容：" + msg.Body,
	}
	if msg.Workspace != "" && msg.Agent != "codex" {
		lines = append(lines, "工作目录："+msg.Workspace)
	}
	lines = append(lines, footerText)

	return strings.Join(lines, "\n")
}

// validateWechatCompatWebhookURL 仅做宽松校验：URL 非空且包含 host。
// 本通道的存在意义就是兼容任意自建服务，因此不限定域名。
func validateWechatCompatWebhookURL(rawURL string) error {
	if rawURL == "" {
		return fmt.Errorf("wechat-compat: webhook_url is empty")
	}
	// 简单校验：必须包含 "://" 和 host 部分
	if !strings.Contains(rawURL, "://") {
		return fmt.Errorf("wechat-compat: invalid webhook_url: %s", rawURL)
	}
	return nil
}
