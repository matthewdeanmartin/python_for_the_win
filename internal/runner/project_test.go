package runner

import (
	"github.com/matthewdeanmartin/python_for_the_win/internal/state"
	"os"
	"path/filepath"
	"testing"
)

func TestProjectVersion(t *testing.T) {
	root := t.TempDir()
	child := filepath.Join(root, "child")
	os.Mkdir(child, 0700)
	os.WriteFile(filepath.Join(root, ".python-version"), []byte("# project\n3.15\n"), 0600)
	if v, e := projectVersion(child); e != nil || v != "3.15" {
		t.Fatal(v, e)
	}
	t.Chdir(child)
	for _, content := range []string{"3.14", "3.14\n3.15", "../python.exe", "", "system"} {
		os.WriteFile(".python-version", []byte(content), 0600)
		v, e := projectVersion(child)
		if content == "3.14" {
			if e != nil || v != content {
				t.Fatal(v, e)
			}
		} else if e == nil {
			t.Fatal("accepted", content)
		}
	}
	os.WriteFile(".python-version", []byte("invalid"), 0600)
	selected, e := Select(root, state.New(), "3.15", nil)
	if e == nil || selected.Reason != "flag" || selected.Spec != "3.15" {
		t.Fatal("explicit selection read invalid project file", selected, e)
	}
	os.WriteFile(".python-version", []byte("3.16"), 0600)
	selected, _ = Select(root, state.New(), "", []string{"-c", "pass"})
	if selected.Reason != ".python-version" || selected.Spec != "3.16" {
		t.Fatal(selected)
	}
	selected, _ = SelectInstalled(root, state.New(), "")
	if selected.Reason != "default" {
		t.Fatal("which used project file", selected)
	}
}

func TestProjectLosesToScriptRequirements(t *testing.T) {
	root := t.TempDir()
	t.Chdir(root)
	os.WriteFile(".python-version", []byte("invalid"), 0600)
	for _, tc := range []struct{ script, reason string }{
		{"#!/usr/bin/env python3.15\npass", "shebang"},
		{"# /// script\n# requires-python = '>=3.15'\n# ///\npass", "723"},
	} {
		os.WriteFile("script.py", []byte(tc.script), 0600)
		got, _ := Select(root, state.New(), "", []string{"script.py"})
		if got.Reason != tc.reason || got.Spec == "" {
			t.Fatal(got)
		}
	}
	os.WriteFile(".python-version", make([]byte, 4097), 0600)
	if _, e := projectVersion(root); e == nil {
		t.Fatal("accepted oversized file")
	}
}
