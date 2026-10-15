package cli

import (
	"github.com/matthewdeanmartin/python_for_the_win/internal/errs"
	"github.com/matthewdeanmartin/python_for_the_win/internal/install"
	"github.com/matthewdeanmartin/python_for_the_win/internal/layout"
	"github.com/matthewdeanmartin/python_for_the_win/internal/proc"
	"github.com/matthewdeanmartin/python_for_the_win/internal/resolve"
	"github.com/spf13/cobra"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

func (o *options) installTarget(c *cobra.Command, args []string) error {
	for _, flag := range []string{"portable", "set-default", "fresh", "no-pip", "path"} {
		if c.Flags().Changed(flag) {
			return errs.New(2, "--target cannot be combined with --%s", flag)
		}
	}
	target, err := filepath.Abs(o.target)
	if err != nil {
		return err
	}
	if _, err = layout.Safe(filepath.Dir(target), filepath.Base(target)); err != nil {
		return err
	}
	if _, err = os.Lstat(target); !os.IsNotExist(err) {
		return errs.New(3, "target already exists or is inaccessible: %s", target)
	}
	parent, err := os.Stat(filepath.Dir(target))
	if err != nil {
		return err
	}
	if !parent.IsDir() {
		return errs.New(3, "target parent must be a directory")
	}
	s, err := o.load()
	if err != nil {
		return err
	}
	readonly := *o
	readonly.dryRun = true
	entries, d, _, err := readonly.entries(c, s)
	if err != nil {
		return err
	}
	spec := "latest"
	if len(args) > 0 {
		spec = args[0]
	}
	entry, err := resolve.Pick(entries, resolve.Options{Spec: spec, Arch: o.arch, Threaded: o.threaded, Prerelease: o.pre})
	if err != nil {
		return err
	}
	changes := []string{"Create temporary download/check directory beside " + target, "Download and verify Python " + entry.Version, "Extract runtime files into " + target, "Run isolated Python smoke test", "Remove temporary download/check directory"}
	if o.dryRun {
		return o.output(c, map[string]any{"dry_run": true, "changes": changes}, "Dry run: no files or settings changed.\n"+strings.Join(changes, "\n"))
	}
	if runtime.GOOS != "windows" {
		return errs.New(8, "Installing Windows Python requires Windows")
	}
	elevated, err := o.info.IsElevated()
	if err != nil {
		return err
	}
	if elevated && !o.allowAdmin {
		return errs.New(3, "Run pftw as yourself, not as administrator (or explicitly use --allow-admin)")
	}
	m := install.Manager{Download: d, Runner: proc.Real{}, FreeBytes: o.info.FreeBytes}
	if err = m.Target(c.Context(), entry, target); err != nil {
		return err
	}
	return o.output(c, map[string]string{"target": target, "version": entry.Version}, "Extracted Python "+entry.Version+" to "+target)
}
