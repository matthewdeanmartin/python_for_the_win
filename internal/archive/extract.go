package archive

import (
	"archive/zip"
	"fmt"
	"github.com/matthewdeanmartin/python_for_the_win/internal/download"
	"github.com/matthewdeanmartin/python_for_the_win/internal/errs"
	"github.com/matthewdeanmartin/python_for_the_win/internal/layout"
	"io"
	"os"
	"path/filepath"
	"strings"
	"unicode"
)

func SafeName(name string) error {
	n := strings.ReplaceAll(name, `\`, "/")
	if n == "" || strings.HasPrefix(n, "/") {
		return fmt.Errorf("absolute or empty archive path")
	}
	for _, s := range strings.Split(strings.TrimSuffix(n, "/"), "/") {
		if s == "" || s == "." || s == ".." || strings.TrimRight(s, " .") != s || strings.ContainsAny(s, `:<>"|?*`) {
			return fmt.Errorf("unsafe archive name %q", name)
		}
		for _, r := range s {
			if unicode.IsControl(r) {
				return fmt.Errorf("control character in archive name")
			}
		}
		base := strings.ToUpper(strings.SplitN(s, ".", 2)[0])
		if base == "CON" || base == "PRN" || base == "AUX" || base == "NUL" || base == "CONIN$" || base == "CONOUT$" {
			return fmt.Errorf("reserved archive name %q", s)
		}
		if len(base) >= 4 && (strings.HasPrefix(base, "COM") || strings.HasPrefix(base, "LPT")) && strings.Contains("123456789¹²³", base[3:]) {
			return fmt.Errorf("reserved archive name %q", s)
		}
	}
	return nil
}
func Extract(path, dest string) error {
	err := extract(path, dest)
	if err != nil {
		return errs.Wrap(6, "cannot safely extract runtime", fmt.Errorf("%w: %v", download.ErrIntegrity, err))
	}
	return nil
}
func extract(path, dest string) error {
	z, e := zip.OpenReader(path)
	if e != nil {
		return e
	}
	defer z.Close()
	if len(z.File) > 100000 {
		return fmt.Errorf("too many archive entries")
	}
	var total uint64
	seen := map[string]bool{}
	for _, f := range z.File {
		if e = SafeName(f.Name); e != nil {
			return e
		}
		if f.Mode()&os.ModeType != 0 && !f.FileInfo().IsDir() {
			return fmt.Errorf("non-regular archive entry")
		}
		if f.ExternalAttrs&0x400 != 0 {
			return fmt.Errorf("reparse archive entry")
		}
		total += f.UncompressedSize64
		if total > 4<<30 || f.UncompressedSize64 > 1<<30 {
			return fmt.Errorf("archive expansion exceeds limit")
		}
		name := strings.ReplaceAll(f.Name, `\`, "/")
		key := strings.ToLower(strings.TrimSuffix(name, "/"))
		if seen[key] {
			return fmt.Errorf("duplicate archive entry %s", name)
		}
		seen[key] = true
		target, e := layout.Safe(dest, filepath.FromSlash(name))
		if e != nil {
			return e
		}
		if f.FileInfo().IsDir() {
			if e = os.MkdirAll(target, 0700); e != nil {
				return e
			}
			continue
		}
		if e = os.MkdirAll(filepath.Dir(target), 0700); e != nil {
			return e
		}
		r, e := f.Open()
		if e != nil {
			return e
		}
		w, e := os.OpenFile(target, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600|f.Mode().Perm()&0111)
		if e != nil {
			r.Close()
			return e
		}
		_, e = io.Copy(w, io.LimitReader(r, int64(f.UncompressedSize64)+1))
		if e == nil {
			e = w.Sync()
		}
		ce := w.Close()
		r.Close()
		if e != nil {
			return e
		}
		if ce != nil {
			return ce
		}
	}
	return nil
}
