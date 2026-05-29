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
	barkHTTPTimeout      = 10 * time.Second
	maxBarkErrorBodySize = 512
)

// BarkSender sends notifications through a Bark push URL.
type BarkSender struct {
	webhookURL string
	httpClient *http.Client
}

// NewBarkSender creates a BarkSender with the provided Bark URL.
func NewBarkSender(webhookURL string) *BarkSender {
	return &BarkSender{
		webhookURL: strings.TrimSpace(webhookURL),
		httpClient: &http.Client{Timeout: barkHTTPTimeout},
	}
}

func (s *BarkSender) Name() string { return "bark" }

func (s *BarkSender) Send(ctx context.Context, msg Message) error {
	endpoint, err := barkEndpoint(s.webhookURL)
	if err != nil {
		return err
	}
	title, payloadBody := barkPayloadText(msg)

	body, err := json.Marshal(map[string]string{
		"title": title,
		"body":  payloadBody,
	})
	if err != nil {
		return fmt.Errorf("bark: marshal payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("bark: create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json; charset=utf-8")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("bark: send request: %w", err)
	}
	defer resp.Body.Close()

	return handleBarkResponse(resp)
}

func barkPayloadText(msg Message) (string, string) {
	if msg.Agent == "codex" && msg.Event == "run_completed" {
		if title := extractJSONTitle(msg.Body); title != "" {
			return title, msg.Title
		}
	}
	return msg.Title, msg.Body
}

func extractJSONTitle(body string) string {
	data := []byte(strings.TrimSpace(body))
	if len(data) == 0 {
		return ""
	}

	var items []struct {
		Title string `json:"title"`
	}
	if err := json.Unmarshal(data, &items); err == nil {
		for _, item := range items {
			if title := strings.TrimSpace(item.Title); title != "" {
				return title
			}
		}
	}

	var item struct {
		Title string `json:"title"`
	}
	if err := json.Unmarshal(data, &item); err != nil {
		return ""
	}
	return strings.TrimSpace(item.Title)
}

func barkEndpoint(rawURL string) (string, error) {
	if rawURL == "" {
		return "", fmt.Errorf("bark: webhook_url is empty")
	}

	u, err := url.Parse(rawURL)
	if err != nil {
		return "", fmt.Errorf("bark: parse webhook_url: %w", err)
	}
	if u.Scheme == "" || u.Host == "" {
		return "", fmt.Errorf("bark: webhook_url must include scheme and host")
	}

	key := firstPathSegment(u.Path)
	if key == "" {
		return "", fmt.Errorf("bark: webhook_url missing key")
	}
	u.Path = "/" + key
	u.RawPath = ""
	return u.String(), nil
}

func firstPathSegment(path string) string {
	for _, part := range strings.Split(strings.Trim(path, "/"), "/") {
		if part != "" {
			return part
		}
	}
	return ""
}

func handleBarkResponse(resp *http.Response) error {
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, maxBarkErrorBodySize))
		return fmt.Errorf("bark: unexpected status %d: %s", resp.StatusCode, string(body))
	}

	var result struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("bark: decode response: %w", err)
	}
	if result.Code != http.StatusOK {
		return fmt.Errorf("bark: api error code=%d msg=%s", result.Code, result.Message)
	}

	return nil
}
