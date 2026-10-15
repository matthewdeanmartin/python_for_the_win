package archive

import (
	"archive/zip"
	"os"
	"path/filepath"
	"testing"
)

func makeZip(t *testing.T, names []string, symlink bool) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "a.zip")
	f, e := os.Create(p)
	if e != nil {
		t.Fatal(e)
	}
	w := zip.NewWriter(f)
	for _, n := range names {
		h := &zip.FileHeader{Name: n, Method: zip.Deflate}
		if symlink {
			h.SetMode(os.ModeSymlink | 0777)
		}
		a, e := w.CreateHeader(h)
		if e != nil {
			t.Fatal(e)
		}
		_, _ = a.Write([]byte("hello"))
	}
	if e = w.Close(); e != nil {
		t.Fatal(e)
	}
	_ = f.Close()
	return p
}
func TestRejectUnsafeArchives(t *testing.T) {
	for _, name := range []string{"../x", "/abs", `C:\x`, `a\..\x`, "a:x", "NUL.txt", "foo/COM1.py", "foo/LPT9", "x.", "x ", "a/./b", "a//b"} {
		t.Run(name, func(t *testing.T) {
			if e := Extract(makeZip(t, []string{name}, false), t.TempDir()); e == nil {
				t.Fatal("accepted", name)
			}
		})
	}
	if e := Extract(makeZip(t, []string{"link"}, true), t.TempDir()); e == nil {
		t.Fatal("symlink accepted")
	}
	if e := Extract(makeZip(t, []string{"a", "A"}, false), t.TempDir()); e == nil {
		t.Fatal("collision accepted")
	}
}
func TestExtract(t *testing.T) {
	root := t.TempDir()
	if e := Extract(makeZip(t, []string{"Lib/site.py", "python.exe"}, false), root); e != nil {
		t.Fatal(e)
	}
	b, e := os.ReadFile(filepath.Join(root, "Lib", "site.py"))
	if e != nil || string(b) != "hello" {
		t.Fatal(e, string(b))
	}
}

func TestDeviceNamesExactMatch(t *testing.T) {
	for _, name := range []string{"COM12.txt", "LPT123.py", "COM10", "LPT0"} {
		if e := SafeName(name); e != nil {
			t.Errorf("valid name %s rejected: %v", name, e)
		}
	}
	for _, name := range []string{"COM¹.txt", "LPT².py", "COM3", "AUX.txt"} {
		if e := SafeName(name); e == nil {
			t.Errorf("device accepted: %s", name)
		}
	}
}
func FuzzSafeName(f *testing.F) {
	for _, s := range []string{"../x", "Lib/site.py", "CON", "a:b", "a/../b", "COM12.txt"} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, s string) {
		if SafeName(s) == nil && (filepath.IsAbs(s) || s == "") {
			t.Fatalf("unsafe accepted: %q", s)
		}
	})
}

func TestExtractionPreservesExistingFiles(t *testing.T) {
	root := t.TempDir()
	file := filepath.Join(root, "python.exe")
	if e := os.WriteFile(file, []byte("foreign"), 0600); e != nil {
		t.Fatal(e)
	}
	if e := Extract(makeZip(t, []string{"python.exe"}, false), root); e == nil {
		t.Fatal("overwrote existing file")
	}
	b, e := os.ReadFile(file)
	if e != nil || string(b) != "foreign" {
		t.Fatal(string(b), e)
	}
}

func TestRejectInvalidUTF8Name(t *testing.T) {
	if e := SafeName(string([]byte{0xff}) + ".py"); e == nil {
		t.Fatal("accepted a name Windows would replace during UTF-16 conversion")
	}
}
