package integration

import (
	"encoding/json"
	"fmt"
	"github.com/matthewdeanmartin/python_for_the_win/internal/envpath"
	"github.com/matthewdeanmartin/python_for_the_win/internal/layout"
	"github.com/matthewdeanmartin/python_for_the_win/internal/logx"
	"github.com/matthewdeanmartin/python_for_the_win/internal/state"
	"os"
	"path/filepath"
	"time"
)

type journal struct {
	Default       string
	Before, After envpath.Value
	Path          state.PathState
}

func Entries(root string, s *state.State) []string {
	if s.Default == "" {
		return nil
	}
	dir := filepath.Join(root, "versions", s.Default)
	return []string{dir, filepath.Join(dir, "Scripts"), filepath.Join(root, "shims")}
}
func Recover(root string, store envpath.Store) error {
	p, e := layout.Safe(root, "path-transaction.json")
	if e != nil {
		return e
	}
	b, e := os.ReadFile(p)
	if os.IsNotExist(e) {
		return nil
	}
	if e != nil {
		return e
	}
	var j journal
	if e = json.Unmarshal(b, &j); e != nil {
		return e
	}
	current, e := store.Get()
	if e != nil {
		return e
	}
	if current == j.After {
		s, e := state.Load(root)
		if e != nil {
			return e
		}
		s.Default = j.Default
		s.SetPath(j.Path)
		if e = s.Save(root); e != nil {
			return e
		}
	} else if current != j.Before {
		return fmt.Errorf("PATH changed during interrupted operation; changes preserved, inspect path-transaction.json before retrying")
	}
	return os.Remove(p)
}
func Apply(root string, s *state.State, store envpath.Store, enable bool) error {
	old := s.Path()
	if s.Portable() {
		enable = false
	}
	if !enable && old.Mode != "first" {
		return s.Save(root)
	}
	before, e := store.Get()
	if e != nil {
		return e
	}
	desired := []string{}
	if enable {
		desired = Entries(root, s)
	}
	after := before
	if after.Kind == 0 {
		after.Kind = 2
	}
	text := envpath.Replace(before.Text, old.Added, desired)
	after.Text = text
	after.Exists = true
	if len(desired) == 0 && text == old.Backup && old.BackedUp {
		after.Exists = old.OriginalExists
		after.Kind = old.OriginalKind
	}
	next := old
	next.Mode = "none"
	next.Added = nil
	if len(desired) > 0 {
		next.Mode = "first"
		for _, p := range desired {
			if envpath.Contains(old.Added, p) || !envpath.Contains(envpath.Split(before.Text), p) {
				next.Added = append(next.Added, p)
			}
		}
	}
	if !old.BackedUp && before != after {
		next.Backup = before.Text
		next.BackedUp = true
		next.OriginalExists = before.Exists
		next.OriginalKind = before.Kind
		dir, e := layout.Safe(root, "logs")
		if e != nil {
			return e
		}
		if e = os.MkdirAll(dir, 0700); e != nil {
			return e
		}
		name := filepath.Join(dir, "path-backup-"+time.Now().UTC().Format("20060102T150405.000000000")+".txt")
		if e = os.WriteFile(name, []byte(before.Text), 0600); e != nil {
			return e
		}
	}
	if before == after {
		s.SetPath(next)
		return s.Save(root)
	}
	if e = state.Write(root, "path-transaction.json", journal{Default: s.Default, Before: before, After: after, Path: next}); e != nil {
		return e
	}
	current, e := store.Get()
	if e != nil {
		return e
	}
	if current != before {
		return fmt.Errorf("PATH changed during update; retry after reviewing path-transaction.json")
	}
	logger, closeLog, e := logx.Open(root)
	if e != nil {
		return e
	}
	defer closeLog()
	logger.Printf("PATH planned before=%q after=%q", before.Text, after.Text)
	if e = store.Set(after); e != nil {
		return e
	}
	current, e = store.Get()
	if e != nil {
		return fmt.Errorf("cannot verify PATH write; recovery journal retained: %w", e)
	}
	if current != after {
		// Do not blindly restore: another process may have edited PATH since our write.
		return fmt.Errorf("PATH read-back differs from the intended value; recovery journal retained, no further PATH changes made")
	}
	logger.Print("PATH write succeeded")
	s.SetPath(next)
	if e = s.Save(root); e != nil {
		return e
	}
	if e = os.Remove(filepath.Join(root, "path-transaction.json")); e != nil {
		return e
	}
	envpath.Notify(store)
	return nil
}
