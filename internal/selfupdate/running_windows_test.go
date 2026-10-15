package selfupdate

import (
	"github.com/matthewdeanmartin/python_for_the_win/internal/shims"
	"github.com/matthewdeanmartin/python_for_the_win/internal/state"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestReplaceRunningManager(t *testing.T) {
	if root := os.Getenv("PFTW_UPDATE_HELPER_ROOT"); root != "" {
		s, e := state.Load(root)
		if e != nil {
			t.Fatal(e)
		}
		if e = shims.Sync(root, filepath.Join(root, "download.exe"), s); e != nil {
			t.Fatal(e)
		}
		return
	}
	root := t.TempDir()
	self, e := os.Executable()
	if e != nil {
		t.Fatal(e)
	}
	s := state.New()
	s.Default = "3.15-64"
	if e = s.Set(s.Default, state.Install{Version: "3.15.0", Arch: "amd64", Dir: filepath.Join("versions", s.Default)}); e != nil {
		t.Fatal(e)
	}
	if e = s.Save(root); e != nil {
		t.Fatal(e)
	}
	if e = shims.Sync(root, self, s); e != nil {
		t.Fatal(e)
	}
	if e = os.WriteFile(filepath.Join(root, "download.exe"), []byte("verified replacement fixture"), 0700); e != nil {
		t.Fatal(e)
	}
	cmd := exec.Command(filepath.Join(root, "shims", "pftw.exe"), "-test.run=^TestReplaceRunningManager$")
	cmd.Env = append(os.Environ(), "PFTW_UPDATE_HELPER_ROOT="+root)
	if b, e := cmd.CombinedOutput(); e != nil {
		t.Fatal(e, string(b))
	}
	for _, name := range append(shims.Names(s), "pftw.exe") {
		b, e := os.ReadFile(filepath.Join(root, "shims", name))
		if e != nil || string(b) != "verified replacement fixture" {
			t.Fatal(name, e)
		}
	}
}
