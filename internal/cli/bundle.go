package cli

import (
	"archive/zip"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/matthewdeanmartin/python_for_the_win/internal/overlay"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/matthewdeanmartin/python_for_the_win/internal/errs"
	"github.com/matthewdeanmartin/python_for_the_win/internal/feed"
	"github.com/matthewdeanmartin/python_for_the_win/internal/resolve"
	"github.com/spf13/cobra"
)

func (o *options) bundleCommand() *cobra.Command {
	var out string
	c := &cobra.Command{Use: "bundle SPEC --out FILE.zip|FILE.exe", Short: "Create a verified offline installation bundle", Args: cobra.ExactArgs(1)}
	c.Flags().StringVar(&out, "out", "", "New output ZIP or EXE path (will not overwrite an existing file)")
	c.RunE = func(c *cobra.Command, args []string) (resultErr error) {
		executable := strings.EqualFold(filepath.Ext(out), ".exe")
		if out == "" || (!executable && !strings.EqualFold(filepath.Ext(out), ".zip")) {
			return errs.New(2, "--out must name a new .zip or .exe file")
		}
		target, e := filepath.Abs(out)
		if e != nil {
			return e
		}
		if _, e = os.Lstat(target); !os.IsNotExist(e) {
			return errs.New(2, "output already exists or cannot be inspected: %s", target)
		}
		s, e := o.load()
		if e != nil {
			return e
		}
		// Bundle reads never populate the installation root's index cache.
		readonly := *o
		readonly.dryRun = true
		entries, d, _, e := readonly.entries(c, s)
		if e != nil {
			return e
		}
		v, e := resolve.Pick(entries, resolve.Options{Spec: args[0], Arch: o.arch, Threaded: o.threaded, Prerelease: o.pre})
		if e != nil {
			return e
		}
		self := o.self
		if self == "" {
			self, e = os.Executable()
			if e != nil {
				return e
			}
		}
		if o.dryRun {
			return o.output(c, map[string]any{"dry_run": true, "output": target, "runtime": v.Version}, fmt.Sprintf("Would download and verify Python %s (%s).\nWould create temporary work files beside %s, then create that bundle containing pftw.exe, index.json and the verified runtime ZIP.\nNo files or settings changed.", v.Version, v.Hash["sha256"], target))
		}
		work, e := os.MkdirTemp(filepath.Dir(target), ".pftw-bundle-")
		if e != nil {
			return e
		}
		defer func() {
			if cleanupErr := os.RemoveAll(work); cleanupErr != nil {
				resultErr = errors.Join(resultErr, fmt.Errorf("temporary directory cleanup: %w", cleanupErr))
			}
		}()
		payload, e := d.Fetch(c.Context(), v.URL, v.Hash["sha256"], work)
		if e != nil {
			return e
		}
		staged := filepath.Join(work, "bundle.zip")
		f, e := os.OpenFile(staged, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
		if e != nil {
			return e
		}
		z := zip.NewWriter(f)
		add := func(name, path string) error {
			in, _, e := overlay.Base(path)
			if strings.HasPrefix(path, overlay.Scheme+"://") {
				in, _, e = overlay.Open(path)
			}
			if e != nil {
				return e
			}
			defer func() { _ = in.Close() }() // Read-only handle; read errors are checked above.
			w, e := z.Create(name)
			if e != nil {
				return e
			}
			_, e = io.Copy(w, in)
			return e
		}
		if e = add("pftw.exe", self); e == nil {
			companion := filepath.Join(filepath.Dir(self), "pftww.exe")
			if info, err := overlay.Inspect(self); err == nil && info != nil {
				companion = overlay.Source(self, "pftww.exe")
			}
			available := false
			if strings.HasPrefix(companion, overlay.Scheme+"://") {
				r, _, err := overlay.Open(companion)
				if err == nil {
					available = true
					_ = r.Close() // Read-only or already-failed operation; preserve the primary error.
				}
			} else {
				_, err := os.Stat(companion)
				available = err == nil
			}
			if available {
				e = add("pftww.exe", companion)
			}
		}
		if e == nil {
			v.URL = v.Hash["sha256"] + ".zip"
			e = add(v.URL, payload)
		}
		if e == nil {
			var w io.Writer
			w, e = z.Create("index.json")
			if e == nil {
				e = json.NewEncoder(w).Encode(feed.Index{Versions: []feed.Entry{v}})
			}
		}
		ze := z.Close()
		if e == nil {
			e = ze
		}
		if e == nil {
			e = f.Sync()
		}
		ce := f.Close()
		if e == nil {
			e = ce
		}
		if e != nil {
			return e
		}
		// Exclusive destination publication preserves a concurrently created foreign file.
		dest, e := os.OpenFile(target, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
		if e != nil {
			return e
		}
		good := false
		defer func() {
			_ = dest.Close()
			if !good {
				_ = os.Remove(target)
			}
		}()
		in, e := os.Open(staged)
		if e != nil {
			return e
		}
		defer func() { _ = in.Close() }() // Read-only handle; read errors are checked above.
		if executable {
			e = overlay.Write(dest, self, staged)
		} else {
			_, e = io.Copy(dest, in)
		}
		if e != nil {
			return e
		}
		if e = dest.Sync(); e != nil {
			return e
		}
		if e = dest.Close(); e != nil {
			return e
		}
		good = true
		message := "Created " + target + ". Extract it, then run its pftw.exe install " + v.Key() + " --path none."
		if executable {
			message = "Created " + target + ". Run it for the offline wizard, or use: install " + v.Key() + " --path none"
		}
		return o.output(c, map[string]string{"bundle": target, "version": v.Version}, message)
	}
	return c
}
