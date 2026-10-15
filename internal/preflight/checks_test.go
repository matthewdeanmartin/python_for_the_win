package preflight

import (
	"errors"
	"github.com/matthewdeanmartin/python_for_the_win/internal/envpath"
	"github.com/matthewdeanmartin/python_for_the_win/internal/state"
	"path/filepath"
	"testing"
)

type fake struct {
	admin   bool
	free    uint64
	unknown bool
}

func (f fake) IsElevated() (bool, error) {
	if f.unknown {
		return false, errors.New("unknown")
	}
	return f.admin, nil
}
func (f fake) FreeBytes(string) (uint64, error) {
	if f.unknown {
		return 0, errors.New("unknown")
	}
	return f.free, nil
}
func (fake) OwnsConsole() (bool, error)         { return false, nil }
func (fake) IsNetworkPath(string) (bool, error) { return false, nil }
func (fake) SystemPath() (envpath.Value, error) { return envpath.Value{}, nil }
func (fake) UserPath() (envpath.Value, error)   { return envpath.Value{}, nil }
func (fake) IsAppExecLink(string) (bool, error) { return false, nil }
func TestDecisions(t *testing.T) {
	for _, tc := range []struct {
		hits       []Hit
		mode, want string
	}{{nil, "auto", "AddNoClash"}, {[]Hit{{Scope: "user", Name: "python.exe"}}, "auto", "MustChoose"}, {[]Hit{{Scope: "system", Name: "python.exe"}}, "auto", "OnlyNone"}, {[]Hit{{Scope: "system", Name: "python.exe", Alias: true}}, "auto", "AliasBlock"}, {[]Hit{{Scope: "user", Name: "python.exe", Alias: true}}, "auto", "AddNoClash"}, {[]Hit{{Scope: "system", Name: "py.exe"}}, "auto", "AddNoClash"}, {[]Hit{{Scope: "system", Name: "python.exe"}}, "none", "AddNoClash"}} {
		if got := PathDecision(tc.hits, tc.mode); got != tc.want {
			t.Fatal(got, tc.want)
		}
	}
}
func TestChecksAndUnknown(t *testing.T) {
	checks := Run(fake{admin: true, free: 1}, t.TempDir(), "", state.New(), false)
	if len(checks) != 2 || Blocking(checks) == "" {
		t.Fatal(checks)
	}
	checks = Run(fake{unknown: true}, t.TempDir(), "", state.New(), false)
	if Blocking(checks) != "" || len(checks) != 2 {
		t.Fatal(checks)
	}
}
func TestOwnershipExact(t *testing.T) {
	root := t.TempDir()
	owned := filepath.Join(root, "versions", "3.15-64")
	foreign := filepath.Join(root, "foreign")
	hits := FindPythons("", owned+";"+foreign, []string{owned}, func(p string) (bool, bool, bool) { return filepath.Base(p) == "python.exe", false, false })
	if len(hits) != 1 || hits[0].Dir != foreign {
		t.Fatal(hits)
	}
}
