//go:build integration && windows

package cli

import (
	"bytes"
	"encoding/json"
	"github.com/matthewdeanmartin/python_for_the_win/internal/envpath"
	"github.com/matthewdeanmartin/python_for_the_win/internal/platform"
	"github.com/matthewdeanmartin/python_for_the_win/internal/state"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

type cleanHost struct{ platform.Native }

func (cleanHost) SystemPath() (envpath.Value, error) {
	return envpath.Value{Text: "", Kind: 2, Exists: true}, nil
}
func TestRealInstallLifecycle(t *testing.T) {
	if os.Getenv("PFTW_INTEGRATION") != "1" {
		t.Skip("set PFTW_INTEGRATION=1 to download official Python")
	}
	root := filepath.Join(t.TempDir(), "root")
	exe := filepath.Join(t.TempDir(), "pftw-setup.exe")
	goExe := filepath.Join(runtime.GOROOT(), "bin", "go.exe")
	build := exec.Command(goExe, "build", "-trimpath", "-ldflags=-s -w", "-o", exe, "./cmd/pftw")
	build.Dir = filepath.Join("..", "..")
	build.Env = append(os.Environ(), "CGO_ENABLED=0")
	if out, e := build.CombinedOutput(); e != nil {
		t.Fatal(e, string(out))
	}
	store := &envpath.MemStore{Value: envpath.Value{Text: `C:\example-unrelated;%USERPROFILE%\example;;`, Kind: 2, Exists: true}}
	original := store.Value
	run := func(args ...string) string {
		t.Helper()
		c := NewWith(cleanHost{}, store, exe)
		b := new(bytes.Buffer)
		c.SetOut(b)
		c.SetErr(b)
		c.SetArgs(append(args, "--root", root, "--allow-admin"))
		if e := c.Execute(); e != nil {
			t.Fatal(e, b.String())
		}
		return b.String()
	}
	run("install", "3.14", "--path", "first", "--yes")
	matches, _ := filepath.Glob(filepath.Join(root, "versions", "3.14-*", "python.exe"))
	if len(matches) != 1 {
		t.Fatal(matches)
	}
	python := matches[0]
	if out, e := exec.Command(python, "-I", "-c", "import sys,pip; print(sys.version)").CombinedOutput(); e != nil {
		t.Fatal(e, string(out))
	}
	shim := filepath.Join(root, "shims", "python3.14.exe")
	cmd := exec.Command(shim, "-c", "import json,sys; print(json.dumps(sys.argv[1:]))", "a b", `c"d`, `trailing\`, "", "λ")
	out, e := cmd.CombinedOutput()
	if e != nil {
		t.Fatal(e, string(out))
	}
	var got []string
	if e = json.Unmarshal(bytes.TrimSpace(out), &got); e != nil {
		t.Fatal(e, string(out))
	}
	want := []string{"a b", `c"d`, `trailing\`, "", "λ"}
	if len(got) != len(want) {
		t.Fatal(got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatal(got, want)
		}
	}
	cmd = exec.Command(shim, "-c", "raise SystemExit(7)")
	if e = cmd.Run(); e == nil || cmd.ProcessState.ExitCode() != 7 {
		t.Fatal("exit propagation", e)
	}
	cmd = exec.Command(shim, "-c", "import sys; print(sys.stdin.read())")
	cmd.Stdin = strings.NewReader("piped input")
	if out, e = cmd.CombinedOutput(); e != nil || !bytes.Contains(out, []byte("piped input")) {
		t.Fatal(e, string(out))
	}
	directDuration, shimDuration := time.Duration(0), time.Duration(0)
	for i := 0; i < 20; i++ {
		start := time.Now()
		if e = exec.Command(python, "-c", "pass").Run(); e != nil {
			t.Fatal(e)
		}
		directDuration += time.Since(start)
		start = time.Now()
		if e = exec.Command(shim, "-c", "pass").Run(); e != nil {
			t.Fatal(e)
		}
		shimDuration += time.Since(start)
	}
	t.Logf("20-run mean shim overhead: %v", (shimDuration-directDuration)/20)
	t.Log(run("doctor", "--json"))
	t.Log(run("list", "--json"))
	run("install", "3.15", "--path", "first", "--yes")
	run("default", "3.15")
	if !strings.Contains(envpath.Split(store.Value.Text)[0], "3.15-") {
		t.Fatal("default PATH not changed", store.Value.Text)
	}
	run("uninstall", "3.14")
	run("uninstall", "--all")
	if store.Value != original {
		t.Fatal("PATH not restored byte-for-byte", store.Value, original)
	}
	if _, e = os.Stat(root); !os.IsNotExist(e) {
		t.Fatal("manager root remains", e)
	}
	// A contained install launched from its shims must discover its own root.
	c := NewWith(cleanHost{}, store, exe)
	c.SetArgs([]string{"install", "3.15", "--portable", "--root", root, "--allow-admin", "--quiet"})
	if e = c.Execute(); e != nil {
		t.Fatal(e)
	}
	s, e := state.Load(root)
	if e != nil || !s.Portable() {
		t.Fatal(s, e)
	}
	if store.Value != original {
		t.Fatal("portable changed PATH")
	}
	cmd = exec.Command(filepath.Join(root, "shims", "pftw.exe"), "list", "--json")
	if out, e = cmd.CombinedOutput(); e != nil || !bytes.Contains(out, []byte(`"3.15-64"`)) {
		t.Fatal(e, string(out))
	}
	run("uninstall", "--all")
}
