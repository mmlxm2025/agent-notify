package cli

import (
	"bytes"
	"testing"

	"github.com/hellolib/agent-notify/internal/config"
)

func TestRunInitBarkWritesBothAgentConfigs(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	wantURL := "https://api.day.app/testkey/replace-me"
	streams := Streams{Stdout: &bytes.Buffer{}}
	prompter := &fakePrompter{
		inputs: []string{wantURL},
	}

	if err := runInitBark(streams, prompter); err != nil {
		t.Fatalf("runInitBark() error = %v", err)
	}

	path, err := config.DefaultPath()
	if err != nil {
		t.Fatalf("DefaultPath() error = %v", err)
	}
	cfg, err := config.Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if !cfg.Notify.ClaudeCode.Channels.Bark.Enabled {
		t.Fatal("ClaudeCode bark should be enabled")
	}
	if !cfg.Notify.Codex.Channels.Bark.Enabled {
		t.Fatal("Codex bark should be enabled")
	}
	if cfg.Notify.Codex.Channels.Bark.WebhookURL != wantURL {
		t.Fatalf("Codex bark URL = %q, want %q", cfg.Notify.Codex.Channels.Bark.WebhookURL, wantURL)
	}
}
