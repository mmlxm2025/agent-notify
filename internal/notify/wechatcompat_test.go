package notify

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestWechatCompatSenderName(t *testing.T) {
	s := NewWechatCompatSender("https://example.com/webhook")
	if got := s.Name(); got != "wechat-compat" {
		t.Fatalf("Name() = %q, want wechat-compat", got)
	}
}

func TestWechatCompatSenderSendEmptyURL(t *testing.T) {
	s := NewWechatCompatSender("")
	err := s.Send(context.Background(), Message{Title: "t", Body: "b"})
	if err == nil {
		t.Fatal("Send() error = nil, want error for empty URL")
	}
	if !strings.Contains(err.Error(), "webhook_url is empty") {
		t.Fatalf("Send() error = %v, want webhook_url is empty", err)
	}
}

func TestWechatCompatSenderSendSuccess(t *testing.T) {
	var gotPayload wechatCompatPayload
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		if ct := r.Header.Get("Content-Type"); !strings.Contains(ct, "application/json") {
			t.Errorf("Content-Type = %s, want application/json", ct)
		}
		if err := json.NewDecoder(r.Body).Decode(&gotPayload); err != nil {
			t.Errorf("decode payload: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"message":"发送成功"}`))
	}))
	defer srv.Close()

	s := NewWechatCompatSender(srv.URL)
	msg := Message{
		Agent:     "zcode",
		Event:     "session_start",
		SessionID: "sess-1",
		Workspace: "/path/to/project",
		Title:     "ZCode 会话开始",
		Body:      "新的 Agent 会话已开始",
	}
	if err := s.Send(context.Background(), msg); err != nil {
		t.Fatalf("Send() error = %v", err)
	}

	// 自建转发服务期望 {title, content}
	if gotPayload.Title != msg.Title {
		t.Errorf("title = %q, want %q", gotPayload.Title, msg.Title)
	}
	if gotPayload.Content == "" {
		t.Fatal("content is empty")
	}
	if !strings.Contains(gotPayload.Content, "ZCode 会话开始") {
		t.Errorf("content should contain title, got: %s", gotPayload.Content)
	}
	if !strings.Contains(gotPayload.Content, "新的 Agent 会话已开始") {
		t.Errorf("content should contain body, got: %s", gotPayload.Content)
	}
	if !strings.Contains(gotPayload.Content, "/path/to/project") {
		t.Errorf("content should contain workspace, got: %s", gotPayload.Content)
	}
	// 微信兼容通道发送纯文本，不应包含 markdown 语法符号
	for _, frag := range []string{"##", "<font", "**", "`", ">"} {
		if strings.Contains(gotPayload.Content, frag) {
			t.Errorf("content should be plain text, but contains %q: %s", frag, gotPayload.Content)
		}
	}
}

func TestWechatCompatSenderSendHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":"消息内容不能为空"}`))
	}))
	defer srv.Close()

	s := NewWechatCompatSender(srv.URL)
	err := s.Send(context.Background(), Message{Title: "t", Body: "b"})
	if err == nil {
		t.Fatal("Send() error = nil, want error for HTTP 400")
	}
	if !strings.Contains(err.Error(), "unexpected status 400") {
		t.Errorf("Send() error = %v, want unexpected status 400", err)
	}
}

func TestValidateWechatCompatWebhookURL(t *testing.T) {
	cases := []struct {
		url    string
		wantErr bool
	}{
		{"https://ssl.example.com/api/notify/abc", false},
		{"http://127.0.0.1:9931/notify", false},
		{"", true},
		{"not-a-url", true},
	}
	for _, tc := range cases {
		err := validateWechatCompatWebhookURL(tc.url)
		gotErr := err != nil
		if gotErr != tc.wantErr {
			t.Errorf("validateWechatCompatWebhookURL(%q) err = %v, wantErr = %v", tc.url, err, tc.wantErr)
		}
	}
}

func TestBuildWechatCompatPayload(t *testing.T) {
	msg := Message{Agent: "claude_code", Event: "run_completed", Title: "运行完成", Body: "任务完成"}
	p := buildWechatCompatPayload(msg)
	if p.Title != msg.Title {
		t.Errorf("title = %q, want %q", p.Title, msg.Title)
	}
	if !strings.Contains(p.Content, "任务完成") {
		t.Errorf("content should contain body, got: %s", p.Content)
	}
	// 纯文本，不应含 markdown 符号
	for _, frag := range []string{"##", "<font", "**", "`"} {
		if strings.Contains(p.Content, frag) {
			t.Errorf("content should be plain text, but contains %q: %s", frag, p.Content)
		}
	}
	if !strings.Contains(p.Content, "事件类型：") {
		t.Errorf("plain text content should contain 事件类型 label, got: %s", p.Content)
	}
}
