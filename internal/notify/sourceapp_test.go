package notify

import (
	"runtime"
	"testing"
)

func withoutProcessTreeFallback(t *testing.T) {
	t.Helper()

	orig := detectBundleIDFromProcessTreeFunc
	detectBundleIDFromProcessTreeFunc = func(int) string { return "" }
	t.Cleanup(func() { detectBundleIDFromProcessTreeFunc = orig })
}

func TestDetectSourceAppReadsBundleIdentifier(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("darwin only")
	}
	withoutProcessTreeFallback(t)
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

func TestParsePSLinePreservesCommandSpaces(t *testing.T) {
	pid, ppid, command, ok := parsePSLine("41706     1 /Applications/Visual Studio Code.app/Contents/MacOS/Code --goto /tmp/a")
	if !ok {
		t.Fatal("parsePSLine() ok = false")
	}
	if pid != 41706 || ppid != 1 {
		t.Fatalf("pid/ppid = %d/%d, want 41706/1", pid, ppid)
	}
	want := "/Applications/Visual Studio Code.app/Contents/MacOS/Code --goto /tmp/a"
	if command != want {
		t.Fatalf("command = %q, want %q", command, want)
	}
}

func TestAppBundlePathFromCommand(t *testing.T) {
	command := "/Applications/Visual Studio Code.app/Contents/Frameworks/Code Helper (Plugin).app/Contents/MacOS/Code Helper (Plugin)"
	got := appBundlePathFromCommand(command)
	want := "/Applications/Visual Studio Code.app"
	if got != want {
		t.Fatalf("appBundlePathFromCommand() = %q, want %q", got, want)
	}
}

func TestBundleIDFromProcessTreeFindsIDEAncestor(t *testing.T) {
	procs := map[int]processInfo{
		101: {ppid: 100, command: "/Users/me/.agent-notify/agent-notify handle-codex-hook"},
		100: {ppid: 99, command: "zsh"},
		99:  {ppid: 42, command: "node /Users/me/Library/Caches/JetBrains/GoLand2026.1/acp-agents/agent.js"},
		42:  {ppid: 1, command: "/Applications/GoLand.app/Contents/MacOS/goland /repo"},
	}

	got := bundleIDFromProcessTree(101, procs)
	if got != "com.jetbrains.goland" {
		t.Fatalf("bundleIDFromProcessTree() = %q, want com.jetbrains.goland", got)
	}
}

func TestBundleIDFromProcessTreeFindsVSCodeHelperAncestor(t *testing.T) {
	procs := map[int]processInfo{
		101: {ppid: 100, command: "/Users/me/.agent-notify/agent-notify handle-codex-hook"},
		100: {ppid: 42, command: "/Applications/Visual Studio Code.app/Contents/Frameworks/Code Helper (Plugin).app/Contents/MacOS/Code Helper (Plugin)"},
		42:  {ppid: 1, command: "/Applications/Visual Studio Code.app/Contents/MacOS/Code"},
	}

	got := bundleIDFromProcessTree(101, procs)
	if got != "com.microsoft.VSCode" {
		t.Fatalf("bundleIDFromProcessTree() = %q, want com.microsoft.VSCode", got)
	}
}

func TestDetectSourceAppFallsBackToTermProgram(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("darwin only")
	}
	withoutProcessTreeFallback(t)
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
	withoutProcessTreeFallback(t)
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
	withoutProcessTreeFallback(t)
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
