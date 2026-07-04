package notify

import (
	"os"
	"runtime"
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

// DetectSourceApp 在 hook 进程内调用，通过继承的环境变量识别宿主应用。
// macOS 上 LaunchServices 会为 GUI 启动的应用注入 __CFBundleIdentifier，
// 终端 / IDE 的子进程（即 hook 处理器）会继承它。识别失败时返回零值，
// 通知照常发送，只是点击无跳转行为。
func DetectSourceApp() SourceApp {
	termProgram := os.Getenv("TERM_PROGRAM")
	app := SourceApp{TermProgram: termProgram}

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

	return app
}
