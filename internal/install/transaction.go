package install

import (
	"encoding/json"
	"fmt"
	"github.com/matthewdeanmartin/python_for_the_win/internal/fsops"
	"github.com/matthewdeanmartin/python_for_the_win/internal/layout"
	"github.com/matthewdeanmartin/python_for_the_win/internal/state"
	"os"
	"path/filepath"
	"strings"
)

type transaction struct{ Operation, Key, Stage, ID, Phase string }

func (t transaction) Validate() error {
	if !layout.ValidKey(t.Key) || (t.Operation != "install" && t.Operation != "uninstall" && t.Operation != "upgrade") || t.ID == "" {
		return fmt.Errorf("invalid transaction; recovery data retained")
	}
	if filepath.Base(t.Stage) != t.Stage || !strings.HasPrefix(t.Stage, "pftw-") {
		return fmt.Errorf("invalid transaction path")
	}
	return nil
}

// Recover must run while holding the installation-root lock, before any cleanup.
func Recover(root string) error {
	p, e := layout.Safe(root, "transaction.json")
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
	var t transaction
	if e = json.Unmarshal(b, &t); e != nil {
		return e
	}
	if e = t.Validate(); e != nil {
		return e
	}
	s, e := state.Load(root)
	if e != nil {
		return e
	}
	dest, e := layout.Safe(root, "versions", t.Key)
	if e != nil {
		return e
	}
	if t.Operation == "upgrade" {
		backup, err := layout.Safe(root, "trash", t.Stage)
		if err != nil {
			return err
		}
		v, err := s.Get(t.Key)
		if err != nil {
			return err
		}
		if v.Transaction == t.ID {
			if _, err = os.Stat(filepath.Join(dest, "python.exe")); err != nil {
				return fmt.Errorf("committed runtime missing; backup retained: %w", err)
			}
			if err = layout.Remove(root, "trash", t.Stage); err != nil {
				return err
			}
		} else if _, err = os.Stat(backup); err == nil {
			if err = layout.Remove(root, "versions", t.Key); err != nil {
				return sharingError(err)
			}
			if err = fsops.Rename(backup, dest); err != nil {
				return sharingError(err)
			}
		} else if !os.IsNotExist(err) {
			return err
		} else if _, err = os.Stat(filepath.Join(dest, "python.exe")); err != nil {
			return fmt.Errorf("upgrade recovery missing old tree; data retained: %w", err)
		}
		if err = layout.Remove(root, "staging", t.Stage); err != nil {
			return err
		}
	} else if t.Operation == "install" {
		v, _ := s.Get(t.Key)
		if v.Transaction != t.ID {
			if _, exists := s.Installs[t.Key]; exists {
				return fmt.Errorf("transaction conflicts with installed state; data retained")
			}
			if e = layout.Remove(root, "versions", t.Key); e != nil {
				return e
			}
		}
		if e = layout.Remove(root, "staging", t.Stage); e != nil {
			return e
		}
	} else {
		trash, e := layout.Safe(root, "trash", t.Stage)
		if e != nil {
			return e
		}
		if _, exists := s.Installs[t.Key]; exists {
			if _, e = os.Stat(trash); e == nil {
				if _, e = os.Stat(dest); !os.IsNotExist(e) {
					return fmt.Errorf("cannot restore uninstall: target exists or is inaccessible")
				}
				if e = fsops.Rename(trash, dest); e != nil {
					return e
				}
			} else if !os.IsNotExist(e) {
				return e
			}
		} else {
			if e = layout.Remove(root, "trash", t.Stage); e != nil {
				return e
			}
		}
	}
	return os.Remove(p)
}
func saveTransaction(root string, t transaction) error {
	if e := t.Validate(); e != nil {
		return e
	}
	return state.Write(root, "transaction.json", t)
}

// Cleanup only follows successful recovery; staging/trash are reserved managed directories.
func Cleanup(root string) error {
	if _, e := os.Stat(filepath.Join(root, "transaction.json")); !os.IsNotExist(e) {
		return fmt.Errorf("recovery must finish before cleanup")
	}
	for _, dir := range []string{"staging", "trash"} {
		p, e := layout.Safe(root, dir)
		if e != nil {
			return e
		}
		entries, e := os.ReadDir(p)
		if os.IsNotExist(e) {
			continue
		}
		if e != nil {
			return e
		}
		for _, entry := range entries {
			if strings.HasPrefix(entry.Name(), "pftw-") || dir == "staging" && strings.HasPrefix(entry.Name(), "check-") {
				if e = layout.Remove(root, dir, entry.Name()); e != nil {
					return e
				}
			}
		}
	}
	return nil
}
