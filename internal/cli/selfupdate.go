package cli

import (
	"errors"
	"fmt"
	"github.com/matthewdeanmartin/python_for_the_win/internal/errs"
	"github.com/matthewdeanmartin/python_for_the_win/internal/install"
	"github.com/matthewdeanmartin/python_for_the_win/internal/platform"
	"github.com/matthewdeanmartin/python_for_the_win/internal/selfupdate"
	"github.com/matthewdeanmartin/python_for_the_win/internal/shims"
	"github.com/matthewdeanmartin/python_for_the_win/internal/state"
	"github.com/spf13/cobra"
	"os"
	"path/filepath"
	"strings"
)

func (o *options) selfUpdate(c *cobra.Command, force bool) error {
	self := o.self
	if self == "" {
		var e error
		self, e = os.Executable()
		if e != nil {
			return e
		}
	}
	if selfupdate.Scoop(self) {
		return fmt.Errorf("use scoop update pftw")
	}
	if selfupdate.Pip(self) {
		return errs.New(3, "pftw was installed as a Python package; update it with: python -m pip install --upgrade pftw (or pipx upgrade pftw, or uv tool upgrade pftw)")
	}
	return o.mutate(c, func(s *state.State, m *install.Manager) (resultErr error) {
		d, _, e := o.client(s)
		if e != nil {
			return e
		}
		exe, version, e := (selfupdate.Client{Download: d}).Latest(c.Context(), Version, platform.NativeArch(), o.pre, force, o.root)
		if e != nil {
			return e
		}
		if exe == "" {
			return o.output(c, map[string]string{"version": version}, "Already up to date")
		}
		defer func() {
			if cleanupErr := os.RemoveAll(filepath.Dir(exe)); cleanupErr != nil {
				resultErr = errors.Join(resultErr, fmt.Errorf("temporary directory cleanup: %w", cleanupErr))
			}
		}()
		if e = shims.Sync(o.root, exe, s); e != nil {
			return e
		}
		if e = s.Save(o.root); e != nil {
			return e
		}
		message := "Updated pftw to " + version + ". Start a new pftw process."
		managed := filepath.Join(o.root, "shims", "pftw.exe")
		abs, _ := filepath.Abs(self)
		if !strings.EqualFold(filepath.Clean(abs), filepath.Clean(managed)) {
			if e = selfupdate.Replace(exe, self); e != nil {
				message += " Managed copy updated; downloaded launcher could not be replaced: " + e.Error()
			}
		}
		return o.output(c, map[string]string{"message": message}, message)
	})
}
