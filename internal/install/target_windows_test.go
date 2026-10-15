package install

import (
	"context"
	"fmt"
	"github.com/matthewdeanmartin/python_for_the_win/internal/download"
	"github.com/matthewdeanmartin/python_for_the_win/internal/proc"
	"golang.org/x/sys/windows"
	"path/filepath"
	"strings"
	"testing"
)

type lockingRunner struct{ handle windows.Handle }

func (r *lockingRunner) Run(_ context.Context, exe string, _ []string, _ proc.Options) (string, string, int, error) {
	p, e := windows.UTF16PtrFromString(exe)
	if e != nil {
		return "", "", 1, e
	}
	r.handle, e = windows.CreateFile(p, windows.GENERIC_READ, 0, nil, windows.OPEN_EXISTING, 0, 0)
	if e != nil {
		return "", "", 1, e
	}
	return "", "", 1, fmt.Errorf("injected smoke failure")
}
func TestTargetReportsRollbackFailure(t *testing.T) {
	parent := t.TempDir()
	d, e := download.New("", false)
	if e != nil {
		t.Fatal(e)
	}
	runner := &lockingRunner{}
	defer func() {
		if runner.handle != 0 && runner.handle != windows.InvalidHandle {
			if e := windows.CloseHandle(runner.handle); e != nil {
				t.Error(e)
			}
		}
	}()
	m := Manager{Download: d, Runner: runner}
	e = m.Target(context.Background(), payload(t), filepath.Join(parent, "python"))
	if e == nil || !strings.Contains(e.Error(), "injected smoke failure") || !strings.Contains(e.Error(), "remove incomplete target") {
		t.Fatal("lost operation or cleanup error", e)
	}
}
