package cli

import (
	"encoding/json"
	"fmt"
	"github.com/matthewdeanmartin/python_for_the_win/internal/layout"
	"github.com/matthewdeanmartin/python_for_the_win/internal/shims"
	"github.com/matthewdeanmartin/python_for_the_win/internal/state"
	"github.com/spf13/cobra"
	"os"
	"path/filepath"
	"regexp"
)

// Only delete known artifacts, then remove directories non-recursively so foreign files survive.
func (o *options) removeManager(c *cobra.Command) error {
	unlock, e := state.Lock(o.root)
	if e != nil {
		return e
	}
	locked := true
	defer func() {
		if locked {
			unlock()
		}
	}()
	s, e := state.Load(o.root)
	if e != nil {
		return e
	}
	if len(s.Installs) > 0 || len(s.Path().Added) > 0 {
		return fmt.Errorf("manager cleanup refused: runtimes or PATH entries remain")
	}
	remove := func(parts ...string) error {
		p, e := layout.Safe(o.root, parts...)
		if e != nil {
			return e
		}
		e = os.Remove(p)
		if os.IsNotExist(e) {
			return nil
		}
		return e
	}
	pending := false
	for _, d := range []string{"cache", "logs"} {
		p, e := layout.Safe(o.root, d)
		if e != nil {
			return e
		}
		entries, e := os.ReadDir(p)
		if e != nil && !os.IsNotExist(e) {
			return e
		}
		pattern := `^(?:[a-f0-9]{64}\.zip|index-[a-f0-9]{12}\.json|download-[0-9]+\.part|index-[0-9]+\.tmp)$`
		if d == "logs" {
			pattern = `^(?:pftw-[0-9]{8}\.log|path-backup-[0-9T.]+\.txt)$`
		}
		re := regexp.MustCompile(pattern)
		for _, entry := range entries {
			if !entry.IsDir() && re.MatchString(entry.Name()) {
				if e = remove(d, entry.Name()); e != nil {
					return e
				}
			}
		}
		if e = remove(d); e != nil {
			pending = true
		}
	}
	var names []string
	b, e := os.ReadFile(filepath.Join(o.root, "shims", "owned.json"))
	if e == nil {
		if e = json.Unmarshal(b, &names); e != nil {
			return e
		}
		for _, n := range names {
			if !shims.IsName(n) {
				return fmt.Errorf("invalid shim ownership record")
			}
			if e = remove("shims", n); e != nil {
				pending = true
			}
		}
		if _, err := os.Stat(filepath.Join(o.root, "shims", "windowed-owned.json")); err == nil {
			for _, n := range []string{"pftww.exe", "pftww.exe.old", "pftww.exe.new", "windowed-owned.json"} {
				if e = remove("shims", n); e != nil {
					pending = true
				}
			}
		}
		for _, n := range []string{"pftw.exe", "pftw.exe.old", "pftw.exe.new"} {
			if e = remove("shims", n); e != nil {
				pending = true
			}
		}
		if !pending {
			_ = remove("shims", "owned.json")
		}
	}
	for _, d := range []string{"shims", "versions", "staging", "trash"} {
		if e = remove(d); e != nil {
			pending = true
		}
	}
	// Retain state when a running executable or foreign file requires later inspection.
	if !pending {
		if e = remove("pftw.json"); e != nil {
			return e
		}
		_ = remove("pftw.json.tmp")
	}
	unlock()
	locked = false
	if !pending {
		if e = remove("pftw.lock"); e != nil {
			pending = true
		}
		if !pending {
			if e = os.Remove(o.root); e != nil {
				pending = true
			}
		}
	}
	if pending && !o.quiet {
		fmt.Fprintln(c.ErrOrStderr(), "The root contains foreign files or files still in use; they were left in place.")
	}
	return nil
}
