package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/hellolib/agent-notify/internal/agentintegrations"
	"github.com/hellolib/agent-notify/internal/app/doctor"
	"github.com/hellolib/agent-notify/internal/app/setup"
	"github.com/hellolib/agent-notify/internal/app/tester"
	"github.com/hellolib/agent-notify/internal/claudehooks"
	"github.com/hellolib/agent-notify/internal/common"
	"github.com/hellolib/agent-notify/internal/config"
)

// cliPrompter adapts CLI Prompter to setup.Prompter
type cliPrompter struct {
	p Prompter
}

func (cp *cliPrompter) Select(message string, options []setup.PromptOption, defaultValue string) (string, error) {
	cliOptions := make([]PromptOption, len(options))
	for i, o := range options {
		cliOptions[i] = PromptOption{Label: o.Label, Value: o.Value}
	}
	return cp.p.Select(message, cliOptions, defaultValue)
}

func (cp *cliPrompter) MultiSelect(message string, options []setup.PromptOption, defaults []string) ([]string, error) {
	cliOptions := make([]PromptOption, len(options))
	for i, o := range options {
		cliOptions[i] = PromptOption{Label: o.Label, Value: o.Value}
	}
	return cp.p.MultiSelect(message, cliOptions, defaults)
}

func (cp *cliPrompter) Confirm(message string, defaultValue bool) (bool, error) {
	return cp.p.Confirm(message, defaultValue)
}

func (cp *cliPrompter) Input(message, defaultValue string) (string, error) {
	return cp.p.Input(message, defaultValue)
}

// cliOutputWriter adapts Streams to setup/doctor OutputWriter
type cliOutputWriter struct {
	streams Streams
}

func (w *cliOutputWriter) Writef(format string, args ...any) {
	fmt.Fprintf(w.streams.Stdout, format, args...)
}

// feishuPreparerAdapter adapts the prepareFeishuCLI function to setup.FeishuPreparer
type feishuPreparerAdapter struct{}

func (f *feishuPreparerAdapter) EnsureReady(ctx context.Context) error {
	return prepareFeishuCLI(ctx)
}

func runInitFlow(ctx context.Context, streams Streams, prompter Prompter, configPath, settingsPath, binaryPath string) error {
	_ = settingsPath // kept for backward compatibility

	svc := setup.NewService(
		setup.WithClaudeIntegration(agentintegrations.NewClaudeIntegration()),
		setup.WithCodexIntegration(agentintegrations.NewCodexIntegration()),
		setup.WithFeishuPreparer(&feishuPreparerAdapter{}),
	)

	cliPrompter := &cliPrompter{p: prompter}
	output := &cliOutputWriter{streams: streams}

	_, err := svc.Run(ctx, cliPrompter, output, configPath, binaryPath)
	return err
}

func runPrintClaudeHooks(streams Streams, binaryPath string) error {
	settings := claudehooks.BuildHookSettings(common.ResolveBinaryPath(binaryPath))
	data, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(streams.Stdout, string(data))
	return err
}

func runInstallClaudeHooks(scope, binaryPath string) error {
	path, err := settingsPathForAgent("claude", scope)
	if err != nil {
		return err
	}
	return claudehooks.Install(path, common.ResolveBinaryPath(binaryPath))
}

func runTestFeishu(ctx context.Context, streams Streams) error {
	svc := tester.NewService(
		tester.WithFeishuPreparer(&feishuPreparerAdapter{}),
	)
	result, err := svc.TestFeishu(ctx)
	if err != nil {
		return err
	}
	fmt.Fprintln(streams.Stdout, "✅ "+result.Message)
	return nil
}

func runTestSystem(ctx context.Context, streams Streams) error {
	svc := tester.NewService()
	result, err := svc.TestSystem(ctx)
	if err != nil {
		return err
	}
	fmt.Fprintln(streams.Stdout, "✅ "+result.Message)
	return nil
}

