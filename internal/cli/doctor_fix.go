package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/matthewdeanmartin/python_for_the_win/internal/envpath"
	"github.com/matthewdeanmartin/python_for_the_win/internal/install"
	"github.com/matthewdeanmartin/python_for_the_win/internal/layout"
	"github.com/matthewdeanmartin/python_for_the_win/internal/shims"
	"github.com/matthewdeanmartin/python_for_the_win/internal/state"
	"github.com/spf13/cobra"
)

// Repairs require ownership evidence. Missing runtimes and foreign files need human inspection.
func (o *options) doctorFix(c *cobra.Command) error {
	s, e := o.load()
	if e != nil {
		return e
	}
	changes, _, e := o.repairs(s, false)
	if e != nil {
		return e
	}
	if o.dryRun || len(changes) == 0 {
		if len(changes) == 0 {
			changes = append(changes, "No automatic repairs identified.")
		}
		return o.output(c, map[string]any{"dry_run": o.dryRun, "repairs": changes}, strings.Join(changes, "\n"))
	}
	fmt.Fprintln(c.OutOrStdout(), "Repairs:\n"+strings.Join(changes, "\n"))
	return o.mutate(c, func(s *state.State, _ *install.Manager) error { _, _, e := o.repairs(s, true); return e })
}

func (o *options) repairs(s *state.State, apply bool) ([]string, bool, error) {
	changes := []string{}
	changed := false
	manifest, e := layout.Safe(o.root, "shims", "owned.json")
	if e != nil {
		return nil, false, e
	}
	b, e := os.ReadFile(manifest)
	if e != nil && !os.IsNotExist(e) {
		return nil, false, e
	}
	var owned []string
	if e == nil {
		if e = json.Unmarshal(b, &owned); e != nil {
			return nil, false, e
		}
	}
	wanted := map[string]bool{}
	for _, n := range shims.Names(s) {
		wanted[n] = true
	}
	manager, e := layout.Safe(o.root, "shims", "pftw.exe")
	if e != nil {
		return nil, false, e
	}
	for _, n := range owned {
		if !shims.IsName(n) {
			return nil, false, fmt.Errorf("invalid owned shim name %q", n)
		}
		p, e := layout.Safe(o.root, "shims", n)
		if e != nil {
			return nil, false, e
		}
		_, statErr := os.Lstat(p)
		if !os.IsNotExist(statErr) {
			continue
		} // Never overwrite or delete files whose identity cannot be verified.
		if !wanted[n] {
			continue
		}
		if i, err := os.Stat(manager); err != nil || !i.Mode().IsRegular() {
			continue
		}
		changes = append(changes, "Restore missing owned command "+p)
		if apply {
			source := manager
			if strings.HasPrefix(n, "pythonw") {
				source = filepath.Join(filepath.Dir(manager), "pftww.exe")
			}
			if e = os.Link(source, p); e != nil {
				return nil, false, e
			}
		}
		changed = true
	}
	old := s.Path()
	if old.Mode == "first" && !s.Portable() {
		before, e := o.store.Get()
		if e != nil {
			return nil, false, e
		}
		after := before
		for _, p := range old.Added {
			// State must name exactly a current pftw directory, not an arbitrary user path.
			valid := envpath.Equal(p, filepath.Join(o.root, "shims"))
			for k := range s.Installs {
				dir := filepath.Join(o.root, "versions", k)
				valid = valid || envpath.Equal(p, dir) || envpath.Equal(p, filepath.Join(dir, "Scripts"))
			}
			if !valid {
				continue
			}
			if !envpath.Contains(envpath.Split(after.Text), p) {
				if _, e = os.Stat(p); e != nil {
					continue
				}
				changes = append(changes, "Restore previously owned user PATH entry "+p)
				if after.Text != "" {
					after.Text += ";"
				}
				after.Text += p
				after.Exists = true
				if after.Kind == 0 {
					after.Kind = 2
				}
				changed = true
			}
		}
		if apply && after != before {
			// Append repairs so unrelated entries retain their positions and precedence.
			if e = state.Write(o.root, "path-transaction.json", map[string]any{"Default": s.Default, "Before": before, "After": after, "Path": old}); e != nil {
				return nil, false, e
			}
			current, e := o.store.Get()
			if e != nil {
				return nil, false, e
			}
			if current != before {
				return nil, false, fmt.Errorf("PATH changed while planning repairs; retry after inspection")
			}
			if e = o.store.Set(after); e != nil {
				return nil, false, e
			}
			if e = os.Remove(filepath.Join(o.root, "path-transaction.json")); e != nil {
				return nil, false, e
			}
			envpath.Notify(o.store)
		}
	}
	return changes, changed, nil
}
