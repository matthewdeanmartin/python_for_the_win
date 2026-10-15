// Package overlay reads an offline ZIP attached to a Windows executable without
// extracting a bootstrapper, executing a temporary program, or writing during previews.
package overlay

import (
	"archive/zip"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

const Scheme = "pftw-bundle"
const magic = "PFTW-OFFLINE-v1!"
const footerSize = 64
const maxArchive = 8 << 30

type Info struct {
	Base, Size int64
	Hash       [32]byte
}

func Inspect(path string) (*Info, error) {
	f, e := os.Open(path)
	if e != nil {
		return nil, e
	}
	defer func() { _ = f.Close() }() // Read-only handle; read errors are checked above.
	stat, e := f.Stat()
	if e != nil {
		return nil, e
	}
	if stat.Size() < footerSize {
		return nil, nil
	}
	b := make([]byte, footerSize)
	if _, e = f.ReadAt(b, stat.Size()-footerSize); e != nil {
		return nil, e
	}
	if string(b[:16]) != magic {
		return nil, nil
	}
	base, size := binary.LittleEndian.Uint64(b[16:24]), binary.LittleEndian.Uint64(b[24:32])
	if base == 0 || size == 0 || size > maxArchive || base > uint64(stat.Size()-footerSize) || size != uint64(stat.Size()-footerSize)-base {
		return nil, fmt.Errorf("invalid offline executable bounds")
	}
	info := &Info{Base: int64(base), Size: int64(size)}
	copy(info.Hash[:], b[32:])
	return info, nil
}
func Source(exe, name string) string {
	return Scheme + "://" + base64.RawURLEncoding.EncodeToString([]byte(exe)) + "/" + name
}
func Parse(source string) (string, string, error) {
	u, e := url.Parse(source)
	if e != nil || u.Scheme != Scheme || u.User != nil || u.RawQuery != "" || u.Fragment != "" {
		return "", "", fmt.Errorf("invalid embedded source")
	}
	b, e := base64.RawURLEncoding.DecodeString(u.Host)
	if e != nil || !filepath.IsAbs(string(b)) {
		return "", "", fmt.Errorf("invalid embedded executable path")
	}
	name := strings.TrimPrefix(u.Path, "/")
	if name == "" || name == "." || name == ".." || strings.ContainsAny(name, "/"+string(rune(92))+":") {
		return "", "", fmt.Errorf("invalid embedded member name")
	}
	return string(b), name, nil
}
func Relative(base, ref string) (string, error) {
	exe, _, e := Parse(base)
	if e != nil {
		return "", e
	}
	source := Source(exe, ref)
	_, _, e = Parse(source)
	if e != nil {
		return "", e
	}
	return source, nil
}

type reader struct {
	io.Reader
	close func() error
}

func (r *reader) Close() error { return r.close() }
func Open(source string) (io.ReadCloser, int64, error) {
	exe, name, e := Parse(source)
	if e != nil {
		return nil, 0, e
	}
	info, e := Inspect(exe)
	if e != nil {
		return nil, 0, e
	}
	if info == nil {
		return nil, 0, fmt.Errorf("offline payload is missing")
	}
	f, e := os.Open(exe)
	if e != nil {
		return nil, 0, e
	}
	good := false
	defer func() {
		if !good {
			_ = f.Close()
		}
	}()
	h := sha256.New()
	if _, e = io.Copy(h, io.NewSectionReader(f, info.Base, info.Size)); e != nil {
		return nil, 0, e
	}
	if string(h.Sum(nil)) != string(info.Hash[:]) {
		return nil, 0, fmt.Errorf("offline archive SHA-256 mismatch")
	}
	z, e := zip.NewReader(io.NewSectionReader(f, info.Base, info.Size), info.Size)
	if e != nil {
		return nil, 0, e
	}
	if len(z.File) > 128 {
		return nil, 0, fmt.Errorf("offline archive has too many members")
	}
	seen := map[string]bool{}
	var match *zip.File
	for _, entry := range z.File {
		key := strings.ToLower(entry.Name)
		if seen[key] {
			return nil, 0, fmt.Errorf("duplicate offline member")
		}
		seen[key] = true
		if _, _, e = Parse(Source(exe, entry.Name)); e != nil || !entry.Mode().IsRegular() || entry.UncompressedSize64 > maxArchive {
			return nil, 0, fmt.Errorf("unsafe offline member")
		}
		if entry.Name == name {
			match = entry
		}
	}
	if match == nil {
		return nil, 0, os.ErrNotExist
	}
	r, e := match.Open()
	if e != nil {
		return nil, 0, e
	}
	good = true
	return &reader{Reader: r, close: func() error { return errors.Join(r.Close(), f.Close()) }}, int64(match.UncompressedSize64), nil
}

// Base opens only the executable prefix when the input is itself an offline bundle.
func Base(path string) (io.ReadCloser, int64, error) {
	info, e := Inspect(path)
	if e != nil {
		return nil, 0, e
	}
	f, e := os.Open(path)
	if e != nil {
		return nil, 0, e
	}
	stat, e := f.Stat()
	if e != nil {
		_ = f.Close() // Read-only or already-failed operation; preserve the primary error.
		return nil, 0, e
	}
	size := stat.Size()
	if info != nil {
		size = info.Base
	}
	return &reader{Reader: io.NewSectionReader(f, 0, size), close: f.Close}, size, nil
}
func Write(out io.Writer, exe, archive string) error {
	in, base, e := Base(exe)
	if e != nil {
		return e
	}
	defer func() { _ = in.Close() }() // Read-only handle; read errors are checked above.
	header := make([]byte, 2)
	if _, e = io.ReadFull(in, header); e != nil {
		return e
	}
	if string(header) != "MZ" {
		return fmt.Errorf("executable bundles require a Windows manager")
	}
	if _, e = out.Write(header); e != nil {
		return e
	}
	if _, e = io.Copy(out, in); e != nil {
		return e
	}
	z, e := os.Open(archive)
	if e != nil {
		return e
	}
	defer func() { _ = z.Close() }() // Read-only handle; read errors are checked above.
	h := sha256.New()
	size, e := io.Copy(io.MultiWriter(out, h), io.LimitReader(z, maxArchive+1))
	if e != nil {
		return e
	}
	if size > maxArchive {
		return fmt.Errorf("offline archive exceeds size limit")
	}
	footer := make([]byte, footerSize)
	copy(footer, magic)
	binary.LittleEndian.PutUint64(footer[16:], uint64(base))
	binary.LittleEndian.PutUint64(footer[24:], uint64(size))
	copy(footer[32:], h.Sum(nil))
	_, e = out.Write(footer)
	return e
}
