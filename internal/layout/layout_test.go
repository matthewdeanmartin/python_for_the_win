package layout

import (
	"os"
	"path/filepath"
	"testing"
)

// link creates a directory symlink, skipping where the OS forbids it (Windows without Developer Mode).
func link(t *testing.T, target, name string) {
	if e := os.Symlink(target, name); e != nil {
		t.Skip("symlinks unavailable:", e)
	}
}

func TestSafeAllowsLinksAboveRoot(t *testing.T) {
	real := t.TempDir()
	alias := filepath.Join(t.TempDir(), "alias")
	link(t, real, alias) // Like macOS /var -> /private/var above every temp directory.
	root := filepath.Join(alias, "pftw")
	if e := os.Mkdir(root, 0o700); e != nil {
		t.Fatal(e)
	}
	if _, e := Safe(root, "versions", "3.15-64"); e != nil {
		t.Fatal(e)
	}
}

func TestSafeRefusesLinksAtOrBelowRoot(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	link(t, outside, filepath.Join(root, "versions"))
	if _, e := Safe(root, "versions", "3.15-64"); e == nil {
		t.Fatal("link inside root accepted")
	}
	linkedRoot := filepath.Join(t.TempDir(), "root")
	link(t, outside, linkedRoot)
	if _, e := Safe(linkedRoot, "versions"); e == nil {
		t.Fatal("linked root accepted")
	}
}

func TestSafeRefusesEscapes(t *testing.T) {
	root := t.TempDir()
	if _, e := Safe(root, "..", "elsewhere"); e == nil {
		t.Fatal("escape accepted")
	}
	if p, e := Safe(root+string(filepath.Separator), "shims"); e != nil || p != filepath.Join(root, "shims") {
		t.Fatal(p, e)
	}
}
