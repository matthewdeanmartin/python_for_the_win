package cli

import (
	"bytes"
	"github.com/matthewdeanmartin/python_for_the_win/internal/platform"
	"github.com/matthewdeanmartin/python_for_the_win/internal/resolve"
	"github.com/matthewdeanmartin/python_for_the_win/internal/state"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestWhichReadOnly(t *testing.T) {
	root := t.TempDir()
	s := state.New()
	key := "3.15-" + resolve.Suffix(platform.NativeArch())
	dir := filepath.Join("versions", key)
	os.MkdirAll(filepath.Join(root, dir), 0700)
	exe := filepath.Join(root, dir, "python.exe")
	os.WriteFile(exe, nil, 0700)
	s.Default = key
	s.Set(key, state.Install{Version: "3.15.0", Dir: dir, Arch: platform.NativeArch()})
	s.Save(root)
	before := tree(t, root)
	for _, args := range [][]string{{"which"}, {"which", "3.15"}, {"which", "3.15", "--dry-mode"}} {
		c := New()
		out := new(bytes.Buffer)
		c.SetOut(out)
		c.SetArgs(append(args, "--root", root))
		if e := c.Execute(); e != nil {
			t.Fatal(e)
		}
		if strings.TrimSpace(out.String()) != exe {
			t.Fatal(out.String())
		}
	}
	if !reflect.DeepEqual(before, tree(t, root)) {
		t.Fatal("which wrote files")
	}
}
