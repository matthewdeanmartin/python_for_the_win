//go:build integration && windows

package cli

import (
	"bytes"
	"encoding/json"
	"github.com/matthewdeanmartin/python_for_the_win/internal/envpath"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
)

func TestRealPatchUpgrade(t *testing.T) {
	if os.Getenv("PFTW_INTEGRATION") != "1" {
		t.Skip("set PFTW_INTEGRATION=1")
	}
	root := filepath.Join(t.TempDir(), "root")
	exe := filepath.Join(t.TempDir(), "pftw.exe")
	build := exec.Command(filepath.Join(runtime.GOROOT(), "bin", "go.exe"), "build", "-o", exe, "./cmd/pftw")
	build.Dir = filepath.Join("..", "..")
	build.Env = append(os.Environ(), "CGO_ENABLED=0")
	if b, e := build.CombinedOutput(); e != nil {
		t.Fatal(e, string(b))
	}
	store := &envpath.MemStore{}
	run := func(args ...string) {
		t.Helper()
		c := NewWith(cleanHost{}, store, exe)
		var out bytes.Buffer
		c.SetOut(&out)
		c.SetErr(&out)
		c.SetArgs(append(args, "--root", root, "--allow-admin"))
		if e := c.Execute(); e != nil {
			t.Fatal(e, out.String())
		}
		t.Log(out.String())
	}
	// Freeze the official feed snapshots so a future feed cannot erase the old patch.
	indexes := []string{}
	for _, name := range []string{"index-windows.page2.json", "index-windows.sample.json"} {
		b, e := os.ReadFile(filepath.Join("..", "..", "testdata", name))
		if e != nil {
			t.Fatal(e)
		}
		var doc map[string]any
		if e = json.Unmarshal(b, &doc); e != nil {
			t.Fatal(e)
		}
		delete(doc, "next")
		b, _ = json.Marshal(doc)
		p := filepath.Join(t.TempDir(), "index.json")
		if e = os.WriteFile(p, b, 0600); e != nil {
			t.Fatal(e)
		}
		indexes = append(indexes, p)
	}
	run("install", "3.14.7", "--path", "none", "--source", indexes[0])
	python := filepath.Join(root, "versions", "3.14-64", "python.exe")
	check := func(exe string, args ...string) {
		t.Helper()
		b, e := exec.Command(exe, args...).CombinedOutput()
		if e != nil {
			t.Fatal(e, string(b))
		}
		t.Log(string(b))
	}
	venv := filepath.Join(t.TempDir(), "existing-venv")
	check(python, "-I", "-m", "venv", "--without-pip", venv)
	marker := filepath.Join(root, "versions", "3.14-64", "Lib", "site-packages", "pftw_marker.py")
	if e := os.WriteFile(marker, []byte("value = 42"), 0600); e != nil {
		t.Fatal(e)
	}
	run("upgrade", "3.14", "--source", indexes[1])
	check(python, "-I", "-c", "import sys,pftw_marker; assert sys.version_info[:3] == (3,14,8); assert pftw_marker.value == 42; print(sys.version)")
	check(filepath.Join(root, "versions", "3.14-64", "Scripts", "pip.exe"), "--version")
	check(filepath.Join(venv, "Scripts", "python.exe"), "-I", "-c", "import sys; assert sys.prefix != sys.base_prefix; assert sys.version_info[:3] == (3,14,8); print(sys.executable)")
}
