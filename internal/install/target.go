package install

import (
	"context"
	"github.com/matthewdeanmartin/python_for_the_win/internal/archive"
	"github.com/matthewdeanmartin/python_for_the_win/internal/errs"
	"github.com/matthewdeanmartin/python_for_the_win/internal/feed"
	"github.com/matthewdeanmartin/python_for_the_win/internal/layout"
	"os"
	"path/filepath"
)

// Target extracts the verified payload without installing pip or manager metadata.
// The parent must exist. The destination is exclusively created and never adopted.
func (m *Manager) Target(ctx context.Context, entry feed.Entry, target string) error {
	target, err := filepath.Abs(target)
	if err != nil {
		return err
	}
	parent := filepath.Dir(target)
	if _, err = layout.Safe(parent, filepath.Base(target)); err != nil {
		return err
	}
	if _, err = os.Lstat(target); !os.IsNotExist(err) {
		return errs.New(3, "target already exists or is inaccessible: %s", target)
	}
	work, err := os.MkdirTemp(parent, ".pftw-target-")
	if err != nil {
		return err
	}
	defer func() { _ = layout.Remove(parent, filepath.Base(work)) }()
	payload, err := m.Download.Fetch(ctx, entry.URL, entry.Hash["sha256"], work)
	if err != nil {
		return err
	}
	if m.FreeBytes != nil {
		st, e := os.Stat(payload)
		if e != nil {
			return e
		}
		free, e := m.FreeBytes(parent)
		if e != nil {
			return e
		}
		if free < uint64(3*st.Size()+150*1024*1024) {
			return errs.New(3, "Not enough free space to extract Python")
		}
	}
	if err = ctx.Err(); err != nil {
		return err
	}
	if err = os.Mkdir(target, 0700); err != nil {
		return err
	}
	complete := false
	defer func() {
		if !complete {
			_ = layout.Remove(parent, filepath.Base(target))
		}
	}()
	if err = archive.Extract(payload, target); err != nil {
		return err
	}
	if err = m.smoke(ctx, target, entry.Version, work); err != nil {
		return err
	}
	if err = ctx.Err(); err != nil {
		return err
	}
	complete = true
	return nil
}
