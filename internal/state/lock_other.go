//go:build !windows

package state

import (
	"fmt"
	"github.com/matthewdeanmartin/python_for_the_win/internal/layout"
	"golang.org/x/sys/unix"
	"os"
)

func Lock(root string) (func(), error) {
	p, e := layout.Safe(root, "pftw.lock")
	if e != nil {
		return nil, e
	}
	if e = os.MkdirAll(root, 0700); e != nil {
		return nil, e
	}
	f, e := os.OpenFile(p, os.O_CREATE|os.O_RDWR, 0600)
	if e != nil {
		return nil, e
	}
	if e = unix.Flock(int(f.Fd()), unix.LOCK_EX|unix.LOCK_NB); e != nil {
		f.Close()
		return nil, fmt.Errorf("another pftw is running: %w", e)
	}
	return func() { _ = unix.Flock(int(f.Fd()), unix.LOCK_UN); _ = f.Close() }, nil
}
func replace(from, to string) error { return os.Rename(from, to) }
