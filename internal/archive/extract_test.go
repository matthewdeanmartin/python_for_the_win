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
