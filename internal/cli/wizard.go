package cli

import (
	"bytes"
	"context"
	"fmt"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/matthewdeanmartin/python_for_the_win/internal/crash"
	"github.com/matthewdeanmartin/python_for_the_win/internal/errs"
	"github.com/matthewdeanmartin/python_for_the_win/internal/feed"
	"github.com/matthewdeanmartin/python_for_the_win/internal/gui"
	"github.com/matthewdeanmartin/python_for_the_win/internal/install"
	integration "github.com/matthewdeanmartin/python_for_the_win/internal/integration"
	"github.com/matthewdeanmartin/python_for_the_win/internal/layout"
	"github.com/matthewdeanmartin/python_for_the_win/internal/logx"
	"github.com/matthewdeanmartin/python_for_the_win/internal/platform"
	"github.com/matthewdeanmartin/python_for_the_win/internal/preflight"
	"github.com/matthewdeanmartin/python_for_the_win/internal/pyver"
	"github.com/matthewdeanmartin/python_for_the_win/internal/resolve"
	"github.com/matthewdeanmartin/python_for_the_win/internal/state"
	"github.com/matthewdeanmartin/python_for_the_win/internal/tui"
	"github.com/spf13/cobra"
	"io"
	"path/filepath"
	"strings"
	"time"
)

func (o *options) wizard(c *cobra.Command) error { return o.runWizard(c, false) }
func (o *options) runWizard(c *cobra.Command, graphical bool) error {
	ctx, cancel := context.WithCancel(c.Context())
	defer cancel()
	c.SetContext(ctx)
	root, e := o.rootPath()
	if e != nil {
		return e
	}
	var send func(tea.Msg)
	var selected feed.Entry
	services := tui.Services{
		Prepare: func(ch tui.Choices) (result tui.Prepared) {
			defer func() {
				if result.Err != nil && result.LogPath != "" {
					logger, closeLog, e := logx.Open(o.root)
					if e == nil {
						logger.Printf("wizard preflight: %v", result.Err)
						closeLog()
					}
				}
			}()
			defer crash.Recover(&result.Err)
			r, e := layout.Root(ch.Root)
			if e != nil {
				result.Err = e
				return result
			}
			o.root = r
			result.LogPath = filepath.Join(r, "logs", "pftw-"+time.Now().Format("20060102")+".log")
			if message := preflight.Blocking(preflight.Run(o.info, r, "", state.New(), o.allowAdmin)); message != "" {
				result.Err = fmt.Errorf("%s", message)
				return result
			}
			unlock, e := state.Lock(r)
			if e != nil {
				result.Err = e
				return result
			}
			e = install.Recover(r)
			if e == nil {
				e = integration.Recover(r, o.store)
			}
			unlock()
			if e != nil {
				result.Err = e
				return result
			}
			s, e := state.Load(r)
			if e != nil {
				result.Err = e
				return result
			}
			entries, _, _, e := o.entries(c, s)
			if e != nil {
				result.Err = e
				return result
			}
			v, e := resolve.Pick(entries, resolve.Options{Spec: ch.Spec, Arch: ch.Arch, Prerelease: ch.Prerelease})
			if e != nil {
				result.Err = e
				return result
			}
			selected = v
			result.Version = v.Version
			pv, _ := pyver.Parse(v.Version)
			result.Pre = pv.Pre()
			checks := preflight.Run(o.info, r, v.Key(), s, o.allowAdmin)
			info := []string{"pftw does not install or replace py/pyw.", "Development upgrades may break packages or external venvs."}
			for _, check := range checks {
				info = append(info, check.Title+" "+check.Detail)
			}
			result.Info = strings.Join(info, "\n")
			if message := preflight.Blocking(checks); message != "" {
				result.Err = fmt.Errorf("%s", message)
				return result
			}
			result.Decision = "AddNoClash"
			if ch.Path != "none" && !s.Portable() {
				sys, se := o.info.SystemPath()
				user, ue := o.store.Get()
				if se != nil || ue != nil {
					result.Decision = "Unknown"
				} else {
					result.Decision = preflight.PathDecision(preflight.FindPythons(sys.Text, user.Text, preflight.Owned(r, s), preflight.FileLookup(o.info)), "auto")
				}
			}
			return result
		},
		Install: func(ch tui.Choices) (result tui.Finished) {
			defer crash.Recover(&result.Err)
			o.root = ch.Root
			o.arch = ch.Arch
			o.pre = ch.Prerelease
			o.threaded = strings.Contains(selected.Key(), "t-")
			o.path = ch.Path
			o.step = func(s string) { send(tui.Step(s)) }
			o.progress = func(done, total int64) { send(tui.Progress{Done: done, Total: total}) }
			var output bytes.Buffer
			c.SetOut(&output)
			c.SetErr(io.Discard)
			e := o.install(c, []string{selected.Version})
			return tui.Finished{Message: strings.TrimSpace(output.String()), Err: e}
		},
	}
	model := tui.New(tui.Choices{Spec: "latest", Arch: platform.NativeArch(), Root: root, Path: "auto"}, services, filepath.Join(root, "logs", "pftw-"+time.Now().Format("20060102")+".log"))
	var final tui.Model
	if graphical {
		final, e = gui.Run(ctx, cancel, model, func(fn func(tea.Msg)) { send = fn })
	} else {
		program := tea.NewProgram(model, tea.WithContext(ctx), tea.WithInput(c.InOrStdin()), tea.WithOutput(c.OutOrStdout()))
		send = program.Send
		var result tea.Model
		result, e = program.Run()
		if e == nil {
			final = result.(tui.Model)
		}
	}
	if e != nil {
		return e
	}
	if final.Result.Err != nil {
		return &errs.Presented{Cause: final.Result.Err}
	}
	return nil
}
