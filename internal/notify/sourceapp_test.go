package notify

import (
	"runtime"
	"testing"
)

func TestDetectSourceAppReadsBundleIdentifier(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("darwin only")
	}
	t.Setenv("__CFBundleIdentifier", "com.googlecode.iterm2")
	t.Setenv("TERM_PROGRAM", "iTerm.app")

	got := DetectSourceApp()
	if got.BundleID != "com.googlecode.iterm2" {
		t.Fatalf("BundleID = %q, want com.googlecode.iterm2", got.BundleID)
	}
	if got.TermProgram != "iTerm.app" {
		t.Fatalf("TermProgram = %q, want iTerm.app", got.TermProgram)
	}
}

func TestDetectSourceAppFallsBackToTermProgram(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("darwin only")
	}
	t.Setenv("__CFBundleIdentifier", "")
	t.Setenv("TERM_PROGRAM", "Apple_Terminal")

	got := DetectSourceApp()
	if got.BundleID != "com.apple.Terminal" {
		t.Fatalf("BundleID = %q, want com.apple.Terminal (fallback)", got.BundleID)
	}
	if got.TermProgram != "Apple_Terminal" {
		t.Fatalf("TermProgram = %q, want Apple_Terminal", got.TermProgram)
	}
}

func TestDetectSourceAppEmptyEnv(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("darwin only")
	}
	t.Setenv("__CFBundleIdentifier", "")
	t.Setenv("TERM_PROGRAM", "")

	got := DetectSourceApp()
	if got.BundleID != "" {
		t.Fatalf("BundleID = %q, want empty", got.BundleID)
	}
}

func TestDetectSourceAppUnknownTermProgram(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("darwin only")
	}
	t.Setenv("__CFBundleIdentifier", "")
	t.Setenv("TERM_PROGRAM", "some-unknown-terminal")

	got := DetectSourceApp()
	if got.BundleID != "" {
		t.Fatalf("BundleID = %q, want empty for unknown TERM_PROGRAM", got.BundleID)
	}
	if got.TermProgram != "some-unknown-terminal" {
		t.Fatalf("TermProgram = %q, want some-unknown-terminal", got.TermProgram)
	}
}
