package install

import (
	"github.com/matthewdeanmartin/python_for_the_win/internal/layout"
	"io"
	"io/fs"
	"os"
	"path/filepath"
)

// Copy, never move, package data. Reparse points and links are rejected on both sides.
func copyPackages(root, source, target string) error {
	if _, e := os.Lstat(source); os.IsNotExist(e) {
		return nil
	}
	return filepath.WalkDir(source, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		if _, err = layout.Safe(root, rel); err != nil {
			return err
		}
		part, err := filepath.Rel(source, path)
		if err != nil {
			return err
		}
		dest := filepath.Join(target, part)
		rel, err = filepath.Rel(root, dest)
		if err != nil {
			return err
		}
		if _, err = layout.Safe(root, rel); err != nil {
			return err
		}
		if d.IsDir() {
			return os.MkdirAll(dest, 0700)
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		if !info.Mode().IsRegular() {
			return fs.ErrInvalid
		}
		in, err := os.Open(path)
		if err != nil {
			return err
		}
		defer in.Close()
		out, err := os.OpenFile(dest, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, info.Mode().Perm())
		if err != nil {
			return err
		}
		_, err = io.Copy(out, in)
		if err == nil {
			err = out.Sync()
		}
		closeErr := out.Close()
		if err != nil {
			return err
		}
		return closeErr
	})
}
