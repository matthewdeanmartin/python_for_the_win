package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/matthewdeanmartin/python_for_the_win/internal/errs"
	"github.com/matthewdeanmartin/python_for_the_win/internal/feed"
	"github.com/matthewdeanmartin/python_for_the_win/internal/platform"
	"github.com/matthewdeanmartin/python_for_the_win/internal/pyver"
	"github.com/matthewdeanmartin/python_for_the_win/internal/resolve"
	"github.com/matthewdeanmartin/python_for_the_win/internal/selfupdate"
	"github.com/spf13/cobra"
)

// plan never enters the mutation pipeline: no locks, journals, cache or logs.
func (o *options) plan(c *cobra.Command, args []string) error {
	s, e := o.load()
	if e != nil {
		return e
	}
	changes := []string{}
	add := func(format string, values ...any) { changes = append(changes, fmt.Sprintf(format, values...)) }
	path := func(parts ...string) string { return filepath.Join(append([]string{o.root}, parts...)...) }
	for _, name := range []string{"transaction.json", "path-transaction.json"} {
		if _, err := os.Stat(path(name)); err == nil {
			return errs.New(3, "Pending recovery at %s; preview cannot predict changes until recovery completes", path(name))
		}
	}
	sync := func() {
		add("Update managed commands and ownership manifest in %s; preserve foreign files", path("shims"))
		add("Save managed state: %s", path("pftw.json"))
		if s.Path().Mode == "first" {
			add("Reconcile only pftw-owned user PATH entries with the resulting default; preserve unrelated entries")
		}
	}
	installation := func(v feed.Entry) error {
		if raw, ok := s.Installs[v.Key()]; ok && raw != nil {
			old, err := s.Get(v.Key())
			if err != nil {
				return err
			}
			before, err := pyver.Parse(old.Version)
			if err != nil {
				return err
			}
			after, err := pyver.Parse(v.Version)
			if err != nil {
				return err
			}
			if after.Compare(before) <= 0 {
				add("Python %s already installed; no runtime replacement", old.Version)
				return nil
			}
			add("Replace Python %s with %s at %s using a recovery journal and temporary backup in %s", old.Version, v.Version, path("versions", v.Key()), path("trash"))
			if !o.fresh {
				add("Preserve existing site-packages and Scripts; do not modify external venvs")
			}
		} else {
			add("Install Python %s into %s", v.Version, path("versions", v.Key()))
		}
		add("Download and verify %s (SHA-256 %s) into %s", v.URL, v.Hash["sha256"], path("cache"))
		add("Extract and smoke-test using temporary files in %s; remove temporary files after commit", path("staging"))
		if !o.noPip {
			add("Bootstrap and smoke-test pip inside the managed runtime")
		}
		return nil
	}
	switch c.Name() {
	case "install":
		spec := "latest"
		if len(args) > 0 {
			spec = args[0]
		}
		entries, _, _, err := o.entries(c, s)
		if err != nil {
			return err
		}
		v, err := resolve.Pick(entries, resolve.Options{Spec: spec, Arch: o.arch, Threaded: o.threaded, Prerelease: o.pre})
		if err != nil {
			return err
		}
		if err = installation(v); err != nil {
			return err
		}
		mode, err := o.choosePath(c, s)
		if err != nil {
			return err
		}
		if mode == "first" {
			add("Add pftw's shims, default runtime and Scripts directories to user PATH (never system PATH)")
		}
		if o.setDefaultFlag || s.Default == "" {
			add("Set default Python to %s", v.Key())
		}
		if o.portable {
			add("Mark root portable; no PATH changes")
		}
		sync()
	case "upgrade", "uninstall":
		keys := []string{}
		if o.all {
			for k := range s.Installs {
				keys = append(keys, k)
			}
			sort.Strings(keys)
		} else {
			k, err := installedKey(s, args[0])
			if err != nil {
				return err
			}
			keys = append(keys, k)
		}
		if c.Name() == "upgrade" {
			entries, _, _, err := o.entries(c, s)
			if err != nil {
				return err
			}
			for _, k := range keys {
				old, err := s.Get(k)
				if err != nil {
					return err
				}
				pv, err := pyver.Parse(old.Version)
				if err != nil {
					return err
				}
				v, err := resolve.Pick(entries, resolve.Options{Spec: k, Arch: old.Arch, Threaded: old.Threaded, Prerelease: pv.Pre()})
				if err != nil {
					return err
				}
				next, _ := pyver.Parse(v.Version)
				if !pv.Pre() && next.Pre() {
					return errs.New(3, "No final upgrade is available for %s", k)
				}
				o.noPip = !old.Pip
				if err = installation(v); err != nil {
					return err
				}
			}
		} else {
			for _, k := range keys {
				add("Remove managed runtime %s using a recovery journal and temporary backup in %s", path("versions", k), path("trash"))
			}
			add("Remove only owned command/PATH entries; choose a remaining default if necessary")
			if o.all {
				add("Remove pftw-owned manager, state, logs and cache; remove %s only if empty; preserve foreign files", o.root)
			}
		}
		sync()
	case "default":
		k, err := installedKey(s, args[0])
		if err != nil {
			return err
		}
		add("Set default Python to %s", k)
		sync()
	case "config":
		if args[0] == "set" {
			if args[1] == "source" {
				d, _, err := o.client(s)
				if err != nil {
					return err
				}
				it, err := feed.Load(d, args[2], "", true)
				if err != nil {
					return err
				}
				if _, err = it.Next(c.Context()); err != nil {
					return err
				}
			}
			if args[1] == "proxy" {
				copy := *o
				copy.proxy = args[2]
				if _, _, err := copy.client(s); err != nil {
					return err
				}
			}
			add("Set config %s = %q in %s", args[1], args[2], path("pftw.json"))
		} else {
			add("Remove config %s from %s", args[1], path("pftw.json"))
		}
	case "self-update":
		d, _, err := o.client(s)
		if err != nil {
			return err
		}
		force, _ := c.Flags().GetBool("force")
		asset, version, err := (selfupdate.Client{Download: d, Preview: true}).Latest(c.Context(), Version, platform.NativeArch(), o.pre, force, o.root)
		if err != nil {
			return err
		}
		if asset == "" {
			add("Already up to date (%s)", version)
			break
		}
		add("Download and verify %s; stage update in %s", asset, path("staging"))
		add("Replace manager with %s at %s and, if writable, %s; retain executable .old backup until next mutating invocation", version, path("shims", "pftw.exe"), func() string {
			if o.self != "" {
				return o.self
			}
			exe, _ := os.Executable()
			return exe
		}())
		sync()
	default:
		return errs.New(2, "No preview is available for %s", c.CommandPath())
	}
	if len(changes) > 0 {
		add("A real mutation also uses %s and writes logs under %s", path("pftw.lock"), path("logs"))
	}
	return o.output(c, map[string]any{"dry_run": true, "changes": changes}, "Dry run: no files or settings changed.\n"+strings.Join(changes, "\n"))
}
