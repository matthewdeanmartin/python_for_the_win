package shims

import (
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"testing"
	"time"

	"golang.org/x/sys/windows"
)

// The test binary doubles as the launched "Python" and its detached grandchild.
const helperRole = "PFTW_SHIM_TEST_ROLE"

func TestMain(m *testing.M) {
	switch os.Getenv(helperRole) {
	case "parent":
		child := exec.Command(os.Args[0])
		child.Env = append(os.Environ(), helperRole+"=grandchild")
		child.SysProcAttr = &syscall.SysProcAttr{CreationFlags: windows.DETACHED_PROCESS | windows.CREATE_NEW_PROCESS_GROUP}
		if child.Start() != nil {
			os.Exit(3)
		}
		os.Exit(0)
	case "grandchild":
		time.Sleep(time.Second)
		if os.WriteFile(os.Getenv("PFTW_SHIM_TEST_MARKER"), []byte("alive"), 0o600) != nil {
			os.Exit(3)
		}
		os.Exit(0)
	}
	os.Exit(m.Run())
}

func TestDetachedGrandchildOutlivesLaunchedProcess(t *testing.T) {
	marker := filepath.Join(t.TempDir(), "marker")
	t.Setenv(helperRole, "parent")
	t.Setenv("PFTW_SHIM_TEST_MARKER", marker)
	code, e := Launch(os.Args[0], nil)
	if e != nil || code != 0 {
		t.Fatal(code, e)
	}
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		if _, e = os.Stat(marker); e == nil {
			return
		}
		time.Sleep(100 * time.Millisecond)
	}
	t.Fatal("detached grandchild was killed with the launched process")
}
