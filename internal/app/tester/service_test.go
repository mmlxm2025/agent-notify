package tester

import (
	"context"
	"testing"

	"github.com/hellolib/agent-notify/internal/config"
	"github.com/hellolib/agent-notify/internal/notify"
)

// mockFeishuPreparer implements FeishuPreparer for testing
type mockFeishuPreparer struct {
	called bool
	err    error
}

func (m *mockFeishuPreparer) EnsureReady(ctx context.Context) error {
	m.called = true
	return m.err
}

type mockConfigLoader struct {
	defaultPath string
	loadPath    string
	cfg         config.Config
}

func (m *mockConfigLoader) Load(path string) (config.Config, error) {
	m.loadPath = path
	return m.cfg, nil
}

func (m *mockConfigLoader) DefaultPath() (string, error) {
	return m.defaultPath, nil
}

type fakeSender struct {
	called bool
	err    error
}

func (f *fakeSender) Name() string { return "fake" }

func (f *fakeSender) Send(ctx context.Context, msg notify.Message) error {
	f.called = true
	return f.err
}

func TestNewService(t *testing.T) {
	svc := NewService()
	if svc == nil {
		t.Fatal("NewService() returned nil")
	}
}

func TestNewServiceWithOptions(t *testing.T) {
	preparer := &mockFeishuPreparer{}
	svc := NewService(WithFeishuPreparer(preparer))
	if svc == nil {
		t.Fatal("NewService() returned nil")
	}
	if svc.feishuPreparer != preparer {
		t.Error("feishuPreparer not set correctly")
	}
}

func TestTestFeishu_SendsNotification(t *testing.T) {
	preparer := &mockFeishuPreparer{}
	sender := &fakeSender{}
	svc := NewService(
		WithFeishuPreparer(preparer),
		WithFeishuSender(sender),
	)

	result, err := svc.TestFeishu(context.Background())
	if err != nil {
		t.Fatalf("TestFeishu() error = %v", err)
	}
	if result == nil || result.Message == "" {
		t.Fatal("expected non-empty result")
	}
	if !preparer.called {
		t.Fatal("expected preparer to be called")
	}
	if !sender.called {
		t.Fatal("expected injected sender to be called")
	}
}

func TestTestSystem_UsesInjectedSender(t *testing.T) {
	sender := &fakeSender{}
	svc := NewService(WithSystemSender(sender))

	result, err := svc.TestSystem(context.Background())
	if err != nil {
		t.Fatalf("TestSystem() error = %v", err)
	}
	if result == nil || result.Message == "" {
		t.Fatal("expected non-empty result")
	}
	if !sender.called {
		t.Fatal("expected injected sender to be called")
	}
}

func TestTestBark_UsesInjectedSender(t *testing.T) {
	sender := &fakeSender{}
	svc := NewService(WithBarkSender(sender))

	result, err := svc.TestBark(context.Background(), "https://api.day.app/key")
	if err != nil {
		t.Fatalf("TestBark() error = %v", err)
	}
	if result == nil || result.Message == "" {
		t.Fatal("expected non-empty result")
	}
	if !sender.called {
		t.Fatal("expected injected sender to be called")
	}
}

func TestTestFeishu_IgnoresEnabledFlag(t *testing.T) {
	// Test notification intentionally ignores the enabled flag in config.
	// This allows users to verify Feishu connectivity before enabling it permanently.
	// Even without config, the test will attempt to send (may fail at the sender level if no Feishu CLI is configured, but not because of the enabled flag)
	sender := &fakeSender{}
	svc := NewService(
		WithFeishuPreparer(&mockFeishuPreparer{}),
		WithFeishuSender(sender),
	)

	// Should not fail due to "feishu disabled" check
	result, err := svc.TestFeishu(context.Background())
	if err != nil {
		t.Fatalf("TestFeishu() should not return 'feishu disabled' error, got = %v", err)
	}
	if result == nil || result.Message == "" {
		t.Fatal("expected non-empty result")
	}
	if !sender.called {
		t.Fatal("expected sender to be called")
	}
}

func TestTestSystem(t *testing.T) {
	svc := NewService(WithSystemSender(&fakeSender{}))

	result, err := svc.TestSystem(context.Background())
	if err != nil {
		t.Fatalf("TestSystem() error = %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if result.Message == "" {
		t.Error("expected non-empty message")
	}
}
