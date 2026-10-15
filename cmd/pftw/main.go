package main

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"github.com/matthewdeanmartin/python_for_the_win/internal/cli"
	"github.com/matthewdeanmartin/python_for_the_win/internal/crash"
	"github.com/matthewdeanmartin/python_for_the_win/internal/errs"
	"github.com/matthewdeanmartin/python_for_the_win/internal/gui"
	"github.com/matthewdeanmartin/python_for_the_win/internal/platform"
	"github.com/matthewdeanmartin/python_for_the_win/internal/shims"
	"github.com/spf13/cobra"
	"os"
	"os/signal"
	"path/filepath"
)

func main() {
	self, err := os.Executable()
	if err == nil && shims.IsName(filepath.Base(self)) {
		os.Exit(shims.Run(self))
	}
	command := cli.New()
	graphical := cli.Graphical(command, os.Args[1:])
	if graphical {
		gui.Main(func() { run(command, true) })
		return
	}
	run(command, false)
}
func run(command *cobra.Command, graphical bool) {
	platform.SetTitle()
	e := execute(command)
	var child *errs.ChildExit
	if errors.As(e, &child) {
		os.Exit(child.Code)
	}
	code := 0
	var presented *errs.Presented
	shown := errors.As(e, &presented)
	if e != nil {
		if !shown {
			_ = cli.ReportError(os.Stderr, e, cli.JSONRequested(command)) // Nothing remains to report a failed stderr write to.
		}
		code = errs.CodeOf(e)
	}
	if own, _ := (platform.Native{}).OwnsConsole(); own && platform.IsTerminal(os.Stdin) && platform.IsTerminal(os.Stdout) && !shown && (!graphical || e != nil) && (e != nil || len(os.Args) > 1) {
		fmt.Fprintln(os.Stderr, "Press Enter to close.")
		_, _ = bufio.NewReader(os.Stdin).ReadString(10)
	}
	os.Exit(code)
}
func execute(command *cobra.Command) (err error) {
	defer crash.Recover(&err)
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()
	return command.ExecuteContext(ctx)
}
