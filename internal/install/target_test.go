package install

import (
	"context"
	"github.com/matthewdeanmartin/python_for_the_win/internal/download"
	"os"
	"path/filepath"
	"testing"
)

func TestTargetIsolationAndRollback(t *testing.T) {
	for _, fail := range []bool{false, true} {
		parent := t.TempDir()
		target := filepath.Join(parent, "python")
		d, _ := download.New("", false)
		r := &fakeRunner{}
		if fail {
			r.fail = 1
		}
		m := Manager{Download: d, Runner: r}
		err := m.Target(context.Background(), payload(t), target)
		if (err != nil) != fail {
			t.Fatal(err)
		}
		files, _ := os.ReadDir(parent)
		if fail {
			if len(files) != 0 {
				t.Fatal("failed extraction left files", files)
			}
			continue
		}
		if len(files) != 1 || files[0].Name() != "python" {
			t.Fatal(files)
		}
		inside, _ := os.ReadDir(target)
		if len(inside) != 1 || inside[0].Name() != "python.exe" {
			t.Fatal("not plain extraction", inside)
		}
		if err = m.Target(context.Background(), payload(t), target); err == nil {
			t.Fatal("replaced existing directory")
		}
		b, _ := os.ReadFile(filepath.Join(target, "python.exe"))
		if string(b) != "fixture" {
			t.Fatal("changed existing target")
		}
	}
}

func TestTargetBadHashLeavesNothing(t *testing.T) {
	parent := t.TempDir()
	d, _ := download.New("", false)
	entry := payload(t)
	entry.Hash["sha256"] = "0000000000000000000000000000000000000000000000000000000000000000"
	m := Manager{Download: d, Runner: &fakeRunner{}}
	if e := m.Target(context.Background(), entry, filepath.Join(parent, "python")); e == nil {
		t.Fatal("accepted bad hash")
	}
	files, _ := os.ReadDir(parent)
	if len(files) != 0 {
		t.Fatal(files)
	}
}
