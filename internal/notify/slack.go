package notify

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	slackHTTPTimeout      = 10 * time.Second
	maxSlackErrorBodySize = 512
)

// SlackSender sends notifications via Slack Incoming Webhook.
type SlackSender struct {
	webhookURL string
	httpClient *http.Client
}

// NewSlackSender creates a new SlackSender with the given webhook URL.
func NewSlackSender(webhookURL string) *SlackSender {
	return &SlackSender{
		webhookURL: strings.TrimSpace(webhookURL),
		httpClient: &http.Client{Timeout: slackHTTPTimeout},
	}
}

func (s *SlackSender) Name() string { return "slack" }

// Send sends a notification message via Slack Incoming Webhook.
func (s *SlackSender) Send(ctx context.Context, msg Message) error {
	if err := validateSlackWebhookURL(s.webhookURL); err != nil {
		return err
	}

	payload := s.buildPayload(msg)

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("slack: marshal payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.webhookURL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("slack: create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json; charset=utf-8")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("slack: send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(io.LimitReader(resp.Body, maxSlackErrorBodySize))
		return fmt.Errorf("slack: unexpected status %d: %s", resp.StatusCode, string(respBody))
	}

	return nil
}

type slackPayload struct {
	Text        string            `json:"text"`
	Attachments []slackAttachment `json:"attachments,omitempty"`
}

type slackAttachment struct {
	Color  string       `json:"color"`
	Fields []slackField `json:"fields,omitempty"`
	Footer string       `json:"footer,omitempty"`
	Ts     int64        `json:"ts,omitempty"`
}

type slackField struct {
	Title string `json:"title"`
	Value string `json:"value"`
	Short bool   `json:"short,omitempty"`
}

func (s *SlackSender) buildPayload(msg Message) slackPayload {
	emoji := eventEmoji(msg.Event)
	eventName := eventDisplayName(msg.Event)

	color := slackColorForEvent(msg.Event)

	text := fmt.Sprintf("%s *%s*", emoji, msg.Title)

	footerText := "Agent Notify"
	if msg.Agent == "codex" {
		footerText = "Codex Agent Notify"
	}

	fields := []slackField{
		{Title: "Event", Value: eventName, Short: true},
		{Title: "Message", Value: msg.Body, Short: false},
	}
	if msg.Workspace != "" && msg.Agent != "codex" {
		fields = append(fields, slackField{Title: "Workspace", Value: "`" + msg.Workspace + "`", Short: false})
	}

	return slackPayload{
		Text: text,
		Attachments: []slackAttachment{
			{
				Color:  color,
				Fields: fields,
				Footer: footerText,
				Ts:     time.Now().Unix(),
			},
		},
	}
}

func slackColorForEvent(event string) string {
	switch event {
	case "permission_required":
		return "#warning"
	case "input_required":
		return "#439FE0"
	case "run_completed":
		return "good"
	case "run_failed":
		return "danger"
	default:
		return "#3AA3E3"
	}
}

func validateSlackWebhookURL(rawURL string) error {
	if rawURL == "" {
		return fmt.Errorf("slack: webhook_url is empty")
	}

	u, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("slack: parse webhook_url: %w", err)
	}
	if u.Scheme != "https" {
		return fmt.Errorf("slack: webhook_url must use https scheme")
	}
	if u.Host == "" {
		return fmt.Errorf("slack: webhook_url must include host")
	}
	if !strings.HasSuffix(u.Host, "slack.com") {
		return fmt.Errorf("slack: webhook_url must be a slack.com URL")
	}
	return nil
}
