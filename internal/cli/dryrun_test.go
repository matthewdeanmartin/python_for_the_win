package cli

import (
	"bytes"
	"crypto/sha256"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/matthewdeanmartin/python_for_the_win/internal/state"
)

func tree(t *testing.T, root string) map[string][32]byte {
	t.Helper()
	files := map[string][32]byte{}
	e := filepath.WalkDir(root, func(p string, d fs.DirEntry, e error) error {
		if e != nil {
			return e
		}
		if d.IsDir() {
			files[p] = [32]byte{}
			return nil
		}
		b, e := os.ReadFile(p)
		if e == nil {
			files[p] = sha256.Sum256(b)
		}
		return e
	})
	if e != nil {
		t.Fatal(e)
	}
	return files
}
func TestDryRunLeavesFilesystemUntouched(t *testing.T) {
	root := t.TempDir()
	s := state.New()
	s.Default = "3.14-64"
	_ = s.Set(s.Default, state.Install{Version: "3.14.7", Arch: "amd64", Dir: "versions/3.14-64", Pip: true})
	if e := s.Save(root); e != nil {
		t.Fatal(e)
	}
	source, e := filepath.Abs("../../testdata/index-windows.sample.json")
	if e != nil {
		t.Fatal(e)
	}
	for _, args := range [][]string{
		{"install", "3.15", "--path", "none"}, {"install", "3.15", "--target", filepath.Join(root, "plain-python")}, {"upgrade", "--all"}, {"uninstall", "--all"}, {"default", "3.14"},
		{"config", "set", "source", source}, {"config", "unset", "proxy"}, {"config", "get", "source"}, {"config", "list"},
		{"doctor"}, {"doctor", "--fix"}, {"version"}, {"list"}, {"list", "--available"},
		{"bundle", "3.15", "--out", filepath.Join(root, "offline.zip")},
	} {
		t.Run(args[0]+" "+args[len(args)-1], func(t *testing.T) {
			before := tree(t, root)
			c := New()
			c.SetOut(new(bytes.Buffer))
			c.SetErr(new(bytes.Buffer))
			c.SetArgs(append(args, "--dry-mode", "--root", root, "--source", source))
			if e := c.Execute(); e != nil {
				t.Fatal(e)
			}
			if !reflect.DeepEqual(before, tree(t, root)) {
				t.Fatal("dry run changed filesystem")
			}
		})
	}
}
func TestDryRunDoesNotCreateRoot(t *testing.T) {
	root := filepath.Join(t.TempDir(), "absent")
	source, _ := filepath.Abs("../../testdata/index-windows.sample.json")
	c := New()
	c.SetOut(new(bytes.Buffer))
	c.SetArgs([]string{"install", "3.15", "--dry-run", "--path", "none", "--root", root, "--source", source})
	if e := c.Execute(); e != nil {
		t.Fatal(e)
	}
	if _, e := os.Stat(root); !os.IsNotExist(e) {
		t.Fatal("created root", e)
	}
}

func TestRunnerOwnFlagsAreParsedBeforeCode(t *testing.T) {
	root := t.TempDir()
	interpreter := filepath.Join(root, "python.exe")
	_ = os.WriteFile(interpreter, []byte("not executable"), 0600)
	c := New()
	out := new(bytes.Buffer)
	c.SetOut(out)
	c.SetArgs([]string{"run", "--root", root, "--python", interpreter, "--dry-run", "-c", "raise Exception('must not run')", "--dry-run=false"})
	if e := c.Execute(); e != nil {
		t.Fatal(e)
	}
	if !bytes.Contains(out.Bytes(), []byte("Python was not started")) {
		t.Fatal(out.String())
	}
}