func runTestWechatWork(ctx context.Context, streams Streams) error {
	cfg, _, err := loadDefaultConfig()
	if err != nil {
		return err
	}

	// Try claude config first, fall back to codex
	webhookURL := cfg.Notify.ClaudeCode.Channels.WechatWork.WebhookURL
	if webhookURL == "" {
		webhookURL = cfg.Notify.Codex.Channels.WechatWork.WebhookURL
	}
	if webhookURL == "" {
		return fmt.Errorf("未配置企业微信 Webhook URL，请先运行配置向导")
	}

	svc := tester.NewService()
	result, err := svc.TestWechatWork(ctx, webhookURL)
	if err != nil {
		return err
	}
	fmt.Fprintln(streams.Stdout, "✅ "+result.Message)
	return nil
}

func runDoctor(streams Streams) error {
	svc := doctor.NewService(
		doctor.WithClaudeIntegration(agentintegrations.NewClaudeIntegration()),
		doctor.WithCodexIntegration(agentintegrations.NewCodexIntegration()),
	)
	result, err := svc.Run()
	if err != nil {
		return err
	}
	output := &cliOutputWriter{streams: streams}
	svc.Print(output, result)
	return nil
}

func loadDefaultConfig() (config.Config, string, error) {
	path, err := config.DefaultPath()
	if err != nil {
		return config.Config{}, "", err
	}
	cfg, err := config.Load(path)
	if err != nil {
		return config.Config{}, "", err
	}
	return cfg, path, nil
}

func printCurrentNotifyConfig(streams Streams) error {
	cfg, path, err := loadDefaultConfig()
	if err != nil {
		return err
	}

	fmt.Fprintf(streams.Stdout, "配置文件: %s\n\n", path)

	// Agent status table
	fmt.Fprintln(streams.Stdout, "┌─────────────┬──────┬──────┬────────┐")
	fmt.Fprintln(streams.Stdout, "│ Agent       │ 飞书   │ 系统   │ 企业微信 │")
	fmt.Fprintln(streams.Stdout, "├─────────────├──────├──────├────────┤")

	// Claude Code
	claudeFeishu := "❌"
	if cfg.Notify.ClaudeCode.Channels.Feishu.Enabled {
		claudeFeishu = "✅"
	}
	claudeSystem := "❌"
	if cfg.Notify.ClaudeCode.Channels.System.Enabled {
		claudeSystem = "✅"
	}
	claudeWechat := "❌"
	if cfg.Notify.ClaudeCode.Channels.WechatWork.Enabled {
		claudeWechat = "✅"
	}
	fmt.Fprintf(streams.Stdout, "│ Claude Code │   %s    │   %s    │   %s      │\n", claudeFeishu, claudeSystem, claudeWechat)

	// Codex
	codexFeishu := "❌"
	if cfg.Notify.Codex.Channels.Feishu.Enabled {
		codexFeishu = "✅"
	}
	codexSystem := "❌"
	if cfg.Notify.Codex.Channels.System.Enabled {
		codexSystem = "✅"
	}
	codexWechat := "❌"
	if cfg.Notify.Codex.Channels.WechatWork.Enabled {
		codexWechat = "✅"
	}
	fmt.Fprintf(streams.Stdout, "│ Codex       │   %s    │   %s    │   %s      │\n", codexFeishu, codexSystem, codexWechat)

	fmt.Fprintln(streams.Stdout, "└─────────────┴──────┴──────┴────────┘")
	return nil
}

// settingsPathForAgent returns the settings path for the given agent and scope.
// Currently only Claude has manual install-hooks subcommands; the Codex path is
// handled exclusively through the init flow + CodexIntegration.
func settingsPathForAgent(agent, scope string) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	switch agent {
	case "claude":
		switch scope {
		case "user":
			return filepath.Join(home, ".claude", "settings.json"), nil
		case "project":
			return filepath.Join(".claude", "settings.json"), nil
		default:
			return "", fmt.Errorf("unsupported scope: %s", scope)
		}
	default:
		return "", fmt.Errorf("unsupported agent: %s", agent)
	}
}
