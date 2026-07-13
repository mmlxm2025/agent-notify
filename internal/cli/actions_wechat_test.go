package cli

import (
	"bytes"
	"testing"

	"github.com/hellolib/agent-notify/internal/config"
	"github.com/hellolib/agent-notify/internal/testutil"
)

func TestRunInitWechatWritesAllAgentConfigs(t *testing.T) {
	testutil.IsolateHome(t)

	wantURL := "https://push.example.com/api/notify/abc"
	if err := runInitWechat(Streams{Stdout: &bytes.Buffer{}}, &fakePrompter{inputs: []string{wantURL}}); err != nil {
		t.Fatalf("runInitWechat() error = %v", err)
	}

	path, err := config.DefaultPath()
	if err != nil {
		t.Fatalf("DefaultPath() error = %v", err)
	}
	cfg, err := config.Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	for _, agent := range []struct {
		name string
		ch   config.WechatChannelConfig
	}{
		{"ClaudeCode", cfg.Notify.ClaudeCode.Channels.Wechat},
		{"Codex", cfg.Notify.Codex.Channels.Wechat},
		{"ZCode", cfg.Notify.ZCode.Channels.Wechat},
		{"Grok", cfg.Notify.Grok.Channels.Wechat},
	} {
		if !agent.ch.Enabled {
			t.Fatalf("%s wechat should be enabled", agent.name)
		}
		if agent.ch.WebhookURL != wantURL {
			t.Fatalf("%s wechat URL = %q, want %q", agent.name, agent.ch.WebhookURL, wantURL)
		}
	}
}

func TestRunInitWechatPreservesExistingBarkAndDoesNotDisableIt(t *testing.T) {
	testutil.IsolateHome(t)

	path, err := config.DefaultPath()
	if err != nil {
		t.Fatalf("DefaultPath() error = %v", err)
	}
	seed := config.Default()
	barkURL := "https://api.day.app/existing-key"
	seed.Notify.ClaudeCode.Channels.Bark.Enabled = true
	seed.Notify.ClaudeCode.Channels.Bark.WebhookURL = barkURL
	seed.Notify.Grok.Channels.Bark.Enabled = true
	seed.Notify.Grok.Channels.Bark.WebhookURL = barkURL
	if err := config.Save(path, seed); err != nil {
		t.Fatalf("Save() seed error = %v", err)
	}

	wechatURL := "https://push.example.com/api/notify/wechat"
	if err := runInitWechat(Streams{Stdout: &bytes.Buffer{}}, &fakePrompter{inputs: []string{wechatURL}}); err != nil {
		t.Fatalf("runInitWechat() error = %v", err)
	}

	cfg, err := config.Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if !cfg.Notify.ClaudeCode.Channels.Wechat.Enabled || cfg.Notify.ClaudeCode.Channels.Wechat.WebhookURL != wechatURL {
		t.Fatal("ClaudeCode wechat should be enabled with new URL")
	}
	// Channel-menu init is additive: must not wipe a previously configured Bark.
	if !cfg.Notify.ClaudeCode.Channels.Bark.Enabled {
		t.Fatal("ClaudeCode bark should remain enabled after wechat init")
	}
	if cfg.Notify.ClaudeCode.Channels.Bark.WebhookURL != barkURL {
		t.Fatalf("ClaudeCode bark URL = %q, want %q", cfg.Notify.ClaudeCode.Channels.Bark.WebhookURL, barkURL)
	}
}

func TestRunInitWechatDoesNotClobberPriorWechatWithEmptyReload(t *testing.T) {
	testutil.IsolateHome(t)

	path, err := config.DefaultPath()
	if err != nil {
		t.Fatalf("DefaultPath() error = %v", err)
	}
	seed := config.Default()
	oldURL := "https://push.example.com/api/notify/old"
	seed.Notify.Grok.Channels.Wechat.Enabled = true
	seed.Notify.Grok.Channels.Wechat.WebhookURL = oldURL
	// Simulate channel-only setup that never wrote events for Grok.
	seed.Notify.Grok.Events = nil
	if err := config.Save(path, seed); err != nil {
		t.Fatalf("Save() seed error = %v", err)
	}

	// Re-init with same default (empty input uses current URL from config).
	if err := runInitWechat(Streams{Stdout: &bytes.Buffer{}}, &fakePrompter{inputs: []string{oldURL}}); err != nil {
		t.Fatalf("runInitWechat() error = %v", err)
	}

	cfg, err := config.Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.Notify.Grok.Channels.Wechat.WebhookURL != oldURL {
		t.Fatalf("Grok wechat URL lost: got %q want %q", cfg.Notify.Grok.Channels.Wechat.WebhookURL, oldURL)
	}
	if len(cfg.Notify.Grok.Events) == 0 {
		t.Fatal("Grok events should be backfilled so wechat notifications can dispatch")
	}
}
