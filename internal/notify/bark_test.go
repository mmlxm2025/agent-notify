package notify

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestBarkSenderName(t *testing.T) {
	s := NewBarkSender("https://api.day.app/key")
	if got := s.Name(); got != "bark" {
		t.Fatalf("Name() = %q, want bark", got)
	}
}

func TestBarkSenderSendEmptyURL(t *testing.T) {
	s := NewBarkSender("")
	err := s.Send(context.Background(), Message{Title: "t", Body: "b"})
	if err == nil {
		t.Fatal("Send() error = nil, want error for empty URL")
	}
	if !strings.Contains(err.Error(), "webhook_url is empty") {
		t.Fatalf("Send() error = %v, want webhook_url is empty", err)
	}
}

func TestBarkSenderSendSuccess(t *testing.T) {
	var gotPayload map[string]string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		if r.URL.Path != "/testkey" {
			t.Errorf("path = %s, want /testkey", r.URL.Path)
		}
		if r.URL.Query().Get("group") != "codex" {
			t.Errorf("group query = %q, want codex", r.URL.Query().Get("group"))
		}
		if ct := r.Header.Get("Content-Type"); !strings.Contains(ct, "application/json") {
			t.Errorf("Content-Type = %s, want application/json", ct)
		}
		if err := json.NewDecoder(r.Body).Decode(&gotPayload); err != nil {
			t.Errorf("decode payload: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"code":200,"message":"success"}`))
	}))
	defer srv.Close()

	s := NewBarkSender(srv.URL + "/testkey/replace-me?group=codex")
	msg := Message{Title: "Codex 运行完成", Body: "任务已完成"}
	if err := s.Send(context.Background(), msg); err != nil {
		t.Fatalf("Send() error = %v", err)
	}
	if gotPayload["title"] != msg.Title {
		t.Fatalf("title = %q, want %q", gotPayload["title"], msg.Title)
	}
	if gotPayload["body"] != msg.Body {
		t.Fatalf("body = %q, want %q", gotPayload["body"], msg.Body)
	}
}

func TestBarkSenderUsesCodexTaskTitle(t *testing.T) {
	var gotPayload map[string]string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&gotPayload); err != nil {
			t.Errorf("decode payload: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"code":200,"message":"success"}`))
	}))
	defer srv.Close()

	s := NewBarkSender(srv.URL + "/testkey")
	msg := Message{
		Agent: "codex",
		Event: "run_completed",
		Title: "Codex 运行完成",
		Body:  `[{"title":"补上 glm-5 计费倍率，别再走 unknown ratio","description":"ignored"}]`,
	}
	if err := s.Send(context.Background(), msg); err != nil {
		t.Fatalf("Send() error = %v", err)
	}
	if gotPayload["title"] != "补上 glm-5 计费倍率，别再走 unknown ratio" {
		t.Fatalf("title = %q, want task title", gotPayload["title"])
	}
	if gotPayload["body"] != "Codex 运行完成" {
		t.Fatalf("body = %q, want event title", gotPayload["body"])
	}
}

func TestBarkSenderSendHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("internal error"))
	}))
	defer srv.Close()

	s := NewBarkSender(srv.URL + "/testkey")
	err := s.Send(context.Background(), Message{Title: "t", Body: "b"})
	if err == nil {
		t.Fatal("Send() error = nil, want HTTP error")
	}
	if !strings.Contains(err.Error(), "unexpected status 500") {
		t.Fatalf("Send() error = %v, want unexpected status 500", err)
	}
}

func TestBarkSenderSendAPIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"code":400,"message":"bad key"}`))
	}))
	defer srv.Close()

	s := NewBarkSender(srv.URL + "/testkey")
	err := s.Send(context.Background(), Message{Title: "t", Body: "b"})
	if err == nil {
		t.Fatal("Send() error = nil, want api error")
	}
	if !strings.Contains(err.Error(), "api error") {
		t.Fatalf("Send() error = %v, want api error", err)
	}
}
