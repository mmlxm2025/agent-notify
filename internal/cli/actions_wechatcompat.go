package cli

import (
	"context"
	"fmt"

	"github.com/hellolib/agent-notify/internal/app/tester"
	"github.com/hellolib/agent-notify/internal/config"
	"github.com/hellolib/agent-notify/internal/i18n"
)

func runTestWechatCompat(ctx context.Context, streams Streams) error {
	cfg, _, err := loadDefaultConfig()
	if err != nil {
		return err
	}

	// Try claude config first, fall back to codex, then zcode
	webhookURL := cfg.Notify.ClaudeCode.Channels.WechatCompat.WebhookURL
	if webhookURL == "" {
		webhookURL = cfg.Notify.Codex.Channels.WechatCompat.WebhookURL
	}
	if webhookURL == "" {
		webhookURL = cfg.Notify.ZCode.Channels.WechatCompat.WebhookURL
	}
	if webhookURL == "" {
		return fmt.Errorf("%s", i18n.T("err.wechatcompat_not_configured"))
	}

	svc := tester.NewService()
	result, err := svc.TestWechatCompat(ctx, webhookURL)
	if err != nil {
		return err
	}
	fmt.Fprintln(streams.Stdout, "✅ "+result.Message)
	return nil
}

func runInitWechatCompat(streams Streams, prompter Prompter) error {
	cfg, path, err := loadDefaultConfig()
	if err != nil {
		return err
	}

	// Get current webhook URL (use claude's if available)
	currentURL := cfg.Notify.ClaudeCode.Channels.WechatCompat.WebhookURL
	if currentURL == "" {
		currentURL = cfg.Notify.Codex.Channels.WechatCompat.WebhookURL
	}
	if currentURL == "" {
		currentURL = cfg.Notify.ZCode.Channels.WechatCompat.WebhookURL
	}

	webhookURL, err := prompter.Input(i18n.T("prompt.wechat_webhook_compat"), currentURL)
	if err != nil {
		return err
	}

	// Update all agents with the same webhook URL
	cfg.Notify.ClaudeCode.Channels.WechatCompat.Enabled = true
	cfg.Notify.ClaudeCode.Channels.WechatCompat.WebhookURL = webhookURL
	cfg.Notify.Codex.Channels.WechatCompat.Enabled = true
	cfg.Notify.Codex.Channels.WechatCompat.WebhookURL = webhookURL
	cfg.Notify.ZCode.Channels.WechatCompat.Enabled = true
	cfg.Notify.ZCode.Channels.WechatCompat.WebhookURL = webhookURL

	if err := config.Save(path, cfg); err != nil {
		return fmt.Errorf("%s: %w", i18n.T("err.save_failed"), err)
	}

	fmt.Fprintln(streams.Stdout, i18n.T("wechatcompat.init_done"))
	fmt.Fprintf(streams.Stdout, i18n.T("msg.config_file")+"\n", path)
	return nil
}
