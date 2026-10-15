package install

import (
	"github.com/matthewdeanmartin/python_for_the_win/internal/state"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestInterruptedProcessRecovery(t *testing.T) {
	if os.Getenv("PFTW_CRASH_HELPER") == "1" {
		root := os.Getenv("PFTW_CRASH_ROOT")
		_, e := state.Lock(root)
		if e != nil {
			os.Exit(80)
		}
		key := "3.15-64"
		if e = os.MkdirAll(filepath.Join(root, "versions", key), 0700); e != nil {
			os.Exit(81)
		}
		if e = saveTransaction(root, transaction{Operation: "install", Key: key, Stage: "pftw-interrupted", ID: "pftw-interrupted"}); e != nil {
			os.Exit(82)
		}
		os.Exit(77)
	}
	root := t.TempDir()
	exe, e := os.Executable()
	if e != nil {
		t.Fatal(e)
	}
	cmd := exec.Command(exe, "-test.run=^TestInterruptedProcessRecovery$")
	cmd.Env = append(os.Environ(), "PFTW_CRASH_HELPER=1", "PFTW_CRASH_ROOT="+root)
	e = cmd.Run()
	if cmd.ProcessState == nil || cmd.ProcessState.ExitCode() != 77 {
		t.Fatal("helper did not reach crash point", e)
	}
	unlock, e := state.Lock(root)
	if e != nil {
		t.Fatal("crashed process retained lock", e)
	}
	defer unlock()
	if e = Recover(root); e != nil {
		t.Fatal(e)
	}
	if _, e = os.Stat(filepath.Join(root, "versions", "3.15-64")); !os.IsNotExist(e) {
		t.Fatal("partial installation remains", e)
	}
}
