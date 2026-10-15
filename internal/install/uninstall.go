package install

import (
	"fmt"
	"github.com/matthewdeanmartin/python_for_the_win/internal/fsops"
	"github.com/matthewdeanmartin/python_for_the_win/internal/layout"
	"github.com/matthewdeanmartin/python_for_the_win/internal/platform"
	"github.com/matthewdeanmartin/python_for_the_win/internal/pyver"
	"github.com/matthewdeanmartin/python_for_the_win/internal/state"
	"os"
	"path/filepath"
	"sort"
)

func (m *Manager) Uninstall(key string) error {
	s, e := state.Load(m.Root)
	if e != nil {
		return e
	}
	if _, e = s.Get(key); e != nil {
		return e
	}
	dest, e := layout.Safe(m.Root, "versions", key)
	if e != nil {
		return e
	}
	trash, e := layout.Safe(m.Root, "trash")
	if e != nil {
		return e
	}
	if e = os.MkdirAll(trash, 0700); e != nil {
		return e
	}
	backup, e := os.MkdirTemp(trash, "pftw-")
	if e != nil {
		return e
	}
	if e = os.Remove(backup); e != nil {
		return e
	}
	t := transaction{Operation: "uninstall", Key: key, Stage: filepath.Base(backup), ID: filepath.Base(backup)}
	if e = saveTransaction(m.Root, t); e != nil {
		return e
	}
	if e = fsops.Rename(dest, backup); e != nil && !os.IsNotExist(e) {
		return fmt.Errorf("close programs using Python %s and retry: %w", key, sharingError(e))
	}
	delete(s.Installs, key)
	if s.Default == key {
		s.Default = ""
		keys := []string{}
		for k := range s.Installs {
			keys = append(keys, k)
		}
		sort.Slice(keys, func(i, j int) bool {
			a, _ := s.Get(keys[i])
			b, _ := s.Get(keys[j])
			v, _ := pyver.Parse(a.Version)
			w, _ := pyver.Parse(b.Version)
			return v.Compare(w) > 0
		})
		for _, k := range keys {
			v, _ := s.Get(k)
			ver, _ := pyver.Parse(v.Version)
			if v.Arch == platform.NativeArch() && !ver.Pre() {
				s.Default = k
				break
			}
		}
	}
	if e = s.Save(m.Root); e != nil {
		re := Recover(m.Root)
		return fmt.Errorf("state save failed: %w (recovery: %v)", e, re)
	}
	m.step("Uninstalled " + key)
	return Recover(m.Root)
}
