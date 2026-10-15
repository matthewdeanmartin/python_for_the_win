package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/matthewdeanmartin/python_for_the_win/internal/envpath"
	"github.com/matthewdeanmartin/python_for_the_win/internal/platform"
	"github.com/matthewdeanmartin/python_for_the_win/internal/state"
)

func TestRepairPreservesForeignAndExistingFiles(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "shims")
	_ = os.MkdirAll(dir, 0700)
	_ = os.WriteFile(filepath.Join(dir, "pftw.exe"), []byte("manager"), 0700)
	s := state.New()
	s.Default = "3.15-64"
	_ = s.Set(s.Default, state.Install{Dir: "versions/3.15-64", Arch: "amd64", Version: "3.15.0"})
	_ = state.Write(dir, "owned.json", []string{"python3.15.exe", "python3.exe"})
	foreign := filepath.Join(dir, "python3.exe")
	_ = os.WriteFile(foreign, []byte("foreign replacement"), 0700)
	o := options{root: root, info: platform.Native{}, store: &envpath.MemStore{}}
	before := tree(t, root)
	changes, _, e := o.repairs(s, false)
	if e != nil || len(changes) != 1 {
		t.Fatal(changes, e)
	}
	if len(before) != len(tree(t, root)) {
		t.Fatal("preview mutated")
	}
	if _, _, e = o.repairs(s, true); e != nil {
		t.Fatal(e)
	}
	b, _ := os.ReadFile(foreign)
	if !bytes.Equal(b, []byte("foreign replacement")) {
		t.Fatal("overwrote foreign replacement")
	}
	if _, e = os.Stat(filepath.Join(dir, "python3.15.exe")); e != nil {
		t.Fatal(e)
	}
}
