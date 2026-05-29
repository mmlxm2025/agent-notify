package cli

import (
	"context"
	"fmt"

	"github.com/hellolib/agent-notify/internal/app/tester"
	"github.com/hellolib/agent-notify/internal/config"
)

func runTestBark(ctx context.Context, streams Streams) error {
	cfg, _, err := loadDefaultConfig()
	if err != nil {
		return err
	}

	webhookURL := barkURLFromConfig(cfg)
	if webhookURL == "" {
		return fmt.Errorf("未配置 Bark Webhook URL，请先运行配置向导")
	}

	svc := tester.NewService()
	result, err := svc.TestBark(ctx, webhookURL)
	if err != nil {
		return err
	}
	fmt.Fprintln(streams.Stdout, "✅ "+result.Message)
	return nil
}

func runInitBark(streams Streams, prompter Prompter) error {
	cfg, path, err := loadDefaultConfig()
	if err != nil {
		return err
	}

	webhookURL, err := prompter.Input("Bark Webhook URL", barkURLFromConfig(cfg))
	if err != nil {
		return err
	}

	cfg.Notify.ClaudeCode.Channels.Bark.Enabled = true
	cfg.Notify.ClaudeCode.Channels.Bark.WebhookURL = webhookURL
	cfg.Notify.Codex.Channels.Bark.Enabled = true
	cfg.Notify.Codex.Channels.Bark.WebhookURL = webhookURL

	if err := config.Save(path, cfg); err != nil {
		return fmt.Errorf("保存配置失败: %w", err)
	}

	fmt.Fprintln(streams.Stdout, "✅ Bark Webhook 配置完成")
	fmt.Fprintf(streams.Stdout, "配置文件: %s\n", path)
	return nil
}

func barkURLFromConfig(cfg config.Config) string {
	if cfg.Notify.ClaudeCode.Channels.Bark.WebhookURL != "" {
		return cfg.Notify.ClaudeCode.Channels.Bark.WebhookURL
	}
	return cfg.Notify.Codex.Channels.Bark.WebhookURL
}
