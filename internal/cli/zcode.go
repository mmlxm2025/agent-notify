package cli

import (
	"github.com/hellolib/agent-notify/internal/common"
	"github.com/spf13/cobra"
)

func newZcodeCmd(streams Streams) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "zcode",
		Short: "Manage ZCode hook integration",
	}
	cmd.AddCommand(newZcodePrintHooksCmd(streams), newZcodeInstallHooksCmd())
	return cmd
}

func newZcodePrintHooksCmd(streams Streams) *cobra.Command {
	var binaryPath string

	cmd := &cobra.Command{
		Use:   "print-hooks",
		Short: "Print ZCode hook settings JSON",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runPrintZcodeHooks(streams, firstNonEmpty(binaryPath))
		},
	}
	cmd.Flags().StringVar(&binaryPath, "binary", common.ResolveBinaryPath(""), "agent-notify binary path")
	return cmd
}

func newZcodeInstallHooksCmd() *cobra.Command {
	var binaryPath string
	var scope string

	cmd := &cobra.Command{
		Use:   "install-hooks",
		Short: "Install ZCode hook settings",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runInstallZcodeHooks(scope, firstNonEmpty(binaryPath))
		},
	}
	cmd.Flags().StringVar(&binaryPath, "binary", common.ResolveBinaryPath(""), "agent-notify binary path")
	cmd.Flags().StringVar(&scope, "scope", "user", "install scope")
	return cmd
}
