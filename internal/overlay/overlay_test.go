package overlay

import (
	"archive/zip"
	"bytes"
	"io"
	"os"
	"path/filepath"
	"testing"
)

func fixture(t *testing.T, names []string) string {
	t.Helper()
	dir := t.TempDir()
	exe := filepath.Join(dir, "pftw.exe")
	_ = os.WriteFile(exe, []byte("MZ-test-prefix"), 0700)
	var archive bytes.Buffer
	z := zip.NewWriter(&archive)
	for _, name := range names {
		w, e := z.Create(name)
		if e != nil {
			t.Fatal(e)
		}
		_, _ = w.Write([]byte("payload"))
	}
	_ = z.Close()
	p := filepath.Join(dir, "bundle.zip")
	_ = os.WriteFile(p, archive.Bytes(), 0600)
	var result bytes.Buffer
	if e := Write(&result, exe, p); e != nil {
		t.Fatal(e)
	}
	out := filepath.Join(dir, "offline.exe")
	_ = os.WriteFile(out, result.Bytes(), 0700)
	return out
}
func TestReadWithoutExtractionAndStripPrefix(t *testing.T) {
	exe := fixture(t, []string{"index.json", "runtime.zip"})
	before, _ := os.ReadDir(filepath.Dir(exe))
	r, n, e := Open(Source(exe, "index.json"))
	if e != nil {
		t.Fatal(e)
	}
	b, e := io.ReadAll(r)
	r.Close()
	if e != nil || n != 7 || string(b) != "payload" {
		t.Fatal(string(b), n, e)
	}
	prefix, _, e := Base(exe)
	if e != nil {
		t.Fatal(e)
	}
	b, _ = io.ReadAll(prefix)
	prefix.Close()
	if string(b) != "MZ-test-prefix" {
		t.Fatal(string(b))
	}
	after, _ := os.ReadDir(filepath.Dir(exe))
	if len(before) != len(after) {
		t.Fatal("read extracted files")
	}
}
func TestCorruptionAndUnsafeMembersRejected(t *testing.T) {
	for _, names := range [][]string{{"../escape"}, {"index.json", "INDEX.JSON"}, {"dir/payload"}} {
		exe := fixture(t, names)
		if r, _, e := Open(Source(exe, "index.json")); e == nil {
			r.Close()
			t.Fatal(names)
		}
	}
	exe := fixture(t, []string{"index.json"})
	b, _ := os.ReadFile(exe)
	b[len("MZ-test-prefix")+5] ^= 1
	_ = os.WriteFile(exe, b, 0700)
	if r, _, e := Open(Source(exe, "index.json")); e == nil {
		r.Close()
		t.Fatal("accepted corrupt archive")
	}
	b[len(b)-40] = 255
	_ = os.WriteFile(exe, b, 0700)
	if _, e := Inspect(exe); e == nil {
		t.Fatal("accepted invalid bounds")
	}
}
func TestRelativeMembersCannotEscape(t *testing.T) {
	exe := fixture(t, []string{"index.json"})
	for _, name := range []string{"../x", "https://example.org/x", "/absolute", `C:\x`, "x?query", "x#fragment"} {
		if _, e := Relative(Source(exe, "index.json"), name); e == nil {
			t.Fatal(name)
		}
	}
}
