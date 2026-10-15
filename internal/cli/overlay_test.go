package cli

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/matthewdeanmartin/python_for_the_win/internal/envpath"
	"github.com/matthewdeanmartin/python_for_the_win/internal/errs"
	"github.com/matthewdeanmartin/python_for_the_win/internal/feed"
	"github.com/matthewdeanmartin/python_for_the_win/internal/overlay"
	"github.com/matthewdeanmartin/python_for_the_win/internal/platform"
)

func TestEmbeddedIndexReadOnlyAndIntegrityExit(t *testing.T) {
	dir := t.TempDir()
	stub := filepath.Join(dir, "stub.exe")
	_ = os.WriteFile(stub, []byte("MZstub"), 0700)
	var b bytes.Buffer
	z := zip.NewWriter(&b)
	w, _ := z.Create("index.json")
	_ = json.NewEncoder(w).Encode(feed.Index{Versions: []feed.Entry{}})
	_ = z.Close()
	archive := filepath.Join(dir, "payload.zip")
	_ = os.WriteFile(archive, b.Bytes(), 0600)
	b.Reset()
	if e := overlay.Write(&b, stub, archive); e != nil {
		t.Fatal(e)
	}
	exe := filepath.Join(dir, "offline.exe")
	_ = os.WriteFile(exe, b.Bytes(), 0700)
	root := filepath.Join(dir, "unused")
	run := func() error {
		c := NewWith(platform.Native{}, &envpath.MemStore{}, exe)
		c.SetOut(new(bytes.Buffer))
		c.SetArgs([]string{"list", "--available", "--dry-run", "--root", root})
		return c.Execute()
	}
	if e := run(); e != nil {
		t.Fatal(e)
	}
	if _, e := os.Stat(root); !os.IsNotExist(e) {
		t.Fatal("embedded preview wrote files")
	}
	bad := append([]byte{}, b.Bytes()...)
	bad[12] ^= 1
	_ = os.WriteFile(exe, bad, 0700)
	var coded *errs.Error
	if e := run(); !errors.As(e, &coded) || coded.Code != 6 {
		t.Fatal("expected integrity exit 6", e)
	}
}

func TestExecutableBundleCommand(t *testing.T) {
	dir := t.TempDir()
	stub := filepath.Join(dir, "pftw.exe")
	_ = os.WriteFile(stub, []byte("MZstub"), 0700)
	payload := filepath.Join(dir, "runtime.zip")
	data := []byte("runtime bytes")
	_ = os.WriteFile(payload, data, 0600)
	hash := fmt.Sprintf("%x", sha256.Sum256(data))
	entry := feed.Entry{Schema: 1, Company: "PythonCore", Tag: "3.15-64", Version: "3.15.0", InstallFor: []string{"3.15-64"}, URL: payload, Hash: map[string]string{"sha256": hash}}
	source := filepath.Join(dir, "index.json")
	b, _ := json.Marshal(feed.Index{Versions: []feed.Entry{entry}})
	_ = os.WriteFile(source, b, 0600)
	out := filepath.Join(dir, "offline.exe")
	c := NewWith(platform.Native{}, &envpath.MemStore{}, stub)
	c.SetOut(new(bytes.Buffer))
	c.SetArgs([]string{"bundle", "3.15-64", "--source", source, "--out", out, "--root", filepath.Join(dir, "unused")})
	if e := c.Execute(); e != nil {
		t.Fatal(e)
	}
	r, _, e := overlay.Open(overlay.Source(out, hash+".zip"))
	if e != nil {
		t.Fatal(e)
	}
	got, e := io.ReadAll(r)
	r.Close()
	if e != nil || !bytes.Equal(got, data) {
		t.Fatal("payload changed", e)
	}
	if _, e := os.Stat(filepath.Join(dir, "unused")); !os.IsNotExist(e) {
		t.Fatal("bundle wrote install root")
	}
}
