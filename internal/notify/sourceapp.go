package notify

import (
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
)

// termProgramBundleID 将常见终端的 TERM_PROGRAM 环境变量值映射为 macOS bundle id。
// __CFBundleIdentifier 缺失时（部分 launchd 场景）作为降级补充。
var termProgramBundleID = map[string]string{
	"Apple_Terminal": "com.apple.Terminal",
	"iTerm.app":      "com.googlecode.iterm2",
	"vscode":         "com.microsoft.VSCode",
	"WarpTerminal":   "dev.warp.Warp",
	"WezTerm":        "com.github.wez.wezterm",
	"ghostty":        "com.mitchellh.ghostty",
	"kitty":          "net.kovidgoyal.kitty",
}

var appNameBundleID = map[string]string{
	"Terminal.app":           "com.apple.Terminal",
	"iTerm.app":              "com.googlecode.iterm2",
	"Visual Studio Code.app": "com.microsoft.VSCode",
	"Code - Insiders.app":    "com.microsoft.VSCodeInsiders",
	"Cursor.app":             "com.todesktop.230313mzl4w4u92",
	"Codex.app":              "com.openai.codex",
	"GoLand.app":             "com.jetbrains.goland",
	"IntelliJ IDEA.app":      "com.jetbrains.intellij",
	"PyCharm.app":            "com.jetbrains.pycharm",
	"WebStorm.app":           "com.jetbrains.WebStorm",
	"PhpStorm.app":           "com.jetbrains.PhpStorm",
	"CLion.app":              "com.jetbrains.CLion",
	"DataGrip.app":           "com.jetbrains.datagrip",
	"RustRover.app":          "com.jetbrains.rustrover",
	"Zed.app":                "dev.zed.Zed",
}

var appPathPattern = regexp.MustCompile(`(/.*?\.app)/Contents/`)

var detectBundleIDFromProcessTreeFunc = detectBundleIDFromProcessTree

// DetectSourceApp 在 hook 进程内调用，通过继承的环境变量识别宿主应用。
// macOS 上 LaunchServices 会为 GUI 启动的应用注入 __CFBundleIdentifier，
// 终端 / IDE 的子进程（即 hook 处理器）会继承它。识别失败时返回零值，
// 通知照常发送，只是点击无跳转行为。
func DetectSourceApp() SourceApp {
	termProgram := os.Getenv("TERM_PROGRAM")
	app := SourceApp{TermProgram: termProgram, TerminalEmulator: os.Getenv("TERMINAL_EMULATOR")}

	if runtime.GOOS != "darwin" {
		return app
	}

	// 主信号：__CFBundleIdentifier（系统权威值）
	if bid := os.Getenv("__CFBundleIdentifier"); bid != "" {
		app.BundleID = bid
		return app
	}

	// 备信号：TERM_PROGRAM → bundle id 映射（主信号缺失时的降级补充）
	if bid, ok := termProgramBundleID[termProgram]; ok {
		app.BundleID = bid
		return app
	}

	// IDE 集成终端或 agent 插件经常不会设置 __CFBundleIdentifier / TERM_PROGRAM。
	// 沿父进程树向上找 .app bundle，可以覆盖 GoLand/VS Code 插件进程等场景。
	if bid := detectBundleIDFromProcessTreeFunc(os.Getpid()); bid != "" {
		app.BundleID = bid
		return app
	}

	return app
}

type processInfo struct {
	ppid    int
	command string
}

func detectBundleIDFromProcessTree(pid int) string {
	procs, err := darwinProcessTable()
	if err != nil {
		return ""
	}
	return bundleIDFromProcessTree(pid, procs)
}

func darwinProcessTable() (map[int]processInfo, error) {
	out, err := exec.Command("ps", "-axo", "pid=,ppid=,command=").Output()
	if err != nil {
		return nil, err
	}

	procs := make(map[int]processInfo)
	for _, line := range strings.Split(string(out), "\n") {
		pid, ppid, command, ok := parsePSLine(line)
		if !ok {
			continue
		}
		procs[pid] = processInfo{ppid: ppid, command: command}
	}
	return procs, nil
}

func parsePSLine(line string) (pid, ppid int, command string, ok bool) {
	rest := strings.TrimSpace(line)
	if rest == "" {
		return 0, 0, "", false
	}

	pidField, rest, ok := cutField(rest)
	if !ok {
		return 0, 0, "", false
	}
	ppidField, rest, ok := cutField(rest)
	if !ok {
		return 0, 0, "", false
	}

	pid, err := strconv.Atoi(pidField)
	if err != nil {
		return 0, 0, "", false
	}
	ppid, err = strconv.Atoi(ppidField)
	if err != nil {
		return 0, 0, "", false
	}

	return pid, ppid, strings.TrimSpace(rest), true
}

func cutField(s string) (field, rest string, ok bool) {
	s = strings.TrimLeft(s, " \t")
	if s == "" {
		return "", "", false
	}
	for i, r := range s {
		if r == ' ' || r == '\t' {
			return s[:i], strings.TrimLeft(s[i:], " \t"), true
		}
	}
	return s, "", true
}

func bundleIDFromProcessTree(pid int, procs map[int]processInfo) string {
	seen := make(map[int]bool)
	for pid > 1 && !seen[pid] {
		seen[pid] = true

		proc, ok := procs[pid]
		if !ok {
			return ""
		}
		if bid := bundleIDFromCommand(proc.command); bid != "" {
			return bid
		}
		pid = proc.ppid
	}
	return ""
}

func bundleIDFromCommand(command string) string {
	appPath := appBundlePathFromCommand(command)
	if appPath == "" {
		return ""
	}
	return bundleIDFromAppPath(appPath)
}

func appBundlePathFromCommand(command string) string {
	matches := appPathPattern.FindStringSubmatch(command)
	if len(matches) < 2 {
		return ""
	}
	return matches[1]
}

func bundleIDFromAppPath(appPath string) string {
	if bid, ok := appNameBundleID[filepath.Base(appPath)]; ok {
		return bid
	}

	info := filepath.Join(appPath, "Contents", "Info.plist")
	out, err := exec.Command("plutil", "-extract", "CFBundleIdentifier", "raw", info).Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}
