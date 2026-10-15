package install

import (
	"fmt"
	"golang.org/x/sys/windows"
	"os"
	"strings"
	"testing"
)

func TestRestartManagerNamesCurrentExecutable(t *testing.T) {
	self, e := os.Executable()
	if e != nil {
		t.Fatal(e)
	}
	names := lockingProcesses(&os.PathError{Op: "remove", Path: self, Err: windows.ERROR_SHARING_VIOLATION})
	wanted := fmt.Sprintf("PID %d", os.Getpid())
	for _, name := range names {
		if strings.Contains(name, wanted) {
			return
		}
	}
	t.Fatalf("Restart Manager did not identify this running executable: %v", names)
}
