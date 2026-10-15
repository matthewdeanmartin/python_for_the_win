package cli

import (
	"github.com/matthewdeanmartin/python_for_the_win/internal/platform"
	"github.com/spf13/cobra"
	"os"
	"runtime"
)

// Graphical prepares root-only entry flags before Gio takes the main thread.
// Explicit commands always retain their own behavior and argument parsing.
func Graphical(command *cobra.Command, args []string) bool {
	selected, _, err := command.Find(args)
	if err != nil {
		return false
	}
	if selected != command {
		return selected.Name() == "gui"
	}
	if err = command.ParseFlags(args); err != nil {
		return false
	}
	if len(command.Flags().Args()) != 0 {
		return false
	}
	return graphicalRoot(command)
}
func graphicalRoot(c *cobra.Command) bool {
	if runtime.GOOS != "windows" {
		return false
	}
	for _, flag := range []string{"tui", "help", "json", "yes", "quiet", "dry-run", "dry-mode"} {
		value, _ := c.Flags().GetBool(flag)
		if value {
			return false
		}
	}
	force, _ := c.Flags().GetBool("gui")
	if force {
		return true
	}
	return platform.IsTerminal(os.Stdin) && platform.IsTerminal(os.Stdout) && platform.DesktopLaunch()
}
